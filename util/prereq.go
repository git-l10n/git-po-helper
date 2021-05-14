package util

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// BackCompatibleGetTextDir is installed dir for gettext 0.14
var BackCompatibleGetTextDir string

func isGetTextBackCompatible(execPath string) bool {
	cmd := exec.Command(execPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	line, err := bytes.NewBuffer(output).ReadString('\n')
	if err != nil {
		return false
	}
	return strings.Contains(line, " 0.14.")
}

func getBackCompatibleGetTextDir() string {
	var getTextDir string

	if viper.GetBool("no-gettext-back-compatible") {
		return ""
	}
	execPath, err := exec.LookPath("gettext")
	if err == nil {
		if isGetTextBackCompatible(execPath) {
			return filepath.Dir(execPath)
		}
	}

	for _, rootDir := range []string{
		"/opt/gettext",
		"/usr/local/Cellar/gettext",
	} {
		filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				return nil
			}
			execPath = filepath.Join(path, "bin", "gettext")
			if fi, err := os.Stat(execPath); err == nil && fi.Mode().IsRegular() {
				if isGetTextBackCompatible(execPath) {
					getTextDir = filepath.Dir(execPath)
					return errors.New("found backward compatible gettext")
				}
			}
			if path == rootDir {
				return nil
			}
			return filepath.SkipDir
		})

		if getTextDir != "" {
			break
		}
	}

	return getTextDir
}

// CheckPrereq checks prerequisites for po-helper.
func CheckPrereq() error {
	var (
		err     error
		cmd     string
		prereqs = []string{
			"git",
			"gettext",
		}
	)

	for _, cmd = range prereqs {
		_, err = exec.LookPath(cmd)
		if err != nil {
			return fmt.Errorf("%s is not installed", cmd)
		}
	}

	BackCompatibleGetTextDir = getBackCompatibleGetTextDir()
	if BackCompatibleGetTextDir == "" {
		log.Warnln("cannot find gettext 0.14, and won't run gettext backward compatible test")
	} else {
		log.Debugf(`find backward compatible gettext at "%s"`, BackCompatibleGetTextDir)
	}
	return nil
}
