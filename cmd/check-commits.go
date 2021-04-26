package cmd

import (
	"errors"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type checkCommitsCommand struct {
	cmd *cobra.Command
	O   struct {
		NoGPG bool
	}
}

func (v *checkCommitsCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "check-commits [rev-list range...]",
		Short:         "Check commits for l10n conventions",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolVar(&v.O.NoGPG,
		"no-gpg",
		false,
		"no gpg verify")

	return v.cmd
}

func (v checkCommitsCommand) Execute(args []string) error {
	var ret bool
	if len(args) == 0 {
		ret = util.CmdCheckCommits("HEAD@{u}..HEAD")
	} else {
		ret = util.CmdCheckCommits(args...)
	}
	if !ret {
		return errors.New("fail to check commits")
	}
	return nil
}

var checkCommitsCmd = checkCommitsCommand{}

func init() {
	rootCmd.AddCommand(checkCommitsCmd.Command())
}
