// Package util provides path and filesystem utilities.
package util

import (
	"os"
	"strings"
)

// DeriveReviewPaths takes a path and returns (jsonFile, poFile). The path may end with
// .json or .po; the extension is stripped to get the base. Returns base+".json" and
// base+".po". Use this to ensure json and po filenames are always consistent.
func DeriveReviewPaths(path string) (jsonFile, poFile string) {
	base := path
	if strings.HasSuffix(base, ".json") {
		base = strings.TrimSuffix(base, ".json")
	} else if strings.HasSuffix(base, ".po") {
		base = strings.TrimSuffix(base, ".po")
	}
	return base + ".json", base + ".po"
}

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
