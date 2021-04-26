package util

import (
	"bufio"
	"io"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func checkPoFile(program string, poFile string) bool {
	var (
		msgs []string
		ret  = true
	)

	cmd := exec.Command(program,
		"-o",
		"-",
		"--check",
		"--statistics",
		poFile)
	cmd.Dir = GitRootDir
	cmd.Stdout = io.Discard
	stderr, err := cmd.StderrPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Errorf("Fail to check '%s': %s", poFile, err)
		return false
	}
	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			msgs = append(msgs, line)
		}
		if err != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		log.Errorf("Fail to check '%s': %s", poFile, err)
		ret = false
	}
	for _, line := range msgs {
		if ret {
			log.Infof("\t%s", line)
		} else {
			log.Errorf("\t%s", line)
		}
	}

	return ret
}

func CheckPoFile(poFile string, localeFullName string) bool {
	var ret = true

	log.Infof("Checking syntax of po file for '%s'", localeFullName)
	ret = checkPoFile("msgfmt", poFile)
	if !ret {
		return ret
	}

	if BackCompatibleGetTextDir == "" {
		return ret
	}
	log.Infof("Checking syntax of po file for '%s' (use gettext 0.14 for backward compatible)", localeFullName)
	return checkPoFile(filepath.Join(BackCompatibleGetTextDir, "msgfmt"), poFile)
}
