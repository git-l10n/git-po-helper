package cmd

import (
	"strings"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type agentTestOptions struct {
	Agent                  string
	Runs                   int
	DangerouslyRemovePoDir bool
	Range                  string
	Commit                 string
	Since                  string
	Prompt                 string
	Output                 string
	UseAgentMd             bool
	UseLocalOrchestration  bool
	BatchSize              int
}

type agentTestCommand struct {
	cmd *cobra.Command
	O   agentTestOptions
}

func (v *agentTestCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "agent-test",
		Short: "Test agent commands with multiple runs",
		Long: `Test agent commands with multiple runs and calculate average scores.

This command runs agent operations multiple times to test reliability and
performance. It calculates an average score where success = 100 points and
failure = 0 points.

The number of runs can be specified via --runs flag or configured in
git-po-helper.yaml. If not specified, the default is 3 runs.

Without a subcommand, -p/--prompt runs the agent that many times with that
prompt text (direct execution). Subcommands use --prompt as an override for
the configured workflow prompt.

Entry count validation can be configured to verify that the agent correctly
updates files with the expected number of entries.`,
		RunE: v.runRoot,
	}

	v.cmd.PersistentFlags().BoolVar(&v.O.DangerouslyRemovePoDir,
		"dangerously-remove-po-directory",
		false,
		"skip confirmation prompt (dangerous: may cause data loss)")

	v.cmd.PersistentFlags().BoolVar(&v.O.DangerouslyRemovePoDir,
		"yes",
		false,
		"")
	_ = v.cmd.PersistentFlags().MarkHidden("yes")

	_ = viper.BindPFlag("agent-test--dangerously-remove-po-directory", v.cmd.PersistentFlags().Lookup("dangerously-remove-po-directory"))

	v.cmd.PersistentFlags().StringVarP(&v.O.Prompt,
		"prompt",
		"p",
		"",
		"prompt: for direct run (no subcommand), text passed to the agent each run; with a subcommand, overrides the prompt from git-po-helper.yaml")

	_ = viper.BindPFlag("agent-test--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	v.cmd.PersistentFlags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-test--agent", v.cmd.PersistentFlags().Lookup("agent"))

	v.cmd.PersistentFlags().IntVar(&v.O.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 3)")

	_ = viper.BindPFlag("agent-test--runs", v.cmd.PersistentFlags().Lookup("runs"))

	v.cmd.AddCommand(newAgentTestUpdatePotCmd(&v.O))
	v.cmd.AddCommand(newAgentTestUpdatePoCmd(&v.O))
	v.cmd.AddCommand(newAgentTestTranslateCmd(&v.O))
	v.cmd.AddCommand(newAgentTestReviewCmd(&v.O))

	return v.cmd
}

func (v *agentTestCommand) runRoot(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return NewErrorWithUsageF("unknown argument %q (use a subcommand or -p/--prompt for direct run)", args[0])
	}
	if strings.TrimSpace(v.O.Prompt) == "" {
		if err := cmd.Help(); err != nil {
			return NewStandardErrorF("%v", err)
		}
		return nil
	}
	if err := util.CmdAgentTestDirect(v.O.Agent, v.O.Prompt, v.O.Runs); err != nil {
		return NewStandardErrorF("%v", err)
	}
	return nil
}

var agentTestCmd = agentTestCommand{}

func init() {
	rootCmd.AddCommand(agentTestCmd.Command())
}
