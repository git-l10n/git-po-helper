package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
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
		ok      = true
		potFile string
		opt     = flag.CheckPotFile()
	)

	// Build or update pot file.
	if opt == flag.CheckPotFileNone {
		return true
	}
	// Try to download pot file.
	if opt == flag.CheckPotFileDownload {
		tmpfile, err := ioutil.TempFile("", "git.pot-*")
		if err != nil {
			log.Error(err)
			return false
		}
		tmpfile.Close()
		potFile = tmpfile.Name()
		defer os.Remove(potFile)
		showHorizontalLine()
		log.Infof("downloading pot file from %s", PotFileURL)
		if err := httpDownload(PotFileURL, potFile, true); err != nil {
			log.Warn(err)
			opt = flag.CheckPotFileCurrent
		}
	}
	// If fail to download, try to use current pot file.
	if opt == flag.CheckPotFileCurrent || opt == flag.CheckPotFileUpdate {
		potFile = "po/git.pot"
		if !Exist(potFile) || opt == flag.CheckPotFileUpdate {
			cmd := exec.Command("make", "pot")
			showHorizontalLine()
			log.Info("update pot file by running: make pot")
			cmd.Dir = repository.WorkDir()
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Error(err)
				return false
			}
		}
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
		if msgs := checkUnfinishedPoFile(poFile, potFile); len(msgs) > 0 {
			reportResultMessages(msgs, prompt, log.ErrorLevel)
			ok = false
		}
	}
	return ok
}

func checkUnfinishedPoFile(poFile, potFile string) []string {
	var errs []string

	// Run msgcmp to find untranslated missing entries in pot file.
	cmd := exec.Command("msgcmp", "-N", poFile, potFile)
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
			errs = append(errs, fmt.Sprintf(
				"Please run \"make po-update PO_FILE=%s\" to update your po file,\n"+
					"and translate the new strings in it.\n",
				poFile))

		case flag.CheckPotFileDownload:
			fallthrough
		default:
			errs = append(errs, fmt.Sprintf(
				"The latest \"po/git.pot\" file can be downloaded from:\n\n\t%s\n",
				PotFileURL))
			if count == 1 {
				errs = append(errs, fmt.Sprintf(
					", and there is %d new string in it missing in your translation.\n",
					count))
			} else {
				errs = append(errs, fmt.Sprintf(
					", and there are %d new strings in it missing in your translation.\n",
					count))
			}
			errs = append(errs, fmt.Sprintf(
				"Please rebase your branch to the latest upstream branch,\n"+
					"run \"make po-update PO_FILE=%s\" to update your po file,\n"+
					"and translate the new strings in it.\n",
				poFile))

		}

		for _, line := range msgs {
			errs = append(errs, fmt.Sprintf("  > %s", line))
		}
		return errs
	}

	return nil
}
