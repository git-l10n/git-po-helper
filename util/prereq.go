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
)

// DirGetText014 is installed dir for gettext 0.14
var DirGetText014 string

func isGetText014(execPath string) bool {
	cmd := exec.Command(execPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	line, err := bytes.NewBuffer(output).ReadString('\n')
	if err != nil {
		return false
	}
	return strings.Contains(line, " 0.14") || strings.Contains(line, " 0.15")
}

func getGetText014() string {
	var getTextDir string

	if FlagNoGettext14() {
		return ""
	}
	if _, ok := os.LookupEnv("NO_GETTEXT_14"); ok {
		return ""
	}
	execPath, err := exec.LookPath("gettext")
	if err == nil {
		if isGetText014(execPath) {
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
				if isGetText014(execPath) {
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

	DirGetText014 = getGetText014()
	if DirGetText014 == "" {
		if !FlagNoGettext14() {
			log.Warnln("cannot find gettext 0.14 or 0.15, and couldn't run some checks. See:")
			log.Warnf("    https://lore.kernel.org/git/874l8rwrh2.fsf@evledraar.gmail.com/")
		}
	} else {
		log.Debugf(`find backward compatible gettext at "%s"`, DirGetText014)
	}
	return nil
}
