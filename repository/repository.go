package repository

import (
	"github.com/jiangxin/goconfig"
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
		panic(theRepository.error)
	} else if theRepository.repository == nil {
		panic("TheRepository is nil")
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
