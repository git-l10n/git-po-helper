package util

import (
	"bytes"
	"testing"
)

const poHeader = `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

`

// TestPoCompare_Added tests PoCompare when dest has new entries.
func TestPoCompare_Added(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"
`
	destContent := srcContent + `msgid "World"
msgstr "世界"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 1 {
		t.Errorf("expected Added=1, got %d", stat.Added)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if len(data) == 0 {
		t.Errorf("expected non-empty review data")
	}
	if !bytes.Contains(data, []byte("World")) {
		t.Errorf("review data should contain new entry 'World', got: %s", data)
	}
}

// TestPoCompare_NoChange tests PoCompare when files are identical.
func TestPoCompare_NoChange(t *testing.T) {
	content := poHeader + `msgid "Hello"
msgstr "你好"
`

	stat, header, entries, err := PoCompare([]byte(content), []byte(content), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 || stat.Changed != 0 || stat.Deleted != 0 {
		t.Errorf("expected all zeros, got Added=%d Changed=%d Deleted=%d",
			stat.Added, stat.Changed, stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if len(data) != 0 {
		t.Errorf("expected empty review data when no change, got %d bytes", len(data))
	}
}

// TestPoCompare_Deleted tests PoCompare when dest has fewer entries.
func TestPoCompare_Deleted(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 {
		t.Errorf("expected Added=0, got %d", stat.Added)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	if stat.Deleted != 1 {
		t.Errorf("expected Deleted=1, got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if len(data) != 0 {
		t.Errorf("expected empty review data (no new/changed), got %d bytes", len(data))
	}
}

// TestPoCompare_Changed tests PoCompare when same msgid has different content.
func TestPoCompare_Changed(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"
`
	destContent := poHeader + `msgid "Hello"
msgstr "您好"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 {
		t.Errorf("expected Added=0, got %d", stat.Added)
	}
	if stat.Changed != 1 {
		t.Errorf("expected Changed=1, got %d", stat.Changed)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if !bytes.Contains(data, []byte("您好")) {
		t.Errorf("review data should contain changed entry, got: %s", data)
	}
}

// TestPoCompare_ObsoleteSkipped tests that obsolete entries are skipped during comparison.
func TestPoCompare_ObsoleteSkipped(t *testing.T) {
	// src: Hello, obsolete, World. dest: Hello, World (obsolete removed).
	// Obsolete in src should be skipped; comparison should show no change.
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "Obsolete"
#~ msgstr "已废弃"

msgid "World"
msgstr "世界"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 {
		t.Errorf("expected Added=0 (obsolete in src skipped), got %d", stat.Added)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0 (obsolete not counted), got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if len(data) != 0 {
		t.Errorf("expected empty review data when no real change, got %d bytes", len(data))
	}
}

// TestPoCompare_ObsoleteInDest tests that obsolete entries in dest are skipped.
func TestPoCompare_ObsoleteInDest(t *testing.T) {
	// src: Hello. dest: Hello, obsolete, World.
	// Obsolete in dest should be skipped; World is new (Added=1).
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "Obsolete"
#~ msgstr "已废弃"

msgid "World"
msgstr "世界"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 1 {
		t.Errorf("expected Added=1 (World), got %d", stat.Added)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if !bytes.Contains(data, []byte("World")) {
		t.Errorf("review data should contain new entry 'World', got: %s", data)
	}
}

// TestPoCompare_ObsoleteOnlyTrailing tests bounds check when entries end with obsolete.
func TestPoCompare_ObsoleteOnlyTrailing(t *testing.T) {
	// src: Hello + trailing obsolete. dest: Hello only.
	// Should not panic; obsolete at end of src skipped, no deleted.
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "Trailing obsolete"
#~ msgstr ""
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"
`

	stat, _, _, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0 (trailing obsolete skipped), got %d", stat.Deleted)
	}
}

// TestPoCompare_ObsoleteInSrcNormalInDest tests: same msgid obsolete in src, normal in dest → Added.
// Obsolete entries are filtered from src, so the entry appears only in dest and is reported as added.
func TestPoCompare_ObsoleteInSrcNormalInDest(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "Revived"
#~ msgstr "复活"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"

msgid "Revived"
msgstr "复活"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 1 {
		t.Errorf("expected Added=1 (Revived: obsolete in src → normal in dest), got %d", stat.Added)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", stat.Deleted)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	data := BuildPoContent(header, entries)
	if !bytes.Contains(data, []byte("Revived")) || !bytes.Contains(data, []byte("复活")) {
		t.Errorf("review data should contain revived entry, got: %s", data)
	}
}

// TestPoCompare_NormalInSrcObsoleteInDest tests: same msgid normal in src, obsolete in dest → Deleted.
// Obsolete entries are filtered from dest, so the entry appears only in src and is reported as deleted.
func TestPoCompare_NormalInSrcObsoleteInDest(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

msgid "Deprecated"
msgstr "已废弃"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "Deprecated"
#~ msgstr "已废弃"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 {
		t.Errorf("expected Added=0, got %d", stat.Added)
	}
	if stat.Deleted != 1 {
		t.Errorf("expected Deleted=1 (Deprecated: normal in src → obsolete in dest), got %d", stat.Deleted)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	data := BuildPoContent(header, entries)
	if len(data) != 0 {
		t.Errorf("expected empty review data (no new/changed entries), got %d bytes", len(data))
	}
}

// TestPoCompare_DifferentObsoleteInBoth tests: different obsolete entries in both files → no added/deleted.
// Obsolete entries are filtered from both, so they do not affect the diff.
func TestPoCompare_DifferentObsoleteInBoth(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "ObsoleteA"
#~ msgstr "废弃A"
`
	destContent := poHeader + `msgid "Hello"
msgstr "你好"

#~ msgid "ObsoleteB"
#~ msgstr "废弃B"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 0 {
		t.Errorf("expected Added=0 (obsolete in both filtered), got %d", stat.Added)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0 (obsolete in both filtered), got %d", stat.Deleted)
	}
	if stat.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", stat.Changed)
	}
	data := BuildPoContent(header, entries)
	if len(data) != 0 {
		t.Errorf("expected empty review data (no new/changed entries), got %d bytes", len(data))
	}
}

// TestPoCompare_EmptySrc tests PoCompare when src is empty (all dest entries are new).
func TestPoCompare_EmptySrc(t *testing.T) {
	destContent := poHeader + `msgid "Hello"
msgstr "你好"
`

	stat, header, entries, err := PoCompare([]byte{}, []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 1 {
		t.Errorf("expected Added=1, got %d", stat.Added)
	}
	if stat.Deleted != 0 {
		t.Errorf("expected Deleted=0, got %d", stat.Deleted)
	}
	data := BuildPoContent(header, entries)
	if !bytes.Contains(data, []byte("Hello")) {
		t.Errorf("review data should contain entry, got: %s", data)
	}
}

// TestPoCompare_OutputRoundTrip verifies the returned data can be parsed back.
func TestPoCompare_OutputRoundTrip(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"
`
	destContent := srcContent + `msgid "World"
msgstr "世界"
`

	_, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), false)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Skip("no data to round-trip")
	}

	data := BuildPoContent(header, entries)
	parsedEntries, parsedHeader, err := ParsePoEntries(data)
	if err != nil {
		t.Fatalf("ParsePoEntries of output failed: %v", err)
	}
	if len(parsedEntries) != 1 {
		t.Errorf("expected 1 entry in review output, got %d", len(parsedEntries))
	}
	if len(parsedEntries) > 0 && parsedEntries[0].MsgID != "World" {
		t.Errorf("expected MsgID 'World', got %q", parsedEntries[0].MsgID)
	}
	if len(parsedHeader) == 0 {
		t.Errorf("expected non-empty header")
	}

	// Round-trip: BuildPoContent should produce same bytes
	rebuilt := BuildPoContent(parsedHeader, parsedEntries)
	if !bytes.Equal(data, rebuilt) {
		t.Errorf("round-trip mismatch: BuildPoContent output differs from PoCompare output")
	}
}

// TestPoCompare_NoHeader tests PoCompare with noHeader=true (empty header in output).
func TestPoCompare_NoHeader(t *testing.T) {
	srcContent := poHeader + `msgid "Hello"
msgstr "你好"
`
	destContent := srcContent + `msgid "World"
msgstr "世界"
`

	stat, header, entries, err := PoCompare([]byte(srcContent), []byte(destContent), true)
	if err != nil {
		t.Fatalf("PoCompare returned error: %v", err)
	}
	if stat.Added != 1 {
		t.Errorf("expected Added=1, got %d", stat.Added)
	}
	data := BuildPoContent(header, entries)
	if !bytes.Contains(data, []byte("World")) {
		t.Errorf("review data should contain new entry 'World', got: %s", data)
	}
	// With noHeader, output should not contain msgid "" header block
	if bytes.Contains(data, []byte("msgid \"\"\nmsgstr \"\"")) {
		t.Errorf("output with noHeader should not contain header block, got: %s", data)
	}
	// Should start with first content entry
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("msgid ")) {
		t.Errorf("output should start with msgid, got: %s", data)
	}
}

// TestGettextEntriesEqual tests GettextEntriesEqual.
func TestGettextEntriesEqual(t *testing.T) {
	tests := []struct {
		name string
		e1   *GettextEntry
		e2   *GettextEntry
		want bool
	}{
		{
			name: "identical",
			e1:   &GettextEntry{MsgID: "a", MsgStr: "x"},
			e2:   &GettextEntry{MsgID: "a", MsgStr: "x"},
			want: true,
		},
		{
			name: "different msgstr",
			e1:   &GettextEntry{MsgID: "a", MsgStr: "x"},
			e2:   &GettextEntry{MsgID: "a", MsgStr: "y"},
			want: false,
		},
		{
			name: "different msgid",
			e1:   &GettextEntry{MsgID: "a", MsgStr: "x"},
			e2:   &GettextEntry{MsgID: "b", MsgStr: "x"},
			want: false,
		},
		{
			name: "different Fuzzy",
			e1:   &GettextEntry{MsgID: "a", MsgStr: "x", Fuzzy: false},
			e2:   &GettextEntry{MsgID: "a", MsgStr: "x", Fuzzy: true},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GettextEntriesEqual(tt.e1, tt.e2)
			if got != tt.want {
				t.Errorf("GettextEntriesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
