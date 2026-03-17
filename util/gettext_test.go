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
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgctxt "Menu"
msgid "File"
msgstr "文件"

msgctxt "Menu"
msgid "Edit"
msgstr "编辑"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "No context"
msgstr "无上下文"

msgctxt ""
msgid "Empty context"
msgstr "空上下文"
`,
	// Phase 3: #= flag lines (gettext format evolution, June 2025)
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#= c-format
msgid "One %s"
msgstr "一个 %s"

msgid "No flags"
msgstr "无标志"
`,
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
#= no-wrap
msgid "Fuzzy no-wrap"
msgstr "模糊不换行"

#, c-format
#= range: 0..1
msgid "%d item"
msgid_plural "%d items"
msgstr[0] "%d 项"
msgstr[1] "%d 项"
`,
	// Phase 4: obsolete with #~ #: and #~ #, (7.2 Option A round-trip via RawLines)
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~ #: branch.c builtin/branch.c
#~ #, fuzzy
#~ msgid "See 'git help check-ref-format'"
#~ msgstr "查阅 'man git check-ref-format'"
`,
	// Phase 4: obsolete with #~| msgctxt and #~| msgid (MsgCtxtPrevious round-trip)
	`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~| msgctxt "OldMenu"
#~| msgid "Old source"
#~ msgctxt "Menu"
#~ msgid "Obsolete"
#~ msgstr "已废弃"
`,
}

func TestParsePoEntriesRoundTripBytes(t *testing.T) {
	for i, poContent := range poRoundTripExamples {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			original := []byte(poContent)
			po, err := ParsePoEntries(original)
			if err != nil {
				t.Fatalf("ParsePoEntries failed: %v", err)
			}
			written := BuildPoContent(po.HeaderLines(), po.EntriesPtr())
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
	po, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries failed: %v", err)
	}
	entries := po.EntriesPtr()
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

func TestParsePoEntriesHashEq(t *testing.T) {
	// Phase 3: #= flag lines (gettext format evolution, June 2025) are stored in Comments and preserved.
	po := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#= c-format
msgid "Format %s"
msgstr "格式 %s"

#, fuzzy
#= no-wrap
msgid "Fuzzy no-wrap"
msgstr "模糊不换行"
`
	parsed, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatalf("ParsePoEntries failed: %v", err)
	}
	if len(parsed.HeaderLines()) == 0 {
		t.Fatal("expected header")
	}
	entries := parsed.EntriesPtr()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// First entry: only #= line; must be in Comments and not set Fuzzy from #=.
	hasHashEq := false
	for _, c := range entries[0].Comments {
		if strings.HasPrefix(strings.TrimSpace(c), "#=") {
			hasHashEq = true
			break
		}
	}
	if !hasHashEq {
		t.Errorf("entry 0: expected a comment line starting with #= in Comments, got %q", entries[0].Comments)
	}
	if entries[0].Fuzzy {
		t.Errorf("entry 0: #= line must not set Fuzzy; fuzzy is only from #,")
	}
	if entries[0].MsgID != "Format %s" {
		t.Errorf("entry 0: MsgID = %q", entries[0].MsgID)
	}
	// Second entry: #, fuzzy and #= no-wrap; both preserved.
	hasHashComma := false
	hasHashEq2 := false
	for _, c := range entries[1].Comments {
		trimmed := strings.TrimSpace(c)
		if strings.HasPrefix(trimmed, "#,") {
			hasHashComma = true
		}
		if strings.HasPrefix(trimmed, "#=") {
			hasHashEq2 = true
		}
	}
	if !hasHashComma || !hasHashEq2 {
		t.Errorf("entry 1: expected both #, and #= in Comments, got %q", entries[1].Comments)
	}
	if !entries[1].Fuzzy {
		t.Errorf("entry 1: expected Fuzzy=true from #, fuzzy")
	}
	// Round-trip: build back and parse again; #= lines must still be present.
	written := BuildPoContent(parsed.HeaderLines(), entries)
	po2, err := ParsePoEntries(written)
	if err != nil {
		t.Fatalf("second ParsePoEntries failed: %v", err)
	}
	entries2 := po2.EntriesPtr()
	if len(entries2) != 2 {
		t.Fatalf("after round-trip expected 2 entries, got %d", len(entries2))
	}
	for i, e := range entries2 {
		var hasEq bool
		for _, c := range e.Comments {
			if strings.HasPrefix(strings.TrimSpace(c), "#=") {
				hasEq = true
				break
			}
		}
		if i == 0 && !hasEq {
			t.Errorf("after round-trip entry 0: #= line lost")
		}
		if i == 1 && !hasEq {
			t.Errorf("after round-trip entry 1: #= line lost")
		}
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
	parsed, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	entries := parsed.EntriesPtr()
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

func TestParsePoEntriesObsoleteComment72(t *testing.T) {
	// gettext-json-format 7.2 Option A: obsolete comment lines stored without "#~ " prefix; writer prepends "#~ " when emitting.
	po := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~ #: branch.c builtin/branch.c
#~ #, fuzzy
#~ msgid "Obsolete"
#~ msgstr "已废弃"
`
	parsedPO, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	entries := parsedPO.EntriesPtr()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	ob := entries[1]
	if !ob.Obsolete || ob.MsgID != "Obsolete" {
		t.Fatalf("entry 1: Obsolete=%v MsgID=%q", ob.Obsolete, ob.MsgID)
	}
	// Comments must be stored without "#~ " prefix (7.2 Option A).
	if len(ob.Comments) != 2 {
		t.Fatalf("expected 2 comment lines, got %d", len(ob.Comments))
	}
	if ob.Comments[0] != "#: branch.c builtin/branch.c" {
		t.Errorf("Comments[0]: got %q", ob.Comments[0])
	}
	if ob.Comments[1] != "#, fuzzy" {
		t.Errorf("Comments[1]: got %q", ob.Comments[1])
	}
	// Build from entry (no RawLines) must emit "#~ " before each comment line.
	obNoRaw := *ob
	obNoRaw.RawLines = nil
	var buf bytes.Buffer
	if err := writeGettextEntryToPO(&buf, obNoRaw); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "#~ #: branch.c") {
		t.Errorf("output should contain #~ #: branch.c, got:\n%s", out)
	}
	if !strings.Contains(out, "#~ #, fuzzy") {
		t.Errorf("output should contain #~ #, fuzzy, got:\n%s", out)
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
	parsedPO, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatal(err)
	}
	entries := parsedPO.EntriesPtr()
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
				if entries[0].MsgStrSingle() != "你好" {
					t.Errorf("expected first entry MsgStr '你好', got '%s'", entries[0].MsgStr)
				}
				if entries[1].MsgID != "World" {
					t.Errorf("expected second entry MsgID 'World', got '%s'", entries[1].MsgID)
				}
				if entries[1].MsgStrSingle() != "世界" {
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
				if entries[0].MsgStrSingle() != expectedMsgStr {
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
				if len(entries[0].MsgStr) != 2 {
					t.Fatalf("expected 2 plural forms, got %d", len(entries[0].MsgStr))
				}
				if entries[0].MsgStr[0] != "一个项目" {
					t.Errorf("expected first plural form '一个项目', got '%s'", entries[0].MsgStr[0])
				}
				if entries[0].MsgStr[1] != "多个项目" {
					t.Errorf("expected second plural form '多个项目', got '%s'", entries[0].MsgStr[1])
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
				if len(entries[0].MsgStr) != 2 {
					t.Fatalf("expected 2 plural forms, got %d", len(entries[0].MsgStr))
				}
				if entries[0].MsgStr[0] != "一个项目" {
					t.Errorf("expected first plural form '一个项目', got '%s'", entries[0].MsgStr[0])
				}
				if entries[0].MsgStr[1] != "多个项目" {
					t.Errorf("expected second plural form '多个项目', got '%s'", entries[0].MsgStr[1])
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
				if entries[0].MsgStrSingle() != "带注释的字符串" {
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
				if entries[0].MsgStrSingle() != "" {
					t.Errorf("expected first entry MsgStr to be empty, got '%s'", entries[0].MsgStr)
				}
				if entries[1].MsgStrSingle() != "已翻译" {
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
		{
			name: "PO file with msgctxt (context)",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "No context"
msgstr "无上下文"

msgctxt "Menu"
msgid "File"
msgstr "文件"

msgctxt "Menu"
msgid "Edit"
msgstr "编辑"

msgctxt ""
msgid "Empty context"
msgstr "空上下文"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 4,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 4 {
					t.Fatalf("expected 4 entries, got %d", len(entries))
				}
				if entries[0].MsgCtxt != nil {
					t.Errorf("entry 0: expected no msgctxt, got %q", *entries[0].MsgCtxt)
				}
				if entries[0].MsgID != "No context" || entries[0].MsgStrSingle() != "无上下文" {
					t.Errorf("entry 0: got MsgID=%q MsgStr=%q", entries[0].MsgID, entries[0].MsgStrSingle())
				}
				if entries[1].MsgCtxt == nil || *entries[1].MsgCtxt != "Menu" {
					t.Errorf("entry 1: expected msgctxt 'Menu', got %v", entries[1].MsgCtxt)
				}
				if entries[1].MsgID != "File" || entries[1].MsgStrSingle() != "文件" {
					t.Errorf("entry 1: got MsgID=%q MsgStr=%q", entries[1].MsgID, entries[1].MsgStrSingle())
				}
				if entries[2].MsgCtxt == nil || *entries[2].MsgCtxt != "Menu" {
					t.Errorf("entry 2: expected msgctxt 'Menu', got %v", entries[2].MsgCtxt)
				}
				if entries[2].MsgID != "Edit" || entries[2].MsgStrSingle() != "编辑" {
					t.Errorf("entry 2: got MsgID=%q MsgStr=%q", entries[2].MsgID, entries[2].MsgStrSingle())
				}
				if entries[3].MsgCtxt == nil {
					t.Errorf("entry 3: expected msgctxt present (empty string)")
				} else if *entries[3].MsgCtxt != "" {
					t.Errorf("entry 3: expected empty msgctxt, got %q", *entries[3].MsgCtxt)
				}
				if entries[3].MsgID != "Empty context" || entries[3].MsgStrSingle() != "空上下文" {
					t.Errorf("entry 3: got MsgID=%q MsgStr=%q", entries[3].MsgID, entries[3].MsgStrSingle())
				}
			},
		},
		{
			name: "PO file with obsolete entry and #~| msgctxt",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~| msgctxt "OldMenu"
#~| msgid "Old source"
#~ msgctxt "Menu"
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
					t.Fatalf("expected 2 entries, got %d", len(entries))
				}
				if entries[1].MsgCtxt == nil || *entries[1].MsgCtxt != "Menu" {
					t.Errorf("entry 1 msgctxt: got %v", entries[1].MsgCtxt)
				}
				if entries[1].MsgCtxtPrevious == nil || *entries[1].MsgCtxtPrevious != "OldMenu" {
					t.Errorf("entry 1 msgctxt_previous: got %v", entries[1].MsgCtxtPrevious)
				}
				if entries[1].MsgID != "Obsolete" || entries[1].MsgIDPrevious != "Old source" || !entries[1].Obsolete {
					t.Errorf("entry 1: MsgID=%q MsgIDPrevious=%q Obsolete=%v", entries[1].MsgID, entries[1].MsgIDPrevious, entries[1].Obsolete)
				}
			},
		},
		{
			name: "blank lines between location comments and msgid are ignored, comments stay with entry",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: file.c:100

#: other.c:50

msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
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
				expectedComments := []string{"#: file.c:100", "#: other.c:50"}
				if len(entries[0].Comments) != len(expectedComments) {
					t.Errorf("expected %d comments for first entry, got %d", len(expectedComments), len(entries[0].Comments))
				} else {
					for i, expected := range expectedComments {
						if entries[0].Comments[i] != expected {
							t.Errorf("comment %d: expected %q, got %q", i, expected, entries[0].Comments[i])
						}
					}
				}
				if entries[0].MsgID != "Hello" || entries[0].MsgStrSingle() != "你好" {
					t.Errorf("first entry: expected MsgID=Hello MsgStr=你好, got MsgID=%q MsgStr=%q", entries[0].MsgID, entries[0].MsgStrSingle())
				}
				if entries[1].MsgID != "World" || entries[1].MsgStrSingle() != "世界" {
					t.Errorf("second entry: expected MsgID=World MsgStr=世界, got MsgID=%q MsgStr=%q", entries[1].MsgID, entries[1].MsgStrSingle())
				}
			},
		},
		{
			name: "blank lines between msgid and msgstr are ignored",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Key"

msgstr "值"

msgid "Another"

msgstr "另一个"
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
				if entries[0].MsgID != "Key" || entries[0].MsgStrSingle() != "值" {
					t.Errorf("first entry: expected MsgID=Key MsgStr=值, got MsgID=%q MsgStr=%q", entries[0].MsgID, entries[0].MsgStrSingle())
				}
				if entries[1].MsgID != "Another" || entries[1].MsgStrSingle() != "另一个" {
					t.Errorf("second entry: expected MsgID=Another MsgStr=另一个, got MsgID=%q MsgStr=%q", entries[1].MsgID, entries[1].MsgStrSingle())
				}
			},
		},
		{
			name: "BuildPoContent omits meaningless blank lines (comments-msgid and msgid-msgstr)",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: a.c

msgid "One"

msgstr "一"
`,
			expectedHeader: []string{
				`msgid ""`,
				`msgstr ""`,
				`"Content-Type: text/plain; charset=UTF-8\n"`,
			},
			expectedCount: 1,
			validateEntry: func(t *testing.T, entries []*GettextEntry) {
				if len(entries) != 1 {
					t.Fatalf("expected 1 entry, got %d", len(entries))
				}
				written := BuildPoContent([]string{`msgid ""`, `msgstr ""`, `"Content-Type: text/plain; charset=UTF-8\n"`}, entries)
				// Output must not contain two consecutive blank lines inside the entry (no "\n\n\n")
				if strings.Contains(string(written), "\n\n\n") {
					t.Errorf("BuildPoContent should not output consecutive blank lines inside entry, got:\n%s", string(written))
				}
				// Re-parse and verify same content
				po2, err := ParsePoEntries(written)
				if err != nil {
					t.Fatalf("ParsePoEntries of built content: %v", err)
				}
				entries2 := po2.EntriesPtr()
				if len(entries2) != 1 {
					t.Fatalf("re-parsed entry count: expected 1, got %d", len(entries2))
				}
				if entries2[0].MsgID != entries[0].MsgID || entries2[0].MsgStrSingle() != entries[0].MsgStrSingle() {
					t.Errorf("round-trip mismatch: got MsgID=%q MsgStr=%q", entries2[0].MsgID, entries2[0].MsgStrSingle())
				}
			},
		},
		{
			name: "blank line between every part: comment, msgid, msgstr, and between entries",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: foo.c

msgid "First"

msgstr "第一个"

#: bar.c

msgid "Second"

msgstr "第二个"
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
				if len(entries[0].Comments) != 1 || entries[0].Comments[0] != "#: foo.c" {
					t.Errorf("entry 0 comments: expected [#: foo.c], got %v", entries[0].Comments)
				}
				if entries[0].MsgID != "First" || entries[0].MsgStrSingle() != "第一个" {
					t.Errorf("entry 0: expected MsgID=First MsgStr=第一个, got MsgID=%q MsgStr=%q", entries[0].MsgID, entries[0].MsgStrSingle())
				}
				if len(entries[1].Comments) != 1 || entries[1].Comments[0] != "#: bar.c" {
					t.Errorf("entry 1 comments: expected [#: bar.c], got %v", entries[1].Comments)
				}
				if entries[1].MsgID != "Second" || entries[1].MsgStrSingle() != "第二个" {
					t.Errorf("entry 1: expected MsgID=Second MsgStr=第二个, got MsgID=%q MsgStr=%q", entries[1].MsgID, entries[1].MsgStrSingle())
				}
			},
		},
		{
			name: "no blank lines: comments, msgid, msgstr and next entry back-to-back",
			poContent: `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: a.c
msgid "Alpha"
msgstr "甲"
#: b.c
msgid "Beta"
msgstr "乙"
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
				if len(entries[0].Comments) != 1 || entries[0].Comments[0] != "#: a.c" {
					t.Errorf("entry 0 comments: expected [#: a.c], got %v", entries[0].Comments)
				}
				if entries[0].MsgID != "Alpha" || entries[0].MsgStrSingle() != "甲" {
					t.Errorf("entry 0: expected MsgID=Alpha MsgStr=甲, got MsgID=%q MsgStr=%q", entries[0].MsgID, entries[0].MsgStrSingle())
				}
				if len(entries[1].Comments) != 1 || entries[1].Comments[0] != "#: b.c" {
					t.Errorf("entry 1 comments: expected [#: b.c], got %v", entries[1].Comments)
				}
				if entries[1].MsgID != "Beta" || entries[1].MsgStrSingle() != "乙" {
					t.Errorf("entry 1: expected MsgID=Beta MsgStr=乙, got MsgID=%q MsgStr=%q", entries[1].MsgID, entries[1].MsgStrSingle())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po, err := ParsePoEntries([]byte(tt.poContent))
			if err != nil {
				t.Fatalf("ParsePoEntries failed: %v", err)
			}
			header := po.HeaderLines()
			entries := po.EntriesPtr()

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
		{"-", 10, nil, true},               // Invalid: both empty
		{"~3", 10, []int{8, 9, 10}, false}, // ~N: last N entries
		{"~5", 10, []int{6, 7, 8, 9, 10}, false},
		{"~10", 10, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false},
		{"~15", 10, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false}, // N > maxEntry: all
		{"~1", 10, []int{10}, false},
		{"~0", 10, []int{}, false}, // N=0: none
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
	if decoded.HeaderMeta != "Content-Type: text/plain; charset=UTF-8\\n" {
		t.Errorf("HeaderMeta: got %q", decoded.HeaderMeta)
	}
	if len(decoded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(decoded.Entries))
	}
	if decoded.Entries[0].MsgID != "First" || decoded.Entries[0].MsgStrSingle() != "第一个" {
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
	// Empty range: no output (empty file semantics for msg-select)
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer when range selects nothing, got %d bytes", buf.Len())
	}
}
