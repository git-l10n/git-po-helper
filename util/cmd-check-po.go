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
	poFile := filepath.Join("po", locale+".po")
	if !Exist(filepath.Join(GitRootDir, poFile)) {
		log.Errorf("fail to check 'po/%s.po', does not exist", locale)
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
