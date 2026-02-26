// Package util provides review report statistics.
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

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
