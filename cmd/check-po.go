package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type checkPoCommand struct {
	cmd *cobra.Command
	O   struct {
		CheckCore bool
	}
}

func (v *checkPoCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "check-po <XX.po>...",
		Short:         "Check syntax of XX.po file",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolVar(&v.O.CheckCore,
		"core",
		false,
		"also check against "+util.CorePot)

	return v.cmd
}

func (v checkPoCommand) Execute(args []string) error {
	var errMsgs []string

	if len(args) == 0 {
		return newUserError("no argument for check-po command")
	}
	for _, locale := range args {
		if !util.CmdCheckPo(locale, v.O.CheckCore) {
			errMsgs = append(errMsgs, fmt.Sprintf(`fail to check "%s"`, locale))
		}
	}
	if len(errMsgs) > 0 {
		return errors.New(strings.Join(errMsgs, "\n"))
	}
	return nil
}

var checkPoCmd = checkPoCommand{}

func init() {
	rootCmd.AddCommand(checkPoCmd.Command())
}
