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
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/gorilla/i18n/gettext"
)

func checkTyposInPoFile(locale, poFile string) ([]error, bool) {
	var errs []error

	if FlagIgnoreTypos() {
		return nil, true
	}

	moFile, err := ioutil.TempFile("", "mofile")
	if err != nil {
		errs = append(errs, err)
		return errs, false
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
		errs = append(errs, fmt.Errorf("fail to compile %s: %s", poFile, err))
	}
	fi, err := os.Stat(moFile.Name())
	if err != nil || fi.Size() == 0 {
		errs = append(errs, fmt.Errorf("no mofile generated, and no scan typos"))
		return errs, false
	}
	return checkTyposInMoFile(locale, moFile.Name())
}

func checkTyposInMoFile(locale, moFile string) ([]error, bool) {
	var errs []error

	if FlagIgnoreTypos() {
		return nil, true
	}

	f, err := os.Open(moFile)
	if err != nil {
		errs = append(errs, fmt.Errorf("cannot open %s: %s", moFile, err))
		return errs, false
	}
	defer f.Close()
	iter := gettext.ReadMo(f)
	for {
		msg, err := iter.Next()
		if err != nil {
			if err != io.EOF {
				errs = append(errs, fmt.Errorf("fail to iterator: %s", err))
			}
			break
		}
		if len(msg.StrPlural) == 0 {
			errs = append(errs,
				checkTypos(locale, string(msg.Id), string(msg.Str))...)
		} else {
			for i := range msg.StrPlural {
				if i == 0 {
					errs = append(errs,
						checkTypos(locale, string(msg.Id), string(msg.StrPlural[i]))...)
				} else {
					errs = append(errs,
						checkTypos(locale, string(msg.IdPlural), string(msg.StrPlural[i]))...)
				}
			}
		}
	}
	if FlagReportTyposAsErrors() && len(errs) > 0 {
		return errs, false
	}
	return errs, true
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

func checkTypos(locale, msgID, msgStr string) (errs []error) {
	var (
		unmatched  []string
		origMsgID  = msgID
		origMsgStr = msgStr
	)

	// Header entry
	if len(msgID) == 0 {
		return
	}
	// Untranslated entry
	if len(msgStr) == 0 {
		return
	}
	for _, re := range dict.GlobalSkipPatterns {
		if re.Pattern.MatchString(msgID) {
			msgID = re.Pattern.ReplaceAllString(msgID, re.Replace)
		}
		if re.Pattern.MatchString(msgStr) {
			msgStr = re.Pattern.ReplaceAllString(msgStr, re.Replace)
		}
	}

	if smudgeMap, ok := dict.SmudgeMaps[locale]; ok {
		for k, v := range smudgeMap {
			if re, ok := k.(*regexp.Regexp); ok {
				msgStr = re.ReplaceAllString(msgStr, v)
			} else {
				msgStr = strings.Replace(msgStr, k.(string), v, -1)
			}
		}
	}

	unmatched = findUnmatchVariables(msgID, msgStr)
	if len(unmatched) > 0 {
		errs = append(errs,
			fmt.Errorf("mismatch variable names: %s",
				strings.Join(unmatched, ", ")))
		errs = append(errs, fmt.Errorf(">> msgid: %s", origMsgID))
		errs = append(errs, fmt.Errorf(">> msgstr: %s", origMsgStr))
		errs = append(errs, nil)
	}
	return
}
