package util

import (
	"errors"
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

func TestGetRepoRelPath(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
	init := exec.Command("git", "init")
	init.Dir = tmpDir
	init.Env = gitEnv
	if err := init.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "po"), 0755); err != nil {
		t.Fatal(err)
	}
	po := filepath.Join(tmpDir, "po", "zh_CN.po")
	if err := os.WriteFile(po, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	repository.OpenRepository(tmpDir)

	t.Run("repo-relative", func(t *testing.T) {
		rel, err := GetRepoRelPath("po/zh_CN.po")
		if err != nil || rel != "po/zh_CN.po" {
			t.Fatalf("GetRepoRelPath = %q, %v; want po/zh_CN.po, nil", rel, err)
		}
	})

	t.Run("absolute inside repo", func(t *testing.T) {
		rel, err := GetRepoRelPath(po)
		if err != nil || rel != "po/zh_CN.po" {
			t.Fatalf("GetRepoRelPath = %q, %v; want po/zh_CN.po, nil", rel, err)
		}
	})

	t.Run("outside worktree", func(t *testing.T) {
		safe := strings.ReplaceAll(t.Name(), "/", "_")
		outside := filepath.Join(tmpDir, "..", "outside-getreporelpath-"+safe+".po")
		if err := os.WriteFile(outside, []byte("y"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(outside) })
		_, err := GetRepoRelPath(outside)
		if !errors.Is(err, ErrOutsideWorktree) {
			t.Fatalf("expected ErrOutsideWorktree, got %v", err)
		}
	})

	t.Run("repo-relative escapes", func(t *testing.T) {
		safe := strings.ReplaceAll(t.Name(), "/", "_")
		escapePo := filepath.Join(filepath.Dir(tmpDir), "git-po-helper-getreporel-"+safe+".po")
		if err := os.WriteFile(escapePo, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(escapePo) })
		_, err := GetRepoRelPath(filepath.Join("..", filepath.Base(escapePo)))
		if !errors.Is(err, ErrOutsideWorktree) {
			t.Fatalf("expected ErrOutsideWorktree, got %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		_, err := GetRepoRelPath("  ")
		if err == nil || !strings.Contains(err.Error(), "empty") {
			t.Fatalf("expected empty path error, got %v", err)
		}
	})
}

func TestGetRepoRelPath_bareRepository(t *testing.T) {
	bareDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(bareDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	init := exec.Command("git", "init", "--bare")
	init.Dir = bareDir
	init.Env = gitTestEnv()
	if err := init.Run(); err != nil {
		t.Fatal(err)
	}
	repository.OpenRepository(bareDir)

	if strings.TrimSpace(repository.WorkDir()) != "" {
		t.Skip("goconfig reports a work tree for this bare repo; skipping bare GetRepoRelPath checks")
	}

	t.Run("non-absolute returns normalized repo path", func(t *testing.T) {
		rel, err := GetRepoRelPath("po/zh_CN.po")
		if err != nil || rel != "po/zh_CN.po" {
			t.Fatalf("GetRepoRelPath = %q, %v; want po/zh_CN.po, nil", rel, err)
		}
	})

	t.Run("absolute returns error", func(t *testing.T) {
		abs := filepath.Join(bareDir, "objects") // exists under bare .git layout
		_, err := GetRepoRelPath(abs)
		if err == nil || !strings.Contains(err.Error(), "bare repository") {
			t.Fatalf("expected bare repository error, got %v", err)
		}
	})
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
	runGit("commit", "--no-verify", "-m", "initial")

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
		runGit("commit", "--no-verify", "-m", "add pot")

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

func TestEnsureGitPathAtRevision(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
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
	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("po/zh_CN.po", []byte("msgid \"\"\nmsgstr \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "po/zh_CN.po")
	runGit("commit", "--no-verify", "-m", "initial")

	repository.OpenRepository(tmpDir)

	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LANG", "zh_CN.UTF-8")
	t.Setenv("LANGUAGE", "zh_CN")

	if err := ensureGitPathAtRevision("HEAD", "po/zh_CN.po"); err != nil {
		t.Fatalf("existing path: %v", err)
	}
	err = ensureGitPathAtRevision("HEAD", "po/pt_BR.po")
	if !errors.Is(err, ErrFileNotInRevision) {
		t.Fatalf("missing path: got %v, want ErrFileNotInRevision", err)
	}
	err = ensureGitPathAtRevision("no-such-ref", "po/zh_CN.po")
	if err == nil || errors.Is(err, ErrFileNotInRevision) {
		t.Fatalf("invalid revision: got %v, want non-ErrFileNotInRevision error", err)
	}
}

// TestFileRevisionGetFile_NotInRevision_LocalizedEnv verifies detection works
// even when the user environment uses a non-English locale for git messages.
func TestFileRevisionGetFile_NotInRevision_LocalizedEnv(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
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
	if err := os.WriteFile("README", []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README")
	runGit("commit", "--no-verify", "-m", "initial")

	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("po/pt_BR.po", []byte("msgid \"\"\nmsgstr \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	repository.OpenRepository(tmpDir)

	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LANG", "zh_CN.UTF-8")
	t.Setenv("LANGUAGE", "zh_CN")

	fr := FileRevision{Revision: "HEAD", File: "po/pt_BR.po"}
	defer fr.Cleanup()
	_, err = fr.GetFile()
	if !errors.Is(err, ErrFileNotInRevision) {
		t.Fatalf("GetFile = %v, want ErrFileNotInRevision (locale-independent)", err)
	}
}

// TestFileRevisionGetFile_NotInRevision verifies GetFile returns ErrFileNotInRevision
// when the path exists on disk but not at the requested commit (newly added file).
func TestFileRevisionGetFile_NotInRevision(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
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
	if err := os.WriteFile("README", []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README")
	runGit("commit", "--no-verify", "-m", "initial")

	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatal(err)
	}
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "Olá"
`
	if err := os.WriteFile("po/pt_BR.po", []byte(poContent), 0644); err != nil {
		t.Fatal(err)
	}

	repository.OpenRepository(tmpDir)

	fr := FileRevision{Revision: "HEAD", File: "po/pt_BR.po"}
	defer fr.Cleanup()
	_, err = fr.GetFile()
	if !errors.Is(err, ErrFileNotInRevision) {
		t.Fatalf("GetFile = %v, want ErrFileNotInRevision", err)
	}
}

// TestPrepareReviewData_NewFile treats a missing old revision path as empty base,
// so all entries in the new file appear as added.
func TestPrepareReviewData_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
	runGit := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = gitEnv
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
		}
		return strings.TrimSpace(string(out))
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")
	if err := os.WriteFile("README", []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README")
	runGit("commit", "--no-verify", "-m", "initial")
	base := runGit("rev-parse", "HEAD")

	if err := os.MkdirAll("po", 0755); err != nil {
		t.Fatal(err)
	}
	poContent := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Hello"
msgstr "Olá"

msgid "World"
msgstr "Mundo"
`
	if err := os.WriteFile("po/pt_BR.po", []byte(poContent), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "po/pt_BR.po")
	runGit("commit", "--no-verify", "-m", "l10n: add pt_BR")
	tip := runGit("rev-parse", "HEAD")

	repository.OpenRepository(tmpDir)

	outPath := filepath.Join(tmpDir, "review-out.json")
	if err := PrepareReviewData(base, "po/pt_BR.po", tip, "po/pt_BR.po", outPath, false, true, false); err != nil {
		t.Fatalf("PrepareReviewData: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty compare output for newly added PO")
	}
	j, err := LoadFileToGettextJSON(data, outPath)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(j.Entries) != 2 {
		t.Fatalf("expected 2 added entries, got %d", len(j.Entries))
	}
}
