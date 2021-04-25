package util

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CmdUpdate implements update sub command.
func CmdUpdate(fileName string) error {
	locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
	potFile := filepath.Join("po", "git.pot")
	poFile := filepath.Join("po", locale+".po")
	if !Exist(filepath.Join(GitRootDir, poFile)) {
		return fmt.Errorf("'po/%s.po' does not exist, try to create one using init command",
			locale)
	}
	cmd := exec.Command("msgmerge",
		"--add-location",
		"--backup=off",
		"-U",
		poFile,
		potFile)
	cmd.Dir = GitRootDir
	if err := cmd.Start(); err != nil {
		return ExecError(err)
	}
	log.Infof("Running: %s ...", strings.Join(cmd.Args, " "))
	if err := cmd.Wait(); err != nil {
		return ExecError(err)
	}
	return nil
}
