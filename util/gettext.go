package util

import (
	"bufio"
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func CheckPoFile(poFile string) bool {
	var ret = true

	cmd := exec.Command("msgfmt",
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
		log.Errorf("fail to check '%s': %s", poFile, err)
		return false
	}
	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			ret = false
			log.Error(line)
		}
		if err != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		log.Errorf("fail to check '%s': %s", poFile, err)
		ret = false
	}
	return ret
}
