package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPoMetaNewlines(t *testing.T) {
	tests := []struct {
		name     string
		metaStr  string // header msgstr in PO format (escaped)
		wantErr  bool
		wantMsgs int
	}{
		{
			name:     "normal meta",
			metaStr:  "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\n",
			wantErr:  false,
			wantMsgs: 0,
		},
		{
			name:     "literal backslash-n in meta line",
			metaStr:  "Project-Id-Version: foo\\\\nbar\nContent-Type: text/plain\n",
			wantErr:  true,
			wantMsgs: 1,
		},
		{
			name:     "multiple lines with literal backslash-n",
			metaStr:  "Key1: value1\nKey2: value\\\\nbroken\nKey3: value3\n",
			wantErr:  true,
			wantMsgs: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := &GettextPO{
				HeaderEntry: GettextEntry{
					MsgStr: []string{tt.metaStr},
				},
				Entries: []GettextEntry{},
			}
			errs, ok := checkPoMetaNewlines(po)
			if ok == tt.wantErr {
				t.Errorf("checkPoMetaNewlines() ok = %v, want %v", ok, !tt.wantErr)
			}
			if tt.wantErr && len(errs) < tt.wantMsgs {
				t.Errorf("checkPoMetaNewlines() got %d errs, want at least %d", len(errs), tt.wantMsgs)
			}
			if !tt.wantErr && len(errs) != 0 {
				t.Errorf("checkPoMetaNewlines() got %d errs, want 0: %v", len(errs), errs)
			}
		})
	}
}

func TestCheckPoFileWithPrompt_MetaNewlines(t *testing.T) {
	// Create a temp PO file with literal \n in header meta.
	// In PO file: \\n -> decoded as backslash+n (abnormal).
	tmpDir := t.TempDir()
	poPath := filepath.Join(tmpDir, "zh_CN.po")
	poContent := "msgid \"\"\nmsgstr \"\"\n\"Project-Id-Version: foo\\\\nbar\\n\"\n\"Content-Type: text/plain; charset=UTF-8\\n\"\n\nmsgid \"Hello\"\nmsgstr \"你好\"\n"
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatalf("write temp po: %v", err)
	}

	ok := CheckPoFileWithPrompt("zh_CN", poPath, "[zh_CN.po]")
	if ok {
		t.Error("CheckPoFileWithPrompt expected to fail for meta with literal \\n, got ok")
	}
}
