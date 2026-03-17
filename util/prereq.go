package util

import (
	"errors"
	"fmt"
	"os/exec"
)

// CheckPrereq checks prerequisites for po-helper.
func CheckPrereq() error {
	var (
		err     error
		cmd     string
		prereqs = []string{
			"git",
		}
	)

	for _, cmd = range prereqs {
		_, err = exec.LookPath(cmd)
		if err != nil {
			return fmt.Errorf("%s is not installed", cmd)
		}
	}

	if _, err := exec.LookPath("msgfmt"); err != nil {
		return errors.New("gettext is not installed")
	}

	return nil
}
