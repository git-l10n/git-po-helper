package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func minimalReviewInputPO() string {
	return `# Translation file
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "hello"
msgstr "你好"
`
}

func TestAgentRunReviewReportFlagMutualExclusion(t *testing.T) {
	opts := &agentRunOptions{}
	cmd := newAgentRunReviewCmd(opts)
	cmd.SetArgs([]string{"--report", "po", "--since", "HEAD"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --report is used with --since")
	}
	if !strings.Contains(err.Error(), "--report cannot be used with --since") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentRunReviewReportFlagNoPositionalArgs(t *testing.T) {
	opts := &agentRunOptions{}
	cmd := newAgentRunReviewCmd(opts)
	cmd.SetArgs([]string{"--report", "po", "po/zh_CN.po"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --report is used with positional args")
	}
	if !strings.Contains(err.Error(), "does not accept positional arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentRunReviewReportFlagRunsReport(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}

	reportDir := filepath.Join(tmpDir, "po")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		t.Fatalf("MkdirAll %s: %v", reportDir, err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "review-input.po"), []byte(minimalReviewInputPO()), 0644); err != nil {
		t.Fatalf("write review-input.po: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "review-result.json"), []byte(`{"total_entries":1,"issues":[]}`), 0644); err != nil {
		t.Fatalf("write review-result.json: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	opts := &agentRunOptions{}
	cmd := newAgentRunReviewCmd(opts)
	cmd.SetArgs([]string{"--report", "po"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	_ = w.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(r)
	output := out.String()
	if !strings.Contains(output, "Review Report") {
		t.Fatalf("expected report output, got: %s", output)
	}
}
