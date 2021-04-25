package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func notesForL10nTeamLeader(locale string) {
	fmt.Printf(`
============================================================
Notes for l10n team leader:

    Since you created an initial locale file, you are likely
    to be the leader of the %s l10n team.

    You should add your team infomation in the "po/TEAMS"
    file, and make a commit for it.

    Please read the file "po/README" first to understand the
    workflow of Git l10n maintenance.
============================================================
`, locale)
}

// CmdInit implements init sub command.
func CmdInit(fileName string) error {
	locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
	potFile := filepath.Join("po", "git.pot")
	poFile := filepath.Join(GitRootDir, "po", locale+".po")
	if Exist(poFile) {
		return fmt.Errorf("fail to init, 'po/%s' is already exist", filepath.Base(poFile))
	}
	cmd := exec.Command("msginit",
		"-i",
		potFile,
		"--locale="+locale,
		"-o",
		"-")
	cmd.Dir = GitRootDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	log.Infof("Running: %s ...", strings.Join(cmd.Args, " "))
	if err = cmd.Start(); err != nil {
		return ExecError(err)
	}
	f, err := os.OpenFile(poFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(stdout)
	fixed := false
	for {
		line, err := reader.ReadString('\n')
		if !fixed && strings.Contains(line, "Project-Id-Version: PACKAGE VERSION") {
			line = strings.Replace(line, "PACKAGE VERSION", "Git", 1)
			fixed = true
		}
		_, err2 := f.WriteString(line)
		if err2 != nil {
			return err2
		}
		if err != nil {
			break
		}
	}
	if err = cmd.Wait(); err != nil {
		f.Close()
		os.Remove(poFile)
		return ExecError(err)
	}
	notesForL10nTeamLeader(locale)
	return nil
}
