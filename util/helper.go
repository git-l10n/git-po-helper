package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/git-l10n/git-po-helper/data"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

var (
	oidRegex = regexp.MustCompile(`^[0-9a-f]{7,40}[0-9]*$`)
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

// ShowExecError will try to return error message of stderr
func ShowExecError(err error) {
	if err == nil {
		return
	}
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return
	}
	buf := bytes.NewBuffer(exitError.Stderr)
	for {
		line, eof := buf.ReadString('\n')
		if len(line) > 0 {
			log.Error(line)
		}
		if eof != nil {
			break
		}
	}
}

// GetPrettyLocaleName shows full language name and location
func GetPrettyLocaleName(locale string) (string, error) {
	var (
		langName string
		locName  string
	)
	items := strings.SplitN(locale, "_", 2)
	langName = data.GetLanguageName(items[0])
	if langName == "" {
		return "", fmt.Errorf("invalid language code for locale \"%s\"", locale)
	}
	if len(items) > 1 && items[1] != "" {
		locName = data.GetLocationName(items[1])
		if locName == "" {
			return "", fmt.Errorf(`invalid country or location code for locale "%s"`, locale)
		}
	}
	if locName != "" {
		return fmt.Sprintf("%s - %s", langName, locName), nil
	}
	return langName, nil
}

// GetUserInput reads user input from stdin.
// Prompt is written to stderr so stdout remains clean for redirects (e.g. compare >file).
func GetUserInput(prompt, defaultValue string) string {
	fmt.Fprint(os.Stderr, prompt)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	if text == "" {
		return defaultValue
	}
	return text
}

// AnswerIsTrue indicates answer is a true value
func AnswerIsTrue(answer string) bool {
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "y" ||
		answer == "yes" ||
		answer == "t" ||
		answer == "true" ||
		answer == "on" ||
		answer == "1" {
		return true
	}
	return false
}

// AbbrevCommit returns abbrev commit id
func AbbrevCommit(oid string) string {
	if oidRegex.MatchString(oid) {
		return oid[:7]
	}
	return oid
}

func ReportInfoAndErrors(errs []string, prompt string, ok bool) {
	if ok {
		reportResultMessages(errs, prompt, log.InfoLevel)
	} else {
		reportResultMessages(errs, prompt, log.ErrorLevel)
	}
}

func ReportWarnAndErrors(errs []string, prompt string, ok bool) {
	if ok {
		reportResultMessages(errs, prompt, log.WarnLevel)
	} else {
		reportResultMessages(errs, prompt, log.ErrorLevel)
	}
}

func reportResultMessages(errs []string, prompt string, level log.Level) {
	var fn func(format string, args ...interface{})

	if len(errs) == 0 {
		return
	}

	switch level {
	case log.InfoLevel:
		fn = log.Printf
	case log.WarnLevel:
		fn = log.Warnf
	default:
		fn = log.Errorf
	}

	showHorizontalLine()

	for _, err := range errs {
		if err == "" {
			fn("%s", prompt)
			continue
		}
		for _, line := range strings.Split(err, "\n") {
			if prompt == "" {
				fn("%s", line)
			} else if line == "" {
				fn("%s", prompt)
			} else {
				fn("%s\t%s", prompt, line)
			}
		}
	}
}

func showHorizontalLine() {
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 78))
}

// CompareTarget holds the resolved old/new commit and file for compare operations.
type CompareTarget struct {
	OldCommit string
	NewCommit string
	OldFile   string
	NewFile   string
}

// ResolveRevisionsAndFiles resolves range/commit/since flags and args into a CompareTarget.
// Exactly one of rangeStr, commitStr, and sinceStr may be non-empty.
// Args may be 0, 1, or 2 po file paths. With 2 args, revisions are not allowed.
// When args is empty, the po file is auto-selected from changed files.
func ResolveRevisionsAndFiles(rangeStr, commitStr, sinceStr string, args []string) (*CompareTarget, error) {
	// --range, --commit, --since are mutually exclusive
	nSet := 0
	if strings.TrimSpace(rangeStr) != "" {
		nSet++
	}
	if strings.TrimSpace(commitStr) != "" {
		nSet++
	}
	if strings.TrimSpace(sinceStr) != "" {
		nSet++
	}
	if nSet > 1 {
		return nil, fmt.Errorf("only one of --range, --commit, or --since may be specified")
	}

	// Resolve range for both modes
	var revRange string
	if c := strings.TrimSpace(commitStr); c != "" {
		revRange = c + "^.." + c
	} else if s := strings.TrimSpace(sinceStr); s != "" {
		revRange = s + ".."
	} else {
		revRange = strings.TrimSpace(rangeStr)
	}
	if revRange == "" {
		switch len(args) {
		case 0:
			revRange = "HEAD.."
		case 1:
			revRange = "HEAD.."
		case 2:
			// Compare two files in worktree
		}
	}

	if len(args) > 2 {
		return nil, fmt.Errorf("too many arguments (%d > 2)", len(args))
	}

	repository.ChdirProjectRoot()

	var (
		oldCommit, newCommit string
		oldFile, newFile     string
	)
	// Parse revision: "a..b", "a..", or "a"
	if strings.Contains(revRange, "..") {
		parts := strings.SplitN(revRange, "..", 2)
		oldCommit = strings.TrimSpace(parts[0])
		newCommit = strings.TrimSpace(parts[1])
	} else if revRange != "" {
		// a : first is a~, second is a
		oldCommit = revRange + "~"
		newCommit = revRange
	}

	// Set File
	switch len(args) {
	case 0:
		// Automatically or manually select PO file from changed files
	case 1:
		oldFile = args[0]
		newFile = args[0]
	case 2:
		oldFile = args[0]
		newFile = args[1]
		if oldCommit != "" || newCommit != "" {
			return nil, fmt.Errorf("cannot specify revision for multiple files: %s and %s",
				oldFile, newFile)
		}
	}

	// Resolve poFile when not specified
	if len(args) == 0 {
		changedPoFiles, err := GetChangedPoFilesRange(oldCommit, newCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to get changed po files: %w", err)
		}

		oldFile, err = ResolvePoFile(oldFile, changedPoFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve default po file: %w", err)
		}
		newFile = oldFile
	}

	return &CompareTarget{
		OldCommit: oldCommit,
		NewCommit: newCommit,
		OldFile:   oldFile,
		NewFile:   newFile,
	}, nil
}
