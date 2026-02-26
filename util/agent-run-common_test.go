package util

import (
	"os"
	"path/filepath"
	"testing"
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
