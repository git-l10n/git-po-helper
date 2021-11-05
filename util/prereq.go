package util

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/git-l10n/git-po-helper/gettext"
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

	if len(gettext.GettextAppMap) == 0 {
		return errors.New("gettext is not installed")
	}

	return nil
}
