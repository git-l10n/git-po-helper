package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type teamCommand struct {
	cmd *cobra.Command
}

func (v *teamCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "team [--leader | --all] [team]...",
		Short:         "Show team leader/members",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolP("leader",
		"l",
		false,
		"show leader")
	v.cmd.Flags().BoolP("members",
		"m",
		false,
		"show all users")
	v.cmd.Flags().BoolP("check",
		"c",
		false,
		"show all users")
	_ = viper.BindPFlag("team-leader", v.cmd.Flags().Lookup("leader"))
	_ = viper.BindPFlag("team-members", v.cmd.Flags().Lookup("members"))
	_ = viper.BindPFlag("team-check", v.cmd.Flags().Lookup("check"))
	return v.cmd
}

func (v teamCommand) Execute(args []string) error {
	// Execute in root of worktree.
	repository.ChdirProjectRoot()

	if !util.ShowTeams(args...) {
		return errExecute
	}
	return nil
}

var teamCmd = teamCommand{}

func init() {
	rootCmd.AddCommand(teamCmd.Command())
}
