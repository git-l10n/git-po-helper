package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// CmdUpdate implements update sub command.
func CmdUpdate(fileName string) bool {
	var (
		cmd            *exec.Cmd
		locale         string
		localeFullName string
		err            error
		poFile         string
		cmdArgs        []string
	)

	locale = strings.TrimSuffix(filepath.Base(fileName), ".po")
	if localeFullName, err = GetPrettyLocaleName(locale); err != nil {
		log.Errorf("fail to update: %s", err)
		return false
	}
	poFile = filepath.Join(PoDir, locale+".po")

	cmd = exec.Command("make", "-n", "po-update", "PO_FILE="+poFile)
	cmd.Dir = repository.WorkDir()
	if err = cmd.Run(); err != nil {
		return cmdUpdateObsolete(locale, localeFullName)
	}

	cmdArgs = []string{"make", "po-update", "PO_FILE=" + poFile}
	log.Infof(`updating po file for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = repository.WorkDir()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if cmd.Run() != nil {
		return false
	}
	return CheckPoFile(locale, poFile)
}

func cmdUpdateObsolete(locale, localeFullName string) bool {
	var (
		cmd             *exec.Cmd
		potFile, poFile string
		cmdArgs         []string
	)

	poFile = filepath.Join(PoDir, locale+".po")
	potFile = filepath.Join(PoDir, GitPot)
	if !Exist(potFile) {
		log.Errorf(`fail to update "%s", pot file does not exist`, poFile)
		return false
	}
	if !Exist(poFile) {
		log.Errorf(`fail to update "%s", does not exist`, poFile)
		return false
	}

	cmdArgs = []string{"msgmerge",
		"--add-location",
		"--backup=off",
		"-U",
		poFile,
		potFile,
	}
	log.Infof(`updating po file for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = repository.WorkDir()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Errorf(`fail to update "%s": %s`, poFile, err)
		return false
	}
	if err := cmd.Wait(); err != nil {
		log.Errorf(`fail to update "%s": %s`, poFile, err)
		return false
	}
	return CheckPoFile(locale, poFile)
}
