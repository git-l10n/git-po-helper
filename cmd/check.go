package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type checkCommand struct {
	cmd *cobra.Command
}

func (v *checkCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "check",
		Short:         `Check all ".po" files and commits`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().Bool("no-gpg",
		false,
		"do no verify gpg-signed commit")
	v.cmd.Flags().BoolP("force",
		"f",
		false,
		"run even too many commits")
	v.cmd.Flags().Bool("core",
		false,
		"also check XX.po against "+util.CorePot)
	viper.BindPFlag("check--no-gpg", v.cmd.Flags().Lookup("no-gpg"))
	viper.BindPFlag("check--force", v.cmd.Flags().Lookup("force"))
	viper.BindPFlag("check--core", v.cmd.Flags().Lookup("core"))

	return v.cmd
}

func (v checkCommand) Execute(args []string) error {
	var err error

	if len(args) != 0 {
		return newUserError("check command needs no arguments")
	}
	if !util.CmdCheckPo() {
		err = errExecute
	}
	if !util.CmdCheckCommits() {
		err = errExecute
	}
	return err
}

var checkCmd = checkCommand{}

func init() {
	rootCmd.AddCommand(checkCmd.Command())
}
