package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/config"
)

func TestValidatePoEntryCount_Disabled(t *testing.T) {
	var expectedNil *int
	if err := ValidatePoEntryCount("/nonexistent/file.po", expectedNil, "before update"); err != nil {
		t.Errorf("Expected no error when validation is disabled with nil expectedCount, got %v", err)
	}

	zero := 0
	if err := ValidatePoEntryCount("/nonexistent/file.po", &zero, "after update"); err != nil {
		t.Errorf("Expected no error when validation is disabled with zero expectedCount, got %v", err)
	}
}

func TestValidatePoEntryCount_BeforeUpdateMissingFile(t *testing.T) {
	expected := 1
	if err := ValidatePoEntryCount("/nonexistent/file.po", &expected, "before update"); err == nil {
		t.Errorf("Expected error when file is missing and expectedCount is non-zero in before update stage, got nil")
	}
}

func TestValidatePoEntryCount_AfterUpdateMissingFile(t *testing.T) {
	expected := 1
	if err := ValidatePoEntryCount("/nonexistent/file.po", &expected, "after update"); err == nil {
		t.Errorf("Expected error when file is missing in after update stage, got nil")
	}
}

func TestValidatePoEntryCount_MatchingAndNonMatching(t *testing.T) {
	const poContent = `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First string"
msgstr ""
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("Failed to create test PO file: %v", err)
	}

	matching := 1
	if err := ValidatePoEntryCount(poFile, &matching, "before update"); err != nil {
		t.Errorf("Expected no error for matching entry count, got %v", err)
	}

	nonMatching := 2
	if err := ValidatePoEntryCount(poFile, &nonMatching, "after update"); err == nil {
		t.Errorf("Expected error for non-matching entry count, got nil")
	}
}

func TestDetectAgentOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "hello\n", ""},
		{"plain text no newline", "hello", ""},
		{"claude", `{"type":"system","claude_code_version":"1.0"}` + "\n", config.AgentKindClaude},
		{"opencode", `{"type":"step_start","sessionID":"x"}` + "\n", config.AgentKindOpencode},
		{"codex", `{"type":"thread.started","thread_id":"x"}` + "\n", config.AgentKindCodex},
		{"qoder provider", `{"provider":"qoder","type":"system"}` + "\n", config.AgentKindQoder},
		{"qoder result", `{"type":"result","subtype":"success","message":{}}` + "\n", config.AgentKindQoder},
		{"gemini", `{"type":"system","subtype":"init"}` + "\n", config.AgentKindGemini},
		{"json but unknown", `{"type":"unknown","x":1}` + "\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectAgentOutputFormat([]byte(tt.input))
			if got != tt.expected {
				t.Errorf("detectAgentOutputFormat(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseBatchOutput(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		input := []byte("hello world\n")
		content, result, err := parseBatchOutput(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(content) != string(input) {
			t.Errorf("expected content %q, got %q", input, content)
		}
		if result != nil {
			t.Errorf("expected nil result for plain text, got %v", result)
		}
	})

	t.Run("qoder single json", func(t *testing.T) {
		input := []byte(`{"type":"result","subtype":"success","message":{"content":[{"type":"text","text":"done"}]},"session_id":"x","done":true}` + "\n")
		content, _, err := parseBatchOutput(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(content), "done") {
			t.Errorf("expected content to contain 'done', got %q", content)
		}
	})
}
