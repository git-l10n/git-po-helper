package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type agentRunOptions struct {
	Agent                 string
	Range                 string
	Commit                string
	Since                 string
	Prompt                string
	Output                string
	UseAgentMd            bool
	UseLocalOrchestration bool
	BatchSize             int
}

type agentRunCommand struct {
	cmd *cobra.Command
	O   agentRunOptions
}

func (v *agentRunCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "agent-run",
		Short: "Run agent commands for automation",
		Long: `Run agent commands for automating localization tasks.

This command uses configured code agents (like Claude, Gemini, etc.) to
automate various localization operations. The agent configuration is
read from git-po-helper.yaml in the repository root or user home directory.`,
	}

	v.cmd.PersistentFlags().StringVar(&v.O.Prompt,
		"prompt",
		"",
		"override prompt from configuration (if provided, overrides the prompt in git-po-helper.yaml)")

	_ = viper.BindPFlag("agent-run--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	v.cmd.AddCommand(newAgentRunUpdatePotCmd(&v.O))
	v.cmd.AddCommand(newAgentRunUpdatePoCmd(&v.O))
	v.cmd.AddCommand(newAgentRunTranslateCmd(&v.O))
	v.cmd.AddCommand(newAgentRunReviewCmd(&v.O))
	v.cmd.AddCommand(newAgentRunReportCmd())
	v.cmd.AddCommand(newAgentRunParseLogCmd())
	v.cmd.AddCommand(newAgentRunShowConfigCmd())

	return v.cmd
}

var agentRunCmd = agentRunCommand{}

func init() {
	rootCmd.AddCommand(agentRunCmd.Command())
}
