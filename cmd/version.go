package cmd

import (
	"github.com/git-l10n/git-po-helper/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Display the version of git-po-helper",
	Run:     func(cmd *cobra.Command, args []string) {},
	Version: version.Version,
}

func init() {
	versionCmd.Flags().Bool("version",
		true,
		"show version")
	versionCmd.SetVersionTemplate(`{{with .Parent.Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`)
	rootCmd.AddCommand(versionCmd)
}
