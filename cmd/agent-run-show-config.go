package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentRunShowConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return NewErrorWithUsage("show-config command needs no arguments")
			}

			if err := util.CmdAgentRunShowConfig(); err != nil {
				return NewStandardErrorF("%v", err)
			}
			return nil
		},
	}

	return cmd
}
