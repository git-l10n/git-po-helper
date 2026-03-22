package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/git-l10n/git-po-helper/util/utiltest"
	"github.com/spf13/viper"
)

// Serializes CmdUpdate tests that tweak viper global state.
var cmdUpdateTestMu sync.Mutex

func TestCmdUpdate_explicitPoPaths(t *testing.T) {
	if _, err := exec.LookPath("msgmerge"); err != nil {
		t.Skip("msgmerge not in PATH")
	}
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
	}()
	viper.Set("pot-file", potPath)
	viper.Set("check--report-typos", "none")

	t.Run("from project root with po/zh_CN.po", func(t *testing.T) {
		utiltest.Chdir(t, root)
		if !CmdUpdate("po/zh_CN.po") {
			t.Fatal("CmdUpdate(po/zh_CN.po) failed from project root")
		}
		assertZHpoStillTranslated(t, filepath.Join(root, "po", "zh_CN.po"))
	})

	t.Run("from po with zh_CN.po", func(t *testing.T) {
		utiltest.Chdir(t, filepath.Join(root, "po"))
		if !CmdUpdate("zh_CN.po") {
			t.Fatal("CmdUpdate(zh_CN.po) failed from po/")
		}
		assertZHpoStillTranslated(t, filepath.Join(root, "po", "zh_CN.po"))
	})

	t.Run("from po with ./zh_CN.po", func(t *testing.T) {
		utiltest.Chdir(t, filepath.Join(root, "po"))
		if !CmdUpdate("./zh_CN.po") {
			t.Fatal("CmdUpdate(./zh_CN.po) failed from po/")
		}
		assertZHpoStillTranslated(t, filepath.Join(root, "po", "zh_CN.po"))
	})
}

func assertZHpoStillTranslated(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.Contains(s, `msgid "hello world"`) {
		t.Errorf("expected msgid hello world in %s", path)
	}
	if !strings.Contains(s, `msgstr "你好世界"`) {
		t.Errorf("expected Chinese translation preserved in %s", path)
	}
	if !strings.Contains(s, "Project-Id-Version: Git") {
		t.Errorf("expected Git Project-Id-Version in %s", path)
	}
}
