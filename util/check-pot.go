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

func getConfigsFromManpage(configsDir string, onlyCamelCase bool) ([]string, error) {
	var (
		err     error
		configs []string
	)

	// Scan *.txt files from Documentation/config/.
	if !IsDir(configsDir) {
		return nil, fmt.Errorf("cannot find dir %s", configsDir)
	}

	files, err := os.ReadDir(configsDir)
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
		items, err := getConfigsFromOneManpage(filepath.Join(configsDir, f.Name()), onlyCamelCase)
		if err != nil {
			return nil, err
		}
		configs = append(configs, items...)
	}

	if !foundSuitable {
		return nil, fmt.Errorf("no .txt or .adoc files found in %s", configsDir)
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
	return checkMissMatchedConfigVariableInPotFile(po, potFile)
}

const configMismatchExcerptLen = 80

// collectConfigMismatchErrs finds config variables in text that appear with wrong casing and returns error messages.
// entryLine is the 1-based line number of the entry; fieldName is "msgid" or "msgid_plural".
// Each error includes entry line, the full config variable name to use, and a short excerpt of the string.
func collectConfigMismatchErrs(text string, configs []string, entryLine int, fieldName string) []string {
	var errs []string
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
				start := idx
				if start > 25 {
					start = idx - 25
				}
				end := idx + len(item) + 35
				if end > len(text) {
					end = len(text)
				}
				excerpt := text[start:end]
				excerpt = strings.ReplaceAll(excerpt, "\n", " ")
				if len(excerpt) > configMismatchExcerptLen {
					excerpt = excerpt[:configMismatchExcerptLen-3] + "..."
				}
				lineInfo := ""
				if entryLine > 0 {
					lineInfo = fmt.Sprintf("entry at L%d ", entryLine)
				}
				errs = append(errs, fmt.Sprintf("%s(%s): should use %q in msgid: %q",
					lineInfo, fieldName, item, excerpt))
			}
			text = text[idx+len(item):]
		}
	}
	return errs
}

// checkMissMatchedConfigVariableInPotFile checks CamelCase config variables using the parsed POT (po).
func checkMissMatchedConfigVariableInPotFile(po *GettextPO, potFile string) error {
	prompt := "[" + filepath.Base(potFile) + "]"
	if !filepath.IsAbs(potFile) {
		absPotFile, err := filepath.Abs(potFile)
		if err != nil {
			return fmt.Errorf("fail to get absolute path of %q: %w", potFile, err)
		}
		potFile = absPotFile
	}
	configsDir := filepath.Join(filepath.Dir(filepath.Dir(potFile)), "Documentation", "config")
	configs, err := getConfigsFromManpage(configsDir, false)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		return fmt.Errorf("no Git config variables were scanned for checking POT msgid")
	}

	var errs []string
	for _, e := range po.Entries {
		errs = append(errs, collectConfigMismatchErrs(e.MsgID, configs, e.EntryLocation, "msgid")...)
		if e.MsgIDPlural != "" {
			errs = append(errs, collectConfigMismatchErrs(e.MsgIDPlural, configs, e.EntryLocation, "msgid_plural")...)
		}
	}

	if len(errs) > 0 {
		ReportSection("CamelCase config variables", false, log.InfoLevel, prompt, errs...)
		return fmt.Errorf("%d entries checked, %d config variables, %d mismatched",
			len(po.Entries), len(configs), len(errs))
	}
	msg := fmt.Sprintf("checked %d entries against %d config variables", len(po.Entries), len(configs))
	ReportSection("CamelCase config variables", true, log.InfoLevel, prompt, msg)
	return nil
}
