package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:           "update <XX.po>...",
	Short:         "Update XX.po file",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if len(args) == 0 {
			return newUserError("no argument for update command")
		}
		for _, locale := range args {
			if !util.CmdUpdate(locale) {
				err = errExecute
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
