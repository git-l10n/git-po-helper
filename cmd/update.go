package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <XX.po>...",
	Short: "Update XX.po file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return newUserError("no argument for update command")
		}
		for _, locale := range args {
			err := util.CmdUpdate(locale)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
