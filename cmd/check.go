package cmd

import (
	"errors"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
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

	return v.cmd
}

func (v checkCommand) Execute(args []string) error {
	var err error

	if !util.CmdCheckPo() {
		err = errors.New("fail to check po")
	}
	if !util.CmdCheckCommits() {
		err = errors.New("fail to check commits")
	}
	return err
}

var checkCmd = checkCommand{}

func init() {
	rootCmd.AddCommand(checkCmd.Command())
}
