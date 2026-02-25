package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReportReviewWithTotalEntries(t *testing.T) {
	// Create a review JSON with total_entries
	review := &ReviewJSONResult{
		TotalEntries: 100,
		Issues: []ReviewIssue{
			{MsgID: "commit", MsgStr: "承诺", Score: 0, Description: "term error", Suggestion: "提交"},
			{MsgID: "file", MsgStr: "文件", Score: 2, Description: "minor", Suggestion: "档案"},
		},
	}
	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "review.json")
	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Verify CalculateReviewScore works
	score, err := CalculateReviewScore(review)
	if err != nil {
		t.Fatalf("CalculateReviewScore failed: %v", err)
	}
	// 100 entries * 3 = 300 max. Issues: 0 deducts 3, 2 deducts 1. Total deduction = 4. Score = (300-4)*100/300 = 98
	if score < 95 || score > 100 {
		t.Errorf("expected score ~98, got %d", score)
	}

	// Test ReportReviewFromJSON with the written file
	result, err := ReportReviewFromJSON(jsonFile)
	if err != nil {
		t.Fatalf("ReportReviewFromJSON failed: %v", err)
	}
	if result.Review.TotalEntries != 100 {
		t.Errorf("expected TotalEntries 100, got %d", result.Review.TotalEntries)
	}
	if len(result.Review.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Review.Issues))
	}
}

func TestReportReviewMarkdownWrappedJSON(t *testing.T) {
	// JSON wrapped in markdown (common LLM output) - tests preprocessing
	validInMarkdown := "```json\n" + `{"total_entries": 5, "issues": [{"msgid": "x", "msgstr": "y", "score": 2, "description": "d", "suggestion": "s"}]}` + "\n```"
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "review.json")
	if err := os.WriteFile(jsonFile, []byte(validInMarkdown), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	result, err := ReportReviewFromJSON(jsonFile)
	if err != nil {
		t.Fatalf("ReportReviewFromJSON failed: %v", err)
	}
	if result.Review.TotalEntries != 5 {
		t.Errorf("expected TotalEntries 5, got %d", result.Review.TotalEntries)
	}
	if len(result.Review.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Review.Issues))
	}
}
