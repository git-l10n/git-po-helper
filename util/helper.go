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
	log "github.com/sirupsen/logrus"
)

var (
	oidRegex = regexp.MustCompile(`^[0-9a-f]{7,40}[0-9]*$`)
)

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
func GetPrettyLocaleName(locale string) (string, []error) {
	var (
		lang     string
		zone     string
		langName string
		zoneName string
		errs     []error
	)
	items := strings.SplitN(locale, "_", 2)
	lang = items[0]
	if len(items) > 1 {
		zone = items[1]
	}
	if lang != strings.ToLower(lang) {
		errs = append(errs, fmt.Errorf(
			`language code %q must be all lowercase`,
			lang))
		lang = strings.ToLower(lang)
	}
	if zone != "" && zone != strings.ToUpper(zone) {
		errs = append(errs, fmt.Errorf(
			`region/territory code %q must be all uppercase`,
			zone))
		zone = strings.ToUpper(zone)
	}
	langName = data.GetLanguageName(lang)
	if langName == "" {
		errs = append(errs, fmt.Errorf(
			`invalid language code for "%s", see ISO 639 for valid codes`,
			lang))
	}
	if zone != "" {
		zoneName = data.GetLocationName(zone)
		if zoneName == "" {
			errs = append(errs, fmt.Errorf(
				`invalid country or location code for "%s", see ISO 3166 for valid codes`,
				zone))
		}
	}
	if zoneName != "" {
		return fmt.Sprintf("%s - %s", langName, zoneName), errs
	}
	return langName, errs
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
