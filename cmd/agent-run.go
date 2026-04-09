package cmd

import (
	"strings"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type agentRunOptions struct {
	Agent                 string
	Range                 string
	Commit                string
	Since                 string
	Report                string
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
read from git-po-helper.yaml in the repository root or user home directory.

Without a subcommand, -p/--prompt runs the agent once with that prompt text
(direct execution). Subcommands (update-pot, update-po, translate, review)
use --prompt as an override for the configured workflow prompt.

Examples:
  git-po-helper agent-run -p "your instruction"
  git-po-helper agent-run --agent claude update-pot`,
		RunE: v.runRoot,
	}

	v.cmd.PersistentFlags().StringVarP(&v.O.Prompt,
		"prompt",
		"p",
		"",
		"prompt: for direct run (no subcommand), text passed to the agent; with a subcommand, overrides the prompt from git-po-helper.yaml")

	_ = viper.BindPFlag("agent-run--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	v.cmd.PersistentFlags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-run--agent", v.cmd.PersistentFlags().Lookup("agent"))

	v.cmd.AddCommand(newAgentRunUpdatePotCmd(&v.O))
	v.cmd.AddCommand(newAgentRunUpdatePoCmd(&v.O))
	v.cmd.AddCommand(newAgentRunTranslateCmd(&v.O))
	v.cmd.AddCommand(newAgentRunReviewCmd(&v.O))
	v.cmd.AddCommand(newAgentRunReportCmd())
	v.cmd.AddCommand(newAgentRunParseLogCmd())

	return v.cmd
}

func (v *agentRunCommand) runRoot(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return NewErrorWithUsageF("unknown argument %q (use a subcommand or -p/--prompt for direct run)", args[0])
	}
	if strings.TrimSpace(v.O.Prompt) == "" {
		if err := cmd.Help(); err != nil {
			return NewStandardErrorF("%v", err)
		}
		return nil
	}
	if err := util.CmdAgentRunDirect(v.O.Agent, v.O.Prompt); err != nil {
		return NewStandardErrorF("%v", err)
	}
	return nil
}

var agentRunCmd = agentRunCommand{}

func init() {
	rootCmd.AddCommand(agentRunCmd.Command())
}
