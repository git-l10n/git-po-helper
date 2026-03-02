package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentTestUpdatePotCmd(opts *agentTestOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-pot",
		Short: "Test update-pot operation multiple times and calculate average score",
		Long: `Test the update-pot operation multiple times and calculate an average score.

This command runs agent-run update-pot multiple times (default: 5, configurable
via --runs or config file) and provides detailed results including:
- Individual run results with validation status
- Success/failure counts
- Average score across all runs
- Entry count validation results (if configured)

Validation can be configured in git-po-helper.yaml:
- pot_entries_before_update: Expected entry count before update
- pot_entries_after_update: Expected entry count after update

If validation is configured:
- Pre-validation failure: Run is marked as failed (score = 0), agent is not executed
- Post-validation failure: Run is marked as failed (score = 0) even if agent succeeded
- Both validations pass: Run is marked as successful (score = 100)

If validation is not configured (null or 0), scoring is based on agent exit code:
- Agent succeeds (exit code 0): score = 100
- Agent fails (non-zero exit code): score = 0

Examples:
  # Run 5 tests with default agent
  git-po-helper agent-test update-pot

  # Run 10 tests with a specific agent
  git-po-helper agent-test update-pot --agent claude --runs 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewErrorWithUsage("update-pot command needs no arguments")
			}

			if err := util.CmdAgentTestUpdatePot(opts.Agent, opts.Runs, opts.DangerouslyRemovePoDir); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	cmd.Flags().IntVar(&opts.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")

	_ = viper.BindPFlag("agent-test--agent", cmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", cmd.Flags().Lookup("runs"))

	return cmd
}
