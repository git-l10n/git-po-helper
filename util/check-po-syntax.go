package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/git-l10n/git-po-helper/repository"
)

func checkPoSyntax(poFile string) ([]error, bool) {
	var (
		progs []string
		errs  []error
	)

	if !Exist(poFile) {
		errs = append(errs, fmt.Errorf(`fail to check "%s", does not exist`, poFile))
		return errs, false
	}

	if DirGetText014 != "" {
		progs = append(progs, filepath.Join(DirGetText014, "msgfmt"))
	}
	execPath, err := exec.LookPath("msgfmt")
	if err == nil {
		if DirGetText014 == "" || DirGetText014 != filepath.Dir(execPath) {
			progs = append(progs, execPath)
		}
	}

	for _, prog := range progs {
		cmd := exec.Command(prog,
			"-o",
			os.DevNull,
			"--check",
			"--statistics",
			poFile)
		cmd.Dir = repository.WorkDir()
		stderr, err := cmd.StderrPipe()
		if err == nil {
			err = cmd.Start()
		}
		if err != nil {
			errs = append(errs, err)
			return errs, false
		}

		scanner := bufio.NewScanner(stderr)
		msgs := []string{}
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 0 {
				msgs = append(msgs, line)
			}
		}
		if err = cmd.Wait(); err != nil {
			for _, line := range msgs {
				errs = append(errs, errors.New(line))
			}
			errs = append(errs, fmt.Errorf("fail to check po: %s", err))
			return errs, false
		}
		// Only append one statistics line.
		if len(errs) == 0 {
			for _, line := range msgs {
				errs = append(errs, errors.New(line))
			}
		}
	}
	return errs, true
}
