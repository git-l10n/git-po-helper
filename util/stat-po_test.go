package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountPoReportStats(t *testing.T) {
	poContent := `# SOME DESCRIPTIVE TITLE.
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

# Translated
msgid "Hello"
msgstr "你好"

# Untranslated
msgid "World"
msgstr ""

# Same as msgid (suspect)
msgid "File"
msgstr "File"

# Fuzzy
#, fuzzy
msgid "Fuzzy entry"
msgstr "模糊"

# Another translated
msgid "Good"
msgstr "好"

#~ msgid "Obsolete entry"
#~ msgstr ""
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stats, err := CountPoReportStats(poFile)
	if err != nil {
		t.Fatalf("CountPoReportStats failed: %v", err)
	}

	// Same-as-source entries are counted as translated (msgfmt compatibility).
	if stats.Translated != 3 {
		t.Errorf("translated: want 3, got %d", stats.Translated)
	}
	if stats.Untranslated != 1 {
		t.Errorf("untranslated: want 1, got %d", stats.Untranslated)
	}
	if stats.Same != 1 {
		t.Errorf("same: want 1, got %d", stats.Same)
	}
	if stats.Fuzzy != 1 {
		t.Errorf("fuzzy: want 1, got %d", stats.Fuzzy)
	}
	if stats.Obsolete != 1 {
		t.Errorf("obsolete: want 1, got %d", stats.Obsolete)
	}
}

// TestReportMatchesMsgfmtStatistics verifies that report output matches
// msgfmt --statistics when there are no "same" (msgstr == msgid) entries.
func TestReportMatchesMsgfmtStatistics(t *testing.T) {
	if _, err := exec.LookPath("msgfmt"); err != nil {
		t.Skip("msgfmt not found, skipping")
	}

	// PO file without "same" entries: translated, untranslated, fuzzy only
	poContent := `# Test file for msgfmt compatibility
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "你好"

msgid "World"
msgstr ""

#, fuzzy
msgid "Fuzzy"
msgstr "模糊"
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Get msgfmt output
	cmd := exec.Command("msgfmt", "--statistics", "-o", os.DevNull, poFile)
	cmd.Dir = tmpDir
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("msgfmt start: %v", err)
	}
	var sb strings.Builder
	buf := make([]byte, 256)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	_ = cmd.Wait()
	msgfmtOut := sb.String()

	// Get our report output
	stats, err := CountPoReportStats(poFile)
	if err != nil {
		t.Fatalf("CountPoReportStats: %v", err)
	}
	ourOut := FormatMsgfmtStatistics(stats)

	if ourOut != msgfmtOut {
		t.Errorf("output mismatch:\nmsgfmt: %q\nours:   %q", msgfmtOut, ourOut)
	}
}

func TestFormatMsgfmtStatistics(t *testing.T) {
	tests := []struct {
		name     string
		stats    *PoReportStats
		expected string
	}{
		{"all zeros", &PoReportStats{}, "0 translated messages.\n"},
		{"one translated", &PoReportStats{Translated: 1}, "1 translated message.\n"},
		{"two translated", &PoReportStats{Translated: 2}, "2 translated messages.\n"},
		{"one fuzzy", &PoReportStats{Fuzzy: 1}, "1 fuzzy translation.\n"},
		{"one untranslated", &PoReportStats{Untranslated: 1}, "1 untranslated message.\n"},
		{"same counts as translated", &PoReportStats{Same: 1}, "1 translated message.\n"},
		{"mixed", &PoReportStats{Translated: 1, Same: 1, Fuzzy: 1, Untranslated: 1},
			"2 translated messages, 1 fuzzy translation, 1 untranslated message.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMsgfmtStatistics(tt.stats)
			if got != tt.expected {
				t.Errorf("FormatMsgfmtStatistics() = %q, want %q", got, tt.expected)
			}
		})
	}
}

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
			tmpDir := t.TempDir()
			potFile := filepath.Join(tmpDir, "test.pot")
			err := os.WriteFile(potFile, []byte(tt.potContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test POT file: %v", err)
			}
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
	_, err := CountPoReportStats("/nonexistent/file.pot")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCountPotEntries_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	potFile := filepath.Join(tmpDir, "empty.pot")
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
			if count := stats.Total(); count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountPoEntries_InvalidFile(t *testing.T) {
	_, err := CountPoReportStats("/nonexistent/file.po")
	if err == nil {
		t.Error("Expected error for non-existent PO file, got nil")
	}
}

func TestCountPoEntries_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "empty.po")
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
