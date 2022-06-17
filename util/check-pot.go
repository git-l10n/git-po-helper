package util

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if path.Ext(f.Name()) != ".txt" {
			continue
		}
		items, err := getConfigsFromOneManpage(filepath.Join(docDir, f.Name()), onlyCamelCase)
		if err != nil {
			return nil, err
		}
		configs = append(configs, items...)
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

func getConfigsFromL10n() ([]string, error) {
	var (
		potFile = filepath.Join(PoDir, GitPot)
		err     error
		configs []string
	)

	if !IsFile(potFile) {
		return nil, fmt.Errorf("cannot find file %s", potFile)
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

func ShowL10nConfigs() error {
	configs, err := getConfigsFromL10n()
	if err != nil {
		return err
	}
	for _, item := range configs {
		fmt.Println(item)
	}
	return nil
}
