// Package util provides git-related utilities for po file operations.
package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// ErrOutsideWorktree is returned by GetRepoRelPath when the path resolves outside
// the repository work tree (including repo-relative paths that escape with "..").
var ErrOutsideWorktree = errors.New("path is outside repository worktree")

// ErrFileNotInRevision is returned by FileRevision.GetFile when the path does not
// exist at the requested revision (e.g. a newly added po/XX.po).
var ErrFileNotInRevision = errors.New("file does not exist in revision")

// FileRevision identifies a path at a revision for materialization via GetFile.
// IsTipCommit is metadata for callers (e.g. check-commits PO policy); GetFile does not use it.
type FileRevision struct {
	Revision string
	File     string
	tmpfile  string
	// cached is true after git show successfully wrote tmpfile for non-empty Revision.
	cached      bool
	IsTipCommit bool
}

// GetFile returns a path to the file contents. When Revision is empty, returns File after
// verifying it exists (worktree path; no temp file is used). When Revision is non-empty,
// runs git show once into a temp file and reuses that path on later GetFile calls without
// running git show again until Cleanup. Call Cleanup when done to remove tmpfile only for
// the non-empty Revision case.
func (f *FileRevision) GetFile() (string, error) {
	if strings.TrimSpace(f.Revision) == "" {
		if f.File == "" {
			return "", fmt.Errorf("file path is empty")
		}
		if _, err := os.Stat(f.File); err != nil {
			return "", fmt.Errorf("fail to access file: %w", err)
		}
		f.tmpfile = ""
		f.cached = false
		log.Debugf("using worktree file %s directly", f.File)
		return f.File, nil
	}

	if f.cached && f.tmpfile != "" {
		if _, err := os.Stat(f.tmpfile); err == nil {
			log.Debugf("reusing cached git show output %s", f.tmpfile)
			return f.tmpfile, nil
		}
		f.tmpfile = ""
		f.cached = false
	}

	if f.tmpfile == "" {
		tf, err := os.CreateTemp("", "*--"+filepath.Base(f.File))
		if err != nil {
			return "", fmt.Errorf("fail to create tmpfile: %s", err)
		}
		f.tmpfile = tf.Name()
		tf.Close()
	}
	if err := repository.RequireOpened(); err != nil {
		return "", fmt.Errorf("git show requires a repository: %w", err)
	}
	object := f.Revision + ":" + f.File
	if err := ensureGitPathAtRevision(f.Revision, f.File); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "show", object)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf(`get StdoutPipe failed: %s`, err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("fail to start git-show command: %s", err)
	}
	data, err := io.ReadAll(out)
	out.Close()
	if err != nil {
		return "", fmt.Errorf("fail to read git-show output: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		log.Debugf("git show %s failed: %v (%s)", object, err, msg)
		if msg != "" {
			return "", fmt.Errorf("fail to wait git-show command: %s: %s", err, msg)
		}
		return "", fmt.Errorf("fail to wait git-show command: %s", err)
	}
	if err := os.WriteFile(f.tmpfile, data, 0644); err != nil {
		return "", fmt.Errorf("fail to write tmpfile: %w", err)
	}
	f.cached = true
	log.Debugf(`creating "%s" file using command: %s`, f.tmpfile, cmd.String())
	return f.tmpfile, nil
}

// ensureGitPathAtRevision checks whether rev:path exists via `git cat-file -e`.
// Detection uses only the exit status (locale-independent). If the revision
// itself is invalid, returns a normal error (not ErrFileNotInRevision).
func ensureGitPathAtRevision(revision, file string) error {
	object := revision + ":" + file
	cmd := exec.Command("git", "cat-file", "-e", object)
	cmd.Stderr = nil // discard localized stderr; exit code is enough
	if err := cmd.Run(); err == nil {
		return nil
	}
	log.Debugf("git cat-file -e %s failed", object)

	revCmd := exec.Command("git", "rev-parse", "--verify", revision+"^{commit}")
	revCmd.Stderr = nil
	if revErr := revCmd.Run(); revErr != nil {
		return fmt.Errorf("fail to resolve %s: revision %q is invalid", object, revision)
	}
	return fmt.Errorf("%w: %s", ErrFileNotInRevision, object)
}

// Cleanup removes the temp file created by GetFile for a non-empty Revision and clears tmpfile.
// When Revision was empty, GetFile leaves tmpfile unset and Cleanup is a no-op.
func (f *FileRevision) Cleanup() {
	if f.tmpfile != "" {
		_ = os.Remove(f.tmpfile)
		f.tmpfile = ""
	}
	f.cached = false
}

// GetRepoRelPath returns file as a slash-separated path relative to the opened repository
// work tree. If file is not absolute, it is interpreted relative to the work tree root
// (same rules as check-po-filter-format / git check-attr path handling). If the resolved
// path lies outside the work tree, the error wraps ErrOutsideWorktree (use errors.Is).
//
// Bare repositories have no work tree (repository.WorkDir is empty). In that case an
// absolute file path cannot be mapped to a repo-relative path and returns an error; a
// non-absolute file is returned normalized with slash separators (the path is used as-is
// for object/commit operations such as git check-attr --source).
func GetRepoRelPath(file string) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", fmt.Errorf("file path is empty")
	}
	if err := repository.RequireOpened(); err != nil {
		return "", err
	}
	workDir := strings.TrimSpace(repository.WorkDir())
	if workDir == "" {
		if filepath.IsAbs(file) {
			return "", fmt.Errorf("bare repository has no work tree: cannot use absolute path %q as a repository-relative path", file)
		}
		clean := filepath.Clean(filepath.FromSlash(filepath.ToSlash(file)))
		return filepath.ToSlash(clean), nil
	}

	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve work tree: %w", err)
	}

	var absJoined string
	if !filepath.IsAbs(file) {
		clean := filepath.Clean(filepath.FromSlash(filepath.ToSlash(file)))
		joined := filepath.Join(workDir, clean)
		absJoined, err = filepath.Abs(joined)
		if err != nil {
			return "", fmt.Errorf("cannot resolve path %q: %w", file, err)
		}
	} else {
		absJoined, err = filepath.Abs(file)
		if err != nil {
			return "", fmt.Errorf("cannot resolve path %q: %w", file, err)
		}
	}

	rel, err := filepath.Rel(absWork, absJoined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: %s", ErrOutsideWorktree, file)
	}
	return filepath.ToSlash(rel), nil
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
