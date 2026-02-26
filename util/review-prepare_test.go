package util

import (
	"os"
	"path/filepath"
	"testing"
)

// validatePoFileForReview validates PO file format using ValidatePoFileFormat.
func validatePoFileForReview(poFile string) error {
	absPath, err := filepath.Abs(poFile)
	if err != nil {
		return err
	}
	return ValidatePoFile(absPath)
}

// TestWriteReviewInputPo tests writeReviewInputPo with various inputs.
func TestWriteReviewInputPo(t *testing.T) {
	tests := []struct {
		name    string
		header  []string
		entries []*PoEntry
	}{
		{
			name:   "simple header and entry",
			header: []string{"msgid \"\"", "msgstr \"Content-Type: text/plain; charset=UTF-8\\n\""},
			entries: []*PoEntry{
				{RawLines: []string{"msgid \"Hello\"", "msgstr \"你好\""}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "test-output.po")

			err := WritePoEntries(outputPath, tt.header, tt.entries)
			if err != nil {
				t.Fatalf("writeReviewInputPo failed: %v", err)
			}

			writtenData, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read written file: %v", err)
			}

			writtenEntries, writtenHeader, err := ParsePoEntries(writtenData)
			if err != nil {
				t.Fatalf("failed to parse written file: %v", err)
			}

			if len(writtenHeader) != len(tt.header) {
				t.Errorf("header length mismatch: expected %d, got %d", len(tt.header), len(writtenHeader))
			}
			if len(writtenEntries) != len(tt.entries) {
				t.Errorf("entry count mismatch: expected %d, got %d", len(tt.entries), len(writtenEntries))
			}
		})
	}
}

// TestParsePoEntriesRoundTrip tests reading a PO file, writing it with writeReviewInputPo,
// and reading again to verify that the round-trip preserves entry count, structure, and header.
func TestParsePoEntriesRoundTrip(t *testing.T) {
	poFilePath := os.Getenv("TEST_PO_FILE")
	if poFilePath == "" {
		t.Skip("TEST_PO_FILE environment variable not set, skipping test")
	}

	testPoContent, err := os.ReadFile(poFilePath)
	if err != nil {
		t.Fatalf("failed to read PO file %s: %v", poFilePath, err)
	}

	tmpDir := t.TempDir()
	originalPoPath := filepath.Join(tmpDir, "original.po")
	writtenPoPath := filepath.Join(tmpDir, "written.po")

	err = os.WriteFile(originalPoPath, testPoContent, 0644)
	if err != nil {
		t.Fatalf("failed to write original PO file: %v", err)
	}

	originalData, err := os.ReadFile(originalPoPath)
	if err != nil {
		t.Fatalf("failed to read original PO file: %v", err)
	}

	originalEntries, originalHeader, err := ParsePoEntries(originalData)
	if err != nil {
		t.Fatalf("failed to parse original PO file: %v", err)
	}

	err = WritePoEntries(writtenPoPath, originalHeader, originalEntries)
	if err != nil {
		t.Fatalf("failed to write PO file: %v", err)
	}

	if err := validatePoFileForReview(writtenPoPath); err != nil {
		t.Errorf("first written PO file format validation failed: %v", err)
	}

	writtenData, err := os.ReadFile(writtenPoPath)
	if err != nil {
		t.Fatalf("failed to read written PO file: %v", err)
	}

	writtenEntries, writtenHeader, err := ParsePoEntries(writtenData)
	if err != nil {
		t.Fatalf("failed to parse written PO file: %v", err)
	}

	if len(originalHeader) != len(writtenHeader) {
		t.Errorf("header length mismatch: original has %d lines, written has %d lines", len(originalHeader), len(writtenHeader))
	} else {
		for i, originalLine := range originalHeader {
			if i < len(writtenHeader) && originalLine != writtenHeader[i] {
				t.Errorf("header line %d mismatch: original '%s', written '%s'", i, originalLine, writtenHeader[i])
			}
		}
	}

	if len(originalEntries) != len(writtenEntries) {
		t.Errorf("entry count mismatch: original has %d entries, written has %d entries", len(originalEntries), len(writtenEntries))
	}

	for i, originalEntry := range originalEntries {
		if i >= len(writtenEntries) {
			t.Errorf("missing entry %d in written file", i)
			continue
		}
		writtenEntry := writtenEntries[i]

		if originalEntry.MsgID != writtenEntry.MsgID {
			t.Errorf("entry %d MsgID mismatch: original '%s', written '%s'", i, originalEntry.MsgID, writtenEntry.MsgID)
		}
		if originalEntry.MsgStr != writtenEntry.MsgStr {
			t.Errorf("entry %d MsgStr mismatch: original '%s', written '%s'", i, originalEntry.MsgStr, writtenEntry.MsgStr)
		}
		if originalEntry.MsgIDPlural != writtenEntry.MsgIDPlural {
			t.Errorf("entry %d MsgIDPlural mismatch: original '%s', written '%s'", i, originalEntry.MsgIDPlural, writtenEntry.MsgIDPlural)
		}
		if len(originalEntry.MsgStrPlural) != len(writtenEntry.MsgStrPlural) {
			t.Errorf("entry %d MsgStrPlural length mismatch: original has %d, written has %d", i, len(originalEntry.MsgStrPlural), len(writtenEntry.MsgStrPlural))
		} else {
			for j, originalPlural := range originalEntry.MsgStrPlural {
				if writtenEntry.MsgStrPlural[j] != originalPlural {
					t.Errorf("entry %d MsgStrPlural[%d] mismatch: original '%s', written '%s'", i, j, originalPlural, writtenEntry.MsgStrPlural[j])
				}
			}
		}
		if len(originalEntry.Comments) != len(writtenEntry.Comments) {
			t.Errorf("entry %d comments count mismatch: original has %d, written has %d", i, len(originalEntry.Comments), len(writtenEntry.Comments))
		} else {
			for j, originalComment := range originalEntry.Comments {
				if writtenEntry.Comments[j] != originalComment {
					t.Errorf("entry %d comment %d mismatch: original '%s', written '%s'", i, j, originalComment, writtenEntry.Comments[j])
				}
			}
		}
	}

	// Test double round-trip
	secondWrittenPoPath := filepath.Join(tmpDir, "written2.po")
	err = WritePoEntries(secondWrittenPoPath, writtenHeader, writtenEntries)
	if err != nil {
		t.Fatalf("failed to write PO file second time: %v", err)
	}

	if err := validatePoFileForReview(secondWrittenPoPath); err != nil {
		t.Errorf("second written PO file format validation failed: %v", err)
	}

	secondReadData, err := os.ReadFile(secondWrittenPoPath)
	if err != nil {
		t.Fatalf("failed to read second written PO file: %v", err)
	}

	secondReadEntries, secondReadHeader, err := ParsePoEntries(secondReadData)
	if err != nil {
		t.Fatalf("failed to parse second written PO file: %v", err)
	}

	if len(originalHeader) != len(secondReadHeader) {
		t.Errorf("second round-trip header length mismatch: original has %d lines, second written has %d lines", len(originalHeader), len(secondReadHeader))
	} else {
		for i, originalLine := range originalHeader {
			if i < len(secondReadHeader) && originalLine != secondReadHeader[i] {
				t.Errorf("second round-trip header line %d mismatch: original '%s', second written '%s'", i, originalLine, secondReadHeader[i])
			}
		}
	}

	if len(originalEntries) != len(secondReadEntries) {
		t.Errorf("second round-trip entry count mismatch: original has %d entries, second written has %d entries", len(originalEntries), len(secondReadEntries))
	}

	for i, originalEntry := range originalEntries {
		if i >= len(secondReadEntries) {
			t.Errorf("missing entry %d in second written file", i)
			continue
		}
		secondReadEntry := secondReadEntries[i]

		if originalEntry.MsgID != secondReadEntry.MsgID {
			t.Errorf("second round-trip entry %d MsgID mismatch: original '%s', second written '%s'", i, originalEntry.MsgID, secondReadEntry.MsgID)
		}
		if originalEntry.MsgStr != secondReadEntry.MsgStr {
			t.Errorf("second round-trip entry %d MsgStr mismatch: original '%s', second written '%s'", i, originalEntry.MsgStr, secondReadEntry.MsgStr)
		}
		if originalEntry.MsgIDPlural != secondReadEntry.MsgIDPlural {
			t.Errorf("second round-trip entry %d MsgIDPlural mismatch: original '%s', second written '%s'", i, originalEntry.MsgIDPlural, secondReadEntry.MsgIDPlural)
		}
		if len(originalEntry.MsgStrPlural) != len(secondReadEntry.MsgStrPlural) {
			t.Errorf("second round-trip entry %d MsgStrPlural length mismatch: original has %d, second written has %d", i, len(originalEntry.MsgStrPlural), len(secondReadEntry.MsgStrPlural))
		} else {
			for j, originalPlural := range originalEntry.MsgStrPlural {
				if secondReadEntry.MsgStrPlural[j] != originalPlural {
					t.Errorf("second round-trip entry %d MsgStrPlural[%d] mismatch: original '%s', second written '%s'", i, j, originalPlural, secondReadEntry.MsgStrPlural[j])
				}
			}
		}
		if len(originalEntry.Comments) != len(secondReadEntry.Comments) {
			t.Errorf("second round-trip entry %d comments count mismatch: original has %d, second written has %d", i, len(originalEntry.Comments), len(secondReadEntry.Comments))
		} else {
			for j, originalComment := range originalEntry.Comments {
				if secondReadEntry.Comments[j] != originalComment {
					t.Errorf("second round-trip entry %d comment %d mismatch: original '%s', second written '%s'", i, j, originalComment, secondReadEntry.Comments[j])
				}
			}
		}
	}
}
