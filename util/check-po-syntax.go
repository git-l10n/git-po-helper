package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func checkPoCommentEntries(poFile string) ([]string, bool) {
	var (
		errs            []string
		ok              = true
		msgCount        = 0
		commentMsgCount = 0
	)

	f, err := os.Open(poFile)
	if err != nil {
		errs = append(errs, err.Error())
		return errs, false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid ") {
			msgCount++
		} else if strings.HasPrefix(line, "#~ msgid ") {
			commentMsgCount++
		}
	}
	if 100*commentMsgCount/msgCount > 1 {
		ok = false
		errs = append(errs, fmt.Sprintf(
			"too many obsolete entries (%d) in comments, please remove them",
			commentMsgCount))
	}
	return errs, ok
}

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

	if msgs, ok := checkPoCommentEntries(poFile); !ok {
		errs = append(errs, msgs...)
		return errs, false
	}

	return errs, true
}
