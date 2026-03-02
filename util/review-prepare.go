// Package util provides review data preparation utilities.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

func PrepareReviewData(oldCommit, oldFile, newCommit, newFile, outputFile string, noHeader, useJSON bool) error {
	var (
		err                    error
		relOldFile, relNewFile string
	)
	if oldCommit != "" || newCommit != "" {
		_ = repository.WorkDir() // assert repo when using revisions
	}

	// Use temp files for orig and new; they are deleted when the function returns
	oldTmpFile, err := os.CreateTemp("", "review-old-*.po")
	if err != nil {
		return fmt.Errorf("failed to create temp old file: %w", err)
	}
	oldTmpFile.Close()
	defer func() {
		os.Remove(oldTmpFile.Name())
	}()

	newTmpFile, err := os.CreateTemp("", "review-new-*.po")
	if err != nil {
		return fmt.Errorf("failed to create temp new file: %w", err)
	}
	newTmpFile.Close()
	defer func() {
		os.Remove(newTmpFile.Name())
	}()

	log.Debugf("preparing review data: orig=%s, new=%s, review-input=%s",
		oldTmpFile.Name(), newTmpFile.Name(), outputFile)

	// Get original file (from git when revision set, else from worktree)
	log.Infof("getting old file from commit: %s", oldCommit)
	if oldCommit != "" {
		workDir := repository.WorkDir()
		if filepath.IsAbs(oldFile) {
			relOldFile, err = filepath.Rel(workDir, oldFile)
			if err != nil {
				return fmt.Errorf("failed to convert PO file path to relative: %w", err)
			}
		} else {
			relOldFile = oldFile
		}
	} else {
		relOldFile = oldFile
	}
	relOldFile = filepath.ToSlash(relOldFile)
	oldFileRevision := FileRevision{
		Revision: oldCommit,
		File:     relOldFile,
		Tmpfile:  oldTmpFile.Name(),
	}
	if err := CheckoutTmpfile(&oldFileRevision); err != nil {
		// Check if error is because file doesn't exist in the commit
		if strings.Contains(err.Error(), "does not exist in") {
			// If file doesn't exist in that commit, create empty file
			log.Infof("file %s not found in commit %s, using empty file as original", relOldFile, oldCommit)
			if err := os.WriteFile(oldFileRevision.Tmpfile, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create empty orig file: %w", err)
			}
		} else {
			// For other errors, return them
			return fmt.Errorf("failed to get original file from commit %s: %w", oldCommit, err)
		}
	}

	log.Infof("getting new file from commit: %s", newCommit)
	if newCommit != "" {
		workDir := repository.WorkDir()
		if filepath.IsAbs(newFile) {
			relNewFile, err = filepath.Rel(workDir, newFile)
			if err != nil {
				return fmt.Errorf("failed to convert PO file path to relative: %w", err)
			}
		} else {
			relNewFile = newFile
		}
	} else {
		relNewFile = newFile
	}
	relNewFile = filepath.ToSlash(relNewFile)
	newFileRevision := FileRevision{
		Revision: newCommit,
		File:     relNewFile,
		Tmpfile:  newTmpFile.Name(),
	}
	if err := CheckoutTmpfile(&newFileRevision); err != nil {
		// Check if error is because file doesn't exist in the commit
		if strings.Contains(err.Error(), "does not exist in") {
			// If file doesn't exist in that commit, create empty file
			log.Infof("file %s not found in commit %s, using empty file as original", relNewFile, newCommit)
			if err := os.WriteFile(newFileRevision.Tmpfile, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create empty new file: %w", err)
			}
		} else {
			// For other errors, return them
			return fmt.Errorf("failed to get new file from commit %s: %w", newCommit, err)
		}
	}

	origData, err := os.ReadFile(oldFileRevision.Tmpfile)
	if err != nil {
		return fmt.Errorf("failed to read orig file: %w", err)
	}
	newData, err := os.ReadFile(newFileRevision.Tmpfile)
	if err != nil {
		return fmt.Errorf("failed to read new file: %w", err)
	}

	oldJ, err := LoadFileToGettextJSON(origData, relOldFile)
	if err != nil {
		return err
	}
	newJ, err := LoadFileToGettextJSON(newData, relNewFile)
	if err != nil {
		return err
	}

	log.Debugf("extracting differences to review-input")
	_, reviewEntries := CompareGettextEntries(oldJ, newJ)

	if len(reviewEntries) == 0 {
		return WriteFile(outputFile, nil)
	}

	out := &GettextJSON{
		HeaderComment: newJ.HeaderComment,
		HeaderMeta:    newJ.HeaderMeta,
		Entries:       reviewEntries,
	}
	if noHeader {
		out.HeaderComment = ""
		out.HeaderMeta = ""
	}

	if useJSON {
		return writeGettextJSONToPath(outputFile, out)
	}

	poData, err := GettextJSONToPoBytes(out, noHeader)
	if err != nil {
		return fmt.Errorf("failed to build PO: %w", err)
	}
	log.Infof("review data prepared: review-input=%s", outputFile)
	return WriteFile(outputFile, poData)
}

func writeGettextJSONToPath(outputFile string, j *GettextJSON) error {
	if outputFile == "-" || outputFile == "" {
		return WriteGettextJSONToJSON(j, os.Stdout)
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	return WriteGettextJSONToJSON(j, f)
}

func WriteFile(outputFile string, data []byte) error {
	if outputFile == "-" || outputFile == "" {
		if len(data) == 0 {
			return nil
		}
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(outputFile, data, 0644)
}

// WritePoEntries writes the review input PO file with header and review entries.
// When outputPath is "-" or "" and entries is empty, writes nothing (for new-entries command).
func WritePoEntries(outputPath string, header []string, entries []*GettextEntry) error {
	data := BuildPoContent(header, entries)
	return WriteFile(outputPath, data)
}
