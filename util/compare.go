package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// CompareTarget holds the resolved old/new commit and file for compare operations.
type CompareTarget struct {
	OldCommit string
	NewCommit string
	OldFile   string
	NewFile   string
}

// ResolveRevisionsAndFiles resolves range/commit/since flags and args into a CompareTarget.
// Exactly one of rangeStr, commitStr, and sinceStr may be non-empty.
// Args may be 0, 1, or 2 po file paths. With 2 args, revisions are not allowed.
// When args is empty, the po file is auto-selected from changed files.
func ResolveRevisionsAndFiles(rangeStr, commitStr, sinceStr string, args []string) (*CompareTarget, error) {
	// --range, --commit, --since are mutually exclusive
	nSet := 0
	if strings.TrimSpace(rangeStr) != "" {
		nSet++
	}
	if strings.TrimSpace(commitStr) != "" {
		nSet++
	}
	if strings.TrimSpace(sinceStr) != "" {
		nSet++
	}
	if nSet > 1 {
		return nil, fmt.Errorf("only one of --range, --commit, or --since may be specified")
	}

	// Resolve range for both modes
	var revRange string
	if c := strings.TrimSpace(commitStr); c != "" {
		revRange = c + "^.." + c
	} else if s := strings.TrimSpace(sinceStr); s != "" {
		revRange = s + ".."
	} else {
		revRange = strings.TrimSpace(rangeStr)
	}
	if revRange == "" {
		switch len(args) {
		case 0:
			revRange = "HEAD.."
		case 1:
			revRange = "HEAD.."
		case 2:
			// Compare two files in worktree
		}
	}

	if len(args) > 2 {
		return nil, fmt.Errorf("too many arguments (%d > 2)", len(args))
	}

	repository.ChdirProjectRoot()

	var (
		oldCommit, newCommit string
		oldFile, newFile     string
	)
	// Parse revision: "a..b", "a..", or "a"
	if strings.Contains(revRange, "..") {
		parts := strings.SplitN(revRange, "..", 2)
		oldCommit = strings.TrimSpace(parts[0])
		newCommit = strings.TrimSpace(parts[1])
	} else if revRange != "" {
		// a : first is a~, second is a
		oldCommit = revRange + "~"
		newCommit = revRange
	}

	// Set File
	switch len(args) {
	case 0:
		// Automatically or manually select PO file from changed files
	case 1:
		oldFile = args[0]
		newFile = args[0]
	case 2:
		oldFile = args[0]
		newFile = args[1]
		if oldCommit != "" || newCommit != "" {
			return nil, fmt.Errorf("cannot specify revision for multiple files: %s and %s",
				oldFile, newFile)
		}
	}

	// Resolve poFile when not specified
	if len(args) == 0 {
		changedPoFiles, err := GetChangedPoFilesRange(oldCommit, newCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to get changed po files: %w", err)
		}

		oldFile, err = ResolvePoFile(oldFile, changedPoFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve default po file: %w", err)
		}
		newFile = oldFile
	}

	return &CompareTarget{
		OldCommit: oldCommit,
		NewCommit: newCommit,
		OldFile:   oldFile,
		NewFile:   newFile,
	}, nil
}

// DiffStat holds the diff statistics between two PO files.
type DiffStat struct {
	Added   int // Entries in dest but not in src
	Changed int // Same msgid but different content
	Deleted int // Entries in src but not in dest
}

// EntriesEqual checks if two PO entries are equal (same msgid and msgstr).
func EntriesEqual(e1, e2 *PoEntry) bool {
	if e1.IsFuzzy != e2.IsFuzzy {
		return false
	}
	if e1.MsgID != e2.MsgID {
		return false
	}
	if e1.MsgStr != e2.MsgStr {
		return false
	}
	if e1.MsgIDPlural != e2.MsgIDPlural {
		return false
	}
	if len(e1.MsgStrPlural) != len(e2.MsgStrPlural) {
		return false
	}
	for i := range e1.MsgStrPlural {
		if e1.MsgStrPlural[i] != e2.MsgStrPlural[i] {
			return false
		}
	}
	return true
}

// PoCompare compares src and dest PO file content. Returns DiffStat, the generated
// review-input data (newHeader + reviewEntries), and error. reviewEntries are entries
// that are new or changed in dest compared to src.
func PoCompare(src, dest []byte) (DiffStat, []byte, error) {
	origEntries, _, err := ParsePoEntries(src)
	if err != nil {
		return DiffStat{}, nil, fmt.Errorf("failed to parse src file: %w", err)
	}
	newEntries, newHeader, err := ParsePoEntries(dest)
	if err != nil {
		return DiffStat{}, nil, fmt.Errorf("failed to parse dest file: %w", err)
	}

	// Sort entries by MsgID for consistent ordering
	sort.Slice(origEntries, func(i, j int) bool {
		return origEntries[i].MsgID < origEntries[j].MsgID
	})
	sort.Slice(newEntries, func(i, j int) bool {
		return newEntries[i].MsgID < newEntries[j].MsgID
	})

	if len(src) == 0 {
		log.Debugf("src file is empty, all entries in dest will be included in review-input")
	}

	// Two-pointer merge of sorted origEntries and newEntries
	var stat DiffStat
	var reviewEntries []*PoEntry
	i, j := 0, 0
	for i < len(origEntries) && j < len(newEntries) {
		cmp := strings.Compare(origEntries[i].MsgID, newEntries[j].MsgID)
		if cmp < 0 {
			stat.Deleted++
			i++
		} else if cmp > 0 {
			stat.Added++
			reviewEntries = append(reviewEntries, newEntries[j])
			j++
		} else {
			if !EntriesEqual(origEntries[i], newEntries[j]) {
				stat.Changed++
				reviewEntries = append(reviewEntries, newEntries[j])
			}
			i++
			j++
		}
	}
	for i < len(origEntries) {
		stat.Deleted++
		i++
	}
	for j < len(newEntries) {
		stat.Added++
		reviewEntries = append(reviewEntries, newEntries[j])
		j++
	}

	log.Debugf("review stats: deleted=%d, added=%d, changed=%d", stat.Deleted, stat.Added, stat.Changed)

	data := BuildPoContent(newHeader, reviewEntries)
	return stat, data, nil
}
