package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CmdUpdate implements update sub command.
func CmdUpdate(fileName string) bool {
	locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName, err := GetPrettyLocaleName(locale)
	potFile := filepath.Join(PoDir, GitPot)
	poFile := filepath.Join(PoDir, locale+".po")
	if err != nil {
		log.Errorf(`fail to update "%s": %s`, poFile, err)
		return false
	}
	if !Exist(poFile) {
		log.Errorf(`fail to update "%s", does not exist`, poFile)
		return false
	}
	cmd := exec.Command("msgmerge",
		"--add-location",
		"--backup=off",
		"-U",
		poFile,
		potFile)
	cmd.Dir = GitRootDir
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Errorf(`fail to update "%s": %s`, poFile, err)
		return false
	}
	log.Infof(`Updating .po file for "%s":`, localeFullName)
	log.Infof("\t%s ...", strings.Join(cmd.Args, " "))
	if err := cmd.Wait(); err != nil {
		log.Errorf(`fail to update "%s": %s`, poFile, err)
		return false
	}
	return CheckPoFile(locale, poFile)
}
