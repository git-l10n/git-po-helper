package util

import (
	"strings"
	"testing"
)

func mustParsePOJSON(t *testing.T, content string) *GettextJSON {
	t.Helper()
	j, err := LoadFileToGettextJSON([]byte(content), "test.po")
	if err != nil {
		t.Fatalf("LoadFileToGettextJSON: %v", err)
	}
	return j
}

func potCheckPOHeader(extra string) string {
	return `msgid ""
msgstr ""
"Project-Id-Version: Git\n"
"Content-Type: text/plain; charset=UTF-8\n"

` + extra
}

func TestCheckPoEntryOrder(t *testing.T) {
	tests := []struct {
		name    string
		pot     string
		po      string
		wantOK  bool
		wantSub string // substring expected in errors when !wantOK
	}{
		{
			name: "same order",
			pot: potCheckPOHeader(`msgid "Hello"
msgstr ""

msgid "World"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`),
			wantOK: true,
		},
		{
			name: "reversed common entries",
			pot: potCheckPOHeader(`msgid "Hello"
msgstr ""

msgid "World"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "World"
msgstr "世界"

msgid "Hello"
msgstr "你好"
`),
			wantOK:  false,
			wantSub: "out of order",
		},
		{
			name: "POT-only entries ignored",
			pot: potCheckPOHeader(`msgid "Hello"
msgstr ""

msgid "OnlyInPOT"
msgstr ""

msgid "World"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"
`),
			wantOK: true,
		},
		{
			name: "PO-only entries ignored",
			pot: potCheckPOHeader(`msgid "Hello"
msgstr ""

msgid "World"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "Hello"
msgstr "你好"

msgid "OnlyInPO"
msgstr "仅 PO"

msgid "World"
msgstr "世界"
`),
			wantOK: true,
		},
		{
			name: "obsolete entries ignored for order",
			pot: potCheckPOHeader(`msgid "Hello"
msgstr ""

msgid "World"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "Hello"
msgstr "你好"

#~ msgid "Obsolete"
#~ msgstr "过时"

msgid "World"
msgstr "世界"
`),
			wantOK: true,
		},
		{
			name: "msgctxt distinguishes entries",
			pot: potCheckPOHeader(`msgctxt "ctx1"
msgid "File"
msgstr ""

msgctxt "ctx2"
msgid "File"
msgstr ""
`),
			po: potCheckPOHeader(`msgctxt "ctx2"
msgid "File"
msgstr "文件2"

msgctxt "ctx1"
msgid "File"
msgstr "文件1"
`),
			wantOK:  false,
			wantSub: "out of order",
		},
		{
			name: "msgctxt same order",
			pot: potCheckPOHeader(`msgctxt "ctx1"
msgid "File"
msgstr ""

msgctxt "ctx2"
msgid "File"
msgstr ""
`),
			po: potCheckPOHeader(`msgctxt "ctx1"
msgid "File"
msgstr "文件1"

msgctxt "ctx2"
msgid "File"
msgstr "文件2"
`),
			wantOK: true,
		},
		{
			name: "middle pair swapped among longer list",
			pot: potCheckPOHeader(`msgid "A"
msgstr ""

msgid "B"
msgstr ""

msgid "C"
msgstr ""

msgid "D"
msgstr ""
`),
			po: potCheckPOHeader(`msgid "A"
msgstr "a"

msgid "C"
msgstr "c"

msgid "B"
msgstr "b"

msgid "D"
msgstr "d"
`),
			wantOK:  false,
			wantSub: "out of order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			potJ := mustParsePOJSON(t, tt.pot)
			poJ := mustParsePOJSON(t, tt.po)
			msgs, ok := checkPoEntryOrder(potJ, poJ)
			if ok != tt.wantOK {
				t.Errorf("checkPoEntryOrder() ok = %v, want %v; msgs=%v", ok, tt.wantOK, msgs)
			}
			if !tt.wantOK {
				joined := strings.Join(msgs, "\n")
				if !strings.Contains(joined, tt.wantSub) {
					t.Errorf("checkPoEntryOrder() msgs = %q, want substring %q", joined, tt.wantSub)
				}
			} else if len(msgs) != 0 {
				t.Errorf("checkPoEntryOrder() got msgs %v, want none", msgs)
			}
		})
	}
}
