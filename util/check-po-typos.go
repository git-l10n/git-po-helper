package util

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/git-l10n/git-po-helper/dict"
	"github.com/git-l10n/git-po-helper/flag"
)

type CheckPoEntryFunc func(string, string, string) ([]string, bool)

// checkEntriesInPoFile returns a list of messages, and a boolean which
// indicates whether the messages are errors (false) or warnings (true).
// Uses util/gettext.go ParsePoEntries to load and parse the PO file.
func checkEntriesInPoFile(locale, poFile string, fn CheckPoEntryFunc) (msgs []string, ok bool) {
	ok = true

	data, err := os.ReadFile(poFile)
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("fail to read %s: %s", poFile, err))
		return msgs, false
	}

	po, err := ParsePoEntries(data)
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("fail to parse %s: %s", poFile, err))
		return msgs, false
	}

	for _, entry := range po.Entries {
		if entry.Obsolete {
			continue
		}
		if len(entry.MsgStr) == 0 {
			// Singular entry with empty msgstr (untranslated)
			output, ignoreError := fn(locale, poUnescape(entry.MsgID), "")
			msgs = append(msgs, output...)
			if !ignoreError {
				ok = false
			}
			continue
		}
		if len(entry.MsgStr) == 1 {
			// Singular entry
			output, ignoreError := fn(locale, poUnescape(entry.MsgID), poUnescape(entry.MsgStr[0]))
			msgs = append(msgs, output...)
			if !ignoreError {
				ok = false
			}
			continue
		}
		// Plural entry
		for i := range entry.MsgStr {
			var msgID, msgStr string
			if i == 0 {
				msgID = poUnescape(entry.MsgID)
			} else {
				msgID = poUnescape(entry.MsgIDPlural)
			}
			msgStr = poUnescape(entry.MsgStr[i])
			output, ignoreError := fn(locale, msgID, msgStr)
			msgs = append(msgs, output...)
			if !ignoreError {
				ok = false
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
