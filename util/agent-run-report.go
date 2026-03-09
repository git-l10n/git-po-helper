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

// DefaultReviewBase is the default base for review paths (po/review).
// Used by agent-run report when no path is given.
var DefaultReviewBase = filepath.Join(PoDir, "review")

// ResolvedReviewPaths holds resolved paths for report. Embeds ReviewPathSet;
// JSONFile and POFileForCount are the actual files to read (may differ from
// PathSet fields for legacy paths like review.json, review.po).
type ResolvedReviewPaths struct {
	ReviewPathSet
	JSONFile       string // JSON file to read
	POFileForCount string // PO file for total_entries count
}

// resolveReviewPaths returns resolved paths for the given path.
// Path may be: base (po/review), result JSON (po/review-result.json), pending PO
// (po/review-pending.po), or legacy (po/review.json, po/review.po).
func resolveReviewPaths(path string) ResolvedReviewPaths {
	dir := filepath.Dir(path)
	baseName := filepath.Base(path)
	var base string
	var jsonFile, poFile string
	switch {
	case strings.HasSuffix(baseName, "-result.json"):
		jsonFile = path
		base = filepath.Join(dir, strings.TrimSuffix(baseName, "-result.json"))
	case strings.HasSuffix(baseName, "-pending.po"):
		poFile = path
		base = filepath.Join(dir, strings.TrimSuffix(baseName, "-pending.po"))
		jsonFile = filepath.Join(dir, filepath.Base(base)+"-result.json")
	case strings.HasSuffix(baseName, "-output.po"):
		poFile = path
		base = filepath.Join(dir, strings.TrimSuffix(baseName, "-output.po"))
		jsonFile = filepath.Join(dir, filepath.Base(base)+"-result.json")
	case strings.HasSuffix(baseName, ".json"):
		jsonFile = path
		base = filepath.Join(dir, strings.TrimSuffix(baseName, ".json"))
	case strings.HasSuffix(baseName, ".po"):
		poFile = path
		base = filepath.Join(dir, strings.TrimSuffix(baseName, ".po"))
		jsonFile = filepath.Join(dir, filepath.Base(base)+".json")
	default:
		ps := ReviewPathSetFromBase(path)
		poForCount := ps.PendingPO
		if !Exist(poForCount) {
			poForCount = ps.OutputPO
		}
		return ResolvedReviewPaths{
			ReviewPathSet:  ps,
			JSONFile:       ps.ResultJSON,
			POFileForCount: poForCount,
		}
	}
	ps := ReviewPathSetFromBase(base)
	if jsonFile == "" {
		jsonFile = ps.ResultJSON
	}
	if poFile == "" {
		poFile = ps.PendingPO
		if !Exist(poFile) {
			poFile = ps.OutputPO
		}
		// Legacy: if path was .json, prefer review.po in same dir for backward compat
		if strings.HasSuffix(path, ".json") {
			legacyPO := filepath.Join(dir, filepath.Base(base)+".po")
			if Exist(legacyPO) {
				poFile = legacyPO
			}
		}
	}
	return ResolvedReviewPaths{
		ReviewPathSet:  ps,
		JSONFile:       jsonFile,
		POFileForCount: poFile,
	}
}

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
		issue := ReviewIssue{
			MsgID:         r.Get("msgid").String(),
			MsgStr:        r.Get("msgstr").String(),
			MsgIDPlural:   r.Get("msgid_plural").String(),
			Score:         int(r.Get("score").Int()),
			Description:   r.Get("description").String(),
			SuggestMsgstr: r.Get("suggest_msgstr").String(),
		}
		if s := r.Get("suggestion").String(); s != "" && issue.SuggestMsgstr == "" {
			issue.SuggestMsgstr = s
		}
		if arr := r.Get("msgstr_plural"); arr.Exists() && arr.IsArray() {
			for _, v := range arr.Array() {
				issue.MsgStrPlural = append(issue.MsgStrPlural, v.String())
			}
		}
		if arr := r.Get("suggest_msgstr_plural"); arr.Exists() && arr.IsArray() {
			for _, v := range arr.Array() {
				issue.SuggestMsgstrPlural = append(issue.SuggestMsgstrPlural, v.String())
			}
		}
		issues = append(issues, issue)
	}
	result := &ReviewJSONResult{TotalEntries: int(totalEntries), Issues: issues}
	normalizeReviewIssuesToPoFormat(result)
	return result
}

// ReportReviewFromJSON reads a review JSON file, optionally fills total_entries
// from a PO file when the JSON has none, and returns the report data.
// path may be: base (po/review), result JSON (po/review-result.json), or pending PO (po/review-pending.po).
// For Task 4 naming, uses review-pending.po or review-output.po for total count.
// Preprocesses LLM-generated JSON (BOM, markdown wrapping, extra text) before parsing.
func ReportReviewFromJSON(path string) (string, *ReviewReportResult, error) {
	resolved := resolveReviewPaths(path)
	data, err := os.ReadFile(resolved.JSONFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read review JSON %s: %w", resolved.JSONFile, err)
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
	normalizeReviewIssuesToPoFormat(&review)

	if Exist(resolved.POFileForCount) {
		stats, err := CountReportStats(resolved.POFileForCount)
		if err != nil {
			return "", nil, fmt.Errorf("failed to count entries in %s: %w", resolved.POFileForCount, err)
		}
		review.TotalEntries = stats.Total()
	} else {
		return "", nil, fmt.Errorf("file does not exist: %s", resolved.POFileForCount)
	}

	score, err := CalculateReviewScore(&review)
	if err != nil {
		return "", nil, fmt.Errorf("failed to calculate review score: %w", err)
	}

	critical, minor, major := CountReviewIssueScores(&review)
	return resolved.JSONFile, &ReviewReportResult{
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
	normalizeReviewIssuesToPoFormat(&review)
	return &review, nil
}

// ReportReviewFromPathWithBatches reports from review-result-*.json files or a single review JSON.
// Path is the base (e.g. "po/review"); uses review-pending.po/review-output.po for total count,
// review-result.json for merged output, review-result-*.json for batch files.
// If any files match "*-result-*.json", they are merged and saved to review-result.json.
// If no batch files exist, falls back to ReportReviewFromJSON(resultJSON path).
func ReportReviewFromPathWithBatches(path string) (string, *ReviewReportResult, error) {
	if path == "" {
		path = DefaultReviewBase
	}
	ps := ReviewPathSetFromBase(path)
	jsonFile := ps.ResultJSON
	// Use pending PO or output PO for total count (pending is source of truth)
	poFile := ps.PendingPO
	if !Exist(poFile) {
		poFile = ps.OutputPO
	}
	dir := filepath.Dir(jsonFile)
	base := strings.TrimSuffix(filepath.Base(jsonFile), ".json")
	pattern := filepath.Join(dir, base+"-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", nil, fmt.Errorf("glob %s: %w", pattern, err)
	}
	// Filter to only *-result-<N>.json (exclude review-result.json itself)
	var batchMatches []string
	for _, m := range matches {
		name := filepath.Base(m)
		if name != filepath.Base(jsonFile) && strings.HasPrefix(name, base+"-") {
			batchMatches = append(batchMatches, m)
		}
	}
	sort.Strings(batchMatches)

	log.Debugf("Call ReportReviewFromPathWithBatches(%s) with %d batch files",
		path, len(batchMatches))
	if len(batchMatches) == 0 {
		return reportReviewFromJSONWithPaths(jsonFile, poFile)
	}

	// Compare timestamps: if result JSON is newer than all batch files, read from it only.
	jsonStat, err := os.Stat(jsonFile)
	if err == nil {
		var maxBatchModTime int64
		for _, f := range batchMatches {
			fi, err := os.Stat(f)
			if err != nil {
				continue
			}
			if t := fi.ModTime().Unix(); t > maxBatchModTime {
				maxBatchModTime = t
			}
		}
		if jsonStat.ModTime().Unix() >= maxBatchModTime {
			return reportReviewFromJSONWithPaths(jsonFile, poFile)
		}
	}

	// Load batch files and merge; for duplicate msgid, AggregateReviewJSON keeps lower score.
	var batchReviews []*ReviewJSONResult
	for _, f := range batchMatches {
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
		return "", nil, fmt.Errorf("file does not exist: %s (need review-pending.po or review-output.po for total count)", poFile)
	}
	if err := saveReviewJSON(merged, jsonFile); err != nil {
		return "", nil, fmt.Errorf("failed to save aggregated review to %s: %w", jsonFile, err)
	}
	if err := applyReviewJSON(merged, ps); err != nil {
		return "", nil, fmt.Errorf("failed to apply review to %s: %w", ps.OutputPO, err)
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

// reportReviewFromJSONWithPaths reads jsonFile and fills total_entries from poFile.
func reportReviewFromJSONWithPaths(jsonFile, poFile string) (string, *ReviewReportResult, error) {
	if !Exist(jsonFile) {
		return "", nil, fmt.Errorf("file does not exist: %s", jsonFile)
	}
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read review JSON %s: %w", jsonFile, err)
	}
	var review ReviewJSONResult
	if err := json.Unmarshal(data, &review); err != nil {
		prepared := PrepareJSONForParse(data, err)
		if err2 := json.Unmarshal(prepared, &review); err2 != nil {
			if parsed := parseReviewJSONWithGjson(prepared, err2); parsed != nil {
				review = *parsed
			} else {
				return "", nil, fmt.Errorf("failed to parse review JSON: %w", err)
			}
		}
	}
	normalizeReviewIssuesToPoFormat(&review)
	if Exist(poFile) {
		stats, err := CountReportStats(poFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to count entries in %s: %w", poFile, err)
		}
		review.TotalEntries = stats.Total()
	} else {
		return "", nil, fmt.Errorf("file does not exist: %s (need for total_entries)", poFile)
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
