package cmd

import (
	"path/filepath"

	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type diffCommand struct {
	cmd *cobra.Command
	O   struct {
		Revisions []string
	}
}

func (v *diffCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:           "diff [-r revision [-r revision]] [[<src>] <target>]",
		Short:         "Show changes between two l10n files",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().StringArrayVarP(&v.O.Revisions, "revision",
		"r",
		nil,
		"revision to compare( default HEAD)")

	return v.cmd
}

func (v diffCommand) Execute(args []string) error {
	var (
		src, dest util.FileRevision
	)

	if len(v.O.Revisions) > 2 {
		return newUserErrorF("too many revisions (%d > 2)", len(v.O.Revisions))
	}
	if len(args) > 2 {
		return newUserErrorF("too many arguments (%d > 2)", len(args))
	}
	// Set Revision
	switch len(args) {
	case 0:
		fallthrough
	case 1:
		switch len(v.O.Revisions) {
		case 0:
			src.Revision = "HEAD"
		case 1:
			src.Revision = v.O.Revisions[0]
			dest.Revision = ""
		case 2:
			src.Revision = v.O.Revisions[0]
			dest.Revision = v.O.Revisions[1]
		}
	case 2:
		switch len(v.O.Revisions) {
		case 1:
			src.Revision = v.O.Revisions[0]
			dest.Revision = v.O.Revisions[0]
		case 2:
			src.Revision = v.O.Revisions[0]
			dest.Revision = v.O.Revisions[1]
		}
	}
	// Set File
	switch len(args) {
	case 0:
		src.File = filepath.Join("po", util.GitPot)
		dest.File = filepath.Join("po", util.GitPot)
	case 1:
		src.File = args[0]
		dest.File = args[0]
	case 2:
		src.File = args[0]
		dest.File = args[1]
	}
	if !util.DiffFileRevision(src, dest) {
		return errExecute
	}
	return nil
}

var diffCmd = diffCommand{}

func init() {
	rootCmd.AddCommand(diffCmd.Command())
}
