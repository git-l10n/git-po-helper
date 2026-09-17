// Package util provides review report statistics.
package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// DefaultReviewBase is the default base for review paths (po/review).
// Used by agent-run report when no path is given.
var DefaultReviewBase = filepath.Join(PoDir, "review")

// ReportLabelWidth is the column width for left-aligned labels in report output
// (agent-run Report, agent-test summary, review stats). Used with "  " prefix for alignment.
var ReportLabelWidth = 22

// reviewReportWrapWidth is the maximum rune width for review report text lines
// (Markdown output for agent-run review --report); long segments break at whitespace.
const reviewReportWrapWidth = 80

func sanitizeReviewReportText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// lastWhitespaceBreak returns an index into chunk (exclusive) such that chunk[:i]
// ends at a non-space rune before whitespace, or 0 when no space is found in (0,len(chunk)].
func lastWhitespaceBreak(chunk []rune) int {
	for i := len(chunk) - 1; i > 0; i-- {
		if unicode.IsSpace(chunk[i]) {
			return i
		}
	}
	return 0
}

// wrapPlainParagraph breaks s into lines of at most maxWidth runes, preferring breaks at spaces.
func wrapPlainParagraph(s string, maxWidth int) []string {
	s = sanitizeReviewReportText(s)
	if maxWidth <= 0 {
		maxWidth = reviewReportWrapWidth
	}
	vr := []rune(s)
	if len(vr) == 0 {
		return nil
	}
	var out []string
	for len(vr) > 0 {
		if len(vr) <= maxWidth {
			out = append(out, string(vr))
			break
		}
		chunk := vr[:maxWidth]
		br := lastWhitespaceBreak(chunk)
		if br <= 0 {
			br = maxWidth
		}
		line := strings.TrimSpace(string(vr[:br]))
		if line == "" {
			br = maxWidth
			line = strings.TrimSpace(string(vr[:br]))
		}
		out = append(out, line)
		vr = []rune(strings.TrimSpace(string(vr[br:])))
	}
	return out
}

// wrapPrefixValue formats value across lines: first line uses prefix, later lines use cont.
// Each output line is at most maxWidth runes; breaks prefer whitespace inside value.
func wrapPrefixValue(prefix, cont string, value string, maxWidth int) []string {
	value = sanitizeReviewReportText(value)
	if maxWidth <= 0 {
		maxWidth = reviewReportWrapWidth
	}
	pr := []rune(prefix)
	cr := []rune(cont)
	vr := []rune(value)
	if len(vr) == 0 {
		return []string{string(pr)}
	}
	var out []string
	for len(vr) > 0 {
		avail := maxWidth - len(pr)
		if avail < 1 {
			avail = 1
		}
		if len(vr) <= avail {
			out = append(out, string(pr)+string(vr))
			break
		}
		chunk := vr[:avail]
		br := lastWhitespaceBreak(chunk)
		if br <= 0 {
			br = avail
		}
		piece := strings.TrimSpace(string(vr[:br]))
		if piece == "" {
			br = avail
			piece = strings.TrimSpace(string(vr[:br]))
		}
		out = append(out, string(pr)+piece)
		vr = []rune(strings.TrimSpace(string(vr[br:])))
		pr = cr
	}
	return out
}

// reviewReportListIndent is two spaces before list markers (- or 1.) for report output.
const reviewReportListIndent = "  "

func printMarkdownReviewBullet(label, value string) {
	prefix := reviewReportListIndent + "- " + label + ": "
	cont := reviewReportListIndent + "  "
	for _, ln := range wrapPrefixValue(prefix, cont, value, reviewReportWrapWidth) {
		fmt.Println(ln)
	}
}

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
// Input is ps.InputPO (.json preferred over .po). Returns (applied, err): applied is true if
// any suggestion was applied. Skips apply if ps.OutputPO has the newest timestamp among
// ResultJSON, InputPO, and OutputPO.
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
// Input/output may be gettext JSON or PO (.json preferred when both exist).
// Returns *ReviewResult with Score, CriticalCount, MajorCount, MinorCount, ReportFile, AppliedFile set.
func GetReviewReport(pathName string) (*ReviewResult, error) {
	ps := GetReviewPathSet(pathName)

	if err := AggregateReviewBatches(ps); err != nil {
		return nil, err
	}

	// Apply review result to ps.OutputPO (.json or .po)
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

	// Set source for lazy init of TotalEntries/Score/counts (default ps.InputPO)
	poFile := ps.InputPO
	if !Exist(poFile) {
		poFile = ps.OutputPO
	}
	if !Exist(poFile) {
		return nil, fmt.Errorf("file does not exist: %s (need review-input.json or review-input.po for total_entries)", poFile)
	}
	review.SetReviewSource(poFile)
	appliedFile := ""
	if Exist(ps.OutputPO) {
		appliedFile = ps.OutputPO
	}
	review.SetReviewPaths(jsonFile, appliedFile)
	return review, nil
}

// PrintReviewReportResult prints a Markdown-style summary for agent-run review --report
// (bullets and numbered lists; section titles are plain text without '#' heading markers).
func PrintReviewReportResult(r *ReviewResult) {
	if r == nil {
		return
	}
	score, errScore := r.GetScore()
	totalEntries, errTotal := r.GetTotalEntries()
	if errScore != nil || errTotal != nil {
		fmt.Println("🔍 Review Report")
		fmt.Println()
		errMsg := ""
		if errTotal != nil {
			errMsg = fmt.Sprintf("Review init error: %v", errTotal)
		} else {
			errMsg = fmt.Sprintf("Review init error: %v", errScore)
		}
		for _, ln := range wrapPlainParagraph(errMsg, reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		fmt.Println()
		return
	}

	fmt.Println("🔍 Review Report")
	fmt.Println()
	printMarkdownReviewBullet("Review score", fmt.Sprintf("%d/100", score))
	printMarkdownReviewBullet("Total entries", fmt.Sprintf("%d", totalEntries))
	printMarkdownReviewBullet("Perfect (no issue)", fmt.Sprintf("%d", r.PerfectCount()))
	printMarkdownReviewBullet("With issues", fmt.Sprintf("%d", r.IssueCount()))
	fmt.Println()
	critical, _ := r.GetCriticalCount()
	major, _ := r.GetMajorCount()
	minor, _ := r.GetMinorCount()
	printMarkdownReviewBullet(fmt.Sprintf("Critical (score %d)", ReviewIssueScoreCritical), fmt.Sprintf("%d", critical))
	printMarkdownReviewBullet(fmt.Sprintf("Major (score %d)", ReviewIssueScoreMajor), fmt.Sprintf("%d", major))
	printMarkdownReviewBullet(fmt.Sprintf("Minor (score %d)", ReviewIssueScoreMinor), fmt.Sprintf("%d", minor))
	fmt.Println()
	appliedFile, _ := r.GetAppliedFile()
	if appliedFile != "" {
		printMarkdownReviewBullet("Applied file", appliedFile)
	}
	reportFile, _ := r.GetReportFile()
	if reportFile != "" {
		printMarkdownReviewBullet("Report JSON", reportFile)
		printReviewIssueDetails(r)
		fmt.Println()
		for _, ln := range wrapPlainParagraph("For full review details, see the report JSON file", reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		fmt.Println()
	}
}

func printReviewIssueDetails(r *ReviewResult) {
	if r == nil || len(r.Issues) == 0 {
		return
	}

	var issues []ReviewIssue
	for _, issue := range r.Issues {
		if issue.Score < ReviewIssueScorePerfect {
			issues = append(issues, issue)
		}
	}
	if len(issues) == 0 {
		return
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Score != issues[j].Score {
			return issues[i].Score < issues[j].Score
		}
		return issues[i].MsgID < issues[j].MsgID
	})

	fmt.Println()
	fmt.Printf("⚠️ Issues (score < %d)\n", ReviewIssueScorePerfect)
	fmt.Println()
	for i, issue := range issues {
		marker := fmt.Sprintf("%s%d. ", reviewReportListIndent, i+1)
		subIndent := strings.Repeat(" ", utf8.RuneCountInString(marker))
		cont := subIndent + "  "
		title := strconv.Quote(issue.MsgID)
		for _, ln := range wrapPrefixValue(marker, cont, title, reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		for _, ln := range wrapPrefixValue(subIndent+"- score: ", cont, strconv.Itoa(issue.Score), reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		for _, ln := range wrapPrefixValue(subIndent+"- description: ", cont, issue.Description, reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		if issue.MsgIDPlural != "" {
			for _, ln := range wrapPrefixValue(subIndent+"- msgid-plural: ", cont, strconv.Quote(issue.MsgIDPlural), reviewReportWrapWidth) {
				fmt.Println(ln)
			}
		}
		for _, ln := range wrapPrefixValue(subIndent+"- suggest-msgstr: ", cont, formatMsgstrForDisplay(issue.SuggestMsgstr), reviewReportWrapWidth) {
			fmt.Println(ln)
		}
		if i < len(issues)-1 {
			fmt.Println()
		}
	}
}

func formatMsgstrForDisplay(msgstr []string) string {
	if len(msgstr) == 0 {
		return "(empty)"
	}
	if len(msgstr) == 1 {
		return fmt.Sprintf("%q", msgstr[0])
	}
	parts := make([]string, 0, len(msgstr))
	for i, s := range msgstr {
		parts = append(parts, fmt.Sprintf("[%d]=%q", i, s))
	}
	return strings.Join(parts, ", ")
}
