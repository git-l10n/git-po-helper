package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	PotFileURL = "https://github.com/git-l10n/pot-changes/raw/pot/master/po/git.pot"
)

// CheckWithPotFile checks a single po file for incomplete translations.
// When commit is "HEAD" or empty, uses the file from disk; otherwise checkouts
// the file from the given commit. projectName is from Project-Id-Version meta.
func CheckWithPotFile(commit, projectName, poFile string) bool {
	cfg := GetProjectPotConfig(projectName, poFile)
	action := cfg.GetEffectiveAction()
	if action == DefaultPotActionNo {
		return true
	}
	poTemplate, err := cfg.AcquirePotFile(projectName, poFile)
	if err != nil {
		log.Error(err)
		return false
	}
	if poTemplate == "" {
		log.Warnf("no pot file found for project %s and po file %s", projectName, poFile)
		return true
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

	msgs, ret := checkUnfinishedPoFile(fileToCheck, poTemplate, projectName, poFile)
	if len(msgs) > 0 {
		ReportSection("Incomplete translations found", ret, log.WarnLevel, prompt, msgs...)
	}
	return ret
}

func CheckUnfinishedPoFiles(commit string, poFiles []string) bool {
	ok := true
	projectName := ""
	if len(poFiles) > 0 {
		projectName = getProjectNameFromPoFile(poFiles[0], commit)
	}
	for _, poFile := range poFiles {
		if !CheckWithPotFile(commit, projectName, poFile) {
			ok = false
		}
	}
	return ok
}

// getProjectNameFromPoFile reads Project-Id-Version from a PO file.
func getProjectNameFromPoFile(poFile, commit string) string {
	fileToRead := poFile
	if commit != "" && commit != "HEAD" {
		tmpFile := FileRevision{Revision: commit, File: poFile}
		if err := CheckoutTmpfile(&tmpFile); err != nil || tmpFile.Tmpfile == "" {
			return ""
		}
		fileToRead = tmpFile.Tmpfile
		defer os.Remove(tmpFile.Tmpfile)
	}
	data, err := os.ReadFile(fileToRead)
	if err != nil {
		return ""
	}
	po, err := ParsePoEntries(data)
	if err != nil {
		return ""
	}
	return po.GetProject()
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

func checkUnfinishedPoFile(fileToCheck, poTemplate, projectName, poFilePath string) ([]string, bool) {
	var errs []string
	ok := true

	potData, err := os.ReadFile(poTemplate)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to read POT file: %v", err))
		return errs, false
	}
	poData, err := os.ReadFile(fileToCheck)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to read PO file: %v", err))
		return errs, false
	}

	potJ, err := LoadFileToGettextJSON(potData, poTemplate)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to parse POT file: %v", err))
		return errs, false
	}
	poJ, err := LoadFileToGettextJSON(poData, fileToCheck)
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
		cfg := GetProjectPotConfig(projectName, poFilePath)
		action := cfg.GetEffectiveAction()
		downloadURL := PotFileURL
		if cfg.DownloadURL != "" {
			downloadURL = cfg.DownloadURL
		}
		switch action {
		case DefaultPotActionUseIfExist:
			fallthrough
		case DefaultPotActionBuild:
			errs = append(errs,
				"Please run \"git-po-helper update PO-FILE\" to update your po file,",
				"and translate the new strings in it.",
				"")

		case DefaultPotActionDownload:
			fallthrough
		default:
			errs = append(errs,
				fmt.Sprintf(
					"You can download the latest POT file from:\n\n\t%s\n",
					downloadURL),
				"Please rebase your branch to the latest upstream branch,",
				"run \"git-po-helper update PO-FILE\" to update your po file,",
				"and translate the new strings in it.",
				"")
		}
	}

	return errs, ok
}
