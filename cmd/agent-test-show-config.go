package cmd

import (
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentTestShowConfigCmd() *cobra.Command {
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
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository.ChdirProjectRoot()

			if len(args) != 0 {
				return newUserError("show-config command needs no arguments")
			}

			if err := util.CmdAgentRunShowConfig(); err != nil {
				return errExecute
			}
			return nil
		},
	}

	return cmd
}
