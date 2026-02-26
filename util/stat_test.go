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
