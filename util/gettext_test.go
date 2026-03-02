package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// poRoundTripExamples are PO file contents for ParsePoEntries round-trip testing.
// Each example is parsed, written back via BuildPoContent, and the result must match the original byte-for-byte.
var poRoundTripExamples = []string{
	`# Header comment
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`,
	`

# Empty line before header comment
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`,
	`# Header comment
# Empty line after comments

# Another empty line after comment

msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "第一个"

msgid "Second"
msgstr "第二个"

msgid "Third"
msgstr "第三个"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid ""
"Multi"
"line"
msgstr ""
"多"
"行"

msgid "Single"
msgstr "单"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "One"
msgid_plural "Many"
msgstr[0] "一个"
msgstr[1] "多个"

msgid "File"
msgid_plural "Files"
msgstr[0] "文件"
msgstr[1] "文件"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Fuzzy string"
msgstr "模糊"

#, fuzzy, c-format
msgid "Fuzzy %s"
msgstr "模糊 %s"
`,
}

func TestParsePoEntriesRoundTripBytes(t *testing.T) {
	for i, poContent := range poRoundTripExamples {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			original := []byte(poContent)
			entries, header, err := ParsePoEntries(original)
			if err != nil {
				t.Fatalf("ParsePoEntries failed: %v", err)
			}
			written := BuildPoContent(header, entries)
			if !bytes.Equal(original, written) {
				diff := bytesDiff(original, written)
				t.Errorf("round-trip mismatch:\n%s", diff)
			}
		})
	}
}

func TestParsePoEntriesFuzzy(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Fuzzy"
msgstr "模糊"

#, fuzzy, c-format
msgid "Fuzzy %s"
msgstr "模糊 %s"

msgid "Normal"
msgstr "正常"
`
	entries, _, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if !entries[0].Fuzzy {
		t.Errorf("entry 0 (Fuzzy): expected Fuzzy=true, got false")
	}
	if !entries[1].Fuzzy {
		t.Errorf("entry 1 (Fuzzy %%s): expected Fuzzy=true, got false")
	}
	if entries[2].Fuzzy {
		t.Errorf("entry 2 (Normal): expected Fuzzy=false, got true")
	}
}

func TestParsePoEntriesObsolete(t *testing.T) {
	// Minimal test for obsolete parsing
	po := "msgid \"\"\n" +
		"msgstr \"\"\n" +
		"\"Content-Type: text/plain; charset=UTF-8\\n\"\n" +
		"\n" +
		"msgid \"Active\"\n" +
		"msgstr \"活跃\"\n" +
		"#~ msgid \"Obsolete\"\n" +
		"#~ msgstr \"已废弃\"\n" +
		"\n"
	entries, _, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].MsgID != "Active" || entries[0].Obsolete {
		t.Errorf("entry 0: got MsgID=%q Obsolete=%v", entries[0].MsgID, entries[0].Obsolete)
	}
	if entries[1].MsgID != "Obsolete" || !entries[1].Obsolete {
		t.Errorf("entry 1: got MsgID=%q Obsolete=%v", entries[1].MsgID, entries[1].Obsolete)
	}
}

func TestParsePoEntriesObsoleteHashTildePipe(t *testing.T) {
	// #~| msgid format (gettext 0.19.8+): previous untranslated string
	po := "msgid \"\"\n" +
		"msgstr \"\"\n" +
		"\"Content-Type: text/plain; charset=UTF-8\\n\"\n" +
		"\n" +
		"msgid \"Active\"\n" +
		"msgstr \"活跃\"\n" +
		"#~| msgid \"Old source\"\n" +
		"#~ msgid \"Obsolete\"\n" +
		"#~ msgstr \"已废弃\"\n" +
		"\n"
	entries, _, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].MsgID != "Obsolete" || !entries[1].Obsolete {
		t.Errorf("entry 1: got MsgID=%q Obsolete=%v", entries[1].MsgID, entries[1].Obsolete)
	}
	if entries[1].MsgIDPrevious != "Old source" {
		t.Errorf("entry 1 MsgIDPrevious: got %q, want %q", entries[1].MsgIDPrevious, "Old source")
	}
}

func TestParsePoEntries(t *testing.T) {
	tests := []struct {
		name           string
		poContent      string
		expectedHeader []string
		expectedCount  int
		validateEntry  func(t *testing.T, entries []*GettextEntry)
	}{
		{
			name: "simple PO file with header and entries",
			poContent: `# SOME DESCRIPTIVE TITLE.
# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER
# This file is distributed under the same license as the PACKAGE package.
# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.
#
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"

msgid "Hello"
msgstr "你好"

msgid ""
"World"
msgstr ""
"世界"
`,
			expectedHeader: []string{
				`# SOME DESCRIPTIVE TITLE.`,
				`# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER`,
				`# This file is distributed under the same license as the PACKAGE package.`,
				`# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.`,
				`#`,
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
				`"Content-Transfer-Encoding: 8bit\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "Hello" {
					t.Errorf("expected first entry MsgID 'Hello', got '%s'", entries[0].MsgID)
				}
				if entries[0].MsgStr != "你好" {
					t.Errorf("expected first entry MsgStr '你好', got '%s'", entries[0].MsgStr)
				}
				if entries[1].MsgID != "World" {
					t.Errorf("expected second entry MsgID 'World', got '%s'", entries[1].MsgID)
				}
				if entries[1].MsgStr != "世界" {
					t.Errorf("expected second entry MsgStr '世界', got '%s'", entries[1].MsgStr)
				}
			},
		},
		{
			name: "PO file with multi-line msgid and msgstr",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid ""
"First line"
"Second line"
msgstr ""
"第一行"
"第二行"

msgid "Single line"
msgstr "单行"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				expectedMsgID := "First lineSecond line"
				if entries[0].MsgID != expectedMsgID {
					t.Errorf("expected first entry MsgID '%s', got '%s'", expectedMsgID, entries[0].MsgID)
				}
				expectedMsgStr := "第一行第二行"
				if entries[0].MsgStr != expectedMsgStr {
					t.Errorf("expected first entry MsgStr '%s', got '%s'", expectedMsgStr, entries[0].MsgStr)
				}
				if entries[1].MsgID != "Single line" {
					t.Errorf("expected second entry MsgID 'Single line', got '%s'", entries[1].MsgID)
				}
			},
		},
		{
			name: "PO file with plural forms",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "One item"
msgid_plural "Many items"
msgstr[0] "一个项目"
msgstr[1] "多个项目"

msgid "File"
msgid_plural "Files"
msgstr[0] "文件"
msgstr[1] "文件"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "One item" {
					t.Errorf("expected first entry MsgID 'One item', got '%s'", entries[0].MsgID)
				}
				if entries[0].MsgIDPlural != "Many items" {
					t.Errorf("expected first entry MsgIDPlural 'Many items', got '%s'", entries[0].MsgIDPlural)
				}
				if len(entries[0].MsgStrPlural) != 2 {
					t.Fatalf("expected 2 plural forms, got %d", len(entries[0].MsgStrPlural))
				}
				if entries[0].MsgStrPlural[0] != "一个项目" {
					t.Errorf("expected first plural form '一个项目', got '%s'", entries[0].MsgStrPlural[0])
				}
				if entries[0].MsgStrPlural[1] != "多个项目" {
					t.Errorf("expected second plural form '多个项目', got '%s'", entries[0].MsgStrPlural[1])
				}
			},
		},
		{
			name: "PO file with plural forms with multiple lines",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid ""
"One "
"item"
msgid_plural ""
"Many "
"items"
msgstr[0] ""
"一个"
"项目"
msgstr[1] ""
"多个"
"项目"

msgid ""
"File"
msgid_plural ""
"Files"
msgstr[0] "文件"
msgstr[1] ""
"文件"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "One item" {
					t.Errorf("expected first entry MsgID 'One item', got '%s'", entries[0].MsgID)
				}
				if entries[0].MsgIDPlural != "Many items" {
					t.Errorf("expected first entry MsgIDPlural 'Many items', got '%s'", entries[0].MsgIDPlural)
				}
				if len(entries[0].MsgStrPlural) != 2 {
					t.Fatalf("expected 2 plural forms, got %d", len(entries[0].MsgStrPlural))
				}
				if entries[0].MsgStrPlural[0] != "一个项目" {
					t.Errorf("expected first plural form '一个项目', got '%s'", entries[0].MsgStrPlural[0])
				}
				if entries[0].MsgStrPlural[1] != "多个项目" {
					t.Errorf("expected second plural form '多个项目', got '%s'", entries[0].MsgStrPlural[1])
				}
			},
		},
		{
			name: "PO file with comments",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

# Translator comment
#. Automatic comment
#: file.c:123
msgid "String with comments"
msgstr "带注释的字符串"

msgid "Simple string"
msgstr "简单字符串"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "String with comments" {
					t.Errorf("expected first entry MsgID 'String with comments', got '%s'", entries[0].MsgID)
				}
				if entries[0].MsgStr != "带注释的字符串" {
					t.Errorf("expected first entry MsgStr '带注释的字符串', got '%s'", entries[0].MsgStr)
				}
				expectedComments := []string{
					"# Translator comment",
					"#. Automatic comment",
					"#: file.c:123",
				}
				if len(entries[0].Comments) != len(expectedComments) {
					t.Errorf("expected %d comments, got %d", len(expectedComments), len(entries[0].Comments))
				} else {
					for i, expectedComment := range expectedComments {
						if entries[0].Comments[i] != expectedComment {
							t.Errorf("comment %d mismatch: expected '%s', got '%s'", i, expectedComment, entries[0].Comments[i])
						}
					}
				}
				if len(entries[1].Comments) != 0 {
					t.Errorf("expected second entry to have no comments, got %d comments", len(entries[1].Comments))
				}
			},
		},
		{
			name: "PO file with only header",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
"Language: zh_CN\n"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
				`"Language: zh_CN\n"`,
			},
			expectedCount: 0,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 0 {
					t.Errorf("expected 0 entries, got %d", len(entries))
				}
			},
		},
		{
			name: "PO file with empty msgstr (untranslated)",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Untranslated"
msgstr ""

msgid "Translated"
msgstr "已翻译"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "Untranslated" {
					t.Errorf("expected first entry MsgID 'Untranslated', got '%s'", entries[0].MsgID)
				}
				if entries[0].MsgStr != "" {
					t.Errorf("expected first entry MsgStr to be empty, got '%s'", entries[0].MsgStr)
				}
				if entries[1].MsgStr != "已翻译" {
					t.Errorf("expected second entry MsgStr '已翻译', got '%s'", entries[1].MsgStr)
				}
			},
		},
		{
			name:           "empty file",
			poContent:      "",
			expectedHeader: []string{},
			expectedCount:  0,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 0 {
					t.Errorf("expected 0 entries, got %d", len(entries))
				}
			},
		},
		{
			name: "PO file with fuzzy entry",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Fuzzy string"
msgstr "模糊字符串"

msgid "Normal string"
msgstr "正常字符串"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "Fuzzy string" {
					t.Errorf("expected first entry MsgID 'Fuzzy string', got '%s'", entries[0].MsgID)
				}
				if entries[1].MsgID != "Normal string" {
					t.Errorf("expected second entry MsgID 'Normal string', got '%s'", entries[1].MsgID)
				}
				if len(entries[0].Comments) != 1 {
					t.Errorf("expected 1 comment for fuzzy entry, got %d", len(entries[0].Comments))
				} else if entries[0].Comments[0] != "#, fuzzy" {
					t.Errorf("expected comment '#, fuzzy', got '%s'", entries[0].Comments[0])
				}
				if !entries[0].Fuzzy {
					t.Errorf("expected first entry Fuzzy=true, got false")
				}
				if entries[1].Fuzzy {
					t.Errorf("expected second entry Fuzzy=false, got true")
				}
				if len(entries[1].Comments) != 0 {
					t.Errorf("expected second entry to have no comments, got %d comments", len(entries[1].Comments))
				}
			},
		},
		{
			name: "PO file with obsolete entries (#~ and #~| format)",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~ msgid "Obsolete"
#~ msgstr "已废弃"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 2,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 2 {
					for i, e := range entries {
						t.Logf("entry %d: MsgID=%q Obsolete=%v", i, e.MsgID, e.Obsolete)
					}
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[0].MsgID != "Active" || entries[0].Obsolete {
					t.Errorf("entry 0: expected active, got MsgID=%q Obsolete=%v", entries[0].MsgID, entries[0].Obsolete)
				}
				if entries[1].MsgID != "Obsolete" || !entries[1].Obsolete {
					t.Errorf("entry 1: expected obsolete, got MsgID=%q Obsolete=%v", entries[1].MsgID, entries[1].Obsolete)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, header, err := ParsePoEntries([]byte(tt.poContent))
			if err != nil {
				t.Fatalf("ParsePoEntries failed: %v", err)
			}

			if tt.name == "empty file" {
				if len(header) > 1 {
					t.Errorf("empty file header should be empty or single line, got %d lines", len(header))
				}
			} else {
				if len(header) != len(tt.expectedHeader) {
					t.Errorf("header length mismatch: expected %d, got %d", len(tt.expectedHeader), len(header))
					t.Logf("Expected header: %v", tt.expectedHeader)
					t.Logf("Got header: %v", header)
				} else {
					for i, expectedLine := range tt.expectedHeader {
						if i < len(header) && header[i] != expectedLine {
							t.Errorf("header line %d mismatch: expected '%s', got '%s'", i, expectedLine, header[i])
						}
					}
				}
			}

			if len(entries) != tt.expectedCount {
				t.Errorf("entry count mismatch: expected %d, got %d", tt.expectedCount, len(entries))
			}

			if tt.validateEntry != nil {
				tt.validateEntry(t, entries)
			}
		})
	}
}

// bytesDiff returns a string describing the first difference between a and b.
func bytesDiff(a, b []byte) string {
	aLines := bytes.Split(a, []byte("\n"))
	bLines := bytes.Split(b, []byte("\n"))
	maxLen := len(aLines)
	if len(bLines) > maxLen {
		maxLen = len(bLines)
	}
	for i := 0; i < maxLen; i++ {
		var aLine, bLine []byte
		if i < len(aLines) {
			aLine = aLines[i]
		}
		if i < len(bLines) {
			bLine = bLines[i]
		}
		if !bytes.Equal(aLine, bLine) {
			return fmt.Sprintf("first difference at line %d:\noriginal (%d bytes): %q\nwritten (%d bytes):  %q\n",
				i+1, len(a), aLine, len(b), bLine)
		}
	}
	return fmt.Sprintf("lengths differ: original %d bytes, written %d bytes", len(a), len(b))
}

func TestStrDeQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`"hello"`, "hello"},
		{`""`, ""},
		{`"a"`, "a"},
		{`"hello`, `"hello`},
		{`hello"`, `hello"`},
		{`hello`, "hello"},
		{`""hello""`, `"hello"`},
		{"", ""},
		{`"`, `"`},
	}
	for _, tt := range tests {
		got := strDeQuote(tt.in)
		if got != tt.want {
			t.Errorf("strDeQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStripFuzzyFromCommentLine(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"#, fuzzy", ""},
		{"#, fuzzy\n", ""},
		{"  #, fuzzy  ", ""},
		{"#, fuzzy, c-format", "#, c-format"},
		{"#, c-format, fuzzy", "#, c-format"},
		{"#, fuzzy, c-format, no-wrap", "#, c-format, no-wrap"},
		{"#, c-format", "#, c-format"},
		{"#: file.c", "#: file.c"},
		{"# normal comment", "# normal comment"},
	}
	for _, tt := range tests {
		got := StripFuzzyFromCommentLine(tt.line)
		if got != tt.want {
			t.Errorf("StripFuzzyFromCommentLine(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestMergeFuzzyIntoFlagLine(t *testing.T) {
	tests := []struct {
		line     string
		addFuzzy bool
		want     string
	}{
		{"#, c-format", true, "#, fuzzy, c-format"},
		{"#, c-format", false, "#, c-format"},
		{"#, no-wrap", true, "#, fuzzy, no-wrap"},
		{"#, fuzzy", true, "#, fuzzy"},
		{"#: file.c", true, "#: file.c"},
	}
	for _, tt := range tests {
		got := MergeFuzzyIntoFlagLine(tt.line, tt.addFuzzy)
		if got != tt.want {
			t.Errorf("MergeFuzzyIntoFlagLine(%q, %v) = %q, want %q", tt.line, tt.addFuzzy, got, tt.want)
		}
	}
}

func TestParseEntryRange(t *testing.T) {
	tests := []struct {
		spec     string
		maxEntry int
		want     []int
		wantErr  bool
	}{
		{"1", 10, []int{1}, false},
		{"0", 10, []int{}, false}, // 0 excluded (header only)
		{"1-3", 10, []int{1, 2, 3}, false},
		{"3,5,9-13", 20, []int{3, 5, 9, 10, 11, 12, 13}, false},
		{"1-3,5", 10, []int{1, 2, 3, 5}, false},
		{"0,2,4", 5, []int{2, 4}, false},                      // 0 excluded
		{"15", 10, []int{}, false},                            // Out of range, silently skipped
		{"1-5", 3, []int{1, 2, 3}, false},                     // Range clipped
		{"", 10, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false}, // Empty = select all
		{"abc", 10, nil, true},
		{"1-2", 10, []int{1, 2}, false},
		{"2-1", 10, nil, true},                  // Invalid: start > end
		{"-5", 10, []int{1, 2, 3, 4, 5}, false}, // -N: from 1 to N
		{"-3", 10, []int{1, 2, 3}, false},
		{"50-", 100, buildRange(50, 100), false}, // N-: from N to last
		{"8-", 10, []int{8, 9, 10}, false},
		{"-", 10, nil, true}, // Invalid: both empty
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			got, err := ParseEntryRange(tt.spec, tt.maxEntry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEntryRange(%q, %d) error = %v, wantErr %v", tt.spec, tt.maxEntry, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !sliceEqual(got, tt.want) {
				t.Errorf("ParseEntryRange(%q, %d) = %v, want %v", tt.spec, tt.maxEntry, got, tt.want)
			}
		})
	}
}

func buildRange(start, end int) []int {
	var r []int
	for i := start; i <= end; i++ {
		r = append(r, i)
	}
	return r
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMsgSelect(t *testing.T) {
	poContent := `# SOME DESCRIPTIVE TITLE.
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "第一个"

msgid "Second"
msgstr "第二个"

msgid "Third"
msgstr "第三个"
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	err := MsgSelect(poFile, "1,3", &buf, false, nil)
	if err != nil {
		t.Fatalf("MsgSelect failed: %v", err)
	}

	output := buf.String()
	// Should contain header (entry 0) and entries 1 and 3
	if !strings.Contains(output, "First") {
		t.Errorf("output should contain 'First', got:\n%s", output)
	}
	if !strings.Contains(output, "Third") {
		t.Errorf("output should contain 'Third', got:\n%s", output)
	}
	if !strings.Contains(output, "Content-Type") {
		t.Errorf("output should contain header, got:\n%s", output)
	}
	if strings.Contains(output, "Second") {
		t.Errorf("output should not contain 'Second' (entry 2), got:\n%s", output)
	}
}

func TestMsgSelect_OpenEndedRange(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "一"

msgid "Second"
msgstr "二"

msgid "Third"
msgstr "三"
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	t.Run("-2 means entries 1-2", func(t *testing.T) {
		var buf bytes.Buffer
		err := MsgSelect(poFile, "-2", &buf, false, nil)
		if err != nil {
			t.Fatalf("MsgSelect failed: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
			t.Errorf("output should contain First and Second, got:\n%s", output)
		}
		if strings.Contains(output, "Third") {
			t.Errorf("output should not contain Third, got:\n%s", output)
		}
	})

	t.Run("2- means entries 2 to last", func(t *testing.T) {
		var buf bytes.Buffer
		err := MsgSelect(poFile, "2-", &buf, false, nil)
		if err != nil {
			t.Fatalf("MsgSelect failed: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Second") || !strings.Contains(output, "Third") {
			t.Errorf("output should contain Second and Third, got:\n%s", output)
		}
		if strings.Contains(output, "First") {
			t.Errorf("output should not contain First (entry 1), got:\n%s", output)
		}
	})
}

func TestMsgSelect_NoContentEntries(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "一"

msgid "Second"
msgstr "二"
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Range selects only out-of-range entries (file has 2 content entries)
	var buf bytes.Buffer
	err := MsgSelect(poFile, "10-20", &buf, false, nil)
	if err != nil {
		t.Fatalf("MsgSelect failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("output should be empty when no content entries selected, got %d bytes:\n%s", buf.Len(), buf.String())
	}
}

func TestMsgSelect_NoHeader(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "一"

msgid "Second"
msgstr "二"
`

	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var buf bytes.Buffer
	err := MsgSelect(poFile, "1-2", &buf, true, nil)
	if err != nil {
		t.Fatalf("MsgSelect failed: %v", err)
	}
	output := buf.String()
	if strings.Contains(output, "Content-Type") {
		t.Errorf("output should not contain header when noHeader=true, got:\n%s", output)
	}
	if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
		t.Errorf("output should contain First and Second, got:\n%s", output)
	}
}

func TestWriteGettextJSONFromPOFile_SingleEntry(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "第一个"

msgid "Second"
msgstr "第二个"
`
	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	var buf bytes.Buffer
	err := WriteGettextJSONFromPOFile(poFile, "1", &buf, nil)
	if err != nil {
		t.Fatalf("WriteGettextJSONFromPOFile failed: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if decoded.HeaderMeta != "Content-Type: text/plain; charset=UTF-8\n" {
		t.Errorf("HeaderMeta: got %q", decoded.HeaderMeta)
	}
	if len(decoded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(decoded.Entries))
	}
	if decoded.Entries[0].MsgID != "First" || decoded.Entries[0].MsgStr != "第一个" {
		t.Errorf("entry: msgid=%q msgstr=%q", decoded.Entries[0].MsgID, decoded.Entries[0].MsgStr)
	}
}

func TestWriteGettextJSONFromPOFile_EmptyRange(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "First"
msgstr "一"

msgid "Second"
msgstr "二"
`
	tmpDir := t.TempDir()
	poFile := filepath.Join(tmpDir, "test.po")
	if err := os.WriteFile(poFile, []byte(poContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	var buf bytes.Buffer
	err := WriteGettextJSONFromPOFile(poFile, "99-100", &buf, nil)
	if err != nil {
		t.Fatalf("WriteGettextJSONFromPOFile failed: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(decoded.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(decoded.Entries))
	}
	if decoded.HeaderMeta != "Content-Type: text/plain; charset=UTF-8\n" {
		t.Errorf("HeaderMeta: got %q", decoded.HeaderMeta)
	}
}
