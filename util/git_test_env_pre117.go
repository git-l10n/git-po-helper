//go:build !go1.17
// +build !go1.17

package util

import (
	"os"
	"testing"
)

// setTestEnv sets an environment variable for the duration of a test and
// restores its previous value (or unsets it) during test cleanup.
// This fallback is used for Go versions older than 1.17, where
// testing.T.Setenv is not available.
func setTestEnv(t *testing.T, key, value string) {
	prev, had := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if had {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}
