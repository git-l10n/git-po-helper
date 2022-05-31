package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// CheckCorePoFile checks syntax of "po/xx.po" against "po/git-core.pot"
func CheckCorePoFile(locale string) bool {
	var prompt = fmt.Sprintf("[%s]", filepath.Join(PoDir, locale+".po"))

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
		filepath.Join(PoDir, CorePot))
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

// genCorePot will generate "po/git-core.pot"
func genCorePot() bool {
	var (
		corePotFile    = filepath.Join(PoDir, CorePot)
		err            error
		localizedFiles = []string{
			"builtin/checkout.c",
			"builtin/clone.c",
			"builtin/index-pack.c",
			"builtin/push.c",
			"builtin/reset.c",
			"remote.c",
			"wt-status.c",
		}
		cmdArgs []string
		cmd     *exec.Cmd
	)

	cmd = exec.Command("make", "-n", "po/git-core.pot")
	if err = cmd.Run(); err == nil {
		cmdArgs = []string{"make", "po/git-core.pot"}
		log.Infof(`creating %s: %s`, corePotFile, strings.Join(cmdArgs, " "))
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run() == nil
	}

	cmdArgs = []string{
		"xgettext",
		"--force-po",
		"--add-comments=TRANSLATORS:",
		"--package-name=Git",
		"--msgid-bugs-address",
		"Git Mailing List <git@vger.kernel.org>",
		"--language=C",
		"--keyword=_",
		"--keyword=N_",
		"--keyword='Q_:1,2'",
		"-o",
		corePotFile,
	}
	cmdArgs = append(cmdArgs, localizedFiles...)
	log.Infof(`creating %s: %s`, corePotFile, "xgettext ...")
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = repository.WorkDir()
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf(`fail to create "%s": %s`, corePotFile, err)
		os.Remove(corePotFile)
		return false
	}
	return true
}
