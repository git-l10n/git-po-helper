package util

import (
	"io/fs"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// CmdCheckPo implements check-po sub command.
func CmdCheckPo(args ...string) bool {
	var ret = true

	if len(args) == 0 {
		filepath.Walk("po", func(path string, info fs.FileInfo, err error) error {
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
		if !CheckPoFile(poFile, localeFullName) {
			ret = false
		}
		if viper.GetBool("core") && !CheckCorePoFile(locale, localeFullName) {
			ret = false
		}
	}
	return ret
}
