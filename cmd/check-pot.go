package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type checkPotCommand struct {
	OptShowAllConfigs       bool
	OptShowCamelCaseConfigs bool

	cmd *cobra.Command
}

func (v *checkPotCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "check-pot <XX.po>...",
		Short: "Check syntax of XX.po file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolVar(&v.OptShowCamelCaseConfigs,
		"show-camel-case-configs",
		false,
		"show CamelCase config variables in config manpage")
	v.cmd.Flags().BoolVar(&v.OptShowAllConfigs,
		"show-all-configs",
		false,
		"show all config variables in config manpage")

	return v.cmd
}

func (v checkPotCommand) Execute(args []string) error {
	n := 0
	if v.OptShowAllConfigs {
		n++
	}
	if v.OptShowCamelCaseConfigs {
		n++
	}
	if n > 1 {
		return NewErrorWithUsage("cannot use --show-all-configs and --show-camel-case-configs at the same time")
	}

	if v.OptShowAllConfigs {
		return util.ShowManpageConfigs(false)
	}
	if v.OptShowCamelCaseConfigs {
		return util.ShowManpageConfigs(true)
	}

	if err := util.CheckCamelCaseConfigVariableInPotFile(); err != nil {
		return NewStandardErrorF("%v", err)
	}
	return nil
}

var checkPotCmd = checkPotCommand{}

func init() {
	rootCmd.AddCommand(checkPotCmd.Command())
}
