// Package util provides review report statistics.
package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var DefaultReviewPoPath = filepath.Join(PoDir, "review.po")

// ReviewReportResult holds the result of reporting from a review JSON file.
// Issue scores: 0 = critical, 1 = minor, 2 = major. Perfect = no issue.
type ReviewReportResult struct {
	Review        *ReviewJSONResult
	Score         int
	CriticalCount int // score 0
	MinorCount    int // score 1
	MajorCount    int // score 2
}

// PerfectCount returns the number of entries with no reported issue:
// review.TotalEntries - (CriticalCount + MinorCount + MajorCount).
func (r *ReviewReportResult) PerfectCount() int {
	if r.Review == nil {
		return 0
	}
	n := r.Review.TotalEntries - (r.CriticalCount + r.MinorCount + r.MajorCount)
	if n < 0 {
		return 0
	}
	return n
}

// CountReviewIssueScores returns counts by issue score from a review.
// Score 0 = critical, 1 = minor, 2 = major. Perfect count is derived: TotalEntries - (critical + minor + major).
func CountReviewIssueScores(review *ReviewJSONResult) (critical, minor, major int) {
	for _, issue := range review.Issues {
		switch issue.Score {
		case 0:
			critical++
		case 1:
			minor++
		case 2:
			major++
		}
	}
	return critical, minor, major
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

// ReportReviewFromJSON reads a review JSON file, optionally fills total_entries
// from a PO file when the JSON has none, and returns the report data.
// path may end with .json or .po; both json and po filenames are derived from it
// via DeriveReviewPaths to avoid inconsistency.
// Preprocesses LLM-generated JSON (BOM, markdown wrapping, extra text) before parsing.
func ReportReviewFromJSON(path string) (string, *ReviewReportResult, error) {
	jsonFile, poFile := DeriveReviewPaths(path)
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read review JSON %s: %w", jsonFile, err)
	}

	var review ReviewJSONResult
	if err := json.Unmarshal(data, &review); err != nil {
		// Retry with preprocessing for common LLM JSON issues
		prepared := PrepareJSONForParse(data, err)
		if err2 := json.Unmarshal(prepared, &review); err2 != nil {
			// Retry with gjson, which tolerates some malformed LLM output (e.g. missing colons)
			if parsed := parseReviewJSONWithGjson(prepared, err2); parsed != nil {
				review = *parsed
			} else {
				return "", nil, fmt.Errorf("failed to parse review JSON: %w (hint: LLM output may have invalid characters or structure; ensure the JSON is valid)", err)
			}
		}
	}

	if Exist(poFile) {
		stats, err := CountReportStats(poFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to count entries in %s: %w", poFile, err)
		}
		review.TotalEntries = stats.Total()
	} else {
		return "", nil, fmt.Errorf("file does not exist: %s", poFile)
	}

	score, err := CalculateReviewScore(&review)
	if err != nil {
		return "", nil, fmt.Errorf("failed to calculate review score: %w", err)
	}

	critical, minor, major := CountReviewIssueScores(&review)
	return jsonFile, &ReviewReportResult{
		Review:        &review,
		Score:         score,
		CriticalCount: critical,
		MinorCount:    minor,
		MajorCount:    major,
	}, nil
}

// loadReviewJSONFromFile reads and parses a single review JSON file with the same
// robustness as ReportReviewFromJSON (BOM, markdown wrapping, gjson fallback).
// It does not fill TotalEntries from a PO file.
func loadReviewJSONFromFile(jsonFile string) (*ReviewJSONResult, error) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", jsonFile, err)
	}
	var review ReviewJSONResult
	if err := json.Unmarshal(data, &review); err != nil {
		prepared := PrepareJSONForParse(data, err)
		if err2 := json.Unmarshal(prepared, &review); err2 != nil {
			if parsed := parseReviewJSONWithGjson(prepared, err2); parsed != nil {
				return parsed, nil
			}
			return nil, fmt.Errorf("failed to parse review JSON %s: %w", jsonFile, err)
		}
	}
	if review.Issues == nil {
		review.Issues = []ReviewIssue{}
	}
	return &review, nil
}

// ReportReviewFromPathWithBatches reports from review-batch-*.json files or a single review JSON.
// Path may be e.g. "po/review.po"; DeriveReviewPaths gives po/review.json and po/review.po.
// If any files match "<dir>/<base>-batch-*.json" (e.g. po/review-batch-1.json), their mtime is
// compared with base+".json": if base+".json" is newer, it is read directly; otherwise
// batch files are loaded, merged (duplicate msgid: keep the issue with lower score), and
// the result is saved to base+".json" then returned.
// If no batch files exist, falls back to ReportReviewFromJSON(path).
func ReportReviewFromPathWithBatches(path string) (string, *ReviewReportResult, error) {
	if path == "" {
		path = DefaultReviewPoPath
	}
	jsonFile, poFile := DeriveReviewPaths(path)
	dir := filepath.Dir(jsonFile)
	base := strings.TrimSuffix(filepath.Base(jsonFile), ".json")
	pattern := filepath.Join(dir, base+"-batch-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", nil, fmt.Errorf("glob %s: %w", pattern, err)
	}
	sort.Strings(matches)

	log.Debugf("Call ReportReviewFromPathWithBatches(%s) with %d batch files",
		path, len(matches))
	if len(matches) == 0 {
		return ReportReviewFromJSON(path)
	}

	// Compare timestamps: if base+".json" is newer than all batch files, read from it only.
	jsonStat, err := os.Stat(jsonFile)
	if err == nil {
		var maxBatchModTime int64
		for _, f := range matches {
			fi, err := os.Stat(f)
			if err != nil {
				continue
			}
			if t := fi.ModTime().Unix(); t > maxBatchModTime {
				maxBatchModTime = t
			}
		}
		if jsonStat.ModTime().Unix() >= maxBatchModTime {
			return ReportReviewFromJSON(path)
		}
	}

	// Load batch files and merge; for duplicate msgid, AggregateReviewJSON keeps lower score.
	var batchReviews []*ReviewJSONResult
	for _, f := range matches {
		r, err := loadReviewJSONFromFile(f)
		if err != nil {
			return "", nil, err
		}
		if r != nil {
			batchReviews = append(batchReviews, r)
		}
	}
	merged := AggregateReviewJSON(batchReviews, true)
	if merged == nil {
		merged = &ReviewJSONResult{Issues: []ReviewIssue{}}
	}
	if Exist(poFile) {
		stats, err := CountReportStats(poFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to count entries in %s: %w", poFile, err)
		}
		merged.TotalEntries = stats.Total()
	} else {
		return "", nil, fmt.Errorf("file does not exist: %s", poFile)
	}
	if err := saveReviewJSON(merged, jsonFile); err != nil {
		return "", nil, fmt.Errorf("failed to save aggregated review to %s: %w", jsonFile, err)
	}
	score, err := CalculateReviewScore(merged)
	if err != nil {
		return "", nil, fmt.Errorf("failed to calculate review score: %w", err)
	}
	critical, minor, major := CountReviewIssueScores(merged)
	return jsonFile, &ReviewReportResult{
		Review:        merged,
		Score:         score,
		CriticalCount: critical,
		MinorCount:    minor,
		MajorCount:    major,
	}, nil
}

// PrintReviewReportResult prints the same "## Review Statistics" report as agent-run report (step 9).
// Used by RunAgentReview after merge and by cmd/agent-run-report.
func PrintReviewReportResult(jsonFile string, result *ReviewReportResult) {
	fmt.Println("## Review Statistics")
	fmt.Println()
	fmt.Printf("  %-22s %d/100\n", "Review score:", result.Score)
	fmt.Printf("  %-22s %d\n", "Total entries:", result.Review.TotalEntries)
	fmt.Printf("  %-22s %d\n", "Perfect (no issue):", result.PerfectCount())
	fmt.Printf("  %-22s %d\n", "With issues:", result.Review.IssueCount())
	fmt.Println()
	fmt.Printf("  %-22s %d\n", "Critical (score 0):", result.CriticalCount)
	fmt.Printf("  %-22s %d\n", "Major (score 1):", result.MajorCount)
	fmt.Printf("  %-22s %d\n", "Minor (score 2):", result.MinorCount)
	fmt.Println()
	fmt.Printf("For full details, see the review JSON file: `%s`\n", jsonFile)
}
