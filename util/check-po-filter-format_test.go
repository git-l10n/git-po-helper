package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/repository"
	"github.com/spf13/viper"
)

func TestCheckPoFilterFormat_repoAttrPathWithTempContent(t *testing.T) {
	if _, err := exec.LookPath("msgcat"); err != nil {
		t.Skip("msgcat not in PATH")
	}
	tmpDir := t.TempDir()
	gitEnv := gitTestEnv()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = gitEnv
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

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

	runGit("init")
	runGit("config", "user.email", "t@t.com")
	runGit("config", "user.name", "T")

	poDir := filepath.Join(tmpDir, "po")
	if err := os.MkdirAll(poDir, 0755); err != nil {
		t.Fatal(err)
	}
	poBody := `msgid ""
msgstr ""
"Project-Id-Version: Git\n"
"Content-Type: text/plain; charset=UTF-8\n"

#: main.c:42
msgid "Hello"
msgstr "Hi"
`
	repoPo := filepath.Join(poDir, "test.po")
	if err := os.WriteFile(repoPo, []byte(poBody), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitattributes"), []byte("po/*.po filter=gettext-no-location\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "po/test.po", ".gitattributes")
	runGit("commit", "--no-verify", "-m", "init")

	outside, err := os.CreateTemp("", "git-po-helper-filter-*.po")
	if err != nil {
		t.Fatal(err)
	}
	outsidePath := outside.Name()
	_ = outside.Close()
	defer os.Remove(outsidePath)
	if err := os.WriteFile(outsidePath, []byte(poBody), 0644); err != nil {
		t.Fatal(err)
	}

	repository.OpenRepository(tmpDir)
	viper.Set("check--report-file-locations", "error")
	defer viper.Set("check--report-file-locations", "")

	errs, ok := checkPoFilterFormat(outsidePath, "po/test.po")
	if ok || len(errs) == 0 {
		t.Fatalf("expected filter format failure, ok=%v errs=%v", ok, errs)
	}
	joined := strings.Join(errs, "\n")
	if !strings.Contains(joined, "does not match expected filter output") ||
		!strings.Contains(joined, "msgcat --no-location -") {
		t.Fatalf("unexpected messages: %s", joined)
	}

	viper.Set("check-po--no-check-filter", true)
	defer viper.Set("check-po--no-check-filter", false)
	errs, ok = checkPoFilterFormat(outsidePath, "po/test.po")
	if !ok || len(errs) > 0 {
		t.Fatalf("with --no-check-filter expected ok, got ok=%v errs=%v", ok, errs)
	}
}

func TestCheckPoFilterFormat_outsideRepoWithoutAttrPathSkips(t *testing.T) {
	tmpDir := t.TempDir()
	outside := filepath.Join(tmpDir, "orphan.po")
	if err := os.WriteFile(outside, []byte(`msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"
`), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
	repoRoot := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		repository.OpenRepository(origWd)
	}()

	gitEnv := gitTestEnv()
	init := exec.Command("git", "init")
	init.Dir = repoRoot
	init.Env = gitEnv
	if err := init.Run(); err != nil {
		t.Fatal(err)
	}
	repository.OpenRepository(repoRoot)
	viper.Set("check--report-file-locations", "error")
	defer viper.Set("check--report-file-locations", "")

	errs, ok := checkPoFilterFormat(outside, "")
	if !ok || len(errs) > 0 {
		t.Fatalf("expected skip (ok=true, no errs), ok=%v errs=%v", ok, errs)
	}
}

func TestCheckPoFilterFormat_noCheckFilterSkipsBeforeExist(t *testing.T) {
	viper.Set("check--no-check-filter", true)
	defer viper.Set("check--no-check-filter", false)
	errs, ok := checkPoFilterFormat("/nonexistent/path/does-not-exist.po", "")
	if !ok || len(errs) > 0 {
		t.Fatalf("NoCheckFilter should skip entire check, ok=%v errs=%v", ok, errs)
	}
}

func TestCheckPoFilterFormat_invalidRepoAttrPath(t *testing.T) {
	tmpDir := t.TempDir()
	poPath := filepath.Join(tmpDir, "x.po")
	if err := os.WriteFile(poPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, _ := os.Getwd()
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
	repository.OpenRepository(tmpDir)
	viper.Set("check--report-file-locations", "error")
	defer viper.Set("check--report-file-locations", "")

	_, ok := checkPoFilterFormat(poPath, "../outside.po")
	if ok {
		t.Fatal("expected failure for attr path escaping repo")
	}
}
