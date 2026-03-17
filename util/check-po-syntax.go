package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

func checkPoSyntax(poFile string) ([]string, bool) {
	var errs []string

	if !Exist(poFile) {
		errs = append(errs, fmt.Sprintf(`fail to check "%s", does not exist`, poFile))
		return errs, false
	}

	msgfmt, err := exec.LookPath("msgfmt")
	if err != nil {
		errs = append(errs, "no gettext programs found")
		return errs, false
	}

	cmd := exec.Command(msgfmt,
		"-o",
		os.DevNull,
		"--check",
		"--statistics",
		poFile)
	stderr, err := cmd.StderrPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		errs = append(errs, err.Error())
		return errs, false
	}

	var msgs []string
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			msgs = append(msgs, line)
		}
	}
	if err = cmd.Wait(); err != nil {
		errs = append(errs, msgs...)
		errs = append(errs, fmt.Sprintf("fail to check po: %s", err))
		return errs, false
	}
	errs = append(errs, msgs...)

	return errs, true
}
