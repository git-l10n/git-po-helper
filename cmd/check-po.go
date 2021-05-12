package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type checkPoCommand struct {
	cmd *cobra.Command
}

func (v *checkPoCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "check-po <XX.po>...",
		Short:         "Check syntax of XX.po file",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().Bool("core",
		false,
		"also check XX.po against "+util.CorePot)
	viper.BindPFlag("check-po--core", v.cmd.Flags().Lookup("core"))

	return v.cmd
}

func (v checkPoCommand) Execute(args []string) error {
	if !util.CmdCheckPo(args...) {
		return executeError
	}
	return nil
}

var checkPoCmd = checkPoCommand{}

func init() {
	rootCmd.AddCommand(checkPoCmd.Command())
}
