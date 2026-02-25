// Package util provides PO file report statistics.
package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
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

// parseReviewJSONWithGjson parses review JSON using gjson, which can tolerate
// some malformed LLM output (e.g. missing colons). Returns nil if parsing fails.
func parseReviewJSONWithGjson(data []byte, err error) *ReviewJSONResult {
	log.Warnf("fall back to gjson to fix json: %v", err)
	totalEntries := gjson.GetBytes(data, "total_entries").Int()
	issuesResult := gjson.GetBytes(data, "issues")
	if !issuesResult.Exists() {
		if totalEntries == 0 {
			return nil
		}
		return &ReviewJSONResult{TotalEntries: int(totalEntries), Issues: nil}
	}
	var issues []ReviewIssue
	for _, r := range issuesResult.Array() {
		issues = append(issues, ReviewIssue{
			MsgID:       r.Get("msgid").String(),
			MsgStr:      r.Get("msgstr").String(),
			Score:       int(r.Get("score").Int()),
			Description: r.Get("description").String(),
			Suggestion:  r.Get("suggestion").String(),
		})
	}
	return &ReviewJSONResult{TotalEntries: int(totalEntries), Issues: issues}
}

// prepareReviewJSONForParse preprocesses LLM-generated JSON for parsing.
// Handles: UTF-8 BOM, markdown code blocks (```json ... ```), leading/trailing text.
// Returns cleaned JSON bytes or original if no preprocessing needed.
func prepareReviewJSONForParse(data []byte, err error) []byte {
	log.Warnf("fall back to prepare (remove BOM and quote) to fix json: %v", err)
	data = bytes.TrimSpace(data)
	// Strip UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	// Extract from markdown code block: ```json ... ``` or ``` ... ```
	if idx := bytes.Index(data, []byte("```")); idx >= 0 {
		data = data[idx+3:]
		if bytes.HasPrefix(data, []byte("json")) {
			data = bytes.TrimSpace(data[4:])
		}
		if end := bytes.Index(data, []byte("```")); end >= 0 {
			data = bytes.TrimSpace(data[:end])
		}
	}
	// Extract JSON object by brace matching (handles leading/trailing text)
	if extracted, err := ExtractJSONFromOutput(data); err == nil {
		return extracted
	}
	return data
}

// ReportReviewFromJSON reads a review JSON file, optionally fills total_entries
// from a PO file when the JSON has none, and returns the report data.
// path may end with .json or .po; both json and po filenames are derived from it
// via DeriveReviewPaths to avoid inconsistency.
// Preprocesses LLM-generated JSON (BOM, markdown wrapping, extra text) before parsing.
func ReportReviewFromJSON(path string) (*ReviewReportResult, error) {
	jsonFile, poFile := DeriveReviewPaths(path)
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read review JSON %s: %w", jsonFile, err)
	}

	var review ReviewJSONResult
	if err := json.Unmarshal(data, &review); err != nil {
		// Retry with preprocessing for common LLM JSON issues
		prepared := prepareReviewJSONForParse(data, err)
		if err2 := json.Unmarshal(prepared, &review); err2 != nil {
			// Retry with gjson, which tolerates some malformed LLM output (e.g. missing colons)
			if parsed := parseReviewJSONWithGjson(prepared, err2); parsed != nil {
				review = *parsed
			} else {
				return nil, fmt.Errorf("failed to parse review JSON: %w (hint: LLM output may have invalid characters or structure; ensure the JSON is valid)", err)
			}
		}
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
