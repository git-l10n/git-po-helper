package main

import (
	"os"

	"github.com/git-l10n/git-po-helper/cmd"
	"github.com/git-l10n/git-po-helper/gettext"
)

const (
	// Program is name for this project
	Program = "git-po-helper"
)

func main() {
	resp := cmd.Execute()

	// Do cleanups
	for _, app := range gettext.GettextAppMap {
		if app.Defer != nil {
			app.Defer()
		}
	}

	if resp.Err != nil {
		if resp.IsUserError() {
			if resp.Cmd.SilenceErrors {
				resp.Cmd.Printf("ERROR: %s\n", resp.Err)
				resp.Cmd.Println("")
			}
			resp.Cmd.Println(resp.Cmd.UsageString())
		} else if resp.Cmd.SilenceErrors {
			resp.Cmd.Println("")
			resp.Cmd.Printf("ERROR: fail to execute \"%s %s\"\n", Program, resp.Cmd.Name())
		}
		os.Exit(-1)
	}
}
