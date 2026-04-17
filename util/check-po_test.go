package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/util/utiltest"
	"github.com/spf13/viper"
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
			errs, ok := checkPoMetaEscapeChars(po)
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

	ok := CheckPoFileWithPrompt("zh_CN", poPath, true, "[zh_CN.po]", "", true, "")
	if ok {
		t.Error("CheckPoFileWithPrompt expected to fail for meta with literal \\n, got ok")
	}
}

func TestCheckPoLocationCommentsNoLineNumbers(t *testing.T) {
	tests := []struct {
		name    string
		entries []GettextEntry
		wantErr bool
		wantMsg string
	}{
		{
			name: "no location comments",
			entries: []GettextEntry{
				{MsgID: "Hello", MsgStr: []string{"你好"}, Comments: []string{"#. extracted comment"}},
			},
			wantErr: false,
		},
		{
			name: "file-only location (no line number)",
			entries: []GettextEntry{
				{MsgID: "Hello", MsgStr: []string{"你好"}, Comments: []string{"#: path/to/file.c"}},
			},
			wantErr: false,
		},
		{
			name: "location with line number",
			entries: []GettextEntry{
				{MsgID: "Hello", MsgStr: []string{"你好"}, Comments: []string{"#: path/to/file.c:116"}},
			},
			wantErr: true,
			wantMsg: "file.c:116",
		},
		{
			name: "location with line and column",
			entries: []GettextEntry{
				{MsgID: "World", MsgStr: []string{"世界"}, Comments: []string{"#: foo.c:123,5"}},
			},
			wantErr: true,
			wantMsg: "foo.c:123,5",
		},
		{
			name: "multiple refs one has line number",
			entries: []GettextEntry{
				{MsgID: "X", MsgStr: []string{"X"}, Comments: []string{"#: a.c b.c:50"}},
			},
			wantErr: true,
			wantMsg: "b.c:50",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := &GettextPO{
				HeaderEntry: GettextEntry{MsgStr: []string{"Content-Type: text/plain; charset=UTF-8\n"}},
				Entries:     tt.entries,
			}
			errs, ok := checkPoLocationCommentsNoLineNumbers(po)
			if ok == tt.wantErr {
				t.Errorf("checkPoLocationCommentsNoLineNumbers() ok = %v, want %v", ok, !tt.wantErr)
			}
			if tt.wantErr && tt.wantMsg != "" {
				if len(errs) == 0 || !strings.Contains(errs[0], tt.wantMsg) {
					t.Errorf("checkPoLocationCommentsNoLineNumbers() errs = %v, want containing %q", errs, tt.wantMsg)
				}
			}
		})
	}
}

func TestCheckPoFileWithPrompt_LocationCommentsNoLineNumbers(t *testing.T) {
	tmpDir := t.TempDir()
	poPath := filepath.Join(tmpDir, "zh_CN.po")
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

#: path/to/file.c:116
msgid "Hello"
msgstr "你好"
`
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatalf("write temp po: %v", err)
	}

	viper.Set("check-po--report-file-locations", "error")
	defer viper.Set("check-po--report-file-locations", "")

	ok := CheckPoFileWithPrompt("zh_CN", poPath, true, "[zh_CN.po]", "", true, "")
	if ok {
		t.Error("CheckPoFileWithPrompt expected to fail for location comment with line number, got ok")
	}
}

func TestCheckPoCompatibility(t *testing.T) {
	menu := "Menu"
	tests := []struct {
		name       string
		minVersion string
		entries    []GettextEntry
		wantErr    bool
		wantMsg    string
	}{
		{
			name:       "no compatibility issues",
			minVersion: "0.16",
			entries:    []GettextEntry{{MsgID: "Hello", MsgStr: []string{"你好"}}},
			wantErr:    false,
		},
		{
			name:       "empty minVersion skips check",
			minVersion: "",
			entries:    []GettextEntry{{MsgID: "File", MsgStr: []string{"文件"}, MsgCtxt: &menu}},
			wantErr:    false,
		},
		{
			name:       "msgctxt not supported below 0.15",
			minVersion: "0.14",
			entries:    []GettextEntry{{MsgID: "File", MsgStr: []string{"文件"}, MsgCtxt: &menu}},
			wantErr:    true,
			wantMsg:    "msgctxt not supported by gettext below 0.15",
		},
		{
			name:       "msgctxt allowed with 0.15",
			minVersion: "0.15",
			entries:    []GettextEntry{{MsgID: "File", MsgStr: []string{"文件"}, MsgCtxt: &menu}},
			wantErr:    false,
		},
		{
			name:       "previous msgctxt (#|) not supported below 0.15",
			minVersion: "0.14",
			entries:    []GettextEntry{{MsgID: "Open", MsgStr: []string{"打开"}, Comments: []string{"#| msgctxt \"old context\"\n", "#| msgid \"Open\"\n"}}},
			wantErr:    true,
			wantMsg:    "previous msgctxt (#|) not supported by gettext below 0.15",
		},
		{
			name:       "#~ msgctxt not supported below 0.15",
			minVersion: "0.14",
			entries:    []GettextEntry{{MsgID: "File", MsgStr: []string{"文件"}, MsgCtxt: &menu, Obsolete: true}},
			wantErr:    true,
			wantMsg:    "#~ msgctxt (obsolete with context) not supported by gettext below 0.15",
		},
		{
			name:       "#~| msgid not supported below 0.16",
			minVersion: "0.15",
			entries:    []GettextEntry{{MsgID: "Old", MsgStr: []string{"旧"}, Obsolete: true, Comments: []string{"#~| msgid \"Previous\"\n"}}},
			wantErr:    true,
			wantMsg:    "#~| msgid (obsolete previous) not supported by gettext below 0.16",
		},
		{
			name:       "#~| msgctxt not supported below 0.16",
			minVersion: "0.15",
			entries:    []GettextEntry{{MsgID: "X", MsgStr: []string{"X"}, Obsolete: true, Comments: []string{"#~| msgctxt \"Menu\"\n"}}},
			wantErr:    true,
			wantMsg:    "#~| msgctxt (obsolete previous) not supported by gettext below 0.16",
		},
		{
			name:       "#~| msgid_plural not supported below 0.16",
			minVersion: "0.15",
			entries:    []GettextEntry{{MsgID: "doc", MsgIDPlural: "docs", MsgStr: []string{"文档", "文档"}, Obsolete: true, Comments: []string{"#~| msgid_plural \"files\"\n"}}},
			wantErr:    true,
			wantMsg:    "#~| msgid_plural (obsolete previous) not supported by gettext below 0.16",
		},
		{
			name:       "all features allowed with 0.16",
			minVersion: "0.16",
			entries: []GettextEntry{
				{MsgID: "File", MsgStr: []string{"文件"}, MsgCtxt: &menu},
				{MsgID: "Old", MsgStr: []string{"旧"}, Obsolete: true, Comments: []string{"#~| msgid \"Previous\"\n"}},
				{MsgID: "X", MsgStr: []string{"X"}, Obsolete: true, Comments: []string{"#~| msgctxt \"Menu\"\n"}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := &GettextPO{
				HeaderEntry: GettextEntry{MsgStr: []string{"Content-Type: text/plain; charset=UTF-8\n"}},
				Entries:     tt.entries,
			}
			errs, ok := checkPoCompatibility(po, tt.minVersion)
			if ok == tt.wantErr {
				t.Errorf("checkPoCompatibility(_, %q) ok = %v, want %v", tt.minVersion, ok, !tt.wantErr)
			}
			if tt.wantErr && tt.wantMsg != "" {
				if len(errs) == 0 || !strings.Contains(errs[0], tt.wantMsg) {
					t.Errorf("checkPoCompatibility() errs = %v, want containing %q", errs, tt.wantMsg)
				}
			}
		})
	}
}

func TestCheckPoFileWithPrompt_Compatibility(t *testing.T) {
	tmpDir := t.TempDir()
	poPath := filepath.Join(tmpDir, "zh_CN.po")
	// PO with msgctxt - should fail when project sets MinGettextVersion (e.g. Git 0.14).
	poContent := `msgid ""
msgstr ""
"Project-Id-Version: Git\n"
"Content-Type: text/plain; charset=UTF-8\n"

msgctxt "Menu"
msgid "File"
msgstr "文件"
`
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatalf("write temp po: %v", err)
	}

	viper.Set("check-po--report-file-locations", "none")
	defer viper.Set("check-po--report-file-locations", "")

	ok := CheckPoFileWithPrompt("zh_CN", poPath, true, "[zh_CN.po]", "", true, "")
	if ok {
		t.Error("CheckPoFileWithPrompt expected to fail for msgctxt (Git min 0.14 < 0.15), got ok")
	}
}

func TestCheckPoFileWithPrompt_CompatibilitySkippedWhenNoMinVersion(t *testing.T) {
	tmpDir := t.TempDir()
	poPath := filepath.Join(tmpDir, "zh_CN.po")
	// Project without MinGettextVersion: compatibility check is skipped, so msgctxt does not cause failure.
	poContent := `msgid ""
msgstr ""
"Project-Id-Version: OtherProj 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"

msgctxt "Menu"
msgid "File"
msgstr "文件"
`
	if err := os.WriteFile(poPath, []byte(poContent), 0644); err != nil {
		t.Fatalf("write temp po: %v", err)
	}

	viper.Set("check-po--report-file-locations", "none")
	defer viper.Set("check-po--report-file-locations", "")

	ok := CheckPoFileWithPrompt("zh_CN", poPath, true, "[zh_CN.po]", "", true, "")
	if !ok {
		t.Error("CheckPoFileWithPrompt expected ok when project has no MinGettextVersion (compatibility skipped), got !ok")
	}
}

func TestCmdCheckPo_pathsUnderGitCeiling(t *testing.T) {
	if _, err := exec.LookPath("msgfmt"); err != nil {
		t.Skip("msgfmt not in PATH")
	}

	cmdUpdateTestMu.Lock()
	defer cmdUpdateTestMu.Unlock()

	root := t.TempDir()
	utiltest.MaterializeCmdUpdateTree(t, root)
	utiltest.SetGitCeilingDirectories(t, root)

	potPath := filepath.Join(root, "po", "git.pot")
	defer func() {
		viper.Set("pot-file", "auto")
		viper.Set("check--report-typos", "")
		viper.Set("check-po--report-file-locations", "")
	}()
	viper.Set("pot-file", potPath)
	viper.Set("check--report-typos", "none")
	viper.Set("check-po--report-file-locations", "none")

	t.Run("from project root with po/zh_CN.po", func(t *testing.T) {
		utiltest.Chdir(t, root)
		if !CmdCheckPo("po/zh_CN.po") {
			t.Fatal("CmdCheckPo(po/zh_CN.po) failed from project root")
		}
	})

	t.Run("from project root with po directory", func(t *testing.T) {
		utiltest.Chdir(t, root)
		if !CmdCheckPo("po") {
			t.Fatal("CmdCheckPo(po) failed from project root")
		}
	})

	t.Run("from po with zh_CN.po", func(t *testing.T) {
		utiltest.Chdir(t, filepath.Join(root, "po"))
		if !CmdCheckPo("zh_CN.po") {
			t.Fatal("CmdCheckPo(zh_CN.po) failed from po/")
		}
	})

	t.Run("from po with current directory", func(t *testing.T) {
		utiltest.Chdir(t, filepath.Join(root, "po"))
		if !CmdCheckPo(".") {
			t.Fatal("CmdCheckPo(.) failed from po/")
		}
	})
}
