package cmd

import (
	"fmt"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type statCommand struct {
	cmd *cobra.Command
}

func (v *statCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "stat <file> [file...]",
		Short: "Report statistics for PO/JSON file(s)",
		Long: `Report entry statistics for PO or gettext JSON files:
  translated   - entries with non-empty translation
  untranslated - entries with empty msgstr
  same         - entries where msgstr equals msgid (suspect untranslated)
  fuzzy        - entries with fuzzy flag
  obsolete     - obsolete entries (#~ format)

Input can be PO/POT files or gettext JSON (same schema as msg-select --json).
Format is auto-detected: JSON if file starts with '{' after whitespace.

When run inside a git worktree, paths are relative to the project root (e.g. po/zh_CN.po).
When run outside a git repository, paths are relative to the current directory or absolute.

For review JSON report, use: git-po-helper agent-run report [path]`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}

	return v.cmd
}

func (v statCommand) Execute(args []string) error {
	if len(args) < 1 {
		return NewErrorWithUsage("stat requires at least one argument: <file> [file...]")
	}

	var errs []string
	for i, file := range args {
		if !util.Exist(file) {
			errs = append(errs, fmt.Sprintf("file does not exist: %s", file))
			continue
		}

		stats, err := util.CountReportStats(file)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", file, err))
			continue
		}

		if flag.Verbose() > 0 {
			if i > 0 {
				fmt.Println()
			}
			title := fmt.Sprintf("File: %s", file)
			fmt.Println(title)
			fmt.Println(strings.Repeat("-", len(title)))
			fmt.Printf("  translated:   %d\n", stats.Translated)
			fmt.Printf("  untranslated: %d\n", stats.Untranslated)
			fmt.Printf("  same:         %d\n", stats.Same)
			fmt.Printf("  fuzzy:        %d\n", stats.Fuzzy)
			fmt.Printf("  obsolete:     %d\n", stats.Obsolete)
		} else {
			fmt.Printf("%s: %s", file, util.FormatStatLine(stats))
		}
	}

	if len(errs) > 0 {
		return NewStandardError(strings.Join(errs, "\n"))
	}
	return nil
}

var statCmd = statCommand{}

func init() {
	rootCmd.AddCommand(statCmd.Command())
}
