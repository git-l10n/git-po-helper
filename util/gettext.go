package util

import (
	"bufio"
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func CheckPoFile(poFile string) error {
	cmd := exec.Command("msgfmt",
		"-o",
		"-",
		"--check",
		"--statistics",
		poFile)
	cmd.Dir = GitRootDir
	cmd.Stdout = io.Discard
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadString('\n')
		log.Error(line)
		if err != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
