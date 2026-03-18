package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

var (
	PotFileURL = "https://github.com/git-l10n/pot-changes/raw/pot/master/po/git.pot"
)

var potTemplateCache struct {
	once sync.Once
	path string
	ok   bool
}

func getPotTemplateForCheck() (string, bool) {
	potTemplateCache.once.Do(func() {
		if flag.GetPotFileFlag() == flag.PotFileFlagNone {
			potTemplateCache.ok = true
			return
		}
		path, ok := UpdatePotFile()
		if !ok {
			potTemplateCache.ok = false
			return
		}
		potTemplateCache.path = path
		potTemplateCache.ok = true
	})
	if !potTemplateCache.ok {
		return "", false
	}
	if potTemplateCache.path != "" {
		return potTemplateCache.path, true
	}
	return flag.GetPotFileLocation(), true
}

// CheckUnfinishedPoFile checks a single po file for incomplete translations.
// When commit is "HEAD" or empty, uses the file from disk; otherwise checkouts
// the file from the given commit.
func CheckUnfinishedPoFile(commit, poFile string) bool {
	if flag.GetPotFileFlag() == flag.PotFileFlagNone {
		return true
	}
	poTemplate, ok := getPotTemplateForCheck()
	if !ok {
		return false
	}
	if poTemplate == "" {
		poTemplate = flag.GetPotFileLocation()
	}

	prompt := ""
	fileToCheck := poFile
	locale := strings.TrimSuffix(filepath.Base(poFile), ".po")

	if commit != "" && commit != "HEAD" {
		tmpFile := FileRevision{
			Revision: commit,
			File:     poFile,
		}
		if err := CheckoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
			ReportSection("Incomplete translations found", false, log.WarnLevel,
				fmt.Sprintf("[%s@%s]", locale+".po", AbbrevCommit(commit)),
				fmt.Sprintf("commit %s: fail to checkout %s of revision %s: %s",
					AbbrevCommit(commit), tmpFile.File, tmpFile.Revision, err))
			return false
		}
		defer os.Remove(tmpFile.Tmpfile)
		fileToCheck = tmpFile.Tmpfile
		prompt = fmt.Sprintf("[%s@%s]", locale+".po", AbbrevCommit(commit))
	} else {
		prompt = fmt.Sprintf("[%s]", filepath.Base(poFile))
	}

	msgs, ret := checkUnfinishedPoFile(fileToCheck, poTemplate)
	if len(msgs) > 0 {
		ReportSection("Incomplete translations found", ret, log.WarnLevel, prompt, msgs...)
	}
	return ret
}

func CheckUnfinishedPoFiles(commit string, poFiles []string) bool {
	ok := true
	for _, poFile := range poFiles {
		if !CheckUnfinishedPoFile(commit, poFile) {
			ok = false
		}
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

	fuzzyEntries, untranslatedEntries, _ := GetEntriesByState(poJ, potKeys)
	// Obsolete (#~) entries are checked by checkPoNoObsoleteEntries in check-po.go.

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
		errs = append(errs, fmt.Sprintf("%d new string(s) in POT file, but not in your PO file", len(missingEntries)))
		errs = append(errs, "")
		appendSamples(missingEntries, "POT file:", addErr)
		errs = append(errs, "")
	}
	if len(fuzzyEntries) > 0 {
		errs = append(errs, fmt.Sprintf("%d fuzzy string(s) in your PO file", len(fuzzyEntries)))
		errs = append(errs, "")
		appendSamples(fuzzyEntries, "PO file:", addErr)
		errs = append(errs, "")
	}
	if len(untranslatedEntries) > 0 {
		errs = append(errs, fmt.Sprintf("%d untranslated string(s) in your PO file", len(untranslatedEntries)))
		errs = append(errs, "")
		appendSamples(untranslatedEntries, "PO file:", addErr)
		errs = append(errs, "")
	}
	// unusedEntries: non-obsolete entries in PO that are not in POT (custom/unused).
	// #~ obsolete entries are checked by checkPoNoObsoleteEntries.
	if len(unusedEntries) > 0 {
		ok = false
		errs = append(errs, fmt.Sprintf("%d obsolete string(s) in your PO file, which must be removed", len(unusedEntries)))
		errs = append(errs, "")
		appendSamples(unusedEntries, "PO file:", addErr)
		errs = append(errs, "")
	}

	if len(missingEntries) > 0 {
		switch flag.GetPotFileFlag() {
		case flag.PotFileFlagLocation:
			fallthrough
		case flag.PotFileFlagUpdate:
			errs = append(errs,
				"Please run \"git-po-helper update PO-FILE\" to update your po file,",
				"and translate the new strings in it.",
				"")

		case flag.PotFileFlagDownload:
			fallthrough
		default:
			errs = append(errs,
				fmt.Sprintf(
					"You can download the latest POT file from:\n\n\t%s\n",
					PotFileURL),
				"Please rebase your branch to the latest upstream branch,",
				"run \"git-po-helper update PO-FILE\" to update your po file,",
				"and translate the new strings in it.",
				"")
		}
	}

	return errs, ok
}
