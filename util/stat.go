// Package util provides PO file report statistics.
package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PoReportStats holds statistics for a PO file.
type PoReportStats struct {
	Translated   int // Entries with non-empty translation, not fuzzy, not same as msgid
	Untranslated int // Entries with empty msgstr
	Same         int // Entries where msgstr equals msgid (suspect untranslated)
	Fuzzy        int // Entries with fuzzy flag
	Obsolete     int // Obsolete entries (#~ format)
}

// CountPoReportStats reads a PO file and returns entry statistics.
func CountPoReportStats(poFile string) (*PoReportStats, error) {
	data, err := os.ReadFile(poFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", poFile, err)
	}

	entries, _, err := ParsePoEntries(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", poFile, err)
	}

	obsolete, err := countObsoleteEntries(poFile)
	if err != nil {
		return nil, fmt.Errorf("failed to count obsolete entries: %w", err)
	}

	stats := &PoReportStats{Obsolete: obsolete}

	for _, e := range entries {
		// Skip header (empty msgid)
		if e.MsgID == "" {
			continue
		}

		hasTranslation := false
		msgstrValue := ""

		if len(e.MsgStrPlural) > 0 {
			for _, s := range e.MsgStrPlural {
				if s != "" {
					hasTranslation = true
					break
				}
			}
			if len(e.MsgStrPlural) > 0 {
				msgstrValue = e.MsgStrPlural[0]
			}
		} else {
			hasTranslation = e.MsgStr != ""
			msgstrValue = e.MsgStr
		}

		if e.IsFuzzy {
			stats.Fuzzy++
			continue
		}
		if !hasTranslation {
			stats.Untranslated++
			continue
		}
		if msgstrValue == e.MsgID {
			stats.Same++
			continue
		}
		stats.Translated++
	}

	return stats, nil
}

// FormatMsgfmtStatistics formats stats to match msgfmt --statistics output.
// For compatibility, "same" (msgstr == msgid) is counted as translated.
func FormatMsgfmtStatistics(stats *PoReportStats) string {
	translated := stats.Translated + stats.Same
	var parts []string
	if translated > 0 {
		if translated == 1 {
			parts = append(parts, "1 translated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d translated messages", translated))
		}
	}
	if stats.Fuzzy > 0 {
		if stats.Fuzzy == 1 {
			parts = append(parts, "1 fuzzy translation")
		} else {
			parts = append(parts, fmt.Sprintf("%d fuzzy translations", stats.Fuzzy))
		}
	}
	if stats.Untranslated > 0 {
		if stats.Untranslated == 1 {
			parts = append(parts, "1 untranslated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d untranslated messages", stats.Untranslated))
		}
	}
	if len(parts) == 0 {
		return "0 translated messages.\n"
	}
	return strings.Join(parts, ", ") + ".\n"
}

// FormatStatLine formats stats in one line, similar to msgfmt --statistics,
// but also includes same and obsolete. Only non-zero categories are shown.
func FormatStatLine(stats *PoReportStats) string {
	var parts []string
	if stats.Translated > 0 {
		if stats.Translated == 1 {
			parts = append(parts, "1 translated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d translated messages", stats.Translated))
		}
	}
	if stats.Fuzzy > 0 {
		if stats.Fuzzy == 1 {
			parts = append(parts, "1 fuzzy translation")
		} else {
			parts = append(parts, fmt.Sprintf("%d fuzzy translations", stats.Fuzzy))
		}
	}
	if stats.Untranslated > 0 {
		if stats.Untranslated == 1 {
			parts = append(parts, "1 untranslated message")
		} else {
			parts = append(parts, fmt.Sprintf("%d untranslated messages", stats.Untranslated))
		}
	}
	if stats.Same > 0 {
		if stats.Same == 1 {
			parts = append(parts, "1 same message")
		} else {
			parts = append(parts, fmt.Sprintf("%d same messages", stats.Same))
		}
	}
	if stats.Obsolete > 0 {
		if stats.Obsolete == 1 {
			parts = append(parts, "1 obsolete entry")
		} else {
			parts = append(parts, fmt.Sprintf("%d obsolete entries", stats.Obsolete))
		}
	}
	if len(parts) == 0 {
		return "0 translated messages.\n"
	}
	return strings.Join(parts, ", ") + ".\n"
}

// ReviewReportResult holds the result of reporting from a review JSON file.
type ReviewReportResult struct {
	Review        *ReviewJSONResult
	Score         int
	CriticalCount int
	MinorCount    int
}

// ReportReviewFromJSON reads a review JSON file, optionally fills total_entries
// from a PO file when the JSON has none, and returns the report data.
// path may end with .json or .po; both json and po filenames are derived from it
// via DeriveReviewPaths to avoid inconsistency.
func ReportReviewFromJSON(path string) (*ReviewReportResult, error) {
	jsonFile, poFile := DeriveReviewPaths(path)
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read review JSON %s: %w", jsonFile, err)
	}

	var review ReviewJSONResult
	if err := json.Unmarshal(data, &review); err != nil {
		return nil, fmt.Errorf("failed to parse review JSON: %w", err)
	}

	if review.TotalEntries <= 0 {
		if !Exist(poFile) {
			return nil, fmt.Errorf("file does not exist: %s", poFile)
		}
		count, err := CountPoEntries(poFile)
		if err != nil {
			return nil, fmt.Errorf("failed to count entries in %s: %w", poFile, err)
		}
		review.TotalEntries = count
	}

	score, err := CalculateReviewScore(&review)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate review score: %w", err)
	}

	criticalCount := 0
	minorCount := 0
	for _, issue := range review.Issues {
		switch issue.Score {
		case 0:
			criticalCount++
		case 2:
			minorCount++
		}
	}

	return &ReviewReportResult{
		Review:        &review,
		Score:         score,
		CriticalCount: criticalCount,
		MinorCount:    minorCount,
	}, nil
}

// countObsoleteEntries counts lines starting with "#~ msgid " in the file.
func countObsoleteEntries(poFile string) (int, error) {
	f, err := os.Open(poFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "#~ msgid ") {
			count++
		}
	}
	return count, scanner.Err()
}
