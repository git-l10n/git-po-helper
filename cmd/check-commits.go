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
		Use:   "check-commits [<range>]",
		Short: "Check commits for l10n conventions",
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
	v.cmd.Flags().String("report-typos",
		"",
		"way to display typos (none, warn, error)")
	v.cmd.Flags().String("report-file-locations",
		"",
		"way to report file-location issues (none, warn, error)")
	_ = viper.BindPFlag("check-commits--no-gpg", v.cmd.Flags().Lookup("no-gpg"))
	_ = viper.BindPFlag("check-commits--force", v.cmd.Flags().Lookup("force"))
	_ = viper.BindPFlag("check-commits--report-typos", v.cmd.Flags().Lookup("report-typos"))
	_ = viper.BindPFlag("check-commits--report-file-locations", v.cmd.Flags().Lookup("report-file-locations"))
	return v.cmd
}

func (v checkCommitsCommand) Execute(args []string) error {
	if !util.CmdCheckCommits(args...) {
		return NewStandardError("check-commits command failed")
	}
	return nil
}

var checkCommitsCmd = checkCommitsCommand{}

func init() {
	rootCmd.AddCommand(checkCommitsCmd.Command())
}
