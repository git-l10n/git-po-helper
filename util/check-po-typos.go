package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/git-l10n/git-po-helper/dict"
	"github.com/git-l10n/git-po-helper/flag"
	"github.com/gorilla/i18n/gettext"
)

type CheckPoEntryFunc func(string, string, string) ([]string, bool)

// checkEntriesInPoFile returns a list of messages, and a boolean which
// indicates whether the messages are errors (false) or warnings (true).
func checkEntriesInPoFile(locale, poFile string, fn CheckPoEntryFunc) (msgs []string, ok bool) {
	ok = true

	// Compile mo-file from po-file
	moFile, err := os.CreateTemp("", "mofile")
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

/*
 * Some languages do not use space character to separate words, so
 * when grep keep words (ascii characters) from the translated message,
 * we need to make sure the extracted message is a proper segment of the
 * original message. E.g. in vi (Vietnamese) translation, has an entry
 * like below:
 *
 *     msgid: create/reset and checkout a branch
 *     msgstr: tạo/đặt_lại và checkout một nhánh"
 *
 * When searching keep words in msgstr, we may get a variable like
 * keep word "t_l", but by checking boundary of the word (đặt_lại),
 * we can exclude this false positive matching result.
 *
 * But for bg (Bulgarian) translations, the "<>" around the placeholders
 * are removed, we should turn off this check for bg translations.
 */
func isCorrectSentenceSegmentation(str, substr string) error {
	var (
		r    rune
		size int
	)
	idx := strings.Index(str, substr)
	if idx < 0 {
		return fmt.Errorf("substr %s not in %s", substr, str)
	}
	head := str[0:idx]
	tail := str[idx+len(substr):]
	if len(head) != 0 {
		r, size = utf8.DecodeLastRuneInString(head)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return fmt.Errorf("find leading unicode frag: %v", r)
			}
		}
	}
	if len(tail) != 0 {
		r, size = utf8.DecodeRuneInString(tail)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return fmt.Errorf("find trailing unicode frag: %v", r)
			}
		}
	}
	return nil
}

func findMismatchedVariables(locale, src, target string) []string {
	var (
		srcMap     = make(map[string]bool)
		targetMap  = make(map[string]bool)
		mismatched []string
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
		switch locale {
		case "bg":
			// Bulgarian (bg) translations removed "<>" boundary characters,
			// so we should not check boundary characters.
		case "vi", "sv":
			// For vi (Vietnamese), sv (Swedish) translations, check the
			// boundary characters for false positive matches.
			fallthrough
		default:
			if err := isCorrectSentenceSegmentation(target, key); err != nil {
				continue
			}
		}
		if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
			key = "$" + key[2:len(key)-1]
		}
		targetMap[key] = false
	}

	for key := range targetMap {
		if _, ok := srcMap[key]; ok {
			srcMap[key] = true
			targetMap[key] = true
		}
	}
	for key := range srcMap {
		if !srcMap[key] {
			mismatched = append(mismatched, key)
		}
	}
	for key := range targetMap {
		if !targetMap[key] {
			mismatched = append(mismatched, key)
		}
	}
	sort.Strings(mismatched)
	return mismatched
}
func checkTyposInPoEntry(locale, msgID, msgStr string) ([]string, bool) {
	var (
		msgs       []string
		mismatched []string
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
				if smudgeMap.Reverse {
					msgID = re.ReplaceAllString(msgID, smudgeMap.Replace)
				} else {
					msgStr = re.ReplaceAllString(msgStr, smudgeMap.Replace)
				}
			} else {
				if smudgeMap.Reverse {
					msgID = strings.Replace(
						msgID,
						smudgeMap.Pattern.(string),
						smudgeMap.Replace,
						-1)
				} else {
					msgStr = strings.Replace(
						msgStr,
						smudgeMap.Pattern.(string),
						smudgeMap.Replace,
						-1)
				}
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

	mismatched = findMismatchedVariables(locale, msgID, msgStr)
	if len(mismatched) > 0 {
		msgs = append(msgs,
			fmt.Sprintf("mismatched patterns: %s",
				strings.Join(mismatched, ", ")))
		msgs = append(msgs, fmt.Sprintf(">> msgid: %s", origMsgID))
		msgs = append(msgs, fmt.Sprintf(">> msgstr: %s", origMsgStr))
		msgs = append(msgs, "")
	}

	if flag.ReportTypos() == flag.ReportIssueError && len(msgs) > 0 {
		return msgs, false
	}

	return msgs, true
}
