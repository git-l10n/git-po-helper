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

// ReportLabelWidth is the column width for left-aligned labels in report output
// (agent-run Report, agent-test summary, review stats). Used with "  " prefix for alignment.
var ReportLabelWidth = 22

// CountReviewIssueScores returns counts by issue score from a review.
// ReviewIssueScoreCritical, ReviewIssueScoreMajor, ReviewIssueScoreMinor. Perfect count is derived: TotalEntries - (critical + major + minor).
func CountReviewIssueScores(review *ReviewResult) (critical, major, minor int) {
	for _, issue := range review.Issues {
		switch issue.Score {
		case ReviewIssueScoreCritical:
			critical++
		case ReviewIssueScoreMajor:
			major++
		case ReviewIssueScoreMinor:
			minor++
		}
	}
	return critical, major, minor
}

// parseReviewJSONWithGjson parses review JSON using gjson, which can tolerate
// some malformed LLM output (e.g. missing colons). Returns nil if parsing fails.
func parseReviewJSONWithGjson(data []byte, err error) *ReviewResult {
	log.Warnf("fall back to gjson to fix json: %v", err)
	totalEntries := gjson.GetBytes(data, "total_entries").Int()
	issuesResult := gjson.GetBytes(data, "issues")
	if !issuesResult.Exists() {
		if totalEntries == 0 {
			return nil
		}
		return &ReviewResult{TotalEntries: int(totalEntries), Issues: nil}
	}
	var issues []ReviewIssue
	for _, r := range issuesResult.Array() {
		issue := ReviewIssue{
			MsgID:       r.Get("msgid").String(),
			MsgIDPlural: r.Get("msgid_plural").String(),
			Score:       int(r.Get("score").Int()),
			Description: r.Get("description").String(),
		}
		if arr := r.Get("msgstr"); arr.Exists() {
			if arr.IsArray() {
				for _, v := range arr.Array() {
					issue.MsgStr = append(issue.MsgStr, v.String())
				}
			} else if s := arr.String(); s != "" {
				issue.MsgStr = []string{s}
			}
		}
		if arr := r.Get("suggest_msgstr"); arr.Exists() {
			if arr.IsArray() {
				for _, v := range arr.Array() {
					issue.SuggestMsgstr = append(issue.SuggestMsgstr, v.String())
				}
			} else if s := arr.String(); s != "" {
				issue.SuggestMsgstr = []string{s}
			}
		}
		if len(issue.SuggestMsgstr) == 0 {
			if s := r.Get("suggestion").String(); s != "" {
				issue.SuggestMsgstr = []string{s}
			}
		}
		issues = append(issues, issue)
	}
	result := &ReviewResult{TotalEntries: int(totalEntries), Issues: issues}
	normalizeReviewIssuesToPoFormat(result)
	return result
}

// DecodeReviewJSONBytes parses review JSON from bytes using the same pipeline as
// loadReviewJSONFromFile: json.Unmarshal (ReviewIssue.UnmarshalJSON normalizes
// msgstr/suggest_msgstr string or array), PrepareJSONForParse retry, then gjson
// fallback. Ensures Issues is non-nil and runs normalizeReviewIssuesToPoFormat.
// All review JSON loading should go through this or ParseReviewJSON (which uses it).
func DecodeReviewJSONBytes(data []byte) (*ReviewResult, error) {
	var review ReviewResult
	if err := json.Unmarshal(data, &review); err != nil {
		prepared := PrepareJSONForParse(data, err)
		if err2 := json.Unmarshal(prepared, &review); err2 != nil {
			if parsed := parseReviewJSONWithGjson(prepared, err2); parsed != nil {
				return parsed, nil
			}
			return nil, fmt.Errorf("decode review JSON: %w", err)
		}
	}
	if review.Issues == nil {
		review.Issues = []ReviewIssue{}
	}
	normalizeReviewIssuesToPoFormat(&review)
	return &review, nil
}

// loadReviewJSONFromFile reads and parses a single review JSON file with the same
// robustness as GetReviewReport (BOM, markdown wrapping, gjson fallback).
// It does not fill TotalEntries from a PO file.
func loadReviewJSONFromFile(jsonFile string) (*ReviewResult, error) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", jsonFile, err)
	}
	review, err := DecodeReviewJSONBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review JSON %s: %w", jsonFile, err)
	}
	return review, nil
}

// AggregateReviewBatches finds *-result-<N>.json batch files, checks timestamps,
// and if aggregation is needed, loads them, merges (same msgid takes lowest score),
// and saves to ps.ResultJSON. Returns merged result when aggregation was performed,
// or (nil, nil) when no aggregation needed (no batch files or result JSON is newer).
func AggregateReviewBatches(ps ReviewPathSet) error {
	resultJSONFile := ps.ResultJSON
	dir := filepath.Dir(resultJSONFile)
	base := strings.TrimSuffix(filepath.Base(resultJSONFile), ".json")
	pattern := filepath.Join(dir, base+"-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}
	// Filter to only *-result-<N>.json (exclude review-result.json itself)
	var batchMatches []string
	for _, m := range matches {
		name := filepath.Base(m)
		if name != filepath.Base(resultJSONFile) && strings.HasPrefix(name, base+"-") {
			batchMatches = append(batchMatches, m)
		}
	}
	sort.Strings(batchMatches)

	log.Debugf("AggregateReviewBatches: %d batch files", len(batchMatches))
	if len(batchMatches) == 0 {
		return nil
	}

	// Compare timestamps: if result JSON is newer than all batch files, skip aggregation.
	resultJSONStat, err := os.Stat(resultJSONFile)
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
		if resultJSONStat.ModTime().Unix() >= maxBatchModTime {
			return nil
		}
	}

	// Load batch files and merge; for duplicate msgid, AggregateReviewJSON keeps lower score.
	var batchReviews []*ReviewResult
	for _, f := range batchMatches {
		r, err := loadReviewJSONFromFile(f)
		if err != nil {
			return fmt.Errorf("failed to load review JSON from %s: %w", f, err)
		}
		if r != nil {
			batchReviews = append(batchReviews, r)
		}
	}
	merged := aggregateReviewJSONResult(batchReviews, true)
	if merged == nil {
		merged = &ReviewResult{Issues: []ReviewIssue{}}
	}
	if err := saveReviewJSON(merged, resultJSONFile); err != nil {
		return fmt.Errorf("failed to save aggregated review to %s: %w", resultJSONFile, err)
	}
	return nil
}

// ApplyReviewFromResultJSON reads review from ps.ResultJSON and applies suggestions to ps.OutputPO.
// Input PO is ps.InputPO. Returns (applied, err): applied is true if any suggestion was applied.
// Skips apply if ps.OutputPO has the newest timestamp among ResultJSON, InputPO, and OutputPO.
func ApplyReviewFromResultJSON(ps ReviewPathSet) (bool, error) {
	outputStat, err := os.Stat(ps.OutputPO)
	if err == nil {
		outputMod := outputStat.ModTime().Unix()
		if jsonStat, err := os.Stat(ps.ResultJSON); err == nil && jsonStat.ModTime().Unix() <= outputMod {
			if inputStat, err := os.Stat(ps.InputPO); err == nil && inputStat.ModTime().Unix() <= outputMod {
				return false, nil
			}
		}
	}
	review, err := loadReviewJSONFromFile(ps.ResultJSON)
	if err != nil {
		return false, err
	}
	return applyReviewJSON(review, ps.InputPO, ps.OutputPO)
}

// GetReviewReport reads ps.ResultJSON and fills total_entries from ps.InputPO (or ps.OutputPO).
// Returns *ReviewJSONResult with Score, CriticalCount, MajorCount, MinorCount, ReportFile, AppliedFile set.
func GetReviewReport() (*ReviewResult, error) {
	ps := GetReviewPathSet()

	if err := AggregateReviewBatches(ps); err != nil {
		return nil, err
	}

	// Apply review result to ps.OutputPO
	if _, err := ApplyReviewFromResultJSON(ps); err != nil {
		return nil, fmt.Errorf("failed to apply review to %s: %w", ps.OutputPO, err)
	}

	// Load review result from ps.ResultJSON
	jsonFile := ps.ResultJSON
	if !Exist(jsonFile) {
		return nil, fmt.Errorf("file does not exist: %s", jsonFile)
	}
	review, err := loadReviewJSONFromFile(jsonFile)
	if err != nil {
		return nil, err
	}

	// Get TotalEntries from ps.InputPO or ps.OutputPO, and fill it to review
	poFile := ps.InputPO
	if !Exist(poFile) {
		poFile = ps.OutputPO
	}
	if Exist(poFile) {
		stats, err := GetPoStats(poFile)
		if err != nil {
			return nil, fmt.Errorf("failed to count entries in %s: %w", poFile, err)
		}
		review.TotalEntries = stats.Total()
	} else {
		return nil, fmt.Errorf("file does not exist: %s (need review-input.po for total_entries)", poFile)
	}

	// Calculate review score and issue counts
	score, err := CalculateReviewScore(review)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate review score: %w", err)
	}
	critical, major, minor := CountReviewIssueScores(review)
	review.Score = score
	review.CriticalCount = critical
	review.MajorCount = major
	review.MinorCount = minor
	review.ReportFile = jsonFile
	if Exist(ps.OutputPO) {
		review.AppliedFile = ps.OutputPO
	}
	return review, nil
}

// PrintReviewReportResult prints "## Review Statistics" when the result has content.
func PrintReviewReportResult(r *ReviewResult) {
	if r == nil {
		return
	}
	w := ReportLabelWidth

	fmt.Println("🔍 Review Report")
	fmt.Println()
	fmt.Printf("  %-*s %d/100\n", w, "Review score:", r.Score)
	fmt.Printf("  %-*s %d\n", w, "Total entries:", r.TotalEntries)
	fmt.Printf("  %-*s %d\n", w, "Perfect (no issue):", r.PerfectCount())
	fmt.Printf("  %-*s %d\n", w, "With issues:", r.IssueCount())
	fmt.Println()
	fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Critical (score %d):", ReviewIssueScoreCritical), r.CriticalCount)
	fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Major (score %d):", ReviewIssueScoreMajor), r.MajorCount)
	fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Minor (score %d):", ReviewIssueScoreMinor), r.MinorCount)
	fmt.Println()
	if r.AppliedFile != "" {
		fmt.Printf("  %-*s %s\n", w, "Applied PO:", r.AppliedFile)
	}
	if r.ReportFile != "" {
		fmt.Printf("  %-*s %s\n", w, "Report JSON:", r.ReportFile)
		fmt.Println()
		fmt.Println("For full review details, see the report JSON file")
		fmt.Println()
	}
}
