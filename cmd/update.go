package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:           "update <XX.po>...",
	Short:         "Update XX.po file",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var errMsgs []string

		if len(args) == 0 {
			return newUserError("no argument for update command")
		}
		for _, locale := range args {
			if !util.CmdUpdate(locale) {
				errMsgs = append(errMsgs, fmt.Sprintf("fail to update '%s'", locale))
			}
		}
		if len(errMsgs) > 0 {
			return errors.New(strings.Join(errMsgs, "\n"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
