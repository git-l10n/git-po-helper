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
		Use:   "check",
		Short: `Check all ".po" files and commits`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().Bool("no-gpg",
		false,
		"do not verify gpg-signed commit")
	v.cmd.Flags().BoolP("force",
		"f",
		false,
		"run even too many commits")
	v.cmd.Flags().Bool("core",
		false,
		"also check XX.po against "+util.CorePot)
	v.cmd.Flags().String("report-typos",
		"",
		"way to display typos (none, warn, error)")
	v.cmd.Flags().String("report-file-locations",
		"",
		"way to report file-location issues (none, warn, error)")
	_ = viper.BindPFlag("check--no-gpg", v.cmd.Flags().Lookup("no-gpg"))
	_ = viper.BindPFlag("check--force", v.cmd.Flags().Lookup("force"))
	_ = viper.BindPFlag("check--core", v.cmd.Flags().Lookup("core"))
	_ = viper.BindPFlag("check--report-typos", v.cmd.Flags().Lookup("report-typos"))
	_ = viper.BindPFlag("check--report-file-locations", v.cmd.Flags().Lookup("report-file-locations"))

	return v.cmd
}

func (v checkCommand) Execute(args []string) error {
	if len(args) != 0 {
		return NewErrorWithUsage("check command needs no arguments")
	}
	if !util.CmdCheckPo() {
		return NewStandardError("check command failed")
	}
	if !util.CmdCheckCommits() {
		return NewStandardError("check command failed")
	}
	return nil
}

var checkCmd = checkCommand{}

func init() {
	rootCmd.AddCommand(checkCmd.Command())
}
