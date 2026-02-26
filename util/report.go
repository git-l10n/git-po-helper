// Package util provides report and message utilities.
package util

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ReportInfoAndErrors reports errors with info or error level based on ok.
func ReportInfoAndErrors(errs []string, prompt string, ok bool) {
	if ok {
		reportResultMessages(errs, prompt, log.InfoLevel)
	} else {
		reportResultMessages(errs, prompt, log.ErrorLevel)
	}
}

// ReportWarnAndErrors reports errors with warn or error level based on ok.
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
