// Package util provides libs for git-po-helper implementation.
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
    is marked and saved in "po/git-core.pot".

    The new generated po file for locale "XX" is stored in
    "po/XX.po" which includes core l10n entries.

    After translate this core po file, send a pull request to
    the l10n coordinator repository.

        https://github.com/git-l10n/git-po/

========================================================================
`
	msg = strings.Replace(msg, "XX", locale, -1)
	fmt.Print(msg)
}

// CmdInit implements init sub command.
func CmdInit(fileName string, onlyCore bool) bool {
	var (
		locale         string
		localeFullName string
		poFile         string
		err            error
		cmd            *exec.Cmd
		cmdArgs        []string
	)

	locale = strings.TrimSuffix(filepath.Base(fileName), ".po")
	localeFullName, err = GetPrettyLocaleName(locale)
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
	}
	poFile = fmt.Sprintf("po/%s.po", locale)
	if Exist(poFile) {
		log.Errorf(`"%s" exists already`, poFile)
		return false
	}
	cmd = exec.Command("make", "-n", "po-init", "PO_FILE="+poFile)
	if err = cmd.Run(); err != nil {
		return cmdInitObsolete(locale, localeFullName, onlyCore)
	}

	if onlyCore {
		cmdArgs = []string{"make", "po-init", "PO_FILE=" + poFile}
		log.Infof(`creating po file for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run() == nil
	}

	cmdArgs = []string{"make", "pot"}
	log.Infof(`creating pot file: %s`, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if cmd.Run() != nil {
		return false
	}

	cmdArgs = []string{"msginit",
		"--input",
		"po/git.pot",
		"--output",
		poFile,
		"--no-translator",
		"--locale",
		locale,
	}
	log.Infof(`creating po file for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if cmd.Run() != nil {
		return false
	}

	if onlyCore {
		notesForCorePoFile(locale)
	} else {
		notesForL10nTeamLeader(locale)
	}
	return true
}

func cmdInitObsolete(locale string, localeFullName string, onlyCore bool) bool {
	var (
		potFile string
		poFile  string
		err     error
		cmdArgs []string
		cmd     *exec.Cmd
	)

	poFile = filepath.Join(PoDir, locale+".po")
	potFile = filepath.Join(PoDir, GitPot)
	if onlyCore {
		if msgs, ok := genCorePot(); !ok {
			for _, msg := range msgs {
				log.Errorf(msg)
			}
			return false
		}
		potFile = filepath.Join(PoDir, CorePot)
	}
	if Exist(poFile) {
		log.Errorf(`"%s" exists already`, poFile)
		return false
	}
	if !Exist(potFile) {
		log.Errorf(`"%s" does not exist`, potFile)
		return false
	}

	cmdArgs = []string{"msginit",
		"--locale=" + locale,
		"--no-translator",
		"-i",
		potFile,
		"-o",
		"-",
	}
	log.Infof(`creating po file for "%s": %s`, localeFullName, strings.Join(cmdArgs, " "))
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("fail to init: %s", err)
		return false
	}
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
