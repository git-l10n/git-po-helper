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
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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
	keepWordsPattern = regexp.MustCompile(`(` +
		`\${[a-zA-Z0-9_]+}` + // match shell variables: ${n}, ...
		`|` +
		`\$[a-zA-Z0-9_]+` + // match shell variables: $PATH, ...
		`|` +
		`\b[a-zA-Z.]+\.[a-zA-Z]+\b` + // match git config variables: color.ui, ...
		`|` +
		`\b[a-zA-Z_]+_[a-zA-Z]+\b` + // match variable names: var_name, ...
		`|` +
		`\bgit-[a-z-]+` + // match git commands: git-log, ...
		`|` +
		`\bgit [a-z]+-[a-z-]+` + // match git commands: git bisect--helper, ...
		`|` +
		`\b[a-z-]+--[a-z-]+` + // match helper commands: bisect--helper, ...
		`|` +
		`--[a-zA-Z-=]+` + // match git options: --option, --option=value, ...
		`)`)
	skipWordsPatterns = []struct {
		Pattern *regexp.Regexp
		Replace string
	}{
		{
			Pattern: regexp.MustCompile(`\b(` +
				`git-directories` +
				`|` +
				`e\.g\.?` +
				`|` +
				`i\.e\.?` +
				`|` +
				`t\.ex\.?` + // "e.g." in Swedish
				`|` +
				`p\.e\.?` + // "e.g." in Portuguese
				`|` +
				`z\.B\.?` + // "e.g." in German
				`|` +
				`v\.d\.?` + // "e.g." in Vietnamese
				`|` +
				`v\.v\.?` + // "etc." in Vietnamese
				`)\b`),
			Replace: "...",
		},
		{
			// <variable_name>
			Pattern: regexp.MustCompile(`<[^>]+>`),
			Replace: "<...>",
		},
		{
			// [variable_name]
			Pattern: regexp.MustCompile(`\[[^]]+\]`),
			Replace: "[...]",
		},
		{
			// %2$s, %3$d, %2$.*1$s, %1$0.1f
			Pattern: regexp.MustCompile(`%[0-9]+(\$\.\*[0-9]*)?\$`),
			Replace: "%...",
		},
		{
			// email: user@example.com, usuari@domini.com
			Pattern: regexp.MustCompile(`[0-9a-za-z.-]+@[0-9a-za-z-]+\.[0-9a-zA-Z.-]+`),
			Replace: "user@email",
		},
		{
			// ---
			Pattern: regexp.MustCompile(`---+`),
			Replace: "——",
		},
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
			checkTypos(string(msg.Id), string(msg.Str))
		} else {
			for i := range msg.StrPlural {
				if i == 0 {
					checkTypos(string(msg.Id), string(msg.StrPlural[i]))
				} else {
					checkTypos(string(msg.IdPlural), string(msg.StrPlural[i]))
				}
			}
		}
	}
}

func isUnicodeFragment(str, substr string) (bool, error) {
	var (
		r    rune
		size int
	)
	idx := strings.Index(str, substr)
	if idx < 0 {
		return false, fmt.Errorf("substr %s not in %s", substr, str)
	}
	head := str[0:idx]
	tail := str[idx+len(substr):]
	if len(head) != 0 {
		r, size = utf8.DecodeLastRuneInString(head)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return true, nil
			}
		}
	}
	if len(tail) != 0 {
		r, size = utf8.DecodeRuneInString(tail)
		if size > 1 {
			if !unicode.IsPunct(r) && !unicode.IsSymbol(r) && !unicode.IsSpace(r) {
				return true, nil
			}
		}
	}
	return false, nil
}

func findUnmatchVariables(src, target string) []string {
	var (
		srcMap    = make(map[string]bool)
		targetMap = make(map[string]bool)
		unmatched []string
	)

	for _, m := range keepWordsPattern.FindAllStringSubmatch(src, -1) {
		key := m[1]
		if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
			key = "$" + key[2:len(key)-1]
		}
		srcMap[key] = false
	}
	for _, m := range keepWordsPattern.FindAllStringSubmatch(target, -1) {
		key := m[1]
		if frag, err := isUnicodeFragment(target, key); err == nil && !frag {
			if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
				key = "$" + key[2:len(key)-1]
			}
			targetMap[key] = false
		}
	}

	for key := range targetMap {
		if _, ok := srcMap[key]; ok {
			srcMap[key] = true
			targetMap[key] = true
		}
	}
	for key := range srcMap {
		if !srcMap[key] {
			unmatched = append(unmatched, key)
		}
	}
	for key := range targetMap {
		if !targetMap[key] {
			unmatched = append(unmatched, key)
		}
	}
	sort.Strings(unmatched)
	return unmatched
}

func checkTypos(msgID, msgStr string) {
	var (
		unmatched  []string
		origMsgID  = msgID
		origMsgStr = msgStr
	)

	// Header entry
	if len(msgID) == 0 {
		return
	}
	// Untranslated entry
	if len(msgStr) == 0 {
		return
	}
	for _, re := range skipWordsPatterns {
		if re.Pattern.MatchString(msgID) {
			msgID = re.Pattern.ReplaceAllString(msgID, re.Replace)
		}
		if re.Pattern.MatchString(msgStr) {
			msgStr = re.Pattern.ReplaceAllString(msgStr, re.Replace)
		}
	}

	unmatched = findUnmatchVariables(msgID, msgStr)
	if len(unmatched) > 0 {
		log.Warnf("mismatch variable names: %s",
			strings.Join(unmatched, ", "))
		log.Warnf(">> msgid: %s", origMsgID)
		log.Warnf(">> msgstr: %s", origMsgStr)
		log.Warnln("")
		return
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
