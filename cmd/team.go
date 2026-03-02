package cmd

import (
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
		Use:   "team [--leader | --all] [team]...",
		Short: "Show team leader/members",
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolP("leader",
		"l",
		false,
		"show leaders only")
	v.cmd.Flags().BoolP("members",
		"m",
		false,
		"show members only")
	v.cmd.Flags().BoolP("all",
		"a",
		false,
		"show all users")
	v.cmd.Flags().BoolP("language",
		"L",
		false,
		"show language")
	v.cmd.Flags().BoolP("check",
		"c",
		false,
		"check team members")
	_ = viper.BindPFlag("team-leader", v.cmd.Flags().Lookup("leader"))
	_ = viper.BindPFlag("team-members", v.cmd.Flags().Lookup("members"))
	_ = viper.BindPFlag("all-team-members", v.cmd.Flags().Lookup("all"))
	_ = viper.BindPFlag("show-language", v.cmd.Flags().Lookup("language"))
	_ = viper.BindPFlag("team-check", v.cmd.Flags().Lookup("check"))
	return v.cmd
}

func (v teamCommand) Execute(args []string) error {
	if !util.ShowTeams(args...) {
		return NewStandardError("team command failed")
	}
	return nil
}

var teamCmd = teamCommand{}

func init() {
	rootCmd.AddCommand(teamCmd.Command())
}
