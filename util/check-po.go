package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// checkPoMetaNewlines reads meta line by line and reports abnormal newline sequences.
// Abnormal: literal "\\n" (backslash+n) in decoded meta, which may indicate double-escape or corruption.
func checkPoMetaNewlines(po *GettextPO) ([]string, bool) {
	var errs []string
	for i, line := range po.Meta() {
		if strings.Contains(line, `\n`) {
			errs = append(errs, fmt.Sprintf("header meta line %d contains literal \\\\n (abnormal; use real newline or proper PO escape): %q", i+1, line))
		}
	}
	return errs, len(errs) == 0
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

	locale = strings.TrimSuffix(filepath.Base(locale), ".po")
	_, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Error(err)
		return false
	}
	if prompt == "" {
		prompt = fmt.Sprintf("[%s]", locale+".po")
	}

	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		return false
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
	errs, ok = checkPoMetaNewlines(po)
	ReportInfoAndErrors(errs, prompt, ok)
	ret = ret && ok

	// Run msgfmt to check syntax of a .po file
	errs, ok = checkPoSyntax(poFile)
	ReportInfoAndErrors(errs, prompt, ok)
	ret = ret && ok

	// No file locations in "po/XX.po".
	errs, ok = checkPoNoFileLocations(poFile)
	ReportInfoAndErrors(errs, prompt, ok)
	ret = ret && ok

	// Check possible typos in a .po file.
	errs, ok = checkTyposInPoFile(locale, poFile)
	ReportWarnAndErrors(errs, prompt, ok)
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
