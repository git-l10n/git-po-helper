package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type updateCommand struct {
	cmd *cobra.Command
}

func (v *updateCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "update <XX.po>...",
		Short: "Update XX.po file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().Bool("no-location",
		false,
		"no filename and location in comment for entry")
	v.cmd.Flags().Bool("no-line-number",
		false,
		"no line number in comment for entry")
	_ = viper.BindPFlag("no-location", v.cmd.Flags().Lookup("no-location"))
	_ = viper.BindPFlag("no-line-number", v.cmd.Flags().Lookup("no-line-number"))
	return v.cmd
}

func (v updateCommand) Execute(args []string) error {
	if len(args) == 0 {
		return NewErrorWithUsage("no argument for update command")
	}
	for _, locale := range args {
		if !util.CmdUpdate(locale) {
			return NewStandardError("update command failed")
		}
	}
	return nil
}

var updateCmd = updateCommand{}

func init() {
	rootCmd.AddCommand(updateCmd.Command())
}
