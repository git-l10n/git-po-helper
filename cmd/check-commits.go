package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type checkCommitsCommand struct {
	cmd *cobra.Command
}

func (v *checkCommitsCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "check-commits [<range>]",
		Short:         "Check commits for l10n conventions",
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
	viper.BindPFlag("check-commits--no-gpg", v.cmd.Flags().Lookup("no-gpg"))
	viper.BindPFlag("check-commits--force", v.cmd.Flags().Lookup("force"))
	return v.cmd
}

func (v checkCommitsCommand) Execute(args []string) error {
	if !util.CmdCheckCommits(args...) {
		return executeError
	}
	return nil
}

var checkCommitsCmd = checkCommitsCommand{}

func init() {
	rootCmd.AddCommand(checkCommitsCmd.Command())
}
