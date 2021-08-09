package util

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/i18n/gettext"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Prompt holds prompt for output of a message.
type Prompt struct {
	ShortPrompt string
	LongPrompt  string
	PromptWidth int
	Silence     bool
}

var (
	varNamePattern = regexp.MustCompile(`[<\[]?` +
		`(` +
		`\${[a-zA-Z0-9_]+}` + // match shell variables
		`|` +
		`\$[a-zA-Z0-9_]+` + // match shell variables
		`|` +
		`\b[a-zA-Z.]+\.[a-zA-Z]+\b` + // match git config variables
		`|` +
		`\b[a-zA-Z_]+_[a-zA-Z]+\b` + // match variable names
		`|` +
		`\b[a-zA-Z-]*--[a-zA-Z-]+\b` + // match git commands or options
		`)` +
		`[>\]]?`)
	varNameExcludeWords = []string{
		"e.g",
		"i.e",
		"example.com",
	}
)

func (v *Prompt) String() string {
	return fmt.Sprintf("[%s]", v.ShortPrompt)
}

// Width is the width for prompt.
func (v *Prompt) Width() int {
	if v.PromptWidth == 0 {
		return 13
	}
	return v.PromptWidth
}

func checkTyposInMoFile(moFile string) {
	if viper.GetBool("check-po--ignore-typos") || viper.GetBool("check--ignore-typos") {
		return
	}

	f, err := os.Open(moFile)
	if err != nil {
		log.Errorf("cannot open %s: %s", moFile, err)
	}
	defer f.Close()
	iter := gettext.ReadMo(f)
	for {
		msg, err := iter.Next()
		if err != nil {
			if err != io.EOF {
				log.Errorf("fail to iterator: %s\n", err)
			}
			break
		}
		if len(msg.StrPlural) == 0 {
			checkTypos(msg.Id, msg.Str)
		} else {
			for i := range msg.StrPlural {
				if i == 0 {
					checkTypos(msg.Id, msg.StrPlural[i])
				} else {
					checkTypos(msg.IdPlural, msg.StrPlural[i])
				}
			}
		}
	}
}

func checkTypos(msgID, msgStr []byte) {
	if len(msgStr) == 0 {
		return
	}

	matchesInID := varNamePattern.FindAllStringSubmatch(string(msgID), -1)
	if len(matchesInID) == 0 {
		return
	}
	matchesInStr := varNamePattern.FindAllStringSubmatch(string(msgStr), -1)
	unmatched := []string{}
	for _, m := range matchesInID {
		// Ignore exclude words
		foundExclude := false
		for _, exclude := range varNameExcludeWords {
			if m[1] == exclude {
				foundExclude = true
				break
			}
		}
		if foundExclude {
			continue
		}

		// Ignore "<var_name>" and "[var_name]" in msgid.
		if len(m[0]) == len(m[1])+2 {
			continue
		}

		found := false
		for _, mStr := range matchesInStr {
			if m[1] == mStr[1] {
				found = true
				break
			}
		}
		if found {
			continue
		}

		unmatched = append(unmatched, m[1])
	}
	if len(unmatched) > 0 {
		log.Warnf("mismatch variable names: %s",
			strings.Join(unmatched, ", "))
		log.Warnf(">> msgid: %s", msgID)
		log.Warnf(">> msgstr: %s", msgStr)
		log.Warnln("")
	}
}

func runPoChecking(poFile string, prompt Prompt, backCompatible bool) bool {
	var (
		msgs            []string
		ret             = true
		execProgram     string
		bannerDisplayed bool
	)

	if backCompatible {
		if BackCompatibleGetTextDir == "" {
			log.Errorf("cannot find gettext 0.14, and won't run gettext backward compatible test")
			return false
		}
		execProgram = filepath.Join(BackCompatibleGetTextDir, "msgfmt")
	} else {
		execProgram = "msgfmt"
	}

	showCheckingBanner := func(err error) {
		if !bannerDisplayed {
			bannerDisplayed = true
			if backCompatible {
				if err != nil {
					log.Infof(`Checking syntax of po file for "%s" (use "%s" for backward compatible)`,
						prompt.LongPrompt, execProgram)
				} else {
					log.Debugf(`Checking syntax of po file for "%s" (use "%s" for backward compatible)`,
						prompt.LongPrompt, execProgram)
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

	if execProgram == "" {
		execProgram = "msgfmt"
	}

	moFile, err := ioutil.TempFile("", "mofile")
	if err != nil {
		log.Error(err)
		return false
	}
	defer os.Remove(moFile.Name())
	moFile.Close()

	cmd := exec.Command(execProgram,
		"-o",
		moFile.Name(),
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

	// Check typos in mo file.
	if !backCompatible {
		checkTyposInMoFile(moFile.Name())
	}

	return ret
}

// CheckPoFile checks syntax of "po/xx.po"
func CheckPoFile(poFile string, prompt Prompt) bool {
	var ret = true

	ret = runPoChecking(poFile, prompt, false)
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
	return runPoChecking(poFile, prompt, true)
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

	return runPoChecking(fout.Name(), prompt, false)
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
