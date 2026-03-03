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

func TestSplitHeader_NoComment(t *testing.T) {
	header := []string{
		"msgid \"\"",
		"msgstr \"\"",
		"\"Project-Id-Version: git\\n\"",
		"\"Content-Type: text/plain; charset=UTF-8\\n\"",
	}
	comment, meta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	if comment != "" {
		t.Errorf("header_comment: expected empty, got %q", comment)
	}
	expectedMeta := "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n"
	if meta != expectedMeta {
		t.Errorf("header_meta: got %q, want %q", meta, expectedMeta)
	}
}

func TestSplitHeader_CommentOnly(t *testing.T) {
	header := []string{
		"# Glossary:",
		"# term1\tTranslation 1",
		"#",
	}
	comment, meta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	expectedComment := "# Glossary:\n# term1\tTranslation 1\n#\n"
	if comment != expectedComment {
		t.Errorf("header_comment: got %q, want %q", comment, expectedComment)
	}
	if meta != "" {
		t.Errorf("header_meta: expected empty, got %q", meta)
	}
}

func TestSplitHeader_CommentAndHeaderBlock(t *testing.T) {
	header := []string{
		"# Glossary:",
		"# term1\tTranslation 1",
		"#",
		"msgid \"\"",
		"msgstr \"\"",
		"\"Project-Id-Version: git\\n\"",
		"\"Content-Type: text/plain; charset=UTF-8\\n\"",
	}
	comment, meta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	expectedComment := "# Glossary:\n# term1\tTranslation 1\n#\n"
	if comment != expectedComment {
		t.Errorf("header_comment: got %q, want %q", comment, expectedComment)
	}
	expectedMeta := "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n"
	if meta != expectedMeta {
		t.Errorf("header_meta: got %q, want %q", meta, expectedMeta)
	}
}

func TestSplitHeader_MultiLineHeaderMeta(t *testing.T) {
	header := []string{
		"msgid \"\"",
		"msgstr \"\"",
		"\"Project-Id-Version: git\\n\"",
		"\"Content-Type: text/plain; charset=UTF-8\\n\"",
		"\"Plural-Forms: nplurals=2; plural=(n != 1);\\n\"",
	}
	_, meta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	expectedMeta := "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\nPlural-Forms: nplurals=2; plural=(n != 1);\\n"
	if meta != expectedMeta {
		t.Errorf("header_meta: got %q, want %q", meta, expectedMeta)
	}
}

func TestSplitHeader_Empty(t *testing.T) {
	comment, meta, err := SplitHeader(nil)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	if comment != "" || meta != "" {
		t.Errorf("expected both empty, got comment=%q meta=%q", comment, meta)
	}
}

func TestBuildGettextJSON_RoundTrip(t *testing.T) {
	entries := []*GettextEntry{
		{
			MsgID:    "Hello",
			MsgStr:   "你好",
			Comments: []string{"#. Comment\n", "#: src/file.c:10\n"},
			Fuzzy:    false,
		},
		{
			MsgID:        "One file",
			MsgStr:       "",
			MsgIDPlural:  "%d files",
			MsgStrPlural: []string{"一个文件", "%d 个文件"},
			Comments:     []string{"#, c-format\n"},
			Fuzzy:        false,
		},
	}
	var buf bytes.Buffer
	err := BuildGettextJSON("", "Project-Id-Version: git\n", entries, &buf)
	if err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	decoded, err := ParseGettextJSON(&buf)
	if err != nil {
		t.Fatalf("ParseGettextJSON: %v", err)
	}
	if decoded.HeaderMeta != "Project-Id-Version: git\\n" {
		t.Errorf("HeaderMeta: got %q", decoded.HeaderMeta)
	}
	if len(decoded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(decoded.Entries))
	}
	e0 := decoded.Entries[0]
	if e0.MsgID != "Hello" || e0.MsgStr != "你好" || e0.Fuzzy != false {
		t.Errorf("entry 0: msgid=%q msgstr=%q fuzzy=%v", e0.MsgID, e0.MsgStr, e0.Fuzzy)
	}
	e1 := decoded.Entries[1]
	if e1.MsgID != "One file" || e1.MsgStr != "" || e1.MsgIDPlural != "%d files" ||
		len(e1.MsgStrPlural) != 2 || e1.MsgStrPlural[0] != "一个文件" || e1.MsgStrPlural[1] != "%d 个文件" {
		t.Errorf("entry 1: msgid=%q msgstr=%q msgid_plural=%q msgstr_plural=%v",
			e1.MsgID, e1.MsgStr, e1.MsgIDPlural, e1.MsgStrPlural)
	}
}

func TestBuildGettextJSON_EmptyEntries(t *testing.T) {
	var buf bytes.Buffer
	err := BuildGettextJSON("# comment\n", "meta\n", nil, &buf)
	if err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if decoded.HeaderComment != "# comment\n" || decoded.HeaderMeta != "meta\n" || len(decoded.Entries) != 0 {
		t.Errorf("got header_comment=%q header_meta=%q entries len=%d",
			decoded.HeaderComment, decoded.HeaderMeta, len(decoded.Entries))
	}
}

func TestPoUnescape_InMsgidMsgstr(t *testing.T) {
	entries := []*GettextEntry{
		{
			MsgID:  "Line one\nLine two\twith tab",
			MsgStr: "第一行\n第二行\t带制表符",
			Fuzzy:  false,
		},
	}
	var buf bytes.Buffer
	err := BuildGettextJSON("", "", entries, &buf)
	if err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	e := decoded.Entries[0]
	wantMsgid := "Line one\nLine two\twith tab"
	wantMsgstr := "第一行\n第二行\t带制表符"
	if e.MsgID != wantMsgid {
		t.Errorf("msgid: got %q, want %q", e.MsgID, wantMsgid)
	}
	if e.MsgStr != wantMsgstr {
		t.Errorf("msgstr: got %q, want %q", e.MsgStr, wantMsgstr)
	}
}

func TestEntryRangeForJSON(t *testing.T) {
	indices, err := EntryRangeForJSON("1,3", 5)
	if err != nil {
		t.Fatalf("EntryRangeForJSON: %v", err)
	}
	if len(indices) != 2 || indices[0] != 1 || indices[1] != 3 {
		t.Errorf("got %v", indices)
	}
}

func TestSplitHeader_RealPOFromDesign(t *testing.T) {
	poContent := `# Glossary:
# term1	Translation 1
#
msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"

#. Comment for translator
#: src/file.c:10
msgid "Hello"
msgstr "你好"
`
	entries, header, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	comment, meta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	expectedComment := "# Glossary:\n# term1\tTranslation 1\n#\n"
	if comment != expectedComment {
		t.Errorf("header_comment: got %q", comment)
	}
	expectedMeta := "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n"
	if meta != expectedMeta {
		t.Errorf("header_meta: got %q", meta)
	}
	var buf bytes.Buffer
	err = BuildGettextJSON(comment, meta, entries, &buf)
	if err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Entries[0].MsgID != "Hello" || decoded.Entries[0].MsgStr != "你好" {
		t.Errorf("entry: %+v", decoded.Entries[0])
	}
	if len(decoded.Entries[0].Comments) != 2 {
		t.Errorf("comments: got %v", decoded.Entries[0].Comments)
	}
	if !strings.HasPrefix(decoded.Entries[0].Comments[0], "#.") ||
		!strings.HasPrefix(decoded.Entries[0].Comments[1], "#:") {
		t.Errorf("comments: %v", decoded.Entries[0].Comments)
	}
}

func TestWriteGettextJSONToPO_Example2RoundTrip(t *testing.T) {
	jsonStr := `{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\n",
  "entries": [
    {
      "msgid": "Line one\nLine two\twith tab, padding for line 2.",
      "msgstr": "第一行\n第二行\t带制表符, 第二行的填充。",
      "comments": ["#, c-format\n"],
      "fuzzy": false
    }
  ]
}`
	j, err := ParseGettextJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	poBytes := poBuf.Bytes()
	entries, header, err := ParsePoEntries(poBytes)
	if err != nil {
		t.Fatalf("ParsePoEntries of converted PO: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after round-trip, got %d", len(entries))
	}
	e := entries[0]
	wantMsgid := "Line one\\nLine two\\twith tab, padding for line 2."
	wantMsgstr := "第一行\\n第二行\\t带制表符, 第二行的填充。"
	if e.MsgID != wantMsgid {
		t.Errorf("msgid round-trip: got %q", e.MsgID)
	}
	if e.MsgStr != wantMsgstr {
		t.Errorf("msgstr round-trip: got %q", e.MsgStr)
	}
	headerComment, headerMeta, _ := SplitHeader(header)
	var jsonBuf bytes.Buffer
	if err := BuildGettextJSON(headerComment, headerMeta, entries, &jsonBuf); err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	var j2 GettextJSON
	if err := json.Unmarshal(jsonBuf.Bytes(), &j2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Both j and j2 have PO format; entries from ParsePoEntries are in PO format
	if j2.Entries[0].MsgID != j.Entries[0].MsgID || j2.Entries[0].MsgStr != j.Entries[0].MsgStr {
		t.Errorf("round-trip JSON: msgid %q vs %q, msgstr %q vs %q",
			j2.Entries[0].MsgID, j.Entries[0].MsgID, j2.Entries[0].MsgStr, j.Entries[0].MsgStr)
	}
}

func TestWriteGettextJSONToPO_Example3PluralRoundTrip(t *testing.T) {
	jsonStr := `{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\nPlural-Forms: nplurals=2; plural=(n != 1);\n",
  "entries": [
    {
      "msgid": "One file",
      "msgstr": "",
      "msgid_plural": "%d files",
      "msgstr_plural": ["一个文件", "%d 个文件"],
      "comments": ["#, c-format\n"],
      "fuzzy": false
    }
  ]
}`
	j, err := ParseGettextJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.MsgID != "One file" || e.MsgStr != "" || e.MsgIDPlural != "%d files" ||
		len(e.MsgStrPlural) != 2 || e.MsgStrPlural[0] != "一个文件" || e.MsgStrPlural[1] != "%d 个文件" {
		t.Errorf("plural entry: msgid=%q msgstr=%q msgid_plural=%q msgstr_plural=%v",
			e.MsgID, e.MsgStr, e.MsgIDPlural, e.MsgStrPlural)
	}
}

func TestWriteGettextJSONToPO_SpecialChars(t *testing.T) {
	j := &GettextJSON{
		HeaderComment: "",
		HeaderMeta:    "",
		Entries: []GettextEntry{{
			MsgID:  "Quote \" and backslash \\ and tab\t and newline\n",
			MsgStr: "相同",
			Fuzzy:  false,
		}},
	}
	var buf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &buf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries, _, err := ParsePoEntries(buf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	want := "Quote \" and backslash \\ and tab\t and newline\n"
	if poUnescape(entries[0].MsgID) != want {
		t.Errorf("msgid: got %q", poUnescape(entries[0].MsgID))
	}
}

// TestEscapeChars_JSONInputWithNewlineTab verifies JSON with decoded \n, \t
// is converted to PO format and round-trips correctly.
func TestEscapeChars_JSONInputWithNewlineTab(t *testing.T) {
	jsonStr := `{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n",
  "entries": [
    {
      "msgid": "A\\nB\\tC\\rD",
      "msgstr": "甲\\n乙\\t丙\\r丁",
      "fuzzy": false
    }
  ]
}`
	j, err := ParseGettextJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	// After parse, strings are in PO format (backslash+n, backslash+t, etc.)
	wantMsgid := `A\nB\tC\rD`
	wantMsgstr := "甲\\n乙\\t丙\\r丁"
	if j.Entries[0].MsgID != wantMsgid {
		t.Errorf("MsgID after parse: got %q, want %q", j.Entries[0].MsgID, wantMsgid)
	}
	if j.Entries[0].MsgStr != wantMsgstr {
		t.Errorf("MsgStr after parse: got %q, want %q", j.Entries[0].MsgStr, wantMsgstr)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if entries[0].MsgID != wantMsgid || entries[0].MsgStr != wantMsgstr {
		t.Errorf("after PO round-trip: msgid=%q msgstr=%q", entries[0].MsgID, entries[0].MsgStr)
	}
}

// TestEscapeChars_POInputAllSequences verifies PO with \n, \t, \r, \", \\
// round-trips through JSON correctly.
func TestEscapeChars_POInputAllSequences(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid ""
"Quote \" and backslash \\ and tab\t and newline\n"
"and carriage return\r end."
msgstr ""
"引号 \" 反斜杠 \\ 制表符\t 换行\n"
"回车\r 结束。"
`
	entries, header, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	// PO format: \" = backslash+quote, \\ = backslash+backslash, \t \n \r as literal
	wantMsgid := `Quote \" and backslash \\ and tab\t and newline\nand carriage return\r end.`
	wantMsgstr := `引号 \" 反斜杠 \\ 制表符\t 换行\n回车\r 结束。`
	if entries[0].MsgID != wantMsgid {
		t.Errorf("MsgID: got %q, want %q", entries[0].MsgID, wantMsgid)
	}
	if entries[0].MsgStr != wantMsgstr {
		t.Errorf("MsgStr: got %q, want %q", entries[0].MsgStr, wantMsgstr)
	}
	headerComment, headerMeta, _ := SplitHeader(header)
	var jsonBuf bytes.Buffer
	if err := BuildGettextJSON(headerComment, headerMeta, entries, &jsonBuf); err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	j, err := ParseGettextJSONBytes(jsonBuf.Bytes())
	if err != nil {
		t.Fatalf("ParseGettextJSONBytes: %v", err)
	}
	if j.Entries[0].MsgID != wantMsgid || j.Entries[0].MsgStr != wantMsgstr {
		t.Errorf("after JSON: msgid=%q msgstr=%q", j.Entries[0].MsgID, j.Entries[0].MsgStr)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries2, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries (second): %v", err)
	}
	if entries2[0].MsgID != wantMsgid || entries2[0].MsgStr != wantMsgstr {
		t.Errorf("after full round-trip: msgid=%q msgstr=%q", entries2[0].MsgID, entries2[0].MsgStr)
	}
}

// TestEscapeChars_HeaderMetaWithNewlines verifies header_meta with \n
// in JSON round-trips correctly.
func TestEscapeChars_HeaderMetaWithNewlines(t *testing.T) {
	jsonStr := `{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\nPlural-Forms: nplurals=2; plural=(n != 1);\n",
  "entries": [
    {"msgid": "Hello", "msgstr": "你好", "fuzzy": false}
  ]
}`
	j, err := ParseGettextJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	wantMeta := "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\nPlural-Forms: nplurals=2; plural=(n != 1);\\n"
	if j.HeaderMeta != wantMeta {
		t.Errorf("HeaderMeta: got %q, want %q", j.HeaderMeta, wantMeta)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	_, header, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	_, meta, _ := SplitHeader(header)
	if meta != wantMeta {
		t.Errorf("header_meta after round-trip: got %q, want %q", meta, wantMeta)
	}
}

// TestJSONEscape_PythonCompatible verifies JSON escape/unescape matches Python json.dumps/json.loads.
// Python: json.dumps("1 \n 2 \r 3 \" 4 \t 5 \a 6 \\") → '"1 \\n 2 \\r 3 \\" 4 \\t 5 \\u0007 6 \\\\"'
// Python: json.loads('"1 \\n 2 \\r 3 \\" 4 \\t 5 \\u0007 6 \\\\"') → '1 \n 2 \r 3 " 4 \t 5 \x07 6 \\'
func TestJSONEscape_PythonCompatible(t *testing.T) {
	// String with newline, cr, quote, tab, bell, backslash (matches Python input)
	pyInput := "1 \n 2 \r 3 \" 4 \t 5 \x07 6 \\"
	// Expected JSON output (matches Python json.dumps)
	pyDumpsExpected := `"1 \n 2 \r 3 \" 4 \t 5 \u0007 6 \\"`

	// Verify json.Marshal produces same as Python json.dumps
	marshaled, err := json.Marshal(pyInput)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(marshaled) != pyDumpsExpected {
		t.Errorf("json.Marshal: got %q, want %q", string(marshaled), pyDumpsExpected)
	}

	// Verify json.Unmarshal produces same as Python json.loads
	var decoded string
	if err := json.Unmarshal([]byte(pyDumpsExpected), &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded != pyInput {
		t.Errorf("json.Unmarshal: got %q, want %q", decoded, pyInput)
	}

	// Verify gettext JSON round-trip with this string
	jsonStr := `{"header_comment":"","header_meta":"","entries":[{"msgid":"1 \n 2 \r 3 \" 4 \t 5 \u0007 6 \\","msgstr":"same","fuzzy":false}]}`
	j, err := ParseGettextJSONBytes([]byte(jsonStr))
	if err != nil {
		t.Fatalf("ParseGettextJSONBytes: %v", err)
	}
	wantMsgid := "1 \\n 2 \\r 3 \\\" 4 \\t 5 \x07 6 \\\\"
	if j.Entries[0].MsgID != wantMsgid {
		t.Errorf("MsgID after parse: got %q, want %q", j.Entries[0].MsgID, wantMsgid)
	}
	var out bytes.Buffer
	if err := WriteGettextJSONToJSON(j, &out, false); err != nil {
		t.Fatalf("WriteGettextJSONToJSON: %v", err)
	}
	j2, err := ParseGettextJSONBytes(out.Bytes())
	if err != nil {
		t.Fatalf("ParseGettextJSONBytes (round-trip): %v", err)
	}
	if j2.Entries[0].MsgID != wantMsgid {
		t.Errorf("MsgID after round-trip: got %q, want %q", j2.Entries[0].MsgID, wantMsgid)
	}
}

func TestRoundTrip_POToJSONToPOToJSON_Example2(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"

#, c-format
msgid ""
"Line one\n"
"Line two\twith tab, "
"padding for line 2."
msgstr ""
"第一行\n"
"第二行\t带制表符, 第二行的填充。"
`
	entries, header, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	headerComment, headerMeta, err := SplitHeader(header)
	if err != nil {
		t.Fatalf("SplitHeader: %v", err)
	}
	var json1Buf bytes.Buffer
	if err := BuildGettextJSON(headerComment, headerMeta, entries, &json1Buf); err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	j1, err := ParseGettextJSONBytes(json1Buf.Bytes())
	if err != nil {
		t.Fatalf("ParseGettextJSONBytes: %v", err)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j1, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries2, header2, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries (second): %v", err)
	}
	headerComment2, headerMeta2, _ := SplitHeader(header2)
	var json2Buf bytes.Buffer
	if err := BuildGettextJSON(headerComment2, headerMeta2, entries2, &json2Buf); err != nil {
		t.Fatalf("BuildGettextJSON (second): %v", err)
	}
	j2, err := ParseGettextJSONBytes(json2Buf.Bytes())
	if err != nil {
		t.Fatalf("ParseGettextJSONBytes (second): %v", err)
	}
	if len(j2.Entries) != len(j1.Entries) {
		t.Fatalf("entries count: %d vs %d", len(j2.Entries), len(j1.Entries))
	}
	if j2.Entries[0].MsgID != j1.Entries[0].MsgID || j2.Entries[0].MsgStr != j1.Entries[0].MsgStr {
		t.Errorf("round-trip: msgid %q vs %q, msgstr %q vs %q",
			j2.Entries[0].MsgID, j1.Entries[0].MsgID, j2.Entries[0].MsgStr, j1.Entries[0].MsgStr)
	}
}

func TestRoundTrip_PluralExample3(t *testing.T) {
	poContent := `msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

#, c-format
msgid "One file"
msgid_plural "%d files"
msgstr[0] "一个文件"
msgstr[1] "%d 个文件"
`
	entries, header, err := ParsePoEntries([]byte(poContent))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	headerComment, headerMeta, _ := SplitHeader(header)
	var jsonBuf bytes.Buffer
	if err := BuildGettextJSON(headerComment, headerMeta, entries, &jsonBuf); err != nil {
		t.Fatalf("BuildGettextJSON: %v", err)
	}
	j, _ := ParseGettextJSONBytes(jsonBuf.Bytes())
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries2, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries2))
	}
	e := entries2[0]
	if e.MsgID != "One file" || e.MsgIDPlural != "%d files" ||
		len(e.MsgStrPlural) != 2 || e.MsgStrPlural[0] != "一个文件" || e.MsgStrPlural[1] != "%d 个文件" {
		t.Errorf("plural round-trip: %+v", e)
	}
}

func TestWriteGettextJSONToPO_EmptyEntries(t *testing.T) {
	j := &GettextJSON{
		HeaderComment: "# empty\n",
		HeaderMeta:    "Project-Id-Version: git\n",
		Entries:       []GettextEntry{},
	}
	var buf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &buf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries, header, err := ParsePoEntries(buf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
	comment, meta, _ := SplitHeader(header)
	if comment != "# empty\n" || meta != "Project-Id-Version: git\\n" {
		t.Errorf("header: comment=%q meta=%q", comment, meta)
	}
}

func TestWriteGettextJSONToPO_ObsoleteRoundTrip(t *testing.T) {
	po := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~ msgid "Obsolete"
#~ msgstr "已废弃"
`
	entries, header, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	headerComment, headerMeta, _ := SplitHeader(header)
	j := GettextJSONFromEntries(headerComment, headerMeta, entries)
	if j.Entries[0].Obsolete || !j.Entries[1].Obsolete {
		t.Errorf("Obsolete flags: entry0=%v entry1=%v", j.Entries[0].Obsolete, j.Entries[1].Obsolete)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	entries2, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries round-trip: %v", err)
	}
	if len(entries2) != 2 {
		t.Fatalf("round-trip: expected 2 entries, got %d", len(entries2))
	}
	if entries2[0].MsgID != "Active" || entries2[0].Obsolete {
		t.Errorf("round-trip entry0: MsgID=%q Obsolete=%v", entries2[0].MsgID, entries2[0].Obsolete)
	}
	if entries2[1].MsgID != "Obsolete" || !entries2[1].Obsolete {
		t.Errorf("round-trip entry1: MsgID=%q Obsolete=%v", entries2[1].MsgID, entries2[1].Obsolete)
	}
	if !strings.Contains(poBuf.String(), "#~ msgid \"Obsolete\"") {
		t.Errorf("output should contain #~ msgid format: %s", poBuf.String())
	}
}

func TestWriteGettextJSONToPO_ObsoleteWithMsgIDPreviousRoundTrip(t *testing.T) {
	po := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Active"
msgstr "活跃"

#~| msgid "Old source"
#~ msgid "Obsolete"
#~ msgstr "已废弃"
`
	entries, header, err := ParsePoEntries([]byte(po))
	if err != nil {
		t.Fatalf("ParsePoEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].MsgIDPrevious != "Old source" {
		t.Errorf("MsgIDPrevious: got %q", entries[1].MsgIDPrevious)
	}
	headerComment, headerMeta, _ := SplitHeader(header)
	j := GettextJSONFromEntries(headerComment, headerMeta, entries)
	if j.Entries[1].MsgIDPrevious != "Old source" {
		t.Errorf("JSON MsgIDPrevious: got %q", j.Entries[1].MsgIDPrevious)
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatalf("WriteGettextJSONToPO: %v", err)
	}
	if !strings.Contains(poBuf.String(), "#~| msgid \"Old source\"") {
		t.Errorf("output should contain #~| msgid format: %s", poBuf.String())
	}
	entries2, _, err := ParsePoEntries(poBuf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries round-trip: %v", err)
	}
	if entries2[1].MsgIDPrevious != "Old source" {
		t.Errorf("round-trip MsgIDPrevious: got %q", entries2[1].MsgIDPrevious)
	}
}

func TestSelectGettextJSONFromFile_JSONInputToPO(t *testing.T) {
	jsonContent := `{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\n",
  "entries": [
    {
      "msgid": "Line one",
      "msgstr": "第一行",
      "comments": ["#, c-format\n"],
      "fuzzy": false
    }
  ]
}`
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "input.json")
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("write JSON file: %v", err)
	}
	var buf bytes.Buffer
	err := SelectGettextJSONFromFile(jsonFile, "1", &buf, false, nil)
	if err != nil {
		t.Fatalf("SelectGettextJSONFromFile: %v", err)
	}
	entries, _, err := ParsePoEntries(buf.Bytes())
	if err != nil {
		t.Fatalf("ParsePoEntries of PO output: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].MsgID != "Line one" || entries[0].MsgStr != "第一行" {
		t.Errorf("entry: msgid=%q msgstr=%q", entries[0].MsgID, entries[0].MsgStr)
	}
}

func TestSelectGettextJSONFromFile_JSONInputToJSON(t *testing.T) {
	jsonContent := `{"header_comment":"","header_meta":"meta\n","entries":[{"msgid":"A","msgstr":"甲","fuzzy":false},{"msgid":"B","msgstr":"乙","fuzzy":false}]}`
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "input.json")
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("write JSON file: %v", err)
	}
	var buf bytes.Buffer
	err := SelectGettextJSONFromFile(jsonFile, "2", &buf, true, nil)
	if err != nil {
		t.Fatalf("SelectGettextJSONFromFile: %v", err)
	}
	var decoded GettextJSON
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(decoded.Entries) != 1 || decoded.Entries[0].MsgID != "B" {
		t.Errorf("expected single entry B, got %d entries: %+v", len(decoded.Entries), decoded.Entries)
	}
}

func TestSelectGettextJSONFromFile_Range(t *testing.T) {
	jsonContent := `{"header_comment":"","header_meta":"","entries":[{"msgid":"One","msgstr":"一","fuzzy":false},{"msgid":"Two","msgstr":"二","fuzzy":false}]}`
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "input.json")
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("write JSON file: %v", err)
	}
	t.Run("range 1", func(t *testing.T) {
		var buf bytes.Buffer
		if err := SelectGettextJSONFromFile(jsonFile, "1", &buf, true, nil); err != nil {
			t.Fatal(err)
		}
		var j GettextJSON
		if err := json.Unmarshal(buf.Bytes(), &j); err != nil {
			t.Fatal(err)
		}
		if len(j.Entries) != 1 || j.Entries[0].MsgID != "One" {
			t.Errorf("got %v", j.Entries)
		}
	})
	t.Run("range 1-2", func(t *testing.T) {
		var buf bytes.Buffer
		if err := SelectGettextJSONFromFile(jsonFile, "1-2", &buf, true, nil); err != nil {
			t.Fatal(err)
		}
		var j GettextJSON
		if err := json.Unmarshal(buf.Bytes(), &j); err != nil {
			t.Fatal(err)
		}
		if len(j.Entries) != 2 {
			t.Errorf("got %d entries", len(j.Entries))
		}
	})
}

func TestMergeGettextJSON(t *testing.T) {
	// First occurrence of each msgid wins.
	a := &GettextJSON{
		HeaderComment: "# first",
		HeaderMeta:    "H: A\n",
		Entries: []GettextEntry{
			{MsgID: "one", MsgStr: "uno"},
			{MsgID: "two", MsgStr: "due"},
		},
	}
	b := &GettextJSON{
		HeaderComment: "# second",
		HeaderMeta:    "H: B\n",
		Entries: []GettextEntry{
			{MsgID: "two", MsgStr: "ZWEI"},
			{MsgID: "three", MsgStr: "tre"},
		},
	}
	merged := MergeGettextJSON([]*GettextJSON{a, b})
	if merged.HeaderComment != "# first" || merged.HeaderMeta != "H: A\n" {
		t.Errorf("header from first: got comment=%q meta=%q", merged.HeaderComment, merged.HeaderMeta)
	}
	if len(merged.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(merged.Entries))
	}
	if merged.Entries[0].MsgID != "one" || merged.Entries[0].MsgStr != "uno" {
		t.Errorf("entry 0: got %q / %q", merged.Entries[0].MsgID, merged.Entries[0].MsgStr)
	}
	if merged.Entries[1].MsgID != "two" || merged.Entries[1].MsgStr != "due" {
		t.Errorf("entry 1 (first occurrence): got %q / %q", merged.Entries[1].MsgID, merged.Entries[1].MsgStr)
	}
	if merged.Entries[2].MsgID != "three" || merged.Entries[2].MsgStr != "tre" {
		t.Errorf("entry 2: got %q / %q", merged.Entries[2].MsgID, merged.Entries[2].MsgStr)
	}
	// Empty and nil
	empty := MergeGettextJSON(nil)
	if empty == nil || len(empty.Entries) != 0 {
		t.Errorf("MergeGettextJSON(nil): got %v", empty)
	}
	single := MergeGettextJSON([]*GettextJSON{a})
	if len(single.Entries) != 2 || single.HeaderComment != "# first" {
		t.Errorf("MergeGettextJSON([a]): got %d entries", len(single.Entries))
	}
}

// gettextJSONEqualForTest compares two GettextJSON for equality (ignores RawLines).
func gettextJSONEqualForTest(a, b *GettextJSON) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.HeaderComment != b.HeaderComment || a.HeaderMeta != b.HeaderMeta {
		return false
	}
	if len(a.Entries) != len(b.Entries) {
		return false
	}
	for i := range a.Entries {
		e1, e2 := &a.Entries[i], &b.Entries[i]
		if !GettextEntriesEqual(e1, e2) {
			return false
		}
		if e1.MsgIDPrevious != e2.MsgIDPrevious {
			return false
		}
		if len(e1.Comments) != len(e2.Comments) {
			return false
		}
		for j := range e1.Comments {
			if e1.Comments[j] != e2.Comments[j] {
				return false
			}
		}
	}
	return true
}

// TestMsgSelectFromFile_POAndJSONRoundTrip verifies: poContent and jsonContent written to
// two files, loaded as gettext objects, entries' msgid/msgstr compared for consistency.
// Both objects written back to JSON and PO; output matches original files (JSON with indent).
func TestMsgSelectFromFile_POAndJSONRoundTrip(t *testing.T) {
	// PO content with \n, \t (backslash+n, backslash+t in PO format)
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: src/a.c
msgid ""
"Line one\n"
"Line two\twith tab"
msgstr ""
"第一行\n"
"第二行\t带制表符"

#, c-format
msgid "Simple %s"
msgstr "简单 %s"
`
	// JSON content with \n, \t (PO format in JSON: \\n, \\t)
	// WriteGettextJSONToJSON with indent=true produces formatted output for consistency
	jsonContent := `{
  "header_comment": "",
  "header_meta": "Content-Type: text/plain; charset=UTF-8\\n",
  "entries": [
    {
      "msgid": "Line one\\nLine two\\twith tab",
      "msgstr": "第一行\\n第二行\\t带制表符",
      "comments": ["#: src/a.c"],
      "fuzzy": false
    },
    {
      "msgid": "Simple %s",
      "msgstr": "简单 %s",
      "comments": ["#, c-format"],
      "fuzzy": false
    }
  ]
}
`

	dir := t.TempDir()
	poPath := filepath.Join(dir, "input.po")
	jsonPath := filepath.Join(dir, "input.json")
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatalf("write PO: %v", err)
	}
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("write JSON: %v", err)
	}

	// Load from both files
	jFromPO, err := ReadFileToGettextJSON(poPath)
	if err != nil {
		t.Fatalf("ReadFileToGettextJSON(PO): %v", err)
	}
	jFromJSON, err := ReadFileToGettextJSON(jsonPath)
	if err != nil {
		t.Fatalf("ReadFileToGettextJSON(JSON): %v", err)
	}

	// Compare entries: msgid and msgstr must be consistent
	if !gettextJSONEqualForTest(jFromPO, jFromJSON) {
		t.Errorf("entries from PO vs JSON: msgid/msgstr differ\nfrom PO: %+v\nfrom JSON: %+v",
			jFromPO, jFromJSON)
	}

	// Write jFromPO to JSON and PO
	outputFromPOJSON := filepath.Join(dir, "output_from_po.json")
	outputFromPOPO := filepath.Join(dir, "output_from_po.po")
	{
		f, err := os.Create(outputFromPOJSON)
		if err != nil {
			t.Fatalf("create output_from_po.json: %v", err)
		}
		if err := WriteGettextJSONToJSON(jFromPO, f, true); err != nil {
			f.Close()
			t.Fatalf("WriteGettextJSONToJSON (indent): %v", err)
		}
		f.Close()
		var poBuf bytes.Buffer
		if err := WriteGettextJSONToPO(jFromPO, &poBuf, false, false); err != nil {
			t.Fatalf("WriteGettextJSONToPO: %v", err)
		}
		if err := os.WriteFile(outputFromPOPO, poBuf.Bytes(), 0644); err != nil {
			t.Fatalf("write output_from_po.po: %v", err)
		}
	}

	// Write jFromJSON to JSON and PO
	outputFromJSONJSON := filepath.Join(dir, "output_from_json.json")
	outputFromJSONPO := filepath.Join(dir, "output_from_json.po")
	{
		f, err := os.Create(outputFromJSONJSON)
		if err != nil {
			t.Fatalf("create output_from_json.json: %v", err)
		}
		if err := WriteGettextJSONToJSON(jFromJSON, f, true); err != nil {
			f.Close()
			t.Fatalf("WriteGettextJSONToJSON (indent): %v", err)
		}
		f.Close()
		var poBuf bytes.Buffer
		if err := WriteGettextJSONToPO(jFromJSON, &poBuf, false, false); err != nil {
			t.Fatalf("WriteGettextJSONToPO: %v", err)
		}
		if err := os.WriteFile(outputFromJSONPO, poBuf.Bytes(), 0644); err != nil {
			t.Fatalf("write output_from_json.po: %v", err)
		}
	}

	// Output JSON must match original JSON (parse and compare structure; formatting may differ)
	origJSON, _ := os.ReadFile(jsonPath)
	outFromPOJSON, _ := os.ReadFile(outputFromPOJSON)
	outFromJSONJSON, _ := os.ReadFile(outputFromJSONJSON)
	jOrigJSON, _ := ParseGettextJSONBytes(origJSON)
	jOutFromPOJSON, _ := ParseGettextJSONBytes(outFromPOJSON)
	jOutFromJSONJSON, _ := ParseGettextJSONBytes(outFromJSONJSON)
	if !gettextJSONEqualForTest(jOrigJSON, jOutFromPOJSON) {
		t.Errorf("output_from_po.json content != input.json")
	}
	if !gettextJSONEqualForTest(jOrigJSON, jOutFromJSONJSON) {
		t.Errorf("output_from_json.json content != input.json")
	}

	// Output PO must match original PO (compare via parsed structure)
	jOrigPO, _ := ReadFileToGettextJSON(poPath)
	jOutFromPOPO, _ := ReadFileToGettextJSON(outputFromPOPO)
	jOutFromJSONPO, _ := ReadFileToGettextJSON(outputFromJSONPO)
	if !gettextJSONEqualForTest(jOrigPO, jOutFromPOPO) {
		t.Errorf("output_from_po.po content != input.po")
	}
	if !gettextJSONEqualForTest(jOrigPO, jOutFromJSONPO) {
		t.Errorf("output_from_json.po content != input.po")
	}
}

func TestReadFileToGettextJSON(t *testing.T) {
	dir := t.TempDir()
	poPath := filepath.Join(dir, "x.po")
	poContent := `# header
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "Ciao"
`
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatal(err)
	}
	j, err := ReadFileToGettextJSON(poPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Entries) != 1 || j.Entries[0].MsgID != "Hello" || j.Entries[0].MsgStr != "Ciao" {
		t.Errorf("PO: got %v", j.Entries)
	}
	jsonPath := filepath.Join(dir, "x.json")
	jsonContent := `{"header_comment":"","header_meta":"","entries":[{"msgid":"Hi","msgstr":"Salut"}]}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}
	j2, err := ReadFileToGettextJSON(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(j2.Entries) != 1 || j2.Entries[0].MsgID != "Hi" || j2.Entries[0].MsgStr != "Salut" {
		t.Errorf("JSON: got %v", j2.Entries)
	}
	_, err = ReadFileToGettextJSON(filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestWriteGettextJSONToJSON(t *testing.T) {
	j := &GettextJSON{
		HeaderComment: "#",
		HeaderMeta:    "H\n",
		Entries:       []GettextEntry{{MsgID: "x", MsgStr: "y"}},
	}
	var buf bytes.Buffer
	if err := WriteGettextJSONToJSON(j, &buf); err != nil {
		t.Fatal(err)
	}
	var decoded GettextJSON
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.HeaderComment != "#" || len(decoded.Entries) != 1 || decoded.Entries[0].MsgID != "x" {
		t.Errorf("round-trip: got %+v", decoded)
	}
}

// TestFuzzySingleSource verifies fuzzy state lives only in GettextEntry.Fuzzy:
// PO with "#, fuzzy" or "#, fuzzy, c-format" -> JSON has Fuzzy=true and Comments without fuzzy line;
// when writing PO, fuzzy is restored (standalone or merged into flag line).
func TestFuzzySingleSource(t *testing.T) {
	// PO with standalone "#, fuzzy"
	poStandalone := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#, fuzzy
msgid "Fuzzy only"
msgstr ""
`
	entries, header, err := ParsePoEntries([]byte(poStandalone))
	if err != nil {
		t.Fatal(err)
	}
	_, headerMeta, _ := SplitHeader(header)
	j := GettextJSONFromEntries("", headerMeta, entries)
	if len(j.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(j.Entries))
	}
	e := &j.Entries[0]
	if !e.Fuzzy {
		t.Error("expected Fuzzy=true for #, fuzzy entry")
	}
	for _, c := range e.Comments {
		if strings.Contains(c, "fuzzy") {
			t.Errorf("fuzzy should not appear in Comments, got %q", c)
		}
	}
	var poBuf bytes.Buffer
	if err := WriteGettextJSONToPO(j, &poBuf, false, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(poBuf.String(), "#, fuzzy\n") {
		t.Error("expected #, fuzzy to be restored in PO output")
	}

	// PO with "#, fuzzy, c-format"
	poMerged := `msgid ""
msgstr ""

#, fuzzy, c-format
msgid "Fuzzy and c-format"
msgstr ""
`
	entries2, header2, err := ParsePoEntries([]byte(poMerged))
	if err != nil {
		t.Fatal(err)
	}
	_, headerMeta2, _ := SplitHeader(header2)
	j2 := GettextJSONFromEntries("", headerMeta2, entries2)
	if len(j2.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(j2.Entries))
	}
	e2 := &j2.Entries[0]
	if !e2.Fuzzy {
		t.Error("expected Fuzzy=true")
	}
	hasCFormat := false
	for _, c := range e2.Comments {
		if strings.Contains(c, "c-format") {
			hasCFormat = true
		}
		if strings.TrimSpace(c) == "#, fuzzy" {
			t.Error("standalone #, fuzzy should be removed from Comments")
		}
	}
	if !hasCFormat {
		t.Error("expected #, c-format to remain in Comments")
	}
	var poBuf2 bytes.Buffer
	if err := WriteGettextJSONToPO(j2, &poBuf2, false, false); err != nil {
		t.Fatal(err)
	}
	out := poBuf2.String()
	if !strings.Contains(out, "fuzzy") || !strings.Contains(out, "c-format") {
		t.Errorf("expected fuzzy and c-format restored in PO, got %q", out)
	}
}

func TestParseGettextJSONBytes_RepairMalformedLLMOutput(t *testing.T) {
	validJSON := `{"header_comment":"","header_meta":"","entries":[{"msgid":"Hello","msgstr":"你好","fuzzy":false}]}`

	t.Run("BOM prefix", func(t *testing.T) {
		bom := []byte{0xEF, 0xBB, 0xBF}
		data := append(bom, []byte(validJSON)...)
		j, err := ParseGettextJSONBytes(data)
		if err != nil {
			t.Fatalf("ParseGettextJSONBytes with BOM: %v", err)
		}
		if len(j.Entries) != 1 || j.Entries[0].MsgID != "Hello" || j.Entries[0].MsgStr != "你好" {
			t.Errorf("got %+v", j)
		}
	})

	t.Run("markdown code block", func(t *testing.T) {
		data := []byte("Here is the JSON:\n```json\n" + validJSON + "\n```\n")
		j, err := ParseGettextJSONBytes(data)
		if err != nil {
			t.Fatalf("ParseGettextJSONBytes with markdown: %v", err)
		}
		if len(j.Entries) != 1 || j.Entries[0].MsgID != "Hello" {
			t.Errorf("got %+v", j)
		}
	})

	t.Run("leading and trailing text", func(t *testing.T) {
		data := []byte("The result is: " + validJSON + " end of output")
		j, err := ParseGettextJSONBytes(data)
		if err != nil {
			t.Fatalf("ParseGettextJSONBytes with extra text: %v", err)
		}
		if len(j.Entries) != 1 || j.Entries[0].MsgID != "Hello" {
			t.Errorf("got %+v", j)
		}
	})
}

func TestFormatGettextJSONParseError(t *testing.T) {
	data := []byte(`{"invalid": json}`)
	path := "test.json"
	parseErr := fmt.Errorf("invalid character 'j' looking for beginning of value")
	err := FormatGettextJSONParseError(data, path, parseErr)
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "failed to parse gettext JSON file") {
		t.Errorf("expected 'failed to parse gettext JSON file' in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, path) {
		t.Errorf("expected path %q in error, got: %s", path, errStr)
	}
	if !strings.Contains(errStr, "Content snippet") {
		t.Errorf("expected 'Content snippet' in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Please fix the JSON") {
		t.Errorf("expected 'Please fix the JSON' in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, parseErr.Error()) {
		t.Errorf("expected parse error in output, got: %s", errStr)
	}
}
