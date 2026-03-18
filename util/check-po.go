package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// checkPoMetaEscapeChars reads meta line by line and reports abnormal newline sequences.
// Abnormal: literal "\\n" (backslash+n) in decoded meta, which may indicate double-escape or corruption.
func checkPoMetaEscapeChars(po *GettextPO) ([]string, bool) {
	var errs []string
	for i, line := range po.Meta() {
		if strings.Contains(line, `\n`) {
			errs = append(errs, fmt.Sprintf("header meta line %d contains literal \\\\n (abnormal; use real newline or proper PO escape): %q", i+1, line))
		}
	}
	return errs, len(errs) == 0
}

// locationLineNumPattern matches a reference that contains a line number (e.g. file.c:116 or file.c:116,5).
var locationLineNumPattern = regexp.MustCompile(`:\d+`)

// checkPoLocationCommentsNoLineNumbers scans each entry's comments for location comments (#:)
// and reports any that contain line numbers. Per Git l10n convention, location comments should
// not include line numbers (use --add-location=file or --no-location).
func checkPoLocationCommentsNoLineNumbers(po *GettextPO) ([]string, bool) {
	var errs []string

	for i, e := range po.Entries {
		msgid := e.MsgID
		if len(msgid) > 30 {
			msgid = msgid[:27] + "..."
		}
		entryDesc := fmt.Sprintf("entry %d (msgid %q)", i+1, msgid)
		for _, c := range e.Comments {
			trimmed := strings.TrimSpace(c)
			if !strings.HasPrefix(trimmed, "#:") {
				continue
			}
			content := strings.TrimPrefix(trimmed, "#:")
			content = strings.TrimSpace(content)
			for _, ref := range strings.Fields(content) {
				if locationLineNumPattern.MatchString(ref) {
					errs = append(errs, fmt.Sprintf("%s: location comment contains line number (use file-only or remove): %q", entryDesc, ref))
					return errs, false
				}
			}
		}
	}

	return errs, true
}

// checkPoCompatibility reports gettext version compatibility issues:
// - msgctxt: gettext below 0.15 does not support
// - #~| (MsgCtxtPrevious, MsgIDPrevious): gettext 0.14 does not support
// - #~ msgctxt (obsolete with context): gettext 0.14 does not support
func checkPoCompatibility(po *GettextPO) ([]string, bool) {
	for i, e := range po.Entries {
		msgid := e.MsgID
		if len(msgid) > 30 {
			msgid = msgid[:27] + "..."
		}
		entryDesc := fmt.Sprintf("entry %d (msgid %q)", i+1, msgid)

		if e.MsgCtxt != nil && !e.Obsolete {
			return []string{fmt.Sprintf("%s: msgctxt not supported by gettext below 0.15", entryDesc)}, false
		}
		if e.MsgCtxtPrevious != nil || e.MsgIDPrevious != "" {
			return []string{fmt.Sprintf("%s: #~| format not supported by gettext 0.14", entryDesc)}, false
		}
		if e.Obsolete && e.MsgCtxt != nil {
			return []string{fmt.Sprintf("%s: #~ msgctxt (obsolete with context) not supported by gettext 0.14", entryDesc)}, false
		}
	}
	return nil, true
}

// checkPoNoObsoleteEntries reports error if any entry has Obsolete=true.
func checkPoNoObsoleteEntries(po *GettextPO) ([]string, bool) {
	var count int
	for _, e := range po.Entries {
		if e.Obsolete {
			count++
		}
	}
	if count > 0 {
		return []string{fmt.Sprintf("you have %d obsolete entries, please remove them", count)}, false
	}
	return nil, true
}

// CheckPoFile checks syntax of "po/xx.po".
func CheckPoFile(locale, poFile string) bool {
	return CheckPoFileWithPrompt(locale, poFile, "")
}

// CheckPoFileWithPrompt checks syntax of "po/xx.po", and use specific prompt.
func CheckPoFileWithPrompt(locale, poFile string, prompt string) bool {
	var (
		ret  = true
		ok   bool
		errs []string
	)

	if prompt == "" {
		prompt = fmt.Sprintf("[%s]", locale+".po")
	}

	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		return false
	}

	// Run msgfmt to check syntax of a .po file
	errs, ok = checkPoWithMsgfmt(poFile)
	ReportSection("Syntax check with msgfmt", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Get pretty locale name, and validate locale name.
	locale = strings.TrimSuffix(filepath.Base(locale), ".po")
	_, err := GetPrettyLocaleName(locale)
	if err != nil {
		ReportSection("Locale name", false, log.InfoLevel, prompt, err.Error())
		ret = false
	}

	poData, err := os.ReadFile(poFile)
	if err != nil {
		log.Errorf(`%s\tfail to read %q: %v`, prompt, poFile, err)
		return false
	}
	po, err := ParsePoEntries(poData)
	if err != nil {
		log.Errorf(`%s\tfail to parse %q: %v`, prompt, poFile, err)
		return false
	}

	// Check header meta for abnormal newline sequences (e.g. literal \n).
	errs, ok = checkPoMetaEscapeChars(po)
	ReportSection("Syntax of PO header meta lines", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Check that location comments (#:) do not contain line numbers.
	if flag.ReportFileLocations() != flag.ReportIssueNone {
		errs, ok = checkPoLocationCommentsNoLineNumbers(po)
		ok = ok || flag.ReportFileLocations() == flag.ReportIssueWarn
		ReportSection("Location comments (#:)", ok, log.InfoLevel, prompt, errs...)
		ret = ret && ok
	}

	// Compatibility checks: msgctxt (gettext 0.15+), #~| and #~ msgctxt (gettext 0.14+).
	errs, ok = checkPoCompatibility(po)
	ReportSection("gettext compatibility", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// No obsolete entries allowed (unless AllowObsoleteEntries, e.g. in update flow).
	if !flag.AllowObsoleteEntries() {
		errs, ok = checkPoNoObsoleteEntries(po)
		ReportSection("Obsolete #~ entries", ok, log.InfoLevel, prompt, errs...)
		ret = ret && ok
	}

	// Format check: use driver return from git-check-attr to format PO file
	errs, ok = checkPoFilterFormat(poFile)
	ReportSection("PO filter (.gitattributes)", ok, log.InfoLevel, prompt, errs...)
	ret = ret && ok

	// Check possible typos in a .po file.
	errs, ok = checkTyposInPo(locale, po)
	ReportSection("msgid/msgstr pattern check", ok, log.WarnLevel, prompt, errs...)
	ret = ret && ok

	return ret
}

// CmdCheckPo implements check-po sub command.
// Args must be non-empty. Each arg is either a directory (scan for *.po in it, no recursion)
// or a file (must have .po extension). All found/listed .po files are checked.
func CmdCheckPo(args ...string) bool {
	var (
		ret     = true
		poFiles []string
	)

	if len(args) == 0 {
		log.Errorf("no arguments given; specify .po files or directories containing them")
		return false
	}

	type checkItem struct{ locale, poFile string }
	var toCheck []checkItem

	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			log.Errorf("cannot access %q: %v", arg, err)
			ret = false
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(arg)
			if err != nil {
				log.Errorf("cannot read directory %q: %v", arg, err)
				ret = false
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if filepath.Ext(e.Name()) != ".po" {
					continue
				}
				poFile := filepath.Join(arg, e.Name())
				locale := strings.TrimSuffix(e.Name(), ".po")
				toCheck = append(toCheck, checkItem{locale: locale, poFile: poFile})
			}
			continue
		}
		// file
		if filepath.Ext(arg) != ".po" {
			log.Errorf("not a .po file: %q", arg)
			ret = false
			continue
		}
		poFile := arg
		locale := strings.TrimSuffix(filepath.Base(arg), ".po")
		toCheck = append(toCheck, checkItem{locale: locale, poFile: poFile})
	}

	if len(toCheck) == 0 {
		log.Errorf("no .po files to check (specify .po files or directories containing them)")
		return false
	}

	for _, item := range toCheck {
		poFiles = append(poFiles, item.poFile)
		if !CheckPoFile(item.locale, item.poFile) {
			ret = false
		}
		if flag.Core() {
			if !CheckCorePoFile(item.locale, item.poFile) {
				ret = false
			}
		}
	}

	// We can disable this check using "--pot-file=no".
	if flag.GetPotFileFlag() != flag.PotFileFlagNone {
		ret = CheckUnfinishedPoFiles("HEAD", poFiles) && ret
	}
	return ret
}
