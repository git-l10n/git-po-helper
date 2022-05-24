package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// CheckPoFile checks syntax of "po/xx.po".
func CheckPoFile(locale, poFile string) bool {
	return CheckPoFileWithPrompt(locale, poFile, "")
}

// CheckPoFileWithPrompt checks syntax of "po/xx.po", and use specific prompt.
func CheckPoFileWithPrompt(locale, poFile string, prompt string) bool {
	var (
		ret  = true
		ok   = true
		errs []string
	)

	locale = strings.TrimSuffix(filepath.Base(locale), ".po")
	_, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Error(err)
		return false
	}
	if prompt == "" {
		prompt = fmt.Sprintf("[%s]", filepath.Join(PoDir, locale+".po"))
	}

	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		return false
	}

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
func CmdCheckPo(args ...string) bool {
	var (
		ret     = true
		poFiles []string
	)

	if len(args) == 0 {
		err := filepath.Walk("po", func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				if filepath.Ext(path) == ".po" {
					args = append(args, path)
				}
				return nil
			}
			if path == "po" {
				return nil
			}
			// skip subdir
			return filepath.SkipDir
		})
		if err != nil {
			log.Errorf("fail to walk po directory: %s", err)
			return false
		}
	}

	if len(args) == 0 {
		log.Errorf(`cannot find any ".po" files to check`)
		ret = false
	}
	for _, arg := range args {
		locale := strings.TrimSuffix(filepath.Base(arg), ".po")
		poFile := filepath.Join(PoDir, locale+".po")
		poFiles = append(poFiles, poFile)
		if !CheckPoFile(locale, poFile) {
			ret = false
		}
		if flag.Core() {
			if !CheckCorePoFile(locale) {
				ret = false
			}
		}
	}
	if flag.CheckPotFile() != flag.CheckPotFileNone {
		ret = CheckUnfinishedPoFiles("HEAD", poFiles) && ret
	}
	return ret
}
