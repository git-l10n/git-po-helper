package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

func CheckCamelCaseConfigVariableInPotFile() error {
	var (
		potFile   = filepath.Join(PoDir, GitPot)
		configs   []string
		err       error
		unmatched = 0
	)

	if !IsFile(potFile) {
		return fmt.Errorf("cannot find file %s", potFile)
	}

	configs, err = getConfigsFromManpage(false)
	if err != nil {
		return err
	}

	// Make sure pot file is pretty formatted.
	cmd := exec.Command("msgcat", "--no-wrap", "--indent", potFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}

	// Scan msgid, which has prefix "msgid", "msgid_plural", and 8 spaces.
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		} else if line[0] == '#' {
			continue
		} else if strings.HasPrefix(line, "msgstr") {
			continue
		}

		for _, item := range configs {
			for len(line) > 0 {
				lowerLine := strings.ToLower(line)
				if idx := strings.Index(lowerLine, strings.ToLower(item)); idx != -1 {
					if strings.HasPrefix(line[idx:], item) {
						log.Debugf("'%s' is found in: %s", item, line)
					} else {
						log.Errorf("config variable '%s' in manpage does not match string in pot file:", item)
						log.Errorf("    >> %s", line)
						unmatched++
					}
					line = line[idx+len(item):]
				} else {
					break
				}
			}
		}
	}

	if unmatched != 0 {
		return fmt.Errorf("%d unmatched config variables", unmatched)
	}
	return nil
}
