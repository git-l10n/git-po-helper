package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentRunReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Report aggregated review statistics from batch or single JSON",
		Long: `Report review statistics for agent-run review output.

Uses ` + util.DefaultReviewBase + ` for paths. Uses review-input.po for total count,
review-result.json for output. If any files match po/review-result-*.json,
they are aggregated; otherwise review-result.json is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := util.GetReviewReport()
			if err != nil {
				return NewStandardErrorF("%v", err)
			}
			util.PrintReviewReportResult(util.WrapReviewReportForPrint(result), nil, nil)
			return nil
		},
	}
}
