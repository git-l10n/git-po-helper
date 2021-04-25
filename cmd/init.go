package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <XX.po>",
	Short: "Create XX.po file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return newUserError("must given 1 argument for init command")
		}
		return util.CmdInit(args[0])
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
