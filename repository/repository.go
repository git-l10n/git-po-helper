// Package repository provides model for repository.
package repository

import (
	"fmt"
	"os"

	"github.com/jiangxin/goconfig"
	log "github.com/sirupsen/logrus"
)

// Repository holds repository and error.
type Repository struct {
	repository *goconfig.Repository
	error      error
}

var theRepository Repository

// Open will try to find repository in dir.
func (v *Repository) Open(dir string) error {
	v.repository, v.error = goconfig.FindRepository(dir)
	return v.error
}

// OpenRepository will try to find repository in dir.
func OpenRepository(dir string) {
	// Will check error in assertRepositoryNotNil
	_ = theRepository.Open(dir)
}

// Opened returns true if a repository was successfully opened (e.g. when running inside a git worktree).
// Commands that can run without a repo (e.g. stat with explicit paths) may skip ChdirProjectRoot when !Opened().
func Opened() bool {
	return theRepository.error == nil && theRepository.repository != nil
}

// Err returns the error from the last OpenRepository call, or nil if open succeeded.
func Err() error {
	return theRepository.error
}

// RequireOpened returns Err() if the repository is not opened.
// Use before git calls in commands that can run without a repo; return the error
// to the user instead of Fatal when repo is required but not available.
func RequireOpened() error {
	if !Opened() {
		if theRepository.error != nil {
			return theRepository.error
		}
		return fmt.Errorf("not in a git repository")
	}
	return nil
}

func assertRepositoryNotNil() {
	if theRepository.error != nil {
		log.Fatal(theRepository.error)
	} else if theRepository.repository == nil {
		log.Fatal("TheRepository is nil")
	}
}

// GitDir returns locations of .git dir.
func GitDir() string {
	assertRepositoryNotNil()
	return theRepository.repository.GitDir()
}

// WorkDir returns root dir of worktree.
func WorkDir() string {
	assertRepositoryNotNil()
	return theRepository.repository.WorkDir()
}

// WorkDirOrCwd returns WorkDir() when a repository is opened, otherwise the current working directory.
// Use this in commands that can run without a repo; paths will be relative to cwd when not in repo.
func WorkDirOrCwd() string {
	if Opened() {
		return theRepository.repository.WorkDir()
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// Config is git config for the repository.
func Config() goconfig.GitConfig {
	return theRepository.repository.Config()
}
