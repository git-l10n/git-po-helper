package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/git-l10n/git-po-helper/data"
	log "github.com/sirupsen/logrus"
)

// Exist check if path is exist.
func Exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// IsFile returns true if path is exist and is a file.
func IsFile(name string) bool {
	fi, err := os.Stat(name)
	if err != nil || fi.IsDir() {
		return false
	}
	return true
}

// IsDir returns true if path is exist and is a directory.
func IsDir(name string) bool {
	fi, err := os.Stat(name)
	if err != nil || !fi.IsDir() {
		return false
	}
	return true
}

// ShowExecError will try to return error message of stderr
func ShowExecError(err error) {
	if err == nil {
		return
	}
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return
	}
	buf := bytes.NewBuffer(exitError.Stderr)
	for {
		line, eof := buf.ReadString('\n')
		if len(line) > 0 {
			log.Error(line)
		}
		if eof != nil {
			break
		}
	}
}

// GetPrettyLocaleName shows full language name and location
func GetPrettyLocaleName(locale string) (string, error) {
	var (
		langName string
		locName  string
	)
	items := strings.SplitN(locale, "_", 2)
	langName = data.GetLanguageName(items[0])
	if langName == "" {
		return "", fmt.Errorf("invalid language code for locale '%s'", locale)
	}
	if len(items) > 1 && items[1] != "" {
		locName = data.GetLocationName(items[1])
		if locName == "" {
			return "", fmt.Errorf("invalid country or location code for locale '%s'", locale)
		}
	}
	if locName != "" {
		return fmt.Sprintf("%s - %s", langName, locName), nil
	} else {
		return langName, nil
	}
}
