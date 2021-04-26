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
func CmdInit(fileName string) bool {
	locale := strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
	}
	potFile := filepath.Join("po", "git.pot")
	poFile := filepath.Join(GitRootDir, "po", locale+".po")
	if Exist(poFile) {
		log.Errorf("fail to init, 'po/%s' is already exist", filepath.Base(poFile))
		return false
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
		log.Errorf("fail to init: %s", err)
		return false
	}
	log.Infof("Creating .po file for '%s':", localeFullName)
	log.Infof("\t%s ...", strings.Join(cmd.Args, " "))
	if err = cmd.Start(); err != nil {
		log.Errorf("fail to init: %s", err)
		ShowExecError(err)
		return false
	}
	f, err := os.OpenFile(poFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
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
			log.Errorf("fail to write 'po/%s.po': %s", locale, err2)
			return false
		}
		if err != nil {
			break
		}
	}
	if err = cmd.Wait(); err != nil {
		f.Close()
		os.Remove(poFile)
		log.Errorf("fail to init: %s", err)
		ShowExecError(err)
		return false
	}
	notesForL10nTeamLeader(locale)
	return true
}
