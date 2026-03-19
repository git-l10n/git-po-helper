package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

// configCmd represents the top-level config command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the full configuration (agent + projects) in YAML format",
	Long: `Display the complete configuration in YAML format.

This command loads the configuration from .git-po-helper.yaml files
(user home directory and repository root, or --config path) and
displays the merged configuration including agent settings and
POT project settings.

The configuration is read from:
- User home directory: ~/.git-po-helper.yaml (lower priority)
- Repository root: <repo-root>/.git-po-helper.yaml (higher priority)
- Or --config <path> when specified (overrides the above)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return NewErrorWithUsage("config command needs no arguments")
		}

		if err := util.CmdShowConfig(); err != nil {
			return NewStandardErrorF("%v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
