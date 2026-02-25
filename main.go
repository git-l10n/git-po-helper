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
		if resp.IsUserError() {
			if resp.Cmd.SilenceErrors {
				fmt.Fprintf(errOut, "ERROR: %s\n\n", resp.Err)
			}
			fmt.Fprint(errOut, resp.Cmd.UsageString())
		} else if resp.Cmd.SilenceErrors {
			fmt.Fprintln(errOut, "")
			// Use CommandPath() to get full command path (e.g., "git-po-helper agent-run translate")
			// Remove Program prefix to get subcommand path (e.g., "agent-run translate")
			cmdPath := resp.Cmd.CommandPath()
			subCmdPath := strings.TrimPrefix(cmdPath, Program+" ")
			if subCmdPath == "" {
				// Fallback to Name() if CommandPath() only contains Program
				subCmdPath = resp.Cmd.Name()
			}
			fmt.Fprintf(errOut, "ERROR: fail to execute \"%s %s\"\n", Program, subCmdPath)
		}
		os.Exit(-1)
	}
}
