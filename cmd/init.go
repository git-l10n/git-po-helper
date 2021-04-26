package cmd

import (
	"errors"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:           "init <XX.po>",
	Short:         "Create XX.po file",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return newUserError("must given 1 argument for init command")
		}
		if util.CmdInit(args[0]) {
			return nil
		}
		return errors.New("fail to execute init command")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
