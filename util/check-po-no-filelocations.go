package util

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
)

func checkPoNoFileLocations(poFile string) ([]string, bool) {
	var (
		err  error
		errs []string
	)
	if !flag.CheckFileLocations() {
		return nil, true
	}

	f, err := os.Open(poFile)
	if err != nil {
		errs = append(errs, fmt.Sprintf("cannot open %s: %s", poFile, err))
		return errs, false
	}
	defer f.Close()

	pattern := regexp.MustCompile(`.*:\d+$`)
	isHeader := true
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if isHeader {
			if line == "" {
				isHeader = false
			}
			continue
		}
		if !strings.HasPrefix(line, "#: ") {
			continue
		}
		locations := strings.Split(line[3:], " ")
		if pattern.MatchString(locations[0]) {
			errs = append(errs,
				"Found file-location comments in po file.\n",
				"Please commit a location-less \"po/XX.po\" file to save repository size.",
				"See: [Updating a \"XX.po\" file] section in \"po/README.md\" for reference.",
			)
			return errs, false
		}
	}
	return nil, true
}
