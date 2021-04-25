package util

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
)

// GitRootDir is the root dir of current worktree.
var GitRootDir string

// OpenRepository will try to find root dir for current workspace.
func OpenRepository(workDir string) error {
	var (
		dir string
		err error
	)

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return errors.New(string(exitError.Stderr))
		}
		return err
	}
	dir = string(bytes.TrimSpace(out))
	if _, err := os.Stat(path.Join(dir, "po", "git.pot")); err != nil {
		return errors.New("cannot find 'po/git.pot', this command is for git project")
	}
	GitRootDir = dir
	return nil
}
