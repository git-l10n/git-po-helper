package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentTestTranslateCmd(opts *agentTestOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "translate [po/XX.po]",
		Short: "Test translate operation multiple times and calculate average score",
		Long: `Test the translate operation multiple times and calculate an average score.

This command runs agent-run translate multiple times (default: 5, configurable
via --runs or config file) and provides detailed results including:
- Individual run results with validation status
- Success/failure counts
- Average score across all runs
- New and fuzzy entry counts before and after translation

Validation logic:
- Translation is considered successful only if both new entries and fuzzy
  entries are reduced to 0 after translation
- If new entries or fuzzy entries remain: score = 0
- If both are 0: score = 100

If no po/XX.po argument is given, the PO file is derived from
default_lang_code in configuration (e.g., po/zh_CN.po).

Examples:
  # Run 5 tests using default_lang_code to locate PO file
  git-po-helper agent-test translate

  # Run 5 tests for a specific PO file
  git-po-helper agent-test translate po/zh_CN.po

  # Run 10 tests with a specific agent
  git-po-helper agent-test translate --agent claude --runs 10 po/zh_CN.po`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return NewErrorWithUsage("translate command expects at most one argument: po/XX.po")
			}
			if opts.UseAgentMd && opts.UseLocalOrchestration {
				return NewErrorWithUsage("--use-agent-md and --use-local-orchestration are mutually exclusive")
			}
			// When neither specified, default to agent-md
			useLocalOrchestration := opts.UseLocalOrchestration

			poFile := ""
			if len(args) == 1 {
				poFile = args[0]
			}

			if err := util.CmdAgentTestTranslate(opts.Agent, poFile, opts.Runs, opts.DangerouslyRemovePoDir, useLocalOrchestration, opts.BatchSize); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.UseAgentMd, "use-agent-md", false,
		"use agent with po/AGENTS.md: agent receives full/extracted PO (default)")
	cmd.Flags().BoolVar(&opts.UseLocalOrchestration, "use-local-orchestration", false,
		"use local orchestration: agent only translates batch JSON files")
	cmd.Flags().IntVar(&opts.BatchSize, "batch-size", 50,
		"min entries per batch when using --use-local-orchestration (default: 50)")
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
	_ = viper.BindPFlag("agent-test--batch-size", cmd.Flags().Lookup("batch-size"))

	return cmd
}
