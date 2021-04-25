package util

import (
	"errors"
	"os"
	"os/exec"
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

// ExecError will try to return error message of stderr
func ExecError(err error) error {
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return errors.New(string(exitError.Stderr))
		}
	}
	return err
}
