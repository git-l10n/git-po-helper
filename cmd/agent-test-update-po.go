package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentTestUpdatePoCmd(opts *agentTestOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-po [po/XX.po]",
		Short: "Test update-po operation multiple times and calculate average score",
		Long: `Test the update-po operation multiple times and calculate an average score.

This command runs agent-run update-po multiple times (default: 5, configurable
via --runs or config file) and provides detailed results including:
- Individual run results with validation status
- Success/failure counts
- Average score across all runs
- Entry count validation results (if configured)

Validation can be configured in git-po-helper.yaml:
- po_entries_before_update: Expected entry count before update
- po_entries_after_update: Expected entry count after update

If validation is configured:
- Pre-validation failure: Run is marked as failed (score = 0), agent is not executed
- Post-validation failure: Run is marked as failed (score = 0) even if agent succeeded
- Both validations pass: Run is marked as successful (score = 100)

If validation is not configured (null or 0), scoring is based on agent exit code:
- Agent succeeds (exit code 0): score = 100
- Agent fails (non-zero exit code): score = 0

If no po/XX.po argument is given, the PO file is derived from
default_lang_code in configuration (e.g., po/zh_CN.po).

Examples:
  # Run 5 tests using default_lang_code to locate PO file
  git-po-helper agent-test update-po

  # Run 5 tests for a specific PO file
  git-po-helper agent-test update-po po/zh_CN.po

  # Run 10 tests with a specific agent
  git-po-helper agent-test update-po --agent claude --runs 10 po/zh_CN.po`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return NewErrorWithUsage("update-po command expects at most one argument: po/XX.po")
			}

			poFile := ""
			if len(args) == 1 {
				poFile = args[0]
			}

			if err := util.CmdAgentTestUpdatePo(opts.Agent, poFile, opts.Runs, opts.DangerouslyRemovePoDir); err != nil {
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
