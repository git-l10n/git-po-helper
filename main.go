package main

import (
	"os"

	"github.com/git-l10n/git-po-helper/cmd"
)

func main() {
	resp := cmd.Execute()

	if resp.Err != nil {
		if resp.IsUserError() {
			resp.Cmd.Println("")
			resp.Cmd.Println(resp.Cmd.UsageString())
		}
		os.Exit(-1)
	}
}
