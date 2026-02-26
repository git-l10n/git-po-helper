package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/spf13/viper"
)

func TestCountPotEntries(t *testing.T) {
	tests := []struct {
		name        string
		potContent  string
		expected    int
		expectError bool
	}{
		{
			name: "normal POT file with multiple entries",
			potContent: `# SOME DESCRIPTIVE TITLE.
# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER
# This file is distributed under the same license as the PACKAGE package.
# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.
#
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First string"
msgstr ""

msgid "Second string"
msgstr ""

msgid "Third string"
msgstr ""
`,
			expected:    3,
			expectError: false,
		},
		{
			name: "POT file with only header",
			potContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "POT file with multi-line msgid",
			potContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First line"
"Second line"
msgstr ""

msgid "Another string"
msgstr ""
`,
			expected:    2,
			expectError: false,
		},
		{
			name:        "empty file",
			potContent:  "",
			expected:    0,
			expectError: false,
		},
		{
			name: "POT file with commented entries",
			potContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#~ msgid "Obsolete entry"
#~ msgstr ""

msgid "Active entry"
msgstr ""
`,
			expected:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			potFile := filepath.Join(tmpDir, "test.pot")
			err := os.WriteFile(potFile, []byte(tt.potContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test POT file: %v", err)
			}

			// Test CountPoReportStats (POT uses same format)
			stats, err := CountPoReportStats(potFile)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if count := stats.Total(); count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountPotEntries_InvalidFile(t *testing.T) {
	// Test with non-existent file
	_, err := CountPoReportStats("/nonexistent/file.pot")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCountPotEntries_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	potFile := filepath.Join(tmpDir, "empty.pot")

	// Create empty file
	file, err := os.Create(potFile)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	file.Close()

	stats, err := CountPoReportStats(potFile)
	if err != nil {
		t.Errorf("Unexpected error for empty file: %v", err)
	}
	if count := stats.Total(); count != 0 {
		t.Errorf("Expected count 0 for empty file, got %d", count)
	}
}

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		template string
		kv       PlaceholderVars
		expected string
		wantErr  bool
	}{
		{
			name:     "all placeholders",
			template: "cmd -p {{.prompt}} -s {{.source}} -c {{.commit}}",
			kv:       PlaceholderVars{"prompt": "update pot", "source": "po/zh_CN.po", "commit": "HEAD"},
			expected: "cmd -p update pot -s po/zh_CN.po -c HEAD",
		},
		{
			name:     "only prompt placeholder",
			template: "cmd -p {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "update pot"},
			expected: "cmd -p update pot",
		},
		{
			name:     "multiple occurrences",
			template: "{{.prompt}} {{.prompt}} {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "test"},
			expected: "test test test",
		},
		{
			name:     "empty values",
			template: "cmd -p {{.prompt}} -s {{.source}} -c {{.commit}}",
			kv:       PlaceholderVars{"prompt": "", "source": "", "commit": ""},
			expected: "cmd -p  -s  -c ",
		},
		{
			name:     "no placeholders",
			template: "cmd -p test",
			kv:       PlaceholderVars{"prompt": "update pot", "source": "po/zh_CN.po", "commit": "HEAD"},
			expected: "cmd -p test",
		},
		{
			name:     "special characters in values",
			template: "cmd -p {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "update 'pot' file"},
			expected: "cmd -p update 'pot' file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReplacePlaceholders(tt.template, tt.kv)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplacePlaceholders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExecutePromptTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     PlaceholderVars
		expected string
		wantErr  bool
	}{
		{
			name:     "source and dest",
			template: `Review "{{.source}}" and fix in "{{.dest}}"`,
			vars:     PlaceholderVars{"prompt": "ignored", "source": "po/zh_CN.po", "dest": "po/zh_CN.po"},
			expected: `Review "po/zh_CN.po" and fix in "po/zh_CN.po"`,
		},
		{
			name:     "no template vars",
			template: "Update file po/git.pot",
			vars:     PlaceholderVars{"prompt": "x"},
			expected: "Update file po/git.pot",
		},
		{
			name:     "literal braces",
			template: `Placeholders like {{` + "`{name}`" + `}} preserved`,
			vars:     PlaceholderVars{},
			expected: `Placeholders like {name} preserved`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExecutePromptTemplate(tt.template, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecutePromptTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExecuteAgentCommand(t *testing.T) {
	// Test successful command execution
	t.Run("successful command", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "test output"}
		} else {
			cmd = []string{"sh", "-c", "echo 'test output'"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "test output") {
			t.Errorf("Expected stdout to contain 'test output', got %q", output)
		}

		if len(stderr) > 0 {
			t.Logf("stderr: %s", string(stderr))
		}
	})

	// Test command with stderr output
	t.Run("command with stderr", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo test >&2"}
		} else {
			cmd = []string{"sh", "-c", "echo 'test error' >&2"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// stderr should contain the error message
		if len(stderr) == 0 {
			t.Log("Note: stderr is empty (this may be expected on some systems)")
		}
		_ = stdout
	})

	// Test command failure (non-zero exit code)
	t.Run("command failure", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "exit 1"}
		} else {
			cmd = []string{"sh", "-c", "exit 1"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err == nil {
			t.Error("Expected error for failing command, got nil")
		}

		// Error should mention exit code
		if !strings.Contains(err.Error(), "exit code") && !strings.Contains(err.Error(), "failed") {
			t.Errorf("Error message should mention exit code or failure: %v", err)
		}

		_ = stdout
		_ = stderr
	})

	// Test empty command
	t.Run("empty command", func(t *testing.T) {
		_, _, err := ExecuteAgentCommand([]string{})
		if err == nil {
			t.Error("Expected error for empty command, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Error should mention 'cannot be empty': %v", err)
		}
	})

	// Test non-existent command
	t.Run("non-existent command", func(t *testing.T) {
		_, _, err := ExecuteAgentCommand([]string{"nonexistent-command-xyz123"})
		if err == nil {
			t.Error("Expected error for non-existent command, got nil")
		}
	})

	// Test command that produces output (pwd/cd)
	t.Run("command produces output", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "cd"}
		} else {
			cmd = []string{"pwd"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		// On Windows, the path format might be different, so just check it's not empty
		if len(output) == 0 {
			t.Error("Expected non-empty output from pwd/cd command")
		}

		_ = stderr
	})
}

func TestExecuteAgentCommand_PlaceholderReplacement(t *testing.T) {
	// This test verifies that placeholder replacement should be done
	// before calling ExecuteAgentCommand, not inside it.
	// ExecuteAgentCommand should execute the command as-is.

	t.Run("command with literal placeholders", func(t *testing.T) {
		// ExecuteAgentCommand should not replace placeholders
		// (that's the caller's responsibility)
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "{prompt}"}
		} else {
			cmd = []string{"sh", "-c", "echo '{prompt}'"}
		}

		stdout, _, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "{prompt}") {
			t.Errorf("Expected literal {prompt} in output, got %q", output)
		}
	})
}

// Test helper: verify that ExecuteAgentCommand works with real commands
func TestExecuteAgentCommand_RealCommand(t *testing.T) {
	// Test with a command that should exist on all systems
	var cmd []string
	if runtime.GOOS == "windows" {
		// On Windows, use cmd /c echo
		cmd = []string{"cmd", "/c", "echo", "Hello World"}
	} else {
		// On Unix, use echo
		cmd = []string{"echo", "Hello World"}
	}

	// Check if command exists
	if _, err := exec.LookPath(cmd[0]); err != nil {
		t.Skipf("Command %s not found, skipping test", cmd[0])
	}

	stdout, stderr, err := ExecuteAgentCommand(cmd)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := strings.TrimSpace(string(stdout))
	if !strings.Contains(output, "Hello World") {
		t.Errorf("Expected 'Hello World' in output, got %q", output)
	}

	if len(stderr) > 0 {
		t.Logf("stderr (non-fatal): %s", string(stderr))
	}
}

func TestCountPoEntries(t *testing.T) {
	tests := []struct {
		name        string
		poContent   string
		expected    int
		expectError bool
	}{
		{
			name: "normal PO file with multiple entries",
			poContent: `# SOME DESCRIPTIVE TITLE.
# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER
# This file is distributed under the same license as the PACKAGE package.
# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.
#
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First string"
msgstr ""

msgid "Second string"
msgstr ""

msgid "Third string"
msgstr ""
`,
			expected:    3,
			expectError: false,
		},
		{
			name: "PO file with only header",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with multi-line msgid",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First line"
"Second line"
msgstr ""

msgid "Another string"
msgstr ""
`,
			expected:    2,
			expectError: false,
		},
		{
			name:        "empty file",
			poContent:   "",
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with commented entries",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#~ msgid "Obsolete entry"
#~ msgstr ""

msgid "Active entry"
msgstr ""
`,
			expected:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			poFile := filepath.Join(tmpDir, "test.po")
			err := os.WriteFile(poFile, []byte(tt.poContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test PO file: %v", err)
			}

			// Test CountPoReportStats
			stats, err := CountPoReportStats(poFile)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if count := stats.Total(); count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountPoEntries_InvalidFile(t *testing.T) {
	// Test with non-existent file
	_, err := CountPoReportStats("/nonexistent/file.po")
	if err == nil {
		t.Error("Expected error for non-existent PO file, got nil")
	}
}

func TestCountPoEntries_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "empty.po")

	// Create empty file
	file, err := os.Create(poFile)
	if err != nil {
		t.Fatalf("Failed to create empty PO file: %v", err)
	}
	file.Close()

	stats, err := CountPoReportStats(poFile)
	if err != nil {
		t.Errorf("Unexpected error for empty PO file: %v", err)
	}
	if count := stats.Total(); count != 0 {
		t.Errorf("Expected count 0 for empty PO file, got %d", count)
	}
}

func TestValidatePoEntryCount_Disabled(t *testing.T) {
	// nil expectedCount
	var expectedNil *int
	if err := ValidatePoEntryCount("/nonexistent/file.po", expectedNil, "before update"); err != nil {
		t.Errorf("Expected no error when validation is disabled with nil expectedCount, got %v", err)
	}

	// zero expectedCount
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
	// Prepare a temporary PO file with a single entry
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

	// Matching expected count
	matching := 1
	if err := ValidatePoEntryCount(poFile, &matching, "before update"); err != nil {
		t.Errorf("Expected no error for matching entry count, got %v", err)
	}

	// Non-matching expected count
	nonMatching := 2
	if err := ValidatePoEntryCount(poFile, &nonMatching, "after update"); err == nil {
		t.Errorf("Expected error for non-matching entry count, got nil")
	}
}

func TestCountNewEntries(t *testing.T) {
	tests := []struct {
		name        string
		poContent   string
		expected    int
		expectError bool
	}{
		{
			name: "PO file with untranslated entries",
			poContent: `# Translation file
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Translated string"
msgstr "已翻译的字符串"

msgid "Untranslated string 1"
msgstr ""

msgid "Untranslated string 2"
msgstr ""
`,
			expected:    2,
			expectError: false,
		},
		{
			name: "PO file with all translated entries",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First string"
msgstr "第一个字符串"

msgid "Second string"
msgstr "第二个字符串"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with only header",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with multi-line untranslated msgid",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Line 1 "
"Line 2"
msgstr ""

msgid "Another string"
msgstr "另一个字符串"
`,
			expected:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			poFile := filepath.Join(tmpDir, "test.po")
			err := os.WriteFile(poFile, []byte(tt.poContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test PO file: %v", err)
			}

			stats, err := CountPoReportStats(poFile)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if count := stats.Untranslated; count != tt.expected {
				t.Errorf("Expected untranslated count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountNewEntries_InvalidFile(t *testing.T) {
	_, err := CountPoReportStats("/nonexistent/file.po")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCountFuzzyEntries(t *testing.T) {
	tests := []struct {
		name        string
		poContent   string
		expected    int
		expectError bool
	}{
		{
			name: "PO file with fuzzy entries",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Normal string"
msgstr "普通字符串"

#, fuzzy
msgid "Fuzzy string 1"
msgstr "模糊字符串 1"

#, fuzzy
msgid "Fuzzy string 2"
msgstr "模糊字符串 2"
`,
			expected:    2,
			expectError: false,
		},
		{
			name: "PO file with no fuzzy entries",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First string"
msgstr "第一个字符串"

msgid "Second string"
msgstr "第二个字符串"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with only header",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
`,
			expected:    0,
			expectError: false,
		},
		{
			name: "PO file with multi-line fuzzy msgid",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Line 1 "
"Line 2"
msgstr "第一行第二行"

msgid "Another string"
msgstr "另一个字符串"
`,
			expected:    1,
			expectError: false,
		},
		{
			name: "PO file with mixed fuzzy and untranslated",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Fuzzy string"
msgstr "模糊字符串"

msgid "Untranslated string"
msgstr ""

msgid "Normal string"
msgstr "普通字符串"
`,
			expected:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			poFile := filepath.Join(tmpDir, "test.po")
			err := os.WriteFile(poFile, []byte(tt.poContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test PO file: %v", err)
			}

			stats, err := CountPoReportStats(poFile)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if count := stats.Fuzzy; count != tt.expected {
				t.Errorf("Expected fuzzy count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountFuzzyEntries_InvalidFile(t *testing.T) {
	_, err := CountPoReportStats("/nonexistent/file.po")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGetPrompt(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		cfg             *config.AgentConfig
		agentRunPrompt  string
		agentTestPrompt string
		expected        string
		expectError     bool
		errorContains   string
	}{
		{
			name:   "use config prompt when no override",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config update pot prompt",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "config update pot prompt",
			expectError:     false,
		},
		{
			name:   "override with agent-run--prompt",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config update pot prompt",
				},
			},
			agentRunPrompt:  "override prompt from agent-run",
			agentTestPrompt: "",
			expected:        "override prompt from agent-run",
			expectError:     false,
		},
		{
			name:   "override with agent-test--prompt when agent-run--prompt is empty",
			action: "update-po",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePo: "config update po prompt",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "override prompt from agent-test",
			expected:        "override prompt from agent-test",
			expectError:     false,
		},
		{
			name:   "agent-run--prompt takes priority over agent-test--prompt",
			action: "translate",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					Translate: "config translate prompt",
				},
			},
			agentRunPrompt:  "override from agent-run",
			agentTestPrompt: "override from agent-test",
			expected:        "override from agent-run",
			expectError:     false,
		},
		{
			name:   "error when config prompt is empty and no override",
			action: "review",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					Review: "",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "",
			expectError:     true,
			errorContains:   "prompt.review is not configured",
		},
		{
			name:   "error for unknown action",
			action: "unknown-action",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "",
			expectError:     true,
			errorContains:   "unknown action",
		},
		{
			name:   "override works for all actions",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config prompt",
					UpdatePo:  "config prompt",
					Translate: "config prompt",
					Review:    "config prompt",
				},
			},
			agentRunPrompt:  "override prompt",
			agentTestPrompt: "",
			expected:        "override prompt",
			expectError:     false,
		},
		{
			name:   "override works even when config is empty",
			action: "update-po",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePo: "",
				},
			},
			agentRunPrompt:  "override prompt",
			agentTestPrompt: "",
			expected:        "override prompt",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original viper values
			originalAgentRunPrompt := viper.GetString("agent-run--prompt")
			originalAgentTestPrompt := viper.GetString("agent-test--prompt")

			// Set viper values for test
			if tt.agentRunPrompt != "" {
				viper.Set("agent-run--prompt", tt.agentRunPrompt)
			} else {
				viper.Set("agent-run--prompt", "")
			}
			if tt.agentTestPrompt != "" {
				viper.Set("agent-test--prompt", tt.agentTestPrompt)
			} else {
				viper.Set("agent-test--prompt", "")
			}

			// Run test
			result, err := GetRawPrompt(tt.cfg, tt.action)

			// Restore original viper values
			if originalAgentRunPrompt != "" {
				viper.Set("agent-run--prompt", originalAgentRunPrompt)
			} else {
				viper.Set("agent-run--prompt", "")
			}
			if originalAgentTestPrompt != "" {
				viper.Set("agent-test--prompt", originalAgentTestPrompt)
			} else {
				viper.Set("agent-test--prompt", "")
			}

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected prompt %q, got %q", tt.expected, result)
			}
		})
	}
}
