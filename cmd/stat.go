package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

type statCommand struct {
	cmd *cobra.Command
	O   struct {
		Review string
	}
}

func (v *statCommand) Command() *cobra.Command {
	if v.cmd != nil {
		return v.cmd
	}

	v.cmd = &cobra.Command{
		Use:   "stat [po-file]",
		Short: "Report statistics for a PO file or review JSON",
		Long: `Report entry statistics for a PO file:
  translated   - entries with non-empty translation
  untranslated - entries with empty msgstr
  same         - entries where msgstr equals msgid (suspect untranslated)
  fuzzy        - entries with fuzzy flag
  obsolete     - obsolete entries (#~ format)

With --review <json-file>: report review results from agent-run review JSON.
Both files are derived from the path: strip .json or .po if present, then use <base>.json and <base>.po.
No args required.`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Execute(args)
		},
	}

	v.cmd.Flags().StringVar(&v.O.Review, "review", "", "report from review; path may end with .json or .po (both <base>.json and <base>.po are used)")

	return v.cmd
}

func (v statCommand) Execute(args []string) error {
	repository.ChdirProjectRoot()

	if v.O.Review != "" {
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "warning: in --review mode, args are ignored\n")
		}
		return v.executeReviewReport(args)
	}

	if len(args) != 1 {
		return newUserError("stat requires exactly one argument: <po-file>")
	}

	poFile := args[0]
	if !util.Exist(poFile) {
		return newUserError("file does not exist:", poFile)
	}

	stats, err := util.CountPoReportStats(poFile)
	if err != nil {
		return err
	}

	if flag.Verbose() > 0 {
		title := fmt.Sprintf("PO file: %s", poFile)
		fmt.Println(title)
		fmt.Println(strings.Repeat("-", len(title)))
		fmt.Printf("  translated:   %d\n", stats.Translated)
		fmt.Printf("  untranslated: %d\n", stats.Untranslated)
		fmt.Printf("  same:         %d\n", stats.Same)
		fmt.Printf("  fuzzy:        %d\n", stats.Fuzzy)
		fmt.Printf("  obsolete:     %d\n", stats.Obsolete)
	} else {
		fmt.Print(util.FormatStatLine(stats))
	}

	return nil
}

func (v statCommand) executeReviewReport(args []string) error {
	// Derive json and po from --review: strip .json/.po if present, then add both
	base := v.O.Review
	if strings.HasSuffix(base, ".json") {
		base = strings.TrimSuffix(base, ".json")
	} else if strings.HasSuffix(base, ".po") {
		base = strings.TrimSuffix(base, ".po")
	}
	jsonFile := base + ".json"
	poFile := base + ".po"

	result, err := util.ReportReviewFromJSON(jsonFile, poFile)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return newUserError(err.Error())
		}
		if strings.Contains(err.Error(), "provide po-file") {
			return newUserError("review JSON has no total_entries; provide <po-file> to count entries")
		}
		return err
	}

	fmt.Printf("Review JSON: %s\n", jsonFile)
	fmt.Printf("  Total entries: %d\n", result.Review.TotalEntries)
	fmt.Printf("  Issues found: %d\n", len(result.Review.Issues))
	fmt.Printf("  Review score: %d/100\n", result.Score)
	if len(result.Review.Issues) > 0 {
		fmt.Printf("  Critical (score 0): %d\n", result.CriticalCount)
		fmt.Printf("  Minor (score 2):   %d\n", result.MinorCount)
	}

	return nil
}

var statCmd = statCommand{}

func init() {
	rootCmd.AddCommand(statCmd.Command())
}
