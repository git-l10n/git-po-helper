package cmd

import (
	"fmt"

	"github.com/git-l10n/git-po-helper/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of git-po-helper",
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func showVersion() {
	fmt.Printf("git-po-helper version %s\n", version.Version)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
