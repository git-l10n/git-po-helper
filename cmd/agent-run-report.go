package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentRunReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report [path]",
		Short: "Report aggregated review statistics from batch or single JSON",
		Long: `Report review statistics for agent-run review output.

Path is the base (e.g. po/review). Uses review-input.po for total count,
review-result.json for output. If any files match po/review-result-*.json,
they are aggregated; otherwise review-result.json is used.

Default path is ` + util.DefaultReviewBase + ` when omitted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := util.DefaultReviewBase
			if len(args) > 0 {
				path = args[0]
			}
			result, err := util.ReportReviewFromPathWithBatches(path)
			if err != nil {
				return NewStandardErrorF("%v", err)
			}
			util.PrintReviewReportResult(result)
			return nil
		},
	}
}
