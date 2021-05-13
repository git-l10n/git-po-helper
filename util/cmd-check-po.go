package util

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// CmdCheckPo implements check-po sub command.
func CmdCheckPo(args ...string) bool {
	var (
		ret       = true
		prompt    Prompt
		checkCore bool
	)

	if viper.GetBool("check--core") || viper.GetBool("check-po--core") {
		checkCore = true
	}
	if checkCore {
		prompt.PromptWidth = 18
	}

	if len(args) == 0 {
		filepath.Walk("po", func(path string, info os.FileInfo, err error) error {
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
	}

	if len(args) == 0 {
		log.Errorf(`cannot find any ".po" files to check`)
		ret = false
	}
	for _, fileName := range args {
		locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
		localeFullName, err := GetPrettyLocaleName(locale)
		if err != nil {
			log.Error(err)
			ret = false
			continue
		}
		poFile := filepath.Join(PoDir, locale+".po")
		if !Exist(poFile) {
			log.Errorf(`fail to check "%s", does not exist`, poFile)
			ret = false
			continue
		}
		prompt.LongPrompt = localeFullName
		prompt.ShortPrompt = poFile
		if !CheckPoFile(poFile, prompt) {
			ret = false
		}
		if checkCore {
			prompt.ShortPrompt = filepath.Join(PoCoreDir, locale+".po")
			if !CheckCorePoFile(locale, prompt) {
				ret = false
			}
		}
	}
	return ret
}
