package cmd

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/version"
	"github.com/spf13/cobra"
)

type versionCommand struct {
	cmd *cobra.Command
	O   struct {
		LT string
		LE string
		GT string
		GE string
		EQ string
	}
}

func (v *versionCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "version",
		Short: "Display the version of git-po-helper",
		Long: `Display the version of git-po-helper on stdout.

Optional comparison flags check the running version against a constraint.
Constraint may be major only (e.g. 1), major.minor (e.g. 0.8), or
major.minor.patch (e.g. 0.8.4). Extra constraint components are ignored.

The running version must be at least major.minor.patch (release tags must
look like v1.0.0, not v1.0). Only those three components are compared;
git describe extras (commit distance, g<oid>, dirty) are ignored.

- --eq compares at the constraint's precision (0.8.4 satisfies --eq 0.8).
- --lt/--le/--gt/--ge pad the shorter side with zeros (0.8.4 satisfies --gt 0.8).

If the condition is not satisfied, the version is still printed on stdout,
an ERROR line is also printed on stdout, and the process exits with status -1.

At most one of --lt, --le, --gt, --ge, --eq may be given.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}

	fs := v.cmd.Flags()
	fs.StringVar(&v.O.LT, "lt", "", "require version < given version")
	fs.StringVar(&v.O.LE, "le", "", "require version <= given version")
	fs.StringVar(&v.O.GT, "gt", "", "require version > given version")
	fs.StringVar(&v.O.GE, "ge", "", "require version >= given version")
	fs.StringVar(&v.O.EQ, "eq", "", "require version equal to given version (at constraint precision)")

	return v.cmd
}

func (v versionCommand) Execute(args []string) error {
	if len(args) > 0 {
		return NewErrorWithUsage("version command needs no arguments")
	}

	op, constraint, err := v.resolveOp()
	if err != nil {
		return err
	}

	// Version is always printed on stdout.
	fmt.Printf("git-po-helper version %s\n", version.Version)

	if op == "" {
		return nil
	}

	ok, err := version.Satisfies(version.Version, op, constraint)
	if err != nil {
		return NewStandardErrorF("%v", err)
	}
	if ok {
		return nil
	}

	// Mismatch: error message on stdout (as requested); exit -1 without
	// main.go also printing to stderr.
	fmt.Fprintf(os.Stdout, "ERROR: version %s does not satisfy --%s %s\n",
		version.Version, op, constraint)
	return NewExitWithoutMessage()
}

func (v versionCommand) resolveOp() (version.Op, string, error) {
	type pair struct {
		op  version.Op
		val string
	}
	set := make([]pair, 0, 5)
	if v.O.LT != "" {
		set = append(set, pair{version.OpLT, v.O.LT})
	}
	if v.O.LE != "" {
		set = append(set, pair{version.OpLE, v.O.LE})
	}
	if v.O.GT != "" {
		set = append(set, pair{version.OpGT, v.O.GT})
	}
	if v.O.GE != "" {
		set = append(set, pair{version.OpGE, v.O.GE})
	}
	if v.O.EQ != "" {
		set = append(set, pair{version.OpEQ, v.O.EQ})
	}
	if len(set) == 0 {
		return "", "", nil
	}
	if len(set) > 1 {
		return "", "", NewErrorWithUsage("--lt, --le, --gt, --ge, and --eq are mutually exclusive")
	}
	return set[0].op, set[0].val, nil
}

var versionCmd = versionCommand{}

func init() {
	rootCmd.AddCommand(versionCmd.Command())
}
