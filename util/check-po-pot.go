package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

var (
	PotFileURL = "https://github.com/git-l10n/pot-changes/raw/pot/master/po/git.pot"
)

func CheckUnfinishedPoFiles(commit string, poFiles []string) bool {
	var (
		ok         = true
		poTemplate string
	)

	// We can disable this check using "--pot-file=no".
	if flag.GetPotFileFlag() == flag.PotFileFlagNone {
		return true
	}

	// Update pot file.
	if poTemplate, ok = UpdatePotFile(); !ok {
		return false
	}
	if poTemplate == "" {
		poTemplate = flag.GetPotFileLocation()
	} else {
		defer os.Remove(poTemplate)
	}

	// Check po file with pot file.
	for _, fileName := range poFiles {
		prompt := ""
		poFile := fileName
		locale := strings.TrimSuffix(filepath.Base(fileName), ".po")

		// Checkout po files from revision <commit>.
		if commit != "" && commit != "HEAD" {
			tmpFile := FileRevision{
				Revision: commit,
				File:     fileName,
			}
			if err := CheckoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
				showHorizontalLine()
				log.Errorf("commit %s: fail to checkout %s of revision %s: %s",
					AbbrevCommit(commit), tmpFile.File, tmpFile.Revision, err)
				ok = false
				continue
			}
			defer os.Remove(tmpFile.Tmpfile)
			poFile = tmpFile.Tmpfile
			prompt = fmt.Sprintf("[%s@%s]",
				filepath.Join(PoDir, locale+".po"),
				AbbrevCommit(commit))
		} else {
			prompt = fmt.Sprintf("[%s]", poFile)
		}

		// Check po file with pot file for missing translations.
		msgs, ret := checkUnfinishedPoFile(poFile, poTemplate)
		if len(msgs) > 0 {
			ReportWarnAndErrors(msgs, prompt, ret)
		}
		ok = ok && ret
	}
	return ok
}

func checkUnfinishedPoFile(poFile, poTemplate string) ([]string, bool) {
	const (
		kindMissing = iota
		kindFuzzy
		kindUntrans
		kindUnused
	)
	var (
		errs       []string
		ok         = true
		patternMap = make(map[int]*regexp.Regexp)
		countMap   = make(map[int]int)
		msgMap     = make(map[int][]string)
	)

	patternMap[kindMissing] = regexp.MustCompile(`[0-9]+: this message is used but not defined in .*`)
	patternMap[kindFuzzy] = regexp.MustCompile(`[0-9]+: this message needs to be reviewed by the translator`)
	patternMap[kindUntrans] = regexp.MustCompile(`[0-9]+: this message is untranslated`)
	patternMap[kindUnused] = regexp.MustCompile(`[0-9]+: warning: this message is not used`)
	countMap[kindMissing] = 0
	countMap[kindFuzzy] = 0
	countMap[kindUntrans] = 0
	countMap[kindUnused] = 0
	msgMap[kindMissing] = make([]string, 0)
	msgMap[kindFuzzy] = make([]string, 0)
	msgMap[kindUntrans] = make([]string, 0)
	msgMap[kindUnused] = make([]string, 0)

	// Run msgcmp to find untranslated missing entries in pot file.
	cmd := exec.Command("msgcmp", "-N", poFile, poTemplate)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	stderr, err := cmd.StderrPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		errs = append(errs, err.Error())
		return errs, false
	}
	scanner := bufio.NewScanner(stderr)

	for scanner.Scan() {
		line := scanner.Text()
		for _, kind := range []int{kindMissing, kindFuzzy, kindUntrans, kindUnused} {
			m := patternMap[kind].FindStringSubmatch(line)
			if len(m) > 0 {
				if countMap[kind] < 3 {
					if kind == kindMissing {
						msgMap[kind] = append(msgMap[kind], "po/git.pot:"+m[0])
					} else {
						msgMap[kind] = append(msgMap[kind], "po/XX.po:"+m[0])
					}
				} else if countMap[kind] == 3 {
					msgMap[kind] = append(msgMap[kind], "...")
				}
				countMap[kind]++
				break
			}
		}
	}
	countMap[kindUnused] = countMap[kindUnused] - countMap[kindFuzzy] - countMap[kindUntrans]

	for _, kind := range []int{kindMissing, kindFuzzy, kindUntrans, kindUnused} {
		if countMap[kind] == 0 {
			continue
		}
		switch kind {
		case kindMissing:
			errs = append(errs, fmt.Sprintf(
				"%d new string(s) in 'po/git.pot', but not in your 'po/XX.po'",
				countMap[kind]))
		case kindFuzzy:
			errs = append(errs, fmt.Sprintf(
				"%d fuzzy string(s) in your 'po/XX.po'",
				countMap[kind]))
		case kindUntrans:
			errs = append(errs, fmt.Sprintf(
				"%d untranslated string(s) in your 'po/XX.po'",
				countMap[kind]))
		case kindUnused:
			errs = append(errs, fmt.Sprintf(
				"%d obsolete string(s) in your 'po/XX.po', which must be removed",
				countMap[kind]))
		}
		errs = append(errs, "")
		for _, line := range msgMap[kind] {
			errs = append(errs, fmt.Sprintf("  > %s", line))
		}
		errs = append(errs, "")
	}
	if countMap[kindUnused] > 0 {
		ok = false
	}
	if countMap[kindMissing] > 0 {
		ok = false

		switch flag.GetPotFileFlag() {
		case flag.PotFileFlagLocation:
			fallthrough
		case flag.PotFileFlagUpdate:
			errs = append(errs,
				"Please run \"git-po-helper update po/XX.po\" to update your po file,",
				"and translate the new strings in it.",
				"")

		case flag.PotFileFlagDownload:
			fallthrough
		default:
			errs = append(errs,
				fmt.Sprintf(
					"You can download the latest \"po/git.pot\" file from:\n\n\t%s\n",
					PotFileURL),
				"Please rebase your branch to the latest upstream branch,",
				"run \"git-po-helper update po/XX.po\" to update your po file,",
				"and translate the new strings in it.",
				"")
		}
	}

	return errs, ok
}
