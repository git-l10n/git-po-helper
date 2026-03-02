package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCountReviewIssueScores(t *testing.T) {
	review := &ReviewJSONResult{
		TotalEntries: 10,
		Issues: []ReviewIssue{
			{Score: 0}, {Score: 0},
			{Score: 1},
			{Score: 2}, {Score: 2}, {Score: 2},
		},
	}
	critical, minor, major := CountReviewIssueScores(review)
	if critical != 2 || minor != 1 || major != 3 {
		t.Errorf("CountReviewIssueScores: critical=%d minor=%d major=%d; want 2,1,3",
			critical, minor, major)
	}
	// Perfect is derived: TotalEntries - (2+1+3) = 4
	if got := review.TotalEntries - (critical + minor + major); got != 4 {
		t.Errorf("derived perfect = TotalEntries - (critical+minor+major) = %d; want 4", got)
	}
}

func TestIssueCount(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *ReviewJSONResult
		if got := r.IssueCount(); got != 0 {
			t.Errorf("(*ReviewJSONResult)(nil).IssueCount() = %d; want 0", got)
		}
	})
	t.Run("empty issues", func(t *testing.T) {
		r := &ReviewJSONResult{TotalEntries: 5, Issues: []ReviewIssue{}}
		if got := r.IssueCount(); got != 0 {
			t.Errorf("IssueCount() = %d; want 0", got)
		}
	})
	t.Run("only score 3 excluded", func(t *testing.T) {
		r := &ReviewJSONResult{
			TotalEntries: 10,
			Issues: []ReviewIssue{
				{Score: 0}, {Score: 1}, {Score: 2}, {Score: 3}, {Score: 3},
			},
		}
		if got := r.IssueCount(); got != 3 {
			t.Errorf("IssueCount() = %d; want 3 (score 0,1,2 count; score 3 does not)", got)
		}
	})
	t.Run("all scores 0-2 count", func(t *testing.T) {
		r := &ReviewJSONResult{
			TotalEntries: 4,
			Issues:       []ReviewIssue{{Score: 0}, {Score: 1}, {Score: 2}, {Score: 2}},
		}
		if got := r.IssueCount(); got != 4 {
			t.Errorf("IssueCount() = %d; want 4", got)
		}
	})
}

// minimalPoWithEntries returns a valid PO file content with n translatable entries
// (header plus n msgid/msgstr pairs). CountPoReportStats(.).Total() will be n.
func minimalPoWithEntries(n int) string {
	const header = `# Translation file
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

`
	var b string
	for i := 0; i < n; i++ {
		b += "msgid \"entry" + string(rune('a'+i%26)) + "\"\nmsgstr \"\"\n\n"
	}
	return header + b
}

func TestReportReviewWithTotalEntries(t *testing.T) {
	// Create a review JSON with 2 issues
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
	poFile := filepath.Join(tmpDir, "review.po")
	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	// ReportReviewFromJSON requires the .po to exist; it sets TotalEntries from PO stats.
	if err := os.WriteFile(poFile, []byte(minimalPoWithEntries(2)), 0644); err != nil {
		t.Fatalf("write po failed: %v", err)
	}

	// Verify CalculateReviewScore works on the in-memory review
	score, err := CalculateReviewScore(review)
	if err != nil {
		t.Fatalf("CalculateReviewScore failed: %v", err)
	}
	if score < 95 || score > 100 {
		t.Errorf("expected score ~98, got %d", score)
	}

	_, result, err := ReportReviewFromJSON(jsonFile)
	if err != nil {
		t.Fatalf("ReportReviewFromJSON failed: %v", err)
	}
	// TotalEntries is taken from the PO file (2 entries), not from JSON
	if result.Review.TotalEntries != 2 {
		t.Errorf("expected TotalEntries 2 (from PO), got %d", result.Review.TotalEntries)
	}
	if len(result.Review.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Review.Issues))
	}
	// PerfectCount is derived: 2 - (1 critical + 1 major) = 0
	if got := result.PerfectCount(); got != 0 {
		t.Errorf("PerfectCount() = %d, want 0", got)
	}
}

func TestReportReviewMarkdownWrappedJSON(t *testing.T) {
	// JSON wrapped in markdown (common LLM output) - tests preprocessing
	validInMarkdown := "```json\n" + `{"total_entries": 5, "issues": [{"msgid": "x", "msgstr": "y", "score": 2, "description": "d", "suggestion": "s"}]}` + "\n```"
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "review.json")
	poFile := filepath.Join(tmpDir, "review.po")
	if err := os.WriteFile(jsonFile, []byte(validInMarkdown), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(poFile, []byte(minimalPoWithEntries(1)), 0644); err != nil {
		t.Fatalf("write po failed: %v", err)
	}
	_, result, err := ReportReviewFromJSON(jsonFile)
	if err != nil {
		t.Fatalf("ReportReviewFromJSON failed: %v", err)
	}
	// TotalEntries comes from the PO file (1 entry)
	if result.Review.TotalEntries != 1 {
		t.Errorf("expected TotalEntries 1 (from PO), got %d", result.Review.TotalEntries)
	}
	if len(result.Review.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Review.Issues))
	}
}
