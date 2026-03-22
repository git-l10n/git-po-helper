package utiltest

import (
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

//go:embed testdata/cmdupdate
var cmdUpdateFixture embed.FS

const cmdUpdateFixtureRoot = "testdata/cmdupdate"

// MaterializeCmdUpdateTree copies the embedded Makefile + po/zh_CN.po + messages.pot
// into dstRoot, then runs `make pot` so po/git.pot exists. Requires make(1) and cp.
func MaterializeCmdUpdateTree(t *testing.T, dstRoot string) {
	t.Helper()
	if err := os.MkdirAll(dstRoot, 0755); err != nil {
		t.Fatalf("mkdir temp root: %v", err)
	}
	if err := copyEmbedDir(cmdUpdateFixture, cmdUpdateFixtureRoot, dstRoot); err != nil {
		t.Fatalf("copy cmdupdate fixture: %v", err)
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not in PATH; skip CmdUpdate fixture build")
	}
	cmd := exec.Command("make", "pot")
	cmd.Dir = dstRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make pot: %v\n%s", err, out)
	}
	pot := filepath.Join(dstRoot, "po", "git.pot")
	if st, err := os.Stat(pot); err != nil || st.IsDir() {
		t.Fatalf("expected file %s after make pot", pot)
	}
}

func copyEmbedDir(src fs.FS, root, dstRoot string) error {
	return fs.WalkDir(src, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		out := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0755)
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0644)
	})
}
