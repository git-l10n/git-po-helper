package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/repository"
)

func requireMsgcmp(t *testing.T) {
	if _, err := exec.LookPath("msgcmp"); err != nil {
		t.Skip("msgcmp not installed, skipping compare --stat tests")
	}
}

// chdirRepoForCompare sets cwd to tmpDir and opens that repo; defer restore
// restores cwd and re-opens the original dir so other tests are not affected.
// Unsets GIT_DIR/GIT_WORK_TREE so production git calls use tmpDir (pre-commit
// sets these and they override cmd.Dir/cwd). All git commits for compare tests
// must run with cmd.Dir = tmpDir only; Chdir must succeed or Execute would run
// against the wrong repository.
func chdirRepoForCompare(t *testing.T, tmpDir string) (restore func()) {
	t.Helper()
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_COMMON_DIR")
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir %s: %v", tmpDir, err)
	}
	repository.OpenRepository(tmpDir)
	return func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Chdir restore %s: %v", origWd, err)
		}
		repository.OpenRepository(origWd)
	}
}

// gitTestEnv isolates git from global config and from GIT_DIR/GIT_WORK_TREE
// (set when tests run under pre-commit hook); otherwise git uses the project repo.
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

// setupCompareRepo creates a temp git repo with po files for compare tests.
func setupCompareRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = gitTestEnv()
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

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

	// Second commit for HEAD~..HEAD tests
	modifiedContent := poContent + "\nmsgid \"World\"\nmsgstr \"世界\"\n"
	if err := os.WriteFile(filepath.Join(poDir, "zh_CN.po"), []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify zh_CN.po: %v", err)
	}
	runGit("add", "po/zh_CN.po")
	runGit("commit", "--no-verify", "-m", "add World")

	// Modify zh_CN.po for HEAD vs worktree tests
	modifiedContent = modifiedContent + "\nmsgid \"Foo\"\nmsgstr \"富\"\n"
	if err := os.WriteFile(filepath.Join(poDir, "zh_CN.po"), []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to modify zh_CN.po: %v", err)
	}

	return tmpDir
}

// TestCompareCommand_Quick runs fast validation tests with a single repo setup.
func TestCompareCommand_Quick(t *testing.T) {
	tmpDir := setupCompareRepo(t)
	restore := chdirRepoForCompare(t, tmpDir)
	defer restore()

	t.Run("too many args", func(t *testing.T) {
		c := compareCommand{}
		err := c.Execute([]string{"po/zh_CN.po", "po/zh_TW.po", "po/extra.po"})
		if err == nil {
			t.Fatal("expected error for too many args")
		}
		if !strings.Contains(err.Error(), "too many arguments") {
			t.Errorf("expected 'too many arguments' in error, got: %v", err)
		}
	})

	t.Run("mutual exclusivity", func(t *testing.T) {
		c := compareCommand{}
		c.O.Range = "HEAD"
		c.O.Commit = "abc123"
		err := c.Execute([]string{})
		if err == nil {
			t.Fatal("expected error for --range and --commit both set")
		}
		if !strings.Contains(err.Error(), "only one of") {
			t.Errorf("expected 'only one of' in error, got: %v", err)
		}
	})

	t.Run("revision with two files error", func(t *testing.T) {
		c := compareCommand{}
		c.O.Stat = true
		c.O.Range = "HEAD~..HEAD"
		err := c.Execute([]string{"po/zh_CN.po", "po/zh_TW.po"})
		if err == nil {
			t.Fatal("expected error when specifying revision with two files")
		}
		if !strings.Contains(err.Error(), "cannot specify revision") {
			t.Errorf("expected 'cannot specify revision' in error, got: %v", err)
		}
	})

	t.Run("default mode with explicit file", func(t *testing.T) {
		c := compareCommand{}
		c.O.Stat = false
		c.O.Range = ""
		c.O.Commit = ""
		c.O.Since = ""
		err := c.Execute([]string{"po/zh_CN.po"})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	})

	t.Run("default mode no args", func(t *testing.T) {
		c := compareCommand{}
		c.O.Stat = false
		c.O.Range = ""
		c.O.Commit = ""
		c.O.Since = ""
		err := c.Execute([]string{})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	})

	t.Run("compare two files with stat", func(t *testing.T) {
		c := compareCommand{}
		c.O.Stat = true
		c.O.Range = ""
		c.O.Commit = ""
		c.O.Since = ""
		err := c.Execute([]string{"po/zh_CN.po", "po/zh_TW.po"})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	})

	t.Run("compare one file with stat", func(t *testing.T) {
		c := compareCommand{}
		c.O.Stat = true
		c.O.Range = "HEAD~..HEAD"
		c.O.Commit = ""
		c.O.Since = ""
		err := c.Execute([]string{"po/zh_CN.po"})
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	})
}

// setupCompareRepoForStatAndNewEntriesOutput creates a repo with multiple commits and working tree changes.
// Commit 1: zh_CN (Hello), zh_TW (Hello)
// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
func setupCompareRepoForStatAndNewEntriesOutput(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	poHeader := `msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

`
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = gitTestEnv()
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

	poDir := filepath.Join(tmpDir, "po")
	if err := os.MkdirAll(poDir, 0755); err != nil {
		t.Fatalf("failed to create po dir: %v", err)
	}

	// Commit 1: both files with Hello
	v1 := poHeader + `msgid "Hello"
msgstr "你好"
`
	for _, f := range []string{"zh_CN.po", "zh_TW.po"} {
		if err := os.WriteFile(filepath.Join(poDir, f), []byte(v1), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}
	runGit("add", "po/")
	runGit("commit", "--no-verify", "-m", "v1")

	// Commit 2: zh_CN adds World, Foo; zh_TW unchanged
	v2 := poHeader + `msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界"

msgid "Foo"
msgstr "酒吧"
`
	if err := os.WriteFile(filepath.Join(poDir, "zh_CN.po"), []byte(v2), 0644); err != nil {
		t.Fatalf("failed to write zh_CN.po v2: %v", err)
	}
	runGit("add", "po/zh_CN.po")
	runGit("commit", "--no-verify", "-m", "v2")

	// Worktree: zh_CN has Foo removed, Bar added, World msgstr modified
	v3 := poHeader + `msgid "Hello"
msgstr "你好"

msgid "World"
msgstr "世界（已修改）"

msgid "Bar"
msgstr "新条"
`
	if err := os.WriteFile(filepath.Join(poDir, "zh_CN.po"), []byte(v3), 0644); err != nil {
		t.Fatalf("failed to write zh_CN.po worktree: %v", err)
	}

	return tmpDir
}

// runCompareWithOptions runs compare with given options and returns captured stdout.
func runCompareWithOptions(t *testing.T, tmpDir string, stat bool, rangeArg, commitArg, sinceArg string, args []string) string {
	t.Helper()
	restore := chdirRepoForCompare(t, tmpDir)
	defer restore()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	c := compareCommand{}
	c.O.Stat = stat
	c.O.Range = rangeArg
	c.O.Commit = commitArg
	c.O.Since = sinceArg
	err := c.Execute(args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func runCompareStatWithOptions(t *testing.T, tmpDir string, rangeArg, commitArg, sinceArg string, args []string) string {
	return runCompareWithOptions(t, tmpDir, true, rangeArg, commitArg, sinceArg, args)
}

func runCompareNewEntriesWithOptions(t *testing.T, tmpDir string, rangeArg, commitArg, sinceArg string, args []string) string {
	return runCompareWithOptions(t, tmpDir, false, rangeArg, commitArg, sinceArg, args)
}

var (
	reStatNew     = regexp.MustCompile(`(\d+)\s+new`)
	reStatRemoved = regexp.MustCompile(`(\d+)\s+removed`)
	reMsgid       = regexp.MustCompile(`msgid "([^"]*)"`)
)

func parseStatOutput(output string) (newCount, removedCount int) {
	if m := reStatNew.FindStringSubmatch(output); m != nil {
		newCount, _ = strconv.Atoi(m[1])
	}
	if m := reStatRemoved.FindStringSubmatch(output); m != nil {
		removedCount, _ = strconv.Atoi(m[1])
	}
	return newCount, removedCount
}

// parseNewEntriesOutput extracts msgids from PO output (excluding empty header msgid).
func parseNewEntriesOutput(output string) []string {
	matches := reMsgid.FindAllStringSubmatch(output, -1)
	var ids []string
	for _, m := range matches {
		if len(m) > 1 && m[1] != "" {
			ids = append(ids, m[1])
		}
	}
	return ids
}

func idsContains(ids []string, s string) bool {
	for _, id := range ids {
		if id == s {
			return true
		}
	}
	return false
}

func TestCompareCommand_StatAndNewEntriesOutput(t *testing.T) {
	tmpDir := setupCompareRepoForStatAndNewEntriesOutput(t)

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("HEAD~..HEAD one file: 2 new", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "HEAD~..HEAD", "", "", []string{"po/zh_CN.po"})
			if output == "" {
				t.Errorf("output should not be empty")
			}
			newCount, removedCount := parseStatOutput(output)
			if newCount != 2 {
				t.Errorf("expected 2 new (World, Foo), got %d. Output: %s", newCount, output)
			}
			if removedCount != 0 {
				t.Errorf("expected 0 removed, got %d. Output: %s", removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "HEAD~..HEAD", "", "", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Foo") {
				t.Errorf("expected World and Foo (new in commit 2), got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("--commit HEAD one file: same as HEAD~..HEAD", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "HEAD", "", []string{"po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			if newCount != 2 || removedCount != 0 {
				t.Errorf("expected 2 new 0 removed, got %d new %d removed. Output: %s", newCount, removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "HEAD", "", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Foo") {
				t.Errorf("expected World and Foo, got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("HEAD.. one file: worktree vs HEAD", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "HEAD..", "", "", []string{"po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			if newCount != 1 {
				t.Errorf("expected 1 new (Bar), got %d. Output: %s", newCount, output)
			}
			if removedCount != 1 {
				t.Errorf("expected 1 removed (Foo), got %d. Output: %s", removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "HEAD..", "", "", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") {
				t.Errorf("output should contain modified entry 'World', got: %v", ids)
			}
			if !idsContains(ids, "Bar") {
				t.Errorf("output should contain new entry 'Bar', got: %v", ids)
			}
			if idsContains(ids, "Foo") {
				t.Errorf("output should not contain deleted entry 'Foo', got: %v", ids)
			}
			if !strings.Contains(output, "msgstr \"世界（已修改）\"") {
				t.Errorf("output should contain modified msgstr for World, got: %s", output)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("--since HEAD one file: same as HEAD..", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "HEAD", []string{"po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			if newCount != 1 || removedCount != 1 {
				t.Errorf("expected 1 new 1 removed, got %d new %d removed. Output: %s", newCount, removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "HEAD", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Bar") || idsContains(ids, "Foo") {
				t.Errorf("expected World and Bar, not Foo; got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	// --since HEAD~: compare HEAD~ vs worktree; zh_CN had only Hello, worktree has Hello+World+Bar
	t.Run("--since HEAD~ one file: HEAD~ vs worktree", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "HEAD~", []string{"po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			if newCount != 2 || removedCount != 0 {
				t.Errorf("expected 2 new 0 removed, got %d new %d removed. Output: %s", newCount, removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "HEAD~", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Bar") || idsContains(ids, "Foo") {
				t.Errorf("expected World and Bar, not Foo; got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("empty range empty args: auto-select zh_CN", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "", []string{})
			if output == "" {
				t.Errorf("output should not be empty")
			}
			newCount, removedCount := parseStatOutput(output)
			if newCount != 1 || removedCount != 1 {
				t.Errorf("expected 1 new 1 removed (auto-selected zh_CN), got %d new %d removed. Output: %s", newCount, removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "", []string{})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Bar") || idsContains(ids, "Foo") {
				t.Errorf("expected World and Bar, not Foo; got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("empty range one file: default HEAD vs worktree", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "", []string{"po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			if newCount != 1 || removedCount != 1 {
				t.Errorf("expected 1 new 1 removed, got %d new %d removed. Output: %s", newCount, removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "", []string{"po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			if !idsContains(ids, "World") || !idsContains(ids, "Bar") || idsContains(ids, "Foo") {
				t.Errorf("expected World and Bar, not Foo; got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("empty range two files: compare worktree zh_CN vs zh_TW", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "", []string{"po/zh_CN.po", "po/zh_TW.po"})
			if output == "" {
				t.Errorf("output should not be empty")
			}
			newCount, removedCount := parseStatOutput(output)
			if newCount != 0 {
				t.Errorf("zh_TW has no extra msgids vs zh_CN, expected 0 new, got %d. Output: %s", newCount, output)
			}
			if removedCount != 2 {
				t.Errorf("zh_CN has World, Bar not in zh_TW, expected 2 removed, got %d. Output: %s", removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "", []string{"po/zh_CN.po", "po/zh_TW.po"})
			ids := parseNewEntriesOutput(output)
			// zh_TW has only Hello; zh_CN has Hello, World, Bar. New entries in zh_TW vs zh_CN: none (zh_TW is subset)
			if len(ids) != 0 {
				t.Errorf("zh_TW is subset of zh_CN, expected 0 new/changed entries, got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("HEAD~..HEAD zh_TW: no change (same in both commits)", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			// When nothing changed, compare --stat returns error; output may be empty
			output := runCompareStatWithOptions(t, tmpDir, "HEAD~..HEAD", "", "", []string{"po/zh_TW.po"})
			if output != "" {
				t.Errorf("zh_TW unchanged, expected empty output, got: %s", output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "HEAD~..HEAD", "", "", []string{"po/zh_TW.po"})
			ids := parseNewEntriesOutput(output)
			if len(ids) != 0 {
				t.Errorf("zh_TW unchanged between commits, expected empty output, got: %v", ids)
			}
		})
	})

	// Commit 1: zh_CN (Hello), zh_TW (Hello)
	// Commit 2: zh_CN (Hello, World, Foo), zh_TW (Hello)
	// Worktree: zh_CN (Hello, World modified, Bar) - Foo removed, Bar added; zh_TW (Hello)
	t.Run("empty range two files: zh_TW vs zh_CN (reverse order)", func(t *testing.T) {
		t.Run("stat", func(t *testing.T) {
			requireMsgcmp(t)
			output := runCompareStatWithOptions(t, tmpDir, "", "", "", []string{"po/zh_TW.po", "po/zh_CN.po"})
			newCount, removedCount := parseStatOutput(output)
			// zh_TW (src) has Hello; zh_CN (dest) has Hello, World, Bar. So 2 new in dest, 0 removed
			if newCount != 2 {
				t.Errorf("expected 2 new (World, Bar in zh_CN), got %d. Output: %s", newCount, output)
			}
			if removedCount != 0 {
				t.Errorf("expected 0 removed, got %d. Output: %s", removedCount, output)
			}
		})
		t.Run("newentries", func(t *testing.T) {
			output := runCompareNewEntriesWithOptions(t, tmpDir, "", "", "", []string{"po/zh_TW.po", "po/zh_CN.po"})
			ids := parseNewEntriesOutput(output)
			// orig=zh_TW (Hello), new=zh_CN (Hello, World, Bar). New/changed in zh_CN: World, Bar
			if !idsContains(ids, "World") || !idsContains(ids, "Bar") {
				t.Errorf("expected World and Bar (new in zh_CN vs zh_TW), got: %v", ids)
			}
		})
	})
}
