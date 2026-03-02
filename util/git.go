// Package util provides git-related utilities for po file operations.
package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// FileRevision is used as an argument for diff function
type FileRevision struct {
	Revision string
	File     string
	Tmpfile  string
}

// GetChangedPoFiles returns the list of changed po/XX.po files between two git versions.
// For commit mode (commit != ""): uses git diff-tree -r --name-only <baseCommit> <commit> -- po/
// For since/default mode: uses git diff -r --name-only <baseCommit> -- po/
// Returns only .po files (not .pot) under po/ directory.
func GetChangedPoFiles(commit, since string) ([]string, error) {
	var rev1, rev2 string

	if commit != "" {
		rev1 = commit + "~"
		rev2 = commit
	} else if since != "" {
		rev1 = since
		rev2 = ""
	} else {
		rev1 = "HEAD"
		rev2 = "" // working tree
	}
	return GetChangedPoFilesRange(rev1, rev2)
}

func GetChangedPoFilesRange(rev1, rev2 string) ([]string, error) {
	if err := repository.RequireOpened(); err != nil {
		return nil, fmt.Errorf("git operation requires a repository: %w", err)
	}

	var (
		cmd     *exec.Cmd
		workDir = repository.WorkDir()
	)

	if rev1 != "" && rev2 != "" {
		cmd = exec.Command("git", "diff-tree", "-r", "--name-only", rev1, rev2, "--", PoDir)
		log.Debugf("getting changed po files: git diff-tree -r --name-only %s %s -- %s", rev1, rev2, PoDir)
	} else if rev1 != "" && rev2 == "" {
		// Since mode: compare since commit with working tree
		cmd = exec.Command("git", "diff", "-r", "--name-only", rev1, "--", PoDir)
		log.Debugf("getting changed po files: git diff -r --name-only %s -- %s", rev1, PoDir)
	} else {
		// Default mode: compare HEAD with working tree
		return nil, fmt.Errorf("rev1 is nil for GetChangedPoFilesRange")
	}

	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed po files: %w", err)
	}

	// Filter to only .po files (not .pot)
	var poFiles []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".po") {
			poFiles = append(poFiles, line)
		}
	}
	return poFiles, nil
}

// CheckoutTmpfile checks out a file revision to a temp file for reading.
func CheckoutTmpfile(f *FileRevision) error {
	if f.Tmpfile == "" {
		tmpfile, err := os.CreateTemp("", "*--"+filepath.Base(f.File))
		if err != nil {
			return fmt.Errorf("fail to create tmpfile: %s", err)
		}
		f.Tmpfile = tmpfile.Name()
		tmpfile.Close()
	}
	if f.Revision == "" {
		// Read file from f.File and write to f.Tmpfile (no git needed)
		data, err := os.ReadFile(f.File)
		if err != nil {
			return fmt.Errorf("fail to read file: %w", err)
		}
		if err := os.WriteFile(f.Tmpfile, data, 0644); err != nil {
			return fmt.Errorf("fail to write tmpfile: %w", err)
		}
		log.Debugf("read file %s from %s and write to %s", f.File, f.Revision, f.Tmpfile)
		return nil
	}
	if err := repository.RequireOpened(); err != nil {
		return fmt.Errorf("git show requires a repository: %w", err)
	}
	cmd := exec.Command("git",
		"show",
		f.Revision+":"+f.File)
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf(`get StdoutPipe failed: %s`, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("fail to start git-show command: %s", err)
	}
	data, err := io.ReadAll(out)
	out.Close()
	if err != nil {
		return fmt.Errorf("fail to read git-show output: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("fail to wait git-show command: %s", err)
	}
	if err := os.WriteFile(f.Tmpfile, data, 0644); err != nil {
		return fmt.Errorf("fail to write tmpfile: %w", err)
	}
	log.Debugf(`creating "%s" file using command: %s`, f.Tmpfile, cmd.String())
	return nil
}
