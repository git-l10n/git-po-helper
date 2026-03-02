package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentRunTranslateCmd(opts *agentRunOptions) *cobra.Command {
	cmd := &cobra.Command{
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

			if err := util.CmdAgentRunTranslate(opts.Agent, poFile, !useLocalOrchestration, useLocalOrchestration, opts.BatchSize); err != nil {
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

	_ = viper.BindPFlag("agent-run--agent", cmd.Flags().Lookup("agent"))
	_ = viper.BindPFlag("agent-run--batch-size", cmd.Flags().Lookup("batch-size"))

	return cmd
}
