package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type agentRunCommand struct {
	cmd *cobra.Command
	O   struct {
		Agent      string
		Range      string
		Commit     string
		Since      string
		Prompt     string
		Output     string
		AllWithLLM bool
	}
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
read from git-po-helper.yaml in the repository root or user home directory.`,
		SilenceErrors: true,
	}

	// Add --prompt flag to root command
	v.cmd.PersistentFlags().StringVar(&v.O.Prompt,
		"prompt",
		"",
		"override prompt from configuration (if provided, overrides the prompt in git-po-helper.yaml)")

	_ = viper.BindPFlag("agent-run--prompt", v.cmd.PersistentFlags().Lookup("prompt"))

	// Add update-pot subcommand
	updatePotCmd := &cobra.Command{
		Use:   "update-pot",
		Short: "Update po/git.pot using an agent",
		Long: `Update the po/git.pot template file using a configured agent.

This command uses an agent with a configured prompt to update the po/git.pot
file according to po/README.md. The agent command is specified in the
git-po-helper.yaml configuration file.

If only one agent is configured, the --agent flag is optional. If multiple
agents are configured, you must specify which agent to use with --agent.

The command performs validation checks if configured:
- Pre-validation: checks entry count before update (if pot_entries_before_update is set)
- Post-validation: checks entry count after update (if pot_entries_after_update is set)
- Syntax validation: validates the POT file using msgfmt

Examples:
  # Use the default agent (if only one is configured)
  git-po-helper agent-run update-pot

  # Use a specific agent
  git-po-helper agent-run update-pot --agent claude`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			if len(args) != 0 {
				return newUserError("update-pot command needs no arguments")
			}

			return util.CmdAgentRunUpdatePot(v.O.Agent)
		},
	}

	updatePotCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-run--agent", updatePotCmd.Flags().Lookup("agent"))

	// Add update-po subcommand
	updatePoCmd := &cobra.Command{
		Use:   "update-po [po/XX.po]",
		Short: "Update a po/XX.po file using an agent",
		Long: `Update a specific po/XX.po file using a configured agent.

This command uses an agent with a configured prompt to update the target
PO file according to po/README.md. The agent command and prompt are
specified in the git-po-helper.yaml configuration file.

If only one agent is configured, the --agent flag is optional. If multiple
agents are configured, you must specify which agent to use with --agent.

If no po/XX.po argument is given, the PO file is derived from
default_lang_code in configuration (e.g., po/zh_CN.po).

The command performs validation checks if configured:
- Pre-validation: checks entry count before update (if po_entries_before_update is set)
- Post-validation: checks entry count after update (if po_entries_after_update is set)
- Syntax validation: validates the PO file using msgfmt

Examples:
  # Use default_lang_code to locate PO file
  git-po-helper agent-run update-po

  # Explicitly specify the PO file
  git-po-helper agent-run update-po po/zh_CN.po

  # Use a specific agent
  git-po-helper agent-run update-po --agent claude po/zh_CN.po`,
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

			return util.CmdAgentRunUpdatePo(v.O.Agent, poFile)
		},
	}

	updatePoCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-run--agent", updatePoCmd.Flags().Lookup("agent"))

	// Add translate subcommand
	translateCmd := &cobra.Command{
		Use:   "translate [po/XX.po]",
		Short: "Translate new and fuzzy entries in a po/XX.po file using an agent",
		Long: `Translate new strings and fix fuzzy translations in a PO file using a configured agent.

This command uses an agent with a configured prompt to translate all untranslated
entries (new strings) and resolve all fuzzy entries in the target PO file.
The agent command and prompt are specified in the git-po-helper.yaml configuration file.

If only one agent is configured, the --agent flag is optional. If multiple
agents are configured, you must specify which agent to use with --agent.

If no po/XX.po argument is given, the PO file is derived from
default_lang_code in configuration (e.g., po/zh_CN.po).

The command performs validation checks:
- Pre-validation: counts new (untranslated) and fuzzy entries before translation
- Post-validation: verifies all new and fuzzy entries are resolved (count must be 0)
- Syntax validation: validates the PO file using msgfmt

The operation is considered successful only if both new entry count and
fuzzy entry count are 0 after translation.

Examples:
  # Use default_lang_code to locate PO file
  git-po-helper agent-run translate

  # Explicitly specify the PO file
  git-po-helper agent-run translate po/zh_CN.po

  # Use a specific agent
  git-po-helper agent-run translate --agent claude po/zh_CN.po`,
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

			return util.CmdAgentRunTranslate(v.O.Agent, poFile)
		},
	}

	translateCmd.Flags().StringVar(&v.O.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-run--agent", translateCmd.Flags().Lookup("agent"))

	// Add review subcommand
	reviewCmd := &cobra.Command{
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
			// Execute in root of worktree.
			repository.ChdirProjectRoot()

			target, err := util.ResolveRevisionsAndFiles(v.O.Range, v.O.Commit, v.O.Since, args)
			if err != nil {
				return newUserErrorF("%v", err)
			}
			return util.CmdAgentRunReview(v.O.Agent, target, v.O.Output, v.O.AllWithLLM)
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

	_ = viper.BindPFlag("agent-run--agent", reviewCmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-run--range", reviewCmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("agent-run--output", reviewCmd.Flags().Lookup("output"))

	// Add parse-log subcommand
	parseLogCmd := &cobra.Command{
		Use:   "parse-log [log-file]",
		Short: "Parse agent JSONL log file and display formatted output",
		Long: `Parse a Claude or Qwen/Gemini agent JSONL log file (one JSON object per line).
Auto-detects format and displays with type-specific icons:
- 🤔 thinking content
- 🔧 tool_use content (tool name and input)
- 🤖 text content
- 💬 user/tool_result (raw size)

If no log file is specified, defaults to /tmp/claude.log.jsonl.

Examples:
  git-po-helper agent-run parse-log
  git-po-helper agent-run parse-log /tmp/claude.log.jsonl
  git-po-helper agent-run parse-log /tmp/qwen.log.jsonl`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logFile := "/tmp/claude.log.jsonl"
			if len(args) > 0 {
				logFile = args[0]
			}
			if len(args) > 1 {
				return newUserError("parse-log expects at most one argument: log-file")
			}
			return util.CmdAgentRunParseLog(logFile)
		},
	}

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
	v.cmd.AddCommand(parseLogCmd)
	v.cmd.AddCommand(showConfigCmd)

	return v.cmd
}

var agentRunCmd = agentRunCommand{}

func init() {
	rootCmd.AddCommand(agentRunCmd.Command())
}
