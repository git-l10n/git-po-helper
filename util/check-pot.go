package util

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	EightSpaces = "        "
)

var (
	gitConfigManpagePattern   = regexp.MustCompile(`^[a-z].*::$`)
	gitConfigCamelCasePattern = regexp.MustCompile(`[a-z][A-Z][a-z]`)
)

func getConfigsFromManpage(onlyCamelCase bool) ([]string, error) {
	var (
		err     error
		configs []string
	)

	// Scan *.txt files from Documentation/config/.
	docDir := filepath.Join("Documentation", "config")
	if !IsDir(docDir) {
		return nil, fmt.Errorf("cannot find dir %s", docDir)
	}

	files, err := os.ReadDir(docDir)
	if err != nil {
		return nil, err
	}

	var foundSuitable bool
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if path.Ext(f.Name()) != ".txt" && path.Ext(f.Name()) != ".adoc" {
			continue
		}
		foundSuitable = true
		items, err := getConfigsFromOneManpage(filepath.Join(docDir, f.Name()), onlyCamelCase)
		if err != nil {
			return nil, err
		}
		configs = append(configs, items...)
	}

	if !foundSuitable {
		return nil, fmt.Errorf("no .txt or .adoc files found in %s", docDir)
	}
	return configs, err
}

func getConfigsFromOneManpage(filename string, onlyCamelCase bool) ([]string, error) {
	var (
		configs []string
		err     error
	)

	f, err := os.Open(filename)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !gitConfigManpagePattern.MatchString(line) {
			continue
		}
		line = strings.TrimSuffix(line, "::")
		for _, word := range strings.Split(line, ",") {
			word = strings.TrimSpace(word)
			// Trim suffix such as " (deprecated)"
			word = strings.Replace(word, " (deprecated)", "", -1)
			// Ignore config variable such as: advice.*
			if strings.Contains(word, "*") {
				continue
			}
			if onlyCamelCase && !gitConfigCamelCasePattern.MatchString(word) {
				continue
			}
			configs = append(configs, word)
		}
	}

	return configs, err
}

func ShowManpageConfigs(onlyCamelCase bool) error {
	configs, err := getConfigsFromManpage(onlyCamelCase)
	if err != nil {
		return err
	}
	for _, item := range configs {
		fmt.Println(item)
	}
	return nil
}

// CheckGitPotFile reads the POT file, verifies it is a Git project, and runs the CamelCase config variable check.
// Returns an error for read/parse failure, missing or non-Git Project-Id-Version, or check failure.
func CheckGitPotFile(potFile string) error {
	poData, err := os.ReadFile(potFile)
	if err != nil {
		return fmt.Errorf("fail to read %q: %w", potFile, err)
	}
	po, err := ParsePoEntries(poData)
	if err != nil {
		return fmt.Errorf("fail to parse %q: %w", potFile, err)
	}
	projectName := po.GetProject()
	if !strings.EqualFold(projectName, "Git") {
		if projectName == "" {
			return fmt.Errorf("do not know how to check .pot for project without Project-Id-Version: %q", potFile)
		}
		return fmt.Errorf("do not know how to check .pot for non-Git project %q: %q", projectName, potFile)
	}
	return CheckCamelCaseConfigVariableInPotFile(po)
}

// countConfigMismatchesInString checks a single string (e.g. msgid or msgid_plural) for config variable casing.
// Returns the number of mismatches and logs each one.
func countConfigMismatchesInString(text string, configs []string) int {
	mismatched := 0
	for _, item := range configs {
		for len(text) > 0 {
			lowerText := strings.ToLower(text)
			idx := strings.Index(lowerText, strings.ToLower(item))
			if idx == -1 {
				break
			}
			if strings.HasPrefix(text[idx:], item) {
				log.Debugf("'%s' is found in: %s", item, text)
			} else {
				log.Errorf("config variable '%s' in manpage does not match string in pot file:", item)
				log.Errorf("    >> %s", text)
				mismatched++
			}
			text = text[idx+len(item):]
		}
	}
	return mismatched
}

// CheckCamelCaseConfigVariableInPotFile checks CamelCase config variables using the parsed POT (po).
// Caller should ensure the PO is a Git project (Project-Id-Version indicates Git).
// Requires Documentation/config to exist and contain at least one .txt or .adoc file; otherwise returns an error.
func CheckCamelCaseConfigVariableInPotFile(po *GettextPO) error {
	configs, err := getConfigsFromManpage(false)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		return fmt.Errorf("no Git config variables were scanned for checking POT msgid")
	}

	mismatched := 0
	for _, e := range po.Entries {
		mismatched += countConfigMismatchesInString(e.MsgID, configs)
		if e.MsgIDPlural != "" {
			mismatched += countConfigMismatchesInString(e.MsgIDPlural, configs)
		}
	}

	if mismatched != 0 {
		return fmt.Errorf("%d mismatched config variables", mismatched)
	}
	return nil
}
