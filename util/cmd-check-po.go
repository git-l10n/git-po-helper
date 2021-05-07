package util

import (
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CmdCheckPo implements check-po sub command.
func CmdCheckPo(fileName string, checkCore bool) bool {
	var err error

	locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Error(err)
	}
	poFile := filepath.Join(PoDir, locale+".po")
	if !Exist(poFile) {
		log.Errorf(`fail to check "%s", does not exist`, poFile)
		return false
	}
	if !CheckPoFile(poFile, localeFullName) {
		return false
	}
	if checkCore && !CheckCorePoFile(locale, localeFullName) {
		return false
	}
	return true
}
