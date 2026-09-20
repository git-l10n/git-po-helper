//go:build go1.17
// +build go1.17

package util

import "testing"

// setTestEnv sets an environment variable for the duration of a test.
// On Go 1.17+, it delegates to testing.T.Setenv, which automatically
// restores the original value during test cleanup.
func setTestEnv(t *testing.T, key, value string) {
	t.Setenv(key, value)
}
