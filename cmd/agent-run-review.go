package cmd

import (
	"strings"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentRunReviewCmd(opts *agentRunOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review XX.po",
		Short: "Review translations in a po/XX.po file using an agent",
		Long: `Review translations in a PO file using a configured agent.

This command uses an agent with a configured review prompt to analyze
translations in a PO file. You can review local changes, a single commit,
or all changes since a given commit.

If only one agent is configured, the --agent flag is optional. If multiple
agents are configured, you must specify which agent to use with --agent.

Review modes:
- --range a..b: compare commit a with commit b
- --range a..: compare commit a with working tree
- --commit <commit>: review the changes in the specified commit
- --since <commit>: review changes since the specified commit
- --report <dir>: print review report under the specified directory
- no --range/--commit/--since: review changes since HEAD (local changes)

Exactly one of --range, --commit and --since may be specified.

When --report is used, it cannot be combined with --since, --range,
--output, or --commit, and behaves the same as "agent-run report".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.Report) != "" {
				if strings.TrimSpace(opts.Since) != "" ||
					strings.TrimSpace(opts.Range) != "" ||
					strings.TrimSpace(opts.Output) != "" ||
					strings.TrimSpace(opts.Commit) != "" {
					return NewErrorWithUsage("--report cannot be used with --since, --range, --output, or --commit")
				}
				if len(args) != 0 {
					return NewErrorWithUsage("review command with --report does not accept positional arguments")
				}
				result, err := util.GetReviewReport(opts.Report)
				if err != nil {
					return NewStandardErrorF("%v", err)
				}
				util.PrintReviewReportResult(result)
				return nil
			}

			if len(args) != 1 {
				return NewErrorWithUsage("review command expects exactly one argument: XX.po")
			}

			target, err := util.ResolveRevisionsAndFiles(opts.Range, opts.Commit, opts.Since, args)
			if err != nil {
				return NewStandardErrorF("%v", err)
			}
			if err := util.CmdAgentRunReview(opts.Agent, target, opts.UseLocalOrchestration, opts.BatchSize); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.UseLocalOrchestration, "use-local-orchestration", false,
		"use local orchestration: agent only reviews batch JSON files")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "",
		"base path for review output files (default: po/review); .po/.json are appended")
	cmd.Flags().StringVar(&opts.Report, "report", "",
		"print review report from the specified directory (same as agent-run report <dir>)")
	cmd.Flags().StringVar(&opts.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	cmd.Flags().IntVar(&opts.BatchSize, "batch-size", 100,
		"min entries per batch when splitting review (default: 100)")
	cmd.Flags().StringVarP(&opts.Range, "range", "r", "",
		"revision range: a..b (a and b), a.. (a and working tree), or a (a~ and a)")
	cmd.Flags().StringVar(&opts.Commit,
		"commit",
		"",
		"equivalent to -r <commit>^..<commit>")
	cmd.Flags().StringVar(&opts.Since,
		"since",
		"",
		"equivalent to -r <commit>.. (compare commit with working tree)")

	_ = viper.BindPFlag("agent-run--agent", cmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-run--batch-size", cmd.Flags().Lookup("batch-size"))
	_ = viper.BindPFlag("agent-run--range", cmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("agent-run--output", cmd.Flags().Lookup("output"))

	return cmd
}
