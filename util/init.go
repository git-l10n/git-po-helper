// Package util provides libs for git-po-helper implementation.
package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

func notesForL10nTeamLeader(locale string) {
	fmt.Printf(`
========================================================================
Notes for l10n team leader:

    Since you created an initial locale file, you are likely to be the
    leader of the %s l10n team.

    You should add your team infomation in the "po/TEAMS" file, and
    make a commit for it.

    Please read the file "po/README" first to understand the workflow
    of Git l10n maintenance.
========================================================================
`, locale)
}

func notesForCorePoFile(locale string) {
	msg := `
========================================================================
Notes for core po file:

    To contribute a new l10n translation for Git, make a full
    translation is not a piece of cake.  A small part of "po/git.pot"
    is marked and saved in "po-core/core.pot".

    The new generated po file for locale "XX" is stored in
    "po-core/XX.po" which includes core l10n entries.

    After translate this core po file, you can merge it to
    "po/XX.po" using the following commands:

        msgcat po-core/XX.po po/XX.po -s -o /tmp/XX.po
        mv /tmp/XX.po po/XX.po
        msgmerge --add-location --backup=off -U po/XX.po po/git.pot
========================================================================
`
	msg = strings.Replace(msg, "XX", locale, -1)
	fmt.Print(msg)
}

// CmdInit implements init sub command.
func CmdInit(fileName string, onlyCore bool) bool {
	var (
		potFile        string
		poFile         string
		locale         string
		localeFullName string
		err            error
	)

	locale = strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName, err = GetPrettyLocaleName(locale)
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
	}

	if onlyCore {
		if !genCorePot() {
			return false
		}
		potFile = filepath.Join(PoCoreDir, CorePot)
		poFile = filepath.Join(PoCoreDir, locale+".po")
	} else {
		potFile = filepath.Join(PoDir, GitPot)
		poFile = filepath.Join(PoDir, locale+".po")
	}
	if Exist(poFile) {
		log.Errorf(`fail to init, "%s" is already exist`, poFile)
		return false
	}
	if !Exist(potFile) {
		log.Errorf(`fail to init, "%s" is not exist`, potFile)
		return false
	}
	cmd := exec.Command("msginit",
		"--locale="+locale,
		"--no-translator",
		"-i",
		potFile,
		"-o",
		"-")
	cmd.Dir = repository.WorkDir()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
	}
	log.Infof(`Creating .po file for "%s":`, localeFullName)
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
			log.Errorf(`fail to write "%s": %s`, poFile, err2)
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
	if onlyCore {
		notesForCorePoFile(locale)
	} else {
		notesForL10nTeamLeader(locale)
	}
	return true
}
