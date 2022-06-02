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
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

var (
	PotFileURL = "https://github.com/git-l10n/git-po/raw/pot/main/po/git.pot"
)

func CheckUnfinishedPoFiles(commit string, poFiles []string) bool {
	var (
		ok         = true
		poTemplate string
		opt        = flag.CheckPotFile()
	)

	// Build or update pot file.
	if opt == flag.CheckPotFileNone {
		return true
	}

	// Update pot file.
	if poTemplate, ok = UpdatePotFile(); !ok {
		return false
	}
	if poTemplate == "" {
		poTemplate = filepath.Join(PoDir, GitPot)
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
			if err := checkoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
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
		if msgs := checkUnfinishedPoFile(poFile, poTemplate); len(msgs) > 0 {
			reportResultMessages(msgs, prompt, log.ErrorLevel)
			ok = false
		}
	}
	return ok
}

func checkUnfinishedPoFile(poFile, poTemplate string) []string {
	var errs []string

	// Run msgcmp to find untranslated missing entries in pot file.
	cmd := exec.Command("msgcmp", "-N", poFile, poTemplate)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.Dir = repository.WorkDir()
	stderr, err := cmd.StderrPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		errs = append(errs, err.Error())
		return errs
	}
	scanner := bufio.NewScanner(stderr)
	pattern := regexp.MustCompile(`[0-9]+: this message is used but not defined in .*`)
	msgs := []string{}
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		m := pattern.FindStringSubmatch(line)
		if len(m) > 0 {
			if count < 3 {
				msgs = append(msgs, "po/git.pot:"+m[0])
			} else if count == 3 {
				msgs = append(msgs, "...")
			}
			count++
		}
	}
	if count > 0 {
		switch flag.CheckPotFile() {
		case flag.CheckPotFileNone:
			return nil
		case flag.CheckPotFileCurrent:
			fallthrough
		case flag.CheckPotFileUpdate:
			if count == 1 {
				errs = append(errs, fmt.Sprintf(
					"There is %d new string in 'po/git.pot' missing in your translation.\n",
					count))
			} else {
				errs = append(errs, fmt.Sprintf(
					"There are %d new strings in 'po/git.pot' missing in your translation.\n",
					count))
			}
			errs = append(errs,
				"Please run \"make po-update PO_FILE=po/XX.po\" to update your po file,",
				"and translate the new strings in it.",
				"")

		case flag.CheckPotFileDownload:
			fallthrough
		default:
			if count == 1 {
				errs = append(errs, fmt.Sprintf(
					"There is %d new string missing in your translation.\n",
					count))
			} else {
				errs = append(errs, fmt.Sprintf(
					"There are %d new strings missing in your translation.\n",
					count))
			}
			errs = append(errs,
				fmt.Sprintf(
					"You can download the latest \"po/git.pot\" file from:\n\n\t%s\n",
					PotFileURL),
				"Please rebase your branch to the latest upstream branch,",
				"run \"make po-update PO_FILE=po/XX.po\" to update your po file,",
				"and translate the new strings in it.",
				"")

		}

		for _, line := range msgs {
			errs = append(errs, fmt.Sprintf("  > %s", line))
		}
		return errs
	}

	return nil
}
