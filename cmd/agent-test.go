package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type agentTestCommand struct {
	cmd *cobra.Command
	O   struct {
		Agent                  string
		Runs                   int
		DangerouslyRemovePoDir bool
		Range                  string
		Commit                 string
		Since                  string
		Prompt                 string
		Output                 string
		AllWithLLM             bool
	}
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
		SilenceErrors: true,
	}

	// Add global flag for --dangerously-remove-po-directory
	v.cmd.PersistentFlags().BoolVar(&v.O.DangerouslyRemovePoDir,
		"dangerously-remove-po-directory",
		false,
		"skip confirmation prompt (dangerous: may cause data loss)")

	// Add --yes as an alias (hidden from help but functional)
	v.cmd.PersistentFlags().BoolVar(&v.O.DangerouslyRemovePoDir,
		"yes",
		false,
		"")
	_ = v.cmd.PersistentFlags().MarkHidden("yes")

	_ = viper.BindPFlag("agent-test--dangerously-remove-po-directory", v.cmd.PersistentFlags().Lookup("dangerously-remove-po-directory"))

	// Add --prompt flag to root command
	v.cmd.PersistentFlags().StringVar(&v.O.Prompt,
		"prompt",
		"",
		"override prompt from configuration (if provided, overrides the prompt in git-po-helper.yaml)")

	_ = viper.BindPFlag("agent-test--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	// Add update-pot subcommand
	updatePotCmd := &cobra.Command{
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
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			if len(args) != 0 {
				return newUserError("update-pot command needs no arguments")
			}

			return util.CmdAgentTestUpdatePot(v.O.Agent, v.O.Runs, v.O.DangerouslyRemovePoDir)
		},
	}

	updatePotCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	updatePotCmd.Flags().IntVar(&v.O.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")

	_ = viper.BindPFlag("agent-test--agent", updatePotCmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", updatePotCmd.Flags().Lookup("runs"))

	// Add update-po subcommand
	updatePoCmd := &cobra.Command{
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
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			if len(args) > 1 {
				return newUserError("update-po command expects at most one argument: po/XX.po")
			}

			poFile := ""
			if len(args) == 1 {
				poFile = args[0]
			}

			return util.CmdAgentTestUpdatePo(v.O.Agent, poFile, v.O.Runs, v.O.DangerouslyRemovePoDir)
		},
	}

	updatePoCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	updatePoCmd.Flags().IntVar(&v.O.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")

	_ = viper.BindPFlag("agent-test--agent", updatePoCmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", updatePoCmd.Flags().Lookup("runs"))

	// Add translate subcommand
	translateCmd := &cobra.Command{
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
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			if len(args) > 1 {
				return newUserError("translate command expects at most one argument: po/XX.po")
			}

			poFile := ""
			if len(args) == 1 {
				poFile = args[0]
			}

			return util.CmdAgentTestTranslate(v.O.Agent, poFile, v.O.Runs, v.O.DangerouslyRemovePoDir)
		},
	}

	translateCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	translateCmd.Flags().IntVar(&v.O.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")

	_ = viper.BindPFlag("agent-test--agent", translateCmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", translateCmd.Flags().Lookup("runs"))

	// Add review subcommand
	reviewCmd := &cobra.Command{
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
			target, err := util.ResolveRevisionsAndFiles(v.O.Range, v.O.Commit, v.O.Since, args)
			if err != nil {
				return newUserErrorF("%v", err)
			}
			return util.CmdAgentTestReview(v.O.Agent, target, v.O.Runs, v.O.DangerouslyRemovePoDir, v.O.Output, v.O.AllWithLLM)
		},
	}

	reviewCmd.Flags().BoolVar(&v.O.AllWithLLM, "all-with-llm", false,
		"use pure LLM approach: agent does extraction, review, and writes review.json")
	reviewCmd.Flags().StringVarP(&v.O.Output, "output", "o", "",
		"base path for review output files (default: po/review); .po/.json are appended")
	reviewCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")
	reviewCmd.Flags().IntVar(&v.O.Runs,
		"runs",
		0,
		"number of test runs (0 means use config file value or default to 5)")
	reviewCmd.Flags().StringVarP(&v.O.Range, "range", "r", "",
		"revision range: a..b (a and b), a.. (a and working tree), or a (a~ and a)")
	reviewCmd.Flags().StringVar(&v.O.Commit,
		"commit",
		"",
		"equivalent to -r <commit>^..<commit>")
	reviewCmd.Flags().StringVar(&v.O.Since,
		"since",
		"",
		"equivalent to -r <commit>.. (compare commit with working tree)")

	_ = viper.BindPFlag("agent-test--agent", reviewCmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-test--runs", reviewCmd.Flags().Lookup("runs"))
	_ = viper.BindPFlag("agent-test--range", reviewCmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("agent-test--output", reviewCmd.Flags().Lookup("output"))

	// Add show-config subcommand
	showConfigCmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show the current agent configuration in YAML format",
		Long: `Display the complete agent configuration in YAML format.

This command loads the configuration from git-po-helper.yaml files
(user home directory and repository root) and displays the merged
configuration in YAML format.

The configuration is read from:
- User home directory: ~/.git-po-helper.yaml (lower priority)
- Repository root: <repo-root>/git-po-helper.yaml (higher priority, overrides user config)

If no configuration files are found, an empty configuration structure
will be displayed.`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			if len(args) != 0 {
				return newUserError("show-config command needs no arguments")
			}

			return util.CmdAgentRunShowConfig()
		},
	}

	v.cmd.AddCommand(updatePotCmd)
	v.cmd.AddCommand(updatePoCmd)
	v.cmd.AddCommand(translateCmd)
	v.cmd.AddCommand(reviewCmd)
	v.cmd.AddCommand(showConfigCmd)

	return v.cmd
}

var agentTestCmd = agentTestCommand{}

func init() {
	rootCmd.AddCommand(agentTestCmd.Command())
}
