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
		ret  bool
		errs []error
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
	errs, ret = checkPoSyntax(poFile)
	for _, err := range errs {
		if !ret {
			log.Errorf("%s\t%s", prompt, err)
		} else {
			log.Printf("%s\t%s", prompt, err)
		}
	}

	// Check possible typos in a .po file.
	errs, typosOK := checkTyposInPoFile(locale, poFile)
	if !typosOK {
		ret = false
	}
	for _, err := range errs {
		if err == nil {
			if !typosOK {
				log.Error("")
			} else {
				log.Warn("")
			}
		} else {
			if !typosOK {
				log.Errorf("%s\t%s", prompt, err)
			} else {
				log.Warnf("%s\t%s", prompt, err)
			}
		}
	}

	return ret
}

// CmdCheckPo implements check-po sub command.
func CmdCheckPo(args ...string) bool {
	var (
		ret = true
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
	for _, fileName := range args {
		locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
		poFile := filepath.Join(PoDir, locale+".po")
		if !CheckPoFile(locale, poFile) {
			ret = false
		}
		if flag.Core() {
			if !CheckCorePoFile(locale) {
				ret = false
			}
		}
	}
	return ret
}
