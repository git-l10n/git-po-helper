package utiltest

import (
	"os"
	"strings"
	"testing"
)

// SetGitCeilingDirectories sets GIT_CEILING_DIRECTORIES so git does not treat
// parent directories (e.g. the git-po-helper worktree) as part of the search
// when tests run under a temp tree. Pass the root of the temp project (the
// directory that contains Makefile and po/). Multiple roots use the platform
// path list separator.
//
// The cleanup runs on t.Cleanup. Intended for unit tests that must behave as if
// there is no enclosing git repository.
func SetGitCeilingDirectories(t *testing.T, roots ...string) {
	t.Helper()
	prev, had := os.LookupEnv("GIT_CEILING_DIRECTORIES")
	sep := string(os.PathListSeparator)
	os.Setenv("GIT_CEILING_DIRECTORIES", strings.Join(roots, sep))
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("GIT_CEILING_DIRECTORIES", prev)
		} else {
			_ = os.Unsetenv("GIT_CEILING_DIRECTORIES")
		}
	})
}
