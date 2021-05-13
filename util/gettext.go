package util

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Prompt struct {
	ShortPrompt string
	LongPrompt  string
	PromptWidth int
	Silence     bool
}

func (v *Prompt) String() string {
	return fmt.Sprintf("[%s]", v.ShortPrompt)
}

func (v *Prompt) Width() int {
	if v.PromptWidth == 0 {
		return 13
	}
	return v.PromptWidth
}

func runPoChecking(backCompatibleGettext string, poFile string, prompt Prompt) bool {
	var (
		msgs            []string
		ret             = true
		execProgram     string
		bannerDisplayed bool
	)

	showCheckingBanner := func(err error) {
		if !bannerDisplayed {
			bannerDisplayed = true
			if backCompatibleGettext != "" {
				if err != nil {
					log.Infof(`Checking syntax of po file for "%s" (use "%s" for backward compatible)`,
						prompt.LongPrompt, backCompatibleGettext)
				} else {
					log.Debugf(`Checking syntax of po file for "%s" (use "%s" for backward compatible)`,
						prompt.LongPrompt, backCompatibleGettext)
				}
			} else {
				if err != nil {
					log.Infof(`Checking syntax of po file for "%s"`, prompt.LongPrompt)
				} else {
					log.Debugf(`Checking syntax of po file for "%s"`, prompt.LongPrompt)
				}
			}
		}
		if err != nil {
			log.Errorf(`Fail to check "%s": %s`, poFile, err)
		}
	}

	execProgram = backCompatibleGettext
	if execProgram == "" {
		execProgram = "msgfmt"
	}
	cmd := exec.Command(execProgram,
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
		showCheckingBanner(err)
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
	if err = cmd.Wait(); err != nil {
		showCheckingBanner(err)
		ret = false
	} else {
		showCheckingBanner(nil)
	}
	for _, line := range msgs {
		if !ret {
			log.Errorf("\t%s", line)
		} else if !prompt.Silence {
			fmt.Printf("%-*s %s", prompt.Width(), prompt.String(), line)
		}
	}
	return ret
}

// CheckPoFile checks syntax of "po/xx.po"
func CheckPoFile(poFile string, prompt Prompt) bool {
	var ret = true

	ret = runPoChecking("", poFile, prompt)
	if !ret {
		return ret
	}

	if BackCompatibleGetTextDir == "" {
		return ret
	}

	// Turn off output of gettext 0.14 if verbose mode is off (default).
	if viper.GetInt("verbose") == 0 {
		prompt.Silence = true
	}
	return runPoChecking(filepath.Join(BackCompatibleGetTextDir, "msgfmt"), poFile, prompt)
}

// CheckCorePoFile checks syntax of "po/xx.po" against "po-core/core.pot"
func CheckCorePoFile(locale string, prompt Prompt) bool {
	log.Debugf(`Checking syntax of po file against %s for "%s"`, CorePot, prompt.LongPrompt)
	if !GenerateCorePot() {
		log.Errorf(`Fail to check core po file for "%s"`, prompt.LongPrompt)
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

	return runPoChecking("", fout.Name(), prompt)
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
