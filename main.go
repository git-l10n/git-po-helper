package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/git-l10n/git-po-helper/cmd"
)

const (
	// Program is name for this project
	Program = "git-po-helper"
)

func main() {
	resp := cmd.Execute()

	if resp.Err != nil {
		errOut := resp.Cmd.ErrOrStderr()
		msg := strings.TrimRight(resp.Err.Error(), "\n")
		fmt.Fprintf(errOut, "ERROR: %s\n", msg)
		if cmd.IsErrorWithUsage(resp.Err) {
			fmt.Fprint(errOut, resp.Cmd.UsageString())
		}
		os.Exit(-1)
	}
}
