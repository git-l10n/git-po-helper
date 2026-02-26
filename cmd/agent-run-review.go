package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentRunReviewCmd(opts *agentRunOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review [-r range | --commit <commit> | --since <commit>] [[<src>] <target>]",
		Short: "Review translations in a po/XX.po file using an agent",
		Long: `Review translations in a PO file using a configured agent.

This command uses an agent with a configured review prompt to analyze
translations in a PO file. You can review local changes, a single commit,
or all changes since a given commit.

If only one agent is configured, the --agent flag is optional. If multiple
agents are configured, you must specify which agent to use with --agent.

If no po/XX.po argument is given, the PO file is derived from changed files
or default_lang_code in configuration.

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
			if err := util.CmdAgentRunReview(opts.Agent, target, opts.Output, opts.AllWithLLM); err != nil {
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
	_ = viper.BindPFlag("agent-run--range", cmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("agent-run--output", cmd.Flags().Lookup("output"))

	return cmd
}
