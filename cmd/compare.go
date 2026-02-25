package cmd

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type compareCommand struct {
	cmd *cobra.Command
	O   struct {
		Range  string
		Commit string
		Since  string
		Stat   bool
		Output string
	}
}

func (v *compareCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "compare [-r range | --commit <commit> | --since <commit>] [[<src>] <target>]",
		Short: "Show changes between two l10n files",
		Long: `By default: output new or changed entries to stdout.
Use -o <file> to write to a file (avoids stderr mixing when redirecting stdout).
With --stat: show diff statistics between two l10n file versions.

If no po/XX.po argument is given, the PO file is selected from changed files
(interactive when multiple, auto when single).

Modes:
- --commit <commit>: compare parent of commit with the specified commit
- --since <commit>: compare since commit with current working tree
- no --commit/--since: compare HEAD with current working tree (local changes)

Exactly one of --range, --commit and --since may be specified.
Output is empty when there are no new or changed entries.`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}
	v.cmd.Flags().BoolVar(&v.O.Stat, "stat", false, "show diff statistics (default: output new or changed entries)")
	v.cmd.Flags().StringVarP(&v.O.Range, "range", "r", "",
		"revision range: a..b (a and b), a.. (a and working tree), or a (a~ and a)")
	v.cmd.Flags().StringVar(&v.O.Commit, "commit", "",
		"equivalent to -r <commit>^..<commit>")
	v.cmd.Flags().StringVar(&v.O.Since, "since", "",
		"equivalent to -r <commit>.. (compare commit with working tree)")
	v.cmd.Flags().StringVarP(&v.O.Output, "output", "o", "",
		"write output to file (use - for stdout); empty output overwrites file")

	_ = viper.BindPFlag("compare--range", v.cmd.Flags().Lookup("range"))
	_ = viper.BindPFlag("compare--commit", v.cmd.Flags().Lookup("commit"))
	_ = viper.BindPFlag("compare--since", v.cmd.Flags().Lookup("since"))
	_ = viper.BindPFlag("compare--output", v.cmd.Flags().Lookup("output"))

	return v.cmd
}

func (v compareCommand) Execute(args []string) error {
	target, err := util.ResolveRevisionsAndFiles(v.O.Range, v.O.Commit, v.O.Since, args)
	if err != nil {
		return newUserErrorF("%v", err)
	}

	if v.O.Stat {
		return v.executeStat(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile)
	}
	return v.executeNew(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile)
}

func (v compareCommand) executeNew(oldCommit, oldFile, newCommit, newFile string) error {
	outputDest := v.O.Output
	if outputDest == "" {
		outputDest = "-"
	}
	log.Debugf("outputting new entries from '%s:%s' to '%s:%s'",
		oldCommit, oldFile, newCommit, newFile)
	err := util.PrepareReviewData(oldCommit, oldFile, newCommit, newFile, outputDest)
	if err != nil {
		return newUserErrorF("failed to prepare review data: %v", err)
	}
	return nil
}

func (v compareCommand) executeStat(oldCommit, oldFile, newCommit, newFile string) error {
	oldRev := util.FileRevision{Revision: oldCommit, File: oldFile}
	newRev := util.FileRevision{Revision: newCommit, File: newFile}
	if err := util.CheckoutTmpfile(&oldRev); err != nil {
		return newUserErrorF("failed to checkout %s@%s: %v", oldFile, oldCommit, err)
	}
	if err := util.CheckoutTmpfile(&newRev); err != nil {
		return newUserErrorF("failed to checkout %s@%s: %v", newFile, newCommit, err)
	}
	defer func() {
		if oldRev.Tmpfile != "" {
			os.Remove(oldRev.Tmpfile)
		}
		if newRev.Tmpfile != "" {
			os.Remove(newRev.Tmpfile)
		}
	}()

	srcData, err := os.ReadFile(oldRev.Tmpfile)
	if err != nil {
		return newUserErrorF("failed to read old file: %v", err)
	}
	destData, err := os.ReadFile(newRev.Tmpfile)
	if err != nil {
		return newUserErrorF("failed to read new file: %v", err)
	}

	stat, _, err := util.PoCompare(srcData, destData)
	if err != nil {
		return newUserErrorF("%v", err)
	}

	diffStat := ""
	if stat.Added != 0 {
		diffStat = fmt.Sprintf("%d new", stat.Added)
	}
	if stat.Changed != 0 {
		if diffStat != "" {
			diffStat += ", "
		}
		diffStat += fmt.Sprintf("%d changed", stat.Changed)
	}
	if stat.Deleted != 0 {
		if diffStat != "" {
			diffStat += ", "
		}
		diffStat += fmt.Sprintf("%d removed", stat.Deleted)
	}
	if diffStat != "" {
		fmt.Println(diffStat)
	} else {
		fmt.Fprintln(os.Stderr, "Nothing changed.")
	}
	return nil
}

var compareCmd = compareCommand{}

func init() {
	rootCmd.AddCommand(compareCmd.Command())
}
