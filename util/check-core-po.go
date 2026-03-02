package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CheckCorePoFile checks syntax of "po/xx.po" against "po/git-core.pot"
func CheckCorePoFile(locale string) bool {
	var (
		prompt = fmt.Sprintf("[%s.po (core)]", locale)
		errs   []string
		infos  []string
	)

	defer func() {
		if len(infos) > 0 {
			reportResultMessages(infos, prompt, log.InfoLevel)
		}
		if len(errs) > 0 {
			reportResultMessages(errs, prompt, log.ErrorLevel)
		}
	}()

	_, err := GetPrettyLocaleName(locale)
	if err != nil {
		errs = append(errs, err.Error())
		return false
	}

	msgs, ok := genCorePot()
	if !ok {
		errs = append(errs, msgs...)
		return false
	}
	if len(msgs) > 0 {
		infos = append(infos, msgs...)
	}

	fin, err := os.Open(filepath.Join(PoDir, locale+".po"))
	if err != nil {
		errs = append(errs, err.Error())
		return false
	}

	fout, err := os.CreateTemp("", "tmp-core-po")
	if err != nil {
		errs = append(errs,
			fmt.Sprintf("fail to create tmpfile: %s", err))
		return false
	}
	defer os.Remove(fout.Name())
	_, err = io.Copy(fout, fin)
	if err != nil {
		errs = append(errs,
			fmt.Sprintf("fail to copy %s/%s.po to tmpfile: %s",
				PoDir, locale, err))
		return false
	}

	cmd := exec.Command("msgmerge",
		"--add-location=file",
		"--backup=off",
		"-U",
		fout.Name(),
		filepath.Join(PoDir, CorePot))
	if err = cmd.Run(); err != nil {
		errs = append(errs,
			fmt.Sprintf("fail to update core po file: %s", err))
		// ShowExecError(err)
		return false
	}

	poFile := fout.Name()
	if !Exist(poFile) {
		errs = append(errs,
			fmt.Sprintf(`fail to check "%s", does not exist`, poFile))
		return false
	}

	// Run msgfmt to check syntax of a .po file
	msgs, ret := checkPoSyntax(poFile)
	for _, msg := range msgs {
		if !ret {
			errs = append(errs, msg)
		} else {
			infos = append(infos, msg)
		}
	}
	return ret
}

// genCorePot will generate "po/git-core.pot"
func genCorePot() ([]string, bool) {
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
		msgs    []string
	)

	cmd = exec.Command("make", "-n", "po/git-core.pot")
	if err = cmd.Run(); err == nil {
		cmdArgs = []string{"make", "po/git-core.pot"}
		log.Debugf(`creating %s: %s`, corePotFile, strings.Join(cmdArgs, " "))
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err = cmd.Run(); err != nil {
			msgs = append(msgs,
				fmt.Sprintf(`fail to create "%s": %s`,
					corePotFile, err))
			return msgs, false
		} else {
			return msgs, true
		}
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
	log.Debugf(`creating %s: %s`, corePotFile, "xgettext ...")
	cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if err := cmd.Run(); err != nil {
		msgs = append(msgs,
			fmt.Sprintf(`fail to create "%s": %s`,
				corePotFile, err))
		os.Remove(corePotFile)
		return msgs, false
	}
	return msgs, true
}
