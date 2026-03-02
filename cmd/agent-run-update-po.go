package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentRunUpdatePoCmd(opts *agentRunOptions) *cobra.Command {
	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return NewErrorWithUsage("update-po command expects at most one argument: po/XX.po")
			}

			poFile := ""
			if len(args) == 1 {
				poFile = args[0]
			}

			if err := util.CmdAgentRunUpdatePo(opts.Agent, poFile); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Agent,
		"agent",
		"",
		"agent name to use (required if multiple agents are configured)")

	_ = viper.BindPFlag("agent-run--agent", cmd.Flags().Lookup("agent"))

	return cmd
}
