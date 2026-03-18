package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

var (
	PotFileURL = "https://github.com/git-l10n/pot-changes/raw/pot/master/po/git.pot"
)

func CheckUnfinishedPoFiles(commit string, poFiles []string) bool {
	var (
		ok         bool
		poTemplate string
		errs       []string
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
				errs = append(errs, fmt.Sprintf("commit %s: fail to checkout %s of revision %s: %s",
					AbbrevCommit(commit), tmpFile.File, tmpFile.Revision, err))
				ok = false
				continue
			}
			defer os.Remove(tmpFile.Tmpfile)
			poFile = tmpFile.Tmpfile
			prompt = fmt.Sprintf("[%s@%s]",
				locale+".po",
				AbbrevCommit(commit))
		} else {
			prompt = fmt.Sprintf("[%s]", filepath.Base(poFile))
		}
		// Check po file with pot file for missing translations.
		msgs, ret := checkUnfinishedPoFile(poFile, poTemplate)
		errs = append(errs, msgs...)
		if len(errs) > 0 {
			ReportSection("Incomplete translations found", ret, log.WarnLevel, prompt, errs...)
		}
		ok = ok && ret
	}
	return ok
}

const maxMsgidSampleLen = 30
const maxSamples = 3

func truncateMsgid(msgid string, maxLen int) string {
	// Sanitize: replace newlines and tabs so output is one line
	s := strings.ReplaceAll(msgid, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func checkUnfinishedPoFile(poFile, poTemplate string) ([]string, bool) {
	var errs []string
	ok := true

	potData, err := os.ReadFile(poTemplate)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to read POT file: %v", err))
		return errs, false
	}
	poData, err := os.ReadFile(poFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to read PO file: %v", err))
		return errs, false
	}

	potJ, err := LoadFileToGettextJSON(potData, poTemplate)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to parse POT file: %v", err))
		return errs, false
	}
	poJ, err := LoadFileToGettextJSON(poData, poFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to parse PO file: %v", err))
		return errs, false
	}

	// POT=old, PO=new: Deleted=Missing (in POT not in PO), Added=Unused (in PO not in POT)
	_, unusedEntries, missingEntries := CompareGettextEntriesWithDeleted(potJ, poJ, true)

	potKeys := make(map[string]bool)
	for _, e := range filterObsolete(potJ.Entries) {
		potKeys[entryKey(e)] = true
	}

	fuzzyEntries, untranslatedEntries, obsoleteEntries := GetEntriesByState(poJ, potKeys)

	appendSamples := func(entries []GettextEntry, prefix string, addErr func(string)) {
		for i, e := range entries {
			if i >= maxSamples {
				addErr("  > ...")
				break
			}
			sample := truncateMsgid(e.MsgID, maxMsgidSampleLen)
			addErr("  > " + prefix + sample)
		}
	}

	addErr := func(s string) { errs = append(errs, s) }

	if len(missingEntries) > 0 {
		ok = false
		errs = append(errs, fmt.Sprintf("%d new string(s) in 'po/git.pot', but not in your 'po/XX.po'", len(missingEntries)))
		errs = append(errs, "")
		appendSamples(missingEntries, "po/git.pot:", addErr)
		errs = append(errs, "")
	}
	if len(fuzzyEntries) > 0 {
		errs = append(errs, fmt.Sprintf("%d fuzzy string(s) in your 'po/XX.po'", len(fuzzyEntries)))
		errs = append(errs, "")
		appendSamples(fuzzyEntries, "po/XX.po:", addErr)
		errs = append(errs, "")
	}
	if len(untranslatedEntries) > 0 {
		errs = append(errs, fmt.Sprintf("%d untranslated string(s) in your 'po/XX.po'", len(untranslatedEntries)))
		errs = append(errs, "")
		appendSamples(untranslatedEntries, "po/XX.po:", addErr)
		errs = append(errs, "")
	}
	obsoleteCount := len(unusedEntries) + len(obsoleteEntries)
	if obsoleteCount > 0 {
		ok = false
		errs = append(errs, fmt.Sprintf("%d obsolete string(s) in your 'po/XX.po', which must be removed", obsoleteCount))
		errs = append(errs, "")
		// Combine samples from unused + obsolete, max 3 total
		allObsolete := make([]GettextEntry, 0, len(unusedEntries)+len(obsoleteEntries))
		allObsolete = append(allObsolete, unusedEntries...)
		allObsolete = append(allObsolete, obsoleteEntries...)
		appendSamples(allObsolete, "po/XX.po:", addErr)
		errs = append(errs, "")
	}

	if len(missingEntries) > 0 {
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
