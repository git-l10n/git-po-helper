package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentTestReviewCmd(opts *agentTestOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review [-r range | --commit <commit> | --since <commit>] [[<src>] <target>]",
		Short: "Test review operation multiple times and calculate average score",
		Long: `Test the review operation multiple times and calculate an average score.

This command runs agent-run review multiple times (default: 5, configurable
via --runs or config file) and provides detailed results including:
- Individual run results with validation status
- Success/failure counts
- Average score across all runs

Review modes:
- --range a..b: compare commit a with commit b
- --range a..: compare commit a with working tree
- --commit <commit>: review the changes in the specified commit
- --since <commit>: review changes since the specified commit
- no --range/--commit/--since: review changes since HEAD (local changes)

Exactly one of --range, --commit and --since may be specified.
With two file arguments, compare worktree files (revisions not allowed).`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository.ChdirProjectRoot()

			target, err := util.ResolveRevisionsAndFiles(opts.Range, opts.Commit, opts.Since, args)
			if err != nil {
				return newUserErrorF("%v", err)
			}
			if err := util.CmdAgentTestReview(opts.Agent, target, opts.Runs, opts.DangerouslyRemovePoDir, opts.Output, opts.AllWithLLM); err != nil {
				return errExecute
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.AllWithLLM, "all-with-llm", false,
		"use pure LLM approach: agent does extraction, review, and writes review.json")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "",
		"base path for review output files (default: po/review); .po/.json are appended")
	cmd.Flags().StringVar(&opts.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	cmd.Flags().IntVar(&opts.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")
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

	_ = viper.BindPFlag("agent-test--agent", cmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", cmd.Flags().Lookup("runs"))
	_ = viper.BindPFlag("agent-test--range", cmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("agent-test--output", cmd.Flags().Lookup("output"))

	return cmd
}
