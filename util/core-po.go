package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// CheckCorePoFile checks syntax of "po/xx.po" against "po-core/core.pot"
func CheckCorePoFile(locale string) bool {
	var prompt = fmt.Sprintf("[%s]", filepath.Join(PoCoreDir, locale+".po"))

	localeFullName, err := GetPrettyLocaleName(locale)
	if err != nil {
		log.Error(err)
		return false
	}

	if !genCorePot() {
		log.Errorf(`%s\tFail to check core po file for "%s"`, prompt, localeFullName)
		return false
	}

	fin, err := os.Open(filepath.Join(PoDir, locale+".po"))
	if err != nil {
		log.Errorf("%s\t%s", prompt, err)
		return false
	}

	fout, err := ioutil.TempFile("", "tmp-core-po")
	if err != nil {
		log.Errorf("%s\tfail to create tmpfile: %s", prompt, err)
		return false
	}
	defer os.Remove(fout.Name())
	_, err = io.Copy(fout, fin)
	if err != nil {
		log.Errorf("%s\tfail to copy %s/%s.po to tmpfile: %s",
			prompt, PoDir, locale, err)
		return false
	}

	cmd := exec.Command("msgmerge",
		"--add-location",
		"--backup=off",
		"-U",
		fout.Name(),
		filepath.Join(PoCoreDir, CorePot))
	if err = cmd.Run(); err != nil {
		log.Errorf("%s\tfail to update core po file: %s", prompt, err)
		ShowExecError(err)
		return false
	}

	poFile := fout.Name()
	if !Exist(poFile) {
		log.Errorf(`%s\tfail to check "%s", does not exist`, prompt, poFile)
		return false
	}

	// Run msgfmt to check syntax of a .po file
	errs, ret := checkPoSyntax(poFile)
	for _, err := range errs {
		if !ret {
			log.Errorf("%s\t%s", prompt, err)
		} else {
			log.Infof("%s\t%s", prompt, err)
		}
	}
	return ret
}

// genCorePot will generate "po-core/core.pot"
func genCorePot() bool {
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
