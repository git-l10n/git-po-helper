package util

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
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
	stderr, err := cmd.StderrPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Errorf(`Fail to check "%s": %s`, poFile, err)
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
		log.Errorf(`Fail to check "%s": %s`, poFile, err)
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

// CheckPoFile checks syntax of "po/xx.po"
func CheckPoFile(poFile string, localeFullName string) bool {
	var ret = true

	log.Infof(`Checking syntax of po file for "%s"`, localeFullName)
	ret = checkPoFile("msgfmt", poFile)
	if !ret {
		return ret
	}

	if BackCompatibleGetTextDir == "" {
		return ret
	}
	log.Infof(`Checking syntax of po file for "%s" (use gettext 0.14 for backward compatible)`, localeFullName)
	return checkPoFile(filepath.Join(BackCompatibleGetTextDir, "msgfmt"), poFile)
}

// CheckCorePoFile checks syntax of "po/xx.po" against "po-core/core.pot"
func CheckCorePoFile(locale string, localeFullName string) bool {
	log.Infof(`Checking syntax of po file against %s for "%s"`, CorePot, localeFullName)
	if !GenerateCorePot() {
		log.Errorf(`Fail to check core po file for "%s"`, localeFullName)
		return false
	}

	fin, err := os.Open(filepath.Join(PoDir, locale+".po"))
	if err != nil {
		log.Error(err)
		return false
	}

	fout, err := ioutil.TempFile("", "tmp-core-po")
	if err != nil {
		log.Errorf("Fail to create tmpfile: %s", err)
		return false
	}
	defer os.Remove(fout.Name())
	_, err = io.Copy(fout, fin)
	if err != nil {
		log.Errorf("Fail to copy %s/%s.po to tmpfile: %s", PoDir, locale, err)
		return false
	}

	cmd := exec.Command("msgmerge",
		"--add-location",
		"--backup=off",
		"-U",
		fout.Name(),
		filepath.Join(PoCoreDir, CorePot))
	if err = cmd.Run(); err != nil {
		log.Errorf("Fail to update core po file: %s", err)
		ShowExecError(err)
		return false
	}

	return checkPoFile("msgfmt", fout.Name())
}

// GenerateCorePot will generate "po-core/core.pot"
func GenerateCorePot() bool {
	var (
		corePotFile    = filepath.Join(PoCoreDir, CorePot)
		err            error
		localizedFiles = []string{
			"remote.c",
			"wt-status.c",
			"builtin/clone.c",
			"builtin/checkout.c",
			"builtin/index-pack.c",
			"builtin/push.c",
			"builtin/reset.c",
		}
	)
	if !Exist(PoCoreDir) {
		err = os.MkdirAll(PoCoreDir, 0755)
		if err != nil {
			log.Error(err)
			return false
		}
	}
	if IsFile(corePotFile) {
		log.Debugf(`"%s" is already exist, not overwrite`, corePotFile)
		return true
	}
	cmdArgs := []string{
		"xgettext",
		"--force-po",
		"--add-comments=TRANSLATORS:",
		"--from-code=UTF-8",
		"--language=C",
		"--keyword=_",
		"--keyword=N_",
		"--keyword='Q_:1,2'",
		"-o",
		corePotFile,
	}
	cmdArgs = append(cmdArgs, localizedFiles...)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = GitRootDir
	cmd.Stderr = os.Stderr
	log.Infof("Creating core pot file in %s", corePotFile)
	if err := cmd.Run(); err != nil {
		log.Errorf(`fail to create "%s": %s`, corePotFile, err)
		os.Remove(corePotFile)
		return false
	}
	return true
}
