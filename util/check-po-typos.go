package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/git-l10n/git-po-helper/dict"
	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/gorilla/i18n/gettext"
)

type CheckPoEntryFunc func(string, string, string) ([]string, bool)

// checkEntriesInPoFile returns a list of messages, and a boolean which
// indicates whether the messages are errors (false) or warnings (true).
func checkEntriesInPoFile(locale, poFile string, fn CheckPoEntryFunc) (msgs []string, ok bool) {
	ok = true

	// Compile mo-file from po-file
	moFile, err := ioutil.TempFile("", "mofile")
	if err != nil {
		msgs = append(msgs, err.Error())
		return
	}
	defer os.Remove(moFile.Name())
	moFile.Close()
	cmd := exec.Command("msgfmt",
		"-o",
		moFile.Name(),
		poFile)
	cmd.Dir = repository.WorkDir()
	err = cmd.Run()
	if err != nil {
		// There may be some non-fatal errors in the po-file.
		// But if the generated mo-file is empty, a fatal error occurs.
		msgs = append(msgs, fmt.Sprintf("fail to compile %s: %s", poFile, err))
	}

	f, err := os.Open(moFile.Name())
	if err != nil {
		msgs = append(msgs, "fail to generate mofile")
		return msgs, false
	}
	// Fail to compile the po-file if the generated mo-file is empty.
	fi, err := f.Stat()
	if err != nil || fi.Size() == 0 {
		msgs = append(msgs, "fail to generate mofile")
		return msgs, false
	}
	defer f.Close()

	iter := gettext.ReadMo(f)
	for {
		entry, err := iter.Next()
		if err != nil {
			if err != io.EOF {
				msgs = append(msgs, fmt.Sprintf("fail to iterator: %s", err))
			}
			break
		}
		if len(entry.StrPlural) == 0 {
			output, ignoreError := fn(locale, string(entry.Id), string(entry.Str))
			msgs = append(msgs, output...)
			if !ignoreError {
				ok = false
			}
		} else {
			for i := range entry.StrPlural {
				if i == 0 {
					output, ignoreError := fn(locale, string(entry.Id), string(entry.StrPlural[i]))
					msgs = append(msgs, output...)
					if !ignoreError {
						ok = false
					}
				} else {
					output, ignoreError := fn(locale, string(entry.IdPlural), string(entry.StrPlural[i]))
					msgs = append(msgs, output...)
					if !ignoreError {
						ok = false
					}
				}
			}
		}
	}

	return msgs, ok
}

func checkTyposInPoFile(locale, poFile string) ([]string, bool) {
	return checkEntriesInPoFile(locale, poFile, checkTyposInPoEntry)
}

func isUnicodeFragment(str, substr string) (bool, error) {
	var (
		r    rune
		size int
	)
	idx := strings.Index(str, substr)
	if idx < 0 {
		return false, fmt.Errorf("substr %s not in %s", substr, str)
	}
	head := str[0:idx]
	tail := str[idx+len(substr):]
	if len(head) != 0 {
		r, size = utf8.DecodeLastRuneInString(head)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return true, nil
			}
		}
	}
	if len(tail) != 0 {
		r, size = utf8.DecodeRuneInString(tail)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return true, nil
			}
		}
	}
	return false, nil
}

func findUnmatchVariables(src, target string) []string {
	var (
		srcMap    = make(map[string]bool)
		targetMap = make(map[string]bool)
		unmatched []string
	)

	for _, m := range dict.KeepWordsPattern.FindAllStringSubmatch(src, -1) {
		key := m[1]
		if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
			key = "$" + key[2:len(key)-1]
		}
		srcMap[key] = false
	}
	for _, m := range dict.KeepWordsPattern.FindAllStringSubmatch(target, -1) {
		key := m[1]
		if frag, err := isUnicodeFragment(target, key); err == nil && !frag {
			if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
				key = "$" + key[2:len(key)-1]
			}
			targetMap[key] = false
		}
	}

	for key := range targetMap {
		if _, ok := srcMap[key]; ok {
			srcMap[key] = true
			targetMap[key] = true
		}
	}
	for key := range srcMap {
		if !srcMap[key] {
			unmatched = append(unmatched, key)
		}
	}
	for key := range targetMap {
		if !targetMap[key] {
			unmatched = append(unmatched, key)
		}
	}
	sort.Strings(unmatched)
	return unmatched
}
func checkTyposInPoEntry(locale, msgID, msgStr string) ([]string, bool) {
	var (
		msgs       []string
		unmatched  []string
		origMsgID  = msgID
		origMsgStr = msgStr
	)

	if flag.ReportTypos() == flag.ReportIssueNone {
		return nil, true
	}

	// Header entry
	if len(msgID) == 0 {
		return nil, true
	}
	// Untranslated entry
	if len(msgStr) == 0 {
		return nil, true
	}

	if smudgeMaps, ok := dict.SmudgeMaps[locale]; ok {
		for _, smudgeMap := range smudgeMaps {
			if re, ok := smudgeMap.Pattern.(*regexp.Regexp); ok {
				msgStr = re.ReplaceAllString(msgStr, smudgeMap.Replace)
			} else {
				msgStr = strings.Replace(
					msgStr,
					smudgeMap.Pattern.(string),
					smudgeMap.Replace,
					-1)
			}
		}
	}

	for _, re := range dict.GlobalSkipPatterns {
		if re.Pattern.MatchString(msgID) {
			msgID = re.Pattern.ReplaceAllString(msgID, re.Replace)
		}
		if re.Pattern.MatchString(msgStr) {
			msgStr = re.Pattern.ReplaceAllString(msgStr, re.Replace)
		}
	}

	unmatched = findUnmatchVariables(msgID, msgStr)
	if len(unmatched) > 0 {
		msgs = append(msgs,
			fmt.Sprintf("mismatch variable names: %s",
				strings.Join(unmatched, ", ")))
		msgs = append(msgs, fmt.Sprintf(">> msgid: %s", origMsgID))
		msgs = append(msgs, fmt.Sprintf(">> msgstr: %s", origMsgStr))
		msgs = append(msgs, "")
	}

	if flag.ReportTypos() == flag.ReportIssueError && len(msgs) > 0 {
		return msgs, false
	}

	return msgs, true
}
