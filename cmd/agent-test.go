package cmd

import (
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
git-po-helper.yaml. If not specified, the default is 5 runs.

Entry count validation can be configured to verify that the agent correctly
updates files with the expected number of entries.`,
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

	v.cmd.PersistentFlags().StringVar(&v.O.Prompt,
		"prompt",
		"",
		"override prompt from configuration (if provided, overrides the prompt in git-po-helper.yaml)")

	_ = viper.BindPFlag("agent-test--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	v.cmd.AddCommand(newAgentTestUpdatePotCmd(&v.O))
	v.cmd.AddCommand(newAgentTestUpdatePoCmd(&v.O))
	v.cmd.AddCommand(newAgentTestTranslateCmd(&v.O))
	v.cmd.AddCommand(newAgentTestReviewCmd(&v.O))
	v.cmd.AddCommand(newAgentTestShowConfigCmd())

	return v.cmd
}

var agentTestCmd = agentTestCommand{}

func init() {
	rootCmd.AddCommand(agentTestCmd.Command())
}
