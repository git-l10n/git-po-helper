// Package util provides file selection utilities for po operations.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

// ResolvePoFile resolves the po file to use: either the user-specified one (verified against
// changed files) or one selected from changed files (auto when 1, interactive when multiple).
// Returns the selected po file path (relative, e.g. po/zh_CN.po) or an error.
func ResolvePoFile(poFileArg string, changedPoFiles []string) (string, error) {
	if poFileArg == "" {
		// No poFile specified: select from changed files
		switch len(changedPoFiles) {
		case 0:
			return "", fmt.Errorf("no changed po files found between git versions\nHint: Specify po/XX.po explicitly or ensure there are changes in po/ directory")
		case 1:
			poFile := changedPoFiles[0]
			log.Infof("auto-selected po file: %s", poFile)
			return poFile, nil
		default:
			// Multiple files: interactive mode asks user, non-interactive errors
			if isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Fprintf(os.Stderr, "Multiple changed po files found:\n")
				for i, f := range changedPoFiles {
					fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, f)
				}
				answer := GetUserInput(fmt.Sprintf("Select file (1-%d): ", len(changedPoFiles)), "1")
				var idx int
				if _, err := fmt.Sscanf(answer, "%d", &idx); err != nil || idx < 1 || idx > len(changedPoFiles) {
					return "", fmt.Errorf("invalid selection: %s", answer)
				}
				poFile := changedPoFiles[idx-1]
				log.Infof("user selected po file: %s", poFile)
				return poFile, nil
			}
			return "", fmt.Errorf("multiple changed po files found (%s), specify one explicitly in non-interactive mode\nHint: Run with po/XX.po argument", strings.Join(changedPoFiles, ", "))
		}
	}

	// poFile specified: verify it appears in changed files
	poFileRel := filepath.ToSlash(poFileArg)
	if !strings.HasPrefix(poFileRel, PoDir+"/") {
		poFileRel = PoDir + "/" + filepath.Base(poFileArg)
	}
	for _, f := range changedPoFiles {
		if filepath.ToSlash(f) == poFileRel {
			return poFileArg, nil
		}
	}
	return "", fmt.Errorf("po file %s is not in the changed files: %s\nHint: The specified file has no changes between the compared git versions", poFileArg, strings.Join(changedPoFiles, ", "))
}
