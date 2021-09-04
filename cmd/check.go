package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
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
		"do not verify gpg-signed commit")
	v.cmd.Flags().BoolP("force",
		"f",
		false,
		"run even too many commits")
	v.cmd.Flags().Bool("core",
		false,
		"also check XX.po against "+util.CorePot)
	v.cmd.Flags().Bool("ignore-typos",
		false,
		"do not check typos in .po file")
	v.cmd.Flags().Bool("report-typos-as-errors",
		false,
		"consider typos as errors")
	viper.BindPFlag("check--no-gpg", v.cmd.Flags().Lookup("no-gpg"))
	viper.BindPFlag("check--force", v.cmd.Flags().Lookup("force"))
	viper.BindPFlag("check--core", v.cmd.Flags().Lookup("core"))
	viper.BindPFlag("check--ignore-typos", v.cmd.Flags().Lookup("ignore-typos"))
	viper.BindPFlag("check--report-typos-as-errors", v.cmd.Flags().Lookup("report-typos-as-errors"))

	return v.cmd
}

func (v checkCommand) Execute(args []string) error {
	var err error

	// Execute in root of worktree.
	repository.ChdirProjectRoot()

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
