package util

import (
	"fmt"
	"path/filepath"
	"strings"

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
		errs []error
	)
	locale = strings.TrimSuffix(filepath.Base(locale), ".po")
	_, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Error(err)
		ret = false
		return ret
	}
	if prompt == "" {
		prompt = fmt.Sprintf("[%s]", filepath.Join(PoDir, locale+".po"))
	}

	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		ret = false
		return ret
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
	errs, typosOK := checkTyposInPoFile(poFile)
	if !typosOK {
		ret = typosOK
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
