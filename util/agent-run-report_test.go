package util

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCountReviewIssueScores(t *testing.T) {
	review := &ReviewResult{
		TotalEntries: 10,
		Issues: []ReviewIssue{
			{Score: 0}, {Score: 0},
			{Score: 1},
			{Score: 2}, {Score: 2}, {Score: 2},
		},
	}
	critical, major, minor := CountReviewIssueScores(review)
	if critical != 2 || major != 1 || minor != 3 {
		t.Errorf("CountReviewIssueScores: critical=%d major=%d minor=%d; want 2,1,3",
			critical, major, minor)
	}
	// Perfect is derived: TotalEntries - (2+1+3) = 4
	if got := review.TotalEntries - (critical + major + minor); got != 4 {
		t.Errorf("derived perfect = TotalEntries - (critical+minor+major) = %d; want 4", got)
	}
}

func TestIssueCount(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var r *ReviewResult
		if got := r.IssueCount(); got != 0 {
			t.Errorf("(*ReviewJSONResult)(nil).IssueCount() = %d; want 0", got)
		}
	})
	t.Run("empty issues", func(t *testing.T) {
		r := &ReviewResult{TotalEntries: 5, Issues: []ReviewIssue{}}
		if got := r.IssueCount(); got != 0 {
			t.Errorf("IssueCount() = %d; want 0", got)
		}
	})
	t.Run("only score 3 excluded", func(t *testing.T) {
		r := &ReviewResult{
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
		r := &ReviewResult{
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
	review := &ReviewResult{
		TotalEntries: 100,
		Issues: []ReviewIssue{
			{MsgID: "commit", Score: 0, Description: "term error", SuggestMsgstr: []string{"提交"}},
			{MsgID: "file", Score: 2, Description: "minor", SuggestMsgstr: []string{"档案"}},
		},
	}
	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll po: %v", err)
	}
	ps := GetReviewPathSet("po")
	if err := os.WriteFile(ps.ResultJSON, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	// GetReviewReport requires the .po to exist; it sets TotalEntries from PO stats.
	if err := os.WriteFile(ps.InputPO, []byte(minimalPoWithEntries(2)), 0644); err != nil {
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

	result, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport failed: %v", err)
	}
	// TotalEntries is taken from the PO file (2 entries), not from JSON
	totalEntries, err := result.GetTotalEntries()
	if err != nil {
		t.Fatalf("GetTotalEntries: %v", err)
	}
	if totalEntries != 2 {
		t.Errorf("expected TotalEntries 2 (from PO), got %d", totalEntries)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
	// PerfectCount is derived: 2 - (1 critical + 1 major) = 0
	if got := result.PerfectCount(); got != 0 {
		t.Errorf("PerfectCount() = %d, want 0", got)
	}
}

// TestApplyReviewJSONWithEscapes verifies that JSON escape sequences (\n, \t, etc.)
// are correctly converted to PO format so entries can be matched and suggestions applied.
func TestApplyReviewJSONWithEscapes(t *testing.T) {
	// PO format uses literal \n (backslash+n); JSON uses \n for newline.
	// After parsing, normalizeReviewIssuesToPoFormat converts JSON newline to PO \n.
	inputPO := `# Translation
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "line1\nline2"
msgstr "old"
`
	// JSON: "line1\nline2" decodes to line1+newline+line2; after normalize -> "line1\nline2" (PO format)
	reviewJSON := `{"total_entries": 1, "issues": [{"msgid": "line1\nline2", "score": 0, "description": "fix", "suggest_msgstr": "new1\nnew2"}]}`

	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll po: %v", err)
	}
	ps := GetReviewPathSet("po")
	if err := os.WriteFile(ps.InputPO, []byte(inputPO), 0644); err != nil {
		t.Fatalf("write input PO: %v", err)
	}
	if err := os.WriteFile(ps.ResultJSON, []byte(reviewJSON), 0644); err != nil {
		t.Fatalf("write review JSON: %v", err)
	}

	review, err := loadReviewJSONFromFile(ps.ResultJSON)
	if err != nil {
		t.Fatalf("loadReviewJSONFromFile: %v", err)
	}
	if _, err := applyReviewJSON(review, ps.InputPO, ps.OutputPO); err != nil {
		t.Fatalf("applyReviewJSON: %v", err)
	}

	outData, err := os.ReadFile(ps.OutputPO)
	if err != nil {
		t.Fatalf("read output PO: %v", err)
	}
	outStr := string(outData)
	// PO format stores newline as literal \n (backslash+n); output should have the applied suggestion
	if !strings.Contains(outStr, "new1") || !strings.Contains(outStr, "new2") {
		t.Errorf("output PO should contain applied suggestion; got:\n%s", outStr)
	}
}

func TestDecodeReviewJSONBytes_MsgstrAndSuggestStringCompat(t *testing.T) {
	// LLM may emit msgstr / suggest_msgstr as strings; must normalize to []string.
	jsonStr := `{"total_entries": 2, "issues": [
		{"msgid": "a", "msgstr": "旧", "score": 1, "description": "d1", "suggest_msgstr": "新"},
		{"msgid": "b", "msgstr": ["x", "y"], "score": 2, "description": "d2", "suggest_msgstr": ["p", "q"]}
	]}`
	r, err := DecodeReviewJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("DecodeReviewJSONBytes: %v", err)
	}
	if len(r.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(r.Issues))
	}
	if len(r.Issues[0].MsgStr) != 1 || r.Issues[0].MsgStr[0] != "旧" {
		t.Errorf("issue0 MsgStr: got %#v", r.Issues[0].MsgStr)
	}
	if len(r.Issues[0].SuggestMsgstr) != 1 || r.Issues[0].SuggestMsgstr[0] != "新" {
		t.Errorf("issue0 SuggestMsgstr: got %#v", r.Issues[0].SuggestMsgstr)
	}
	if len(r.Issues[1].MsgStr) != 2 || r.Issues[1].SuggestMsgstr[0] != "p" {
		t.Errorf("issue1 arrays: MsgStr=%#v SuggestMsgstr=%#v", r.Issues[1].MsgStr, r.Issues[1].SuggestMsgstr)
	}
	// ParseReviewJSON uses DecodeReviewJSONBytes then validates
	r2, err := ParseReviewJSON([]byte(`{"total_entries": 1, "issues": [{"msgid": "m", "msgstr": "s", "score": 0, "description": "ok", "suggest_msgstr": "t"}]}`))
	if err != nil {
		t.Fatalf("ParseReviewJSON string fields: %v", err)
	}
	if len(r2.Issues) != 1 || len(r2.Issues[0].SuggestMsgstr) != 1 || r2.Issues[0].SuggestMsgstr[0] != "t" {
		t.Errorf("ParseReviewJSON: got %#v", r2.Issues[0].SuggestMsgstr)
	}
}

func TestReportReviewMarkdownWrappedJSON(t *testing.T) {
	// JSON wrapped in markdown (common LLM output) - tests preprocessing
	validInMarkdown := "```json\n" + `{"total_entries": 5, "issues": [{"msgid": "x", "score": 2, "description": "d", "suggest_msgstr": ["s"]}]}` + "\n```"
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll po: %v", err)
	}
	ps := GetReviewPathSet("po")
	if err := os.WriteFile(ps.ResultJSON, []byte(validInMarkdown), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(ps.InputPO, []byte(minimalPoWithEntries(1)), 0644); err != nil {
		t.Fatalf("write po failed: %v", err)
	}
	result, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport failed: %v", err)
	}
	// TotalEntries comes from the PO file (1 entry)
	totalEntries, err := result.GetTotalEntries()
	if err != nil {
		t.Fatalf("GetTotalEntries: %v", err)
	}
	if totalEntries != 1 {
		t.Errorf("expected TotalEntries 1 (from PO), got %d", totalEntries)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestPrintReviewReportResult_IncludesIssueDetails(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll po: %v", err)
	}
	ps := GetReviewPathSet("po")
	if err := os.WriteFile(ps.InputPO, []byte(minimalPoWithEntries(2)), 0644); err != nil {
		t.Fatalf("write po failed: %v", err)
	}
	reviewJSON := `{"total_entries":2,"issues":[{"msgid":"bad 1","score":2,"description":"desc2","suggest_msgstr":["fix2"]},{"msgid":"bad 0","score":0,"description":"desc0","suggest_msgstr":["fix0"]},{"msgid":"ok","score":3,"description":"desc3","suggest_msgstr":["fix3"]}]}`
	if err := os.WriteFile(ps.ResultJSON, []byte(reviewJSON), 0644); err != nil {
		t.Fatalf("write review-result.json: %v", err)
	}

	result, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport failed: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	PrintReviewReportResult(result)
	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	output := out.String()
	if !strings.Contains(output, "🔍 Review Report") || strings.Contains(output, "## ") {
		t.Fatalf("expected plain title line without '#', got: %s", output)
	}
	if !strings.Contains(output, "  - Review score:") || !strings.Contains(output, "  - Total entries:") {
		t.Fatalf("expected indented markdown list bullets, got: %s", output)
	}
	if !strings.Contains(output, "Issues (score < 3)") || strings.Contains(output, "### ") {
		t.Fatalf("expected plain issues section title without '#', got: %s", output)
	}
	if !strings.Contains(output, `  1. "bad 0"`) || !strings.Contains(output, `  2. "bad 1"`) {
		t.Fatalf("expected indented ordered list titles as quoted msgid, got: %s", output)
	}
	if !strings.Contains(output, "- score: 0") || !strings.Contains(output, "- score: 2") {
		t.Fatalf("expected score as first sub-item, got: %s", output)
	}
	if !strings.Contains(output, "description: desc0") ||
		!strings.Contains(output, `suggest-msgstr: "fix0"`) {
		t.Fatalf("expected score<3 issue details in output, got: %s", output)
	}
	if strings.Contains(output, `  1. "ok"`) || strings.Contains(output, `suggest-msgstr: "fix3"`) {
		t.Fatalf("did not expect score=3 issue in details, got: %s", output)
	}
}

func TestWrapPrefixValue_respectsWidth(t *testing.T) {
	prefix := "  - Report JSON: "
	cont := "    "
	long := strings.Repeat("abcdef ", 20) // many short tokens with spaces
	lines := wrapPrefixValue(prefix, cont, long, reviewReportWrapWidth)
	for _, ln := range lines {
		if utf8.RuneCountInString(ln) > reviewReportWrapWidth {
			t.Errorf("line longer than %d runes: %q", reviewReportWrapWidth, ln)
		}
	}
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines for long value, got %d lines", len(lines))
	}
}

func TestPrintReviewReportResult_longDescriptionWrapped(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll po: %v", err)
	}
	ps := GetReviewPathSet("po")
	if err := os.WriteFile(ps.InputPO, []byte(minimalPoWithEntries(1)), 0644); err != nil {
		t.Fatalf("write po failed: %v", err)
	}
	longDesc := strings.Repeat("word ", 30) // > 80 runes, breaks at spaces
	reviewJSON := `{"total_entries":1,"issues":[{"msgid":"x","score":0,"description":` + jsonString(longDesc) + `,"suggest_msgstr":["y"]}]}`
	if err := os.WriteFile(ps.ResultJSON, []byte(reviewJSON), 0644); err != nil {
		t.Fatalf("write review-result.json: %v", err)
	}
	result, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport failed: %v", err)
	}
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	PrintReviewReportResult(result)
	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	output := out.String()
	for _, ln := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if utf8.RuneCountInString(ln) > reviewReportWrapWidth {
			t.Errorf("line exceeds %d runes: %q", reviewReportWrapWidth, ln)
		}
	}
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// minimalGettextJSONWithEntries returns gettext JSON with n content entries.
func minimalGettextJSONWithEntries(n int) string {
	entries := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := "entry" + string(rune('a'+i%26))
		entries = append(entries, `{"msgid":`+jsonString(id)+`,"msgstr":[""]}`)
	}
	return `{"header_comment":"","header_meta":"Content-Type: text/plain; charset=UTF-8\n","entries":[` +
		strings.Join(entries, ",") + `]}`
}

func TestGetReviewPathSet_PreferJSONOverPO(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	t.Run("neither exists defaults to po", func(t *testing.T) {
		ps := GetReviewPathSet("po")
		if !strings.HasSuffix(ps.InputPO, "review-input.po") {
			t.Errorf("InputPO = %q, want .../review-input.po", ps.InputPO)
		}
		if !strings.HasSuffix(ps.OutputPO, "review-output.po") {
			t.Errorf("OutputPO = %q, want .../review-output.po", ps.OutputPO)
		}
	})

	t.Run("json preferred when both exist", func(t *testing.T) {
		if err := os.WriteFile("po/review-input.po", []byte(minimalPoWithEntries(1)), 0644); err != nil {
			t.Fatalf("write po: %v", err)
		}
		if err := os.WriteFile("po/review-input.json", []byte(minimalGettextJSONWithEntries(1)), 0644); err != nil {
			t.Fatalf("write json: %v", err)
		}
		if err := os.WriteFile("po/review-output.po", []byte(minimalPoWithEntries(1)), 0644); err != nil {
			t.Fatalf("write output po: %v", err)
		}
		if err := os.WriteFile("po/review-output.json", []byte(minimalGettextJSONWithEntries(1)), 0644); err != nil {
			t.Fatalf("write output json: %v", err)
		}
		ps := GetReviewPathSet("po")
		if !strings.HasSuffix(ps.InputPO, "review-input.json") {
			t.Errorf("InputPO = %q, want .../review-input.json", ps.InputPO)
		}
		if !strings.HasSuffix(ps.OutputPO, "review-output.json") {
			t.Errorf("OutputPO = %q, want .../review-output.json", ps.OutputPO)
		}
	})

	t.Run("json input implies json output default", func(t *testing.T) {
		_ = os.Remove("po/review-input.po")
		_ = os.Remove("po/review-output.po")
		_ = os.Remove("po/review-output.json")
		if err := os.WriteFile("po/review-input.json", []byte(minimalGettextJSONWithEntries(1)), 0644); err != nil {
			t.Fatalf("write json: %v", err)
		}
		ps := GetReviewPathSet("po")
		if !strings.HasSuffix(ps.InputPO, "review-input.json") {
			t.Errorf("InputPO = %q, want .../review-input.json", ps.InputPO)
		}
		if !strings.HasSuffix(ps.OutputPO, "review-output.json") {
			t.Errorf("OutputPO = %q, want .../review-output.json", ps.OutputPO)
		}
	})
}

func TestGetReviewReport_JSONInputAndOutput(t *testing.T) {
	review := &ReviewResult{
		TotalEntries: 2,
		Issues: []ReviewIssue{
			{MsgID: "entrya", Score: 0, Description: "fix", SuggestMsgstr: []string{"修好"}},
		},
	}
	data, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile("po/review-input.json", []byte(minimalGettextJSONWithEntries(2)), 0644); err != nil {
		t.Fatalf("write input json: %v", err)
	}
	if err := os.WriteFile("po/review-result.json", data, 0644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	result, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport: %v", err)
	}
	total, err := result.GetTotalEntries()
	if err != nil {
		t.Fatalf("GetTotalEntries: %v", err)
	}
	if total != 2 {
		t.Errorf("TotalEntries = %d, want 2", total)
	}
	applied, err := result.GetAppliedFile()
	if err != nil {
		t.Fatalf("GetAppliedFile: %v", err)
	}
	if !strings.HasSuffix(applied, "review-output.json") {
		t.Errorf("AppliedFile = %q, want .../review-output.json", applied)
	}
	outData, err := os.ReadFile(applied)
	if err != nil {
		t.Fatalf("read applied: %v", err)
	}
	if !strings.Contains(string(outData), "修好") {
		t.Errorf("output JSON missing applied suggestion; got:\n%s", outData)
	}
	// Prefer json over po when both exist for source count
	if err := os.WriteFile("po/review-input.po", []byte(minimalPoWithEntries(9)), 0644); err != nil {
		t.Fatalf("write po: %v", err)
	}
	result2, err := GetReviewReport("po")
	if err != nil {
		t.Fatalf("GetReviewReport with both: %v", err)
	}
	total2, err := result2.GetTotalEntries()
	if err != nil {
		t.Fatalf("GetTotalEntries: %v", err)
	}
	if total2 != 2 {
		t.Errorf("with both formats, TotalEntries = %d, want 2 from json", total2)
	}
}
