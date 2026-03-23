// Package util provides path and filesystem utilities.
package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// Exist check if path is exist.
func Exist(name string) bool {
	if _, err := os.Stat(name); err == nil {
		return true
	}
	return false
}

// IsFile returns true if path is exist and is a file.
func IsFile(name string) bool {
	fi, err := os.Stat(name)
	if err != nil || fi.IsDir() {
		return false
	}
	return true
}

// IsDir returns true if path is exist and is a directory.
func IsDir(name string) bool {
	fi, err := os.Stat(name)
	if err != nil || !fi.IsDir() {
		return false
	}
	return true
}

// HasGitProjectLayout reports whether root looks like an upstream Git
// localization tree: Makefile, po/, and po/README.md.
func HasGitProjectLayout(root string) bool {
	mk := filepath.Join(root, "Makefile")
	st, err := os.Stat(mk)
	if err != nil || st.IsDir() {
		return false
	}
	poPath := filepath.Join(root, PoDir)
	st2, err := os.Stat(poPath)
	if err != nil || !st2.IsDir() {
		return false
	}
	readme := filepath.Join(poPath, "README.md")
	return Exist(readme)
}

// EnsureInGitProjectRootDir resolves the Git work tree root from the current
// working directory, verifies layout at that root via HasGitProjectLayout,
// requires po/AGENTS.md, then chdirs there. The returned cleanup restores the
// previous working directory. If repository.Opened() is already true, OpenRepository
// is skipped; otherwise OpenRepository(absWd) is used.
func EnsureInGitProjectRootDir() (func(), error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cannot get working directory: %w", err)
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve working directory: %w", err)
	}
	if !repository.Opened() {
		repository.OpenRepository(absWd)
	}
	if err := repository.RequireOpened(); err != nil {
		return nil, err
	}
	repoRoot := filepath.Clean(repository.WorkDir())
	if !HasGitProjectLayout(repoRoot) {
		return nil, fmt.Errorf(
			"this command is a demo for Git project, and must be run in a Git project source tree")
	}
	agentsPath := filepath.Join(repoRoot, PoDir, "AGENTS.md")
	if !Exist(agentsPath) {
		return nil, fmt.Errorf(
			"required file missing: %s",
			filepath.Join(PoDir, "AGENTS.md"))
	}
	if repoRoot != absWd {
		log.Infof("using Git work tree root: %s", repoRoot)
	}
	if err := os.Chdir(repoRoot); err != nil {
		return nil, fmt.Errorf("cannot change to work tree root %s: %w", repoRoot, err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			log.Errorf("cannot restore working directory %s: %v", wd, err)
		}
	}, nil
}
