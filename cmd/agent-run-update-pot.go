package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentRunUpdatePotCmd(opts *agentRunOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-pot",
		Short: "Update po/git.pot using an agent",
		Long: `Update the po/git.pot template file using a configured agent.

This command uses an agent with a configured prompt to update the po/git.pot
file according to po/README.md and po/AGENTS.md. The agent command is specified
in the git-po-helper.yaml configuration file.

You must run this from inside a Git work tree. The command opens the repository
from the current directory and uses the work tree root, then requires at that
root: Makefile, po/, po/README.md, and po/AGENTS.md (upstream Git po/ layout).
The process working directory is switched to that root for the run.

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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewErrorWithUsage("update-pot command needs no arguments")
			}

			if err := util.CmdAgentRunUpdatePot(opts.Agent); err != nil {
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
