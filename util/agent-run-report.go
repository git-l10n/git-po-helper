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

// ReviewStatLabelWidth is the column width for left-aligned review statistic labels
// (PrintReviewReportResult, agent-test review display). Change this to adjust alignment.
var ReviewStatLabelWidth = 22

// CountReviewIssueScores returns counts by issue score from a review.
// ReviewIssueScoreCritical, ReviewIssueScoreMajor, ReviewIssueScoreMinor. Perfect count is derived: TotalEntries - (critical + major + minor).
func CountReviewIssueScores(review *ReviewJSONResult) (critical, major, minor int) {
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

// loadReviewJSONFromFile reads and parses a single review JSON file with the same
// robustness as GetReviewReport (BOM, markdown wrapping, gjson fallback).
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
	var batchReviews []*ReviewJSONResult
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
		merged = &ReviewJSONResult{Issues: []ReviewIssue{}}
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
func GetReviewReport() (*ReviewReport, error) {
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

	// Calculate review score
	score, err := CalculateReviewScore(review)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate review score: %w", err)
	}

	// Count review issue numbers
	critical, major, minor := CountReviewIssueScores(review)

	appliedFile := ""
	if Exist(ps.OutputPO) {
		appliedFile = ps.OutputPO
	}
	return &ReviewReport{
		ReviewResult:  review,
		Score:         score,
		CriticalCount: critical,
		MajorCount:    major,
		MinorCount:    minor,
		ReportFile:    jsonFile,
		AppliedFile:   appliedFile,
	}, nil
}

// WrapReviewReportForPrint builds an AgentRunResult with only ReviewReport fields set,
// for callers that have *ReviewReport (e.g. cmd agent-run report).
func WrapReviewReportForPrint(r *ReviewReport) *AgentRunResult {
	if r == nil {
		return nil
	}
	ar := &AgentRunResult{AgentExecuted: true}
	ar.ReviewResult = r.ReviewResult
	ar.Score = r.Score
	ar.CriticalCount = r.CriticalCount
	ar.MajorCount = r.MajorCount
	ar.MinorCount = r.MinorCount
	ar.ReportFile = r.ReportFile
	ar.AppliedFile = r.AppliedFile
	return ar
}

// PrintReviewReportResult prints "## Review Statistics" when ReviewResult is present,
// then agent execution / validation lines (PreValidationError, PostValidationError, runErr).
// ar is typically RunAgentReview's return value; runErr is result.RunError for agent-test.
// cmd/agent-run-report uses WrapReviewReportForPrint when only *ReviewReport is available.
func PrintReviewReportResult(ar *AgentRunResult, runErr error) {
	if ar == nil {
		return
	}
	w := ReviewStatLabelWidth

	// "## Review Statistics" block whenever review JSON was loaded
	if ar.ReviewResult != nil {
		fmt.Println("## Review Statistics")
		fmt.Println()
		fmt.Printf("  %-*s %d/100\n", w, "Review score:", ar.Score)
		fmt.Printf("  %-*s %d\n", w, "Total entries:", ar.ReviewResult.TotalEntries)
		fmt.Printf("  %-*s %d\n", w, "Perfect (no issue):", ar.PerfectCount())
		fmt.Printf("  %-*s %d\n", w, "With issues:", ar.ReviewResult.IssueCount())
		fmt.Println()
		fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Critical (score %d):", ReviewIssueScoreCritical), ar.CriticalCount)
		fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Major (score %d):", ReviewIssueScoreMajor), ar.MajorCount)
		fmt.Printf("  %-*s %d\n", w, fmt.Sprintf("Minor (score %d):", ReviewIssueScoreMinor), ar.MinorCount)
		fmt.Println()
		if ar.AppliedFile != "" {
			fmt.Printf("  %-*s %s\n", w, "Applied PO:", ar.AppliedFile)
		}
		if ar.ReportFile != "" {
			fmt.Printf("  %-*s %s\n", w, "Report JSON:", ar.ReportFile)
			fmt.Println()
			fmt.Println("For full review details, see the report JSON file")
		}
	}

	// Execution / validation (aligned with agent-test-review display)
	if ar.AgentExecuted {
		if runErr == nil {
			fmt.Printf("  %-*s %s\n", w, "Agent execution:", "PASS")
		} else {
			fmt.Printf("  %-*s FAIL - %v\n", w, "Agent execution:", runErr)
		}
		if ar.PostValidationError == nil {
			if ar.Score > 0 && ar.ReviewResult != nil {
				fmt.Printf("  %-*s PASS (%d/100)\n", w, "Validation:", ar.Score)
			} else {
				fmt.Printf("  %-*s FAIL (no valid JSON or score 0)\n", w, "Validation:")
			}
		} else {
			fmt.Printf("  %-*s FAIL - %s\n", w, "Post-validation:", ar.PostValidationError)
		}
	} else {
		if ar.PreValidationError != nil {
			fmt.Printf("  %-*s SKIPPED (pre-validation failed)\n", w, "Agent execution:")
			fmt.Printf("  %-*s %s\n", w, "Pre-validation:", ar.PreValidationError)
		} else {
			fmt.Printf("  %-*s SKIPPED\n", w, "Agent execution:")
		}
	}
}
