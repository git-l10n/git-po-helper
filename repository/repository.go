package repository

import (
	"os"
	"path/filepath"

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
func OpenRepository(dir string) error {
	return theRepository.Open(dir)
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

// IsGitProject checks current workdir is belong to git project.
func IsGitProject() bool {
	poFile := filepath.Join(WorkDir(), "po", "git.pot")
	if _, err := os.Stat(poFile); err != nil {
		log.Debugf("'%s' is not belong to git project: %s", WorkDir(), err)
		return false
	}
	return true
}

// ChdirProjectRoot changes current dir to project root.
func ChdirProjectRoot() {
	if theRepository.repository.IsBare() {
		log.Fatal("fail to change workdir, you are in a bare repository")
	}
	if !IsGitProject() {
		log.Fatal("git-po-helper only works for git project.")
	}
	if err := os.Chdir(WorkDir()); err != nil {
		log.Fatal(err)
	}
}

// Config is git config for the repository.
func Config() goconfig.GitConfig {
	return theRepository.repository.Config()
}
