package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/repository"
)

// gitTestEnv returns an environment that isolates git from global config and from
// GIT_DIR/GIT_WORK_TREE (set when tests run under pre-commit hook).
func gitTestEnv() []string {
	env := os.Environ()
	filtered := make([]string, 0, len(env)+2)
	for _, e := range env {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") ||
			strings.HasPrefix(e, "GIT_INDEX_FILE=") || strings.HasPrefix(e, "GIT_COMMON_DIR=") {
			continue
		}
		filtered = append(filtered, e)
	}
	return append(filtered, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
}

// TestGetChangedPoFiles tests GetChangedPoFiles with a temporary git repository.
// It creates a repo with po files, makes commits, and verifies the changed files list.
// Uses GIT_CONFIG_GLOBAL/SYSTEM=/dev/null to avoid global config (e.g. hooks) affecting the test.
func TestGetChangedPoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Unset GIT_DIR/GIT_WORK_TREE so production git calls use tmpDir (pre-commit sets these).
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")
	// Chdir to tmpDir so repository and git operations use the sandbox; restore on exit.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	// Isolate from global git config and parent git context (GIT_DIR, GIT_WORK_TREE
	// set by pre-commit hook); otherwise git uses the project repo instead of tmpDir.
	gitEnv := gitTestEnv()

	// Initialize git repository
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = gitEnv
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

	// Create po directory and initial files
	poDir := filepath.Join(tmpDir, "po")
	if err := os.MkdirAll(poDir, 0755); err != nil {
		t.Fatalf("failed to create po dir: %v", err)
	}

	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "你好"
`

	for _, f := range []string{"zh_CN.po", "zh_TW.po"} {
		if err := os.WriteFile(filepath.Join(poDir, f), []byte(poContent), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	runGit("add", "po/")
	runGit("commit", "-m", "initial")

	// Modify only zh_CN.po
	modifiedContent := poContent + "\nmsgid \"World\"\nmsgstr \"世界\"\n"
	if err := os.WriteFile(filepath.Join(poDir, "zh_CN.po"), []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify zh_CN.po: %v", err)
	}

	// Open repository for testing (must be done before GetChangedPoFiles)
	repository.OpenRepository(tmpDir)

	t.Run("default mode (HEAD vs working tree)", func(t *testing.T) {
		files, err := GetChangedPoFiles("", "")
		if err != nil {
			t.Fatalf("GetChangedPoFiles failed: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("expected 1 changed file, got %d: %v", len(files), files)
		}
		if len(files) > 0 && files[0] != "po/zh_CN.po" {
			t.Errorf("expected po/zh_CN.po, got %s", files[0])
		}
	})

	t.Run("excludes .pot files", func(t *testing.T) {
		// Add git.pot and modify it
		potContent := `# Copyright (C) 2024
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "test"
msgstr ""
`
		if err := os.WriteFile(filepath.Join(poDir, "git.pot"), []byte(potContent), 0644); err != nil {
			t.Fatalf("failed to write git.pot: %v", err)
		}
		runGit("add", "po/git.pot")
		runGit("commit", "-m", "add pot")

		// Modify pot file
		if err := os.WriteFile(filepath.Join(poDir, "git.pot"), []byte(potContent+"\nmsgid \"extra\"\nmsgstr \"\"\n"), 0644); err != nil {
			t.Fatalf("failed to modify git.pot: %v", err)
		}

		files, err := GetChangedPoFiles("", "")
		if err != nil {
			t.Fatalf("GetChangedPoFiles failed: %v", err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, ".pot") {
				t.Errorf("GetChangedPoFiles should not return .pot files, got %s", f)
			}
		}
	})
}
