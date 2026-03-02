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

If path is given (e.g. po/review.po), derives po/review.json and po/review.po.
If any files match po/review-batch-*.json, they are loaded and aggregated
into one result; otherwise po/review.json is used.

Default path is ` + util.DefaultReviewPoPath + ` when omitted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := util.DefaultReviewPoPath
			if len(args) > 0 {
				path = args[0]
			}
			jsonFile, result, err := util.ReportReviewFromPathWithBatches(path)
			if err != nil {
				return NewStandardErrorF("%v", err)
			}
			util.PrintReviewReportResult(jsonFile, result)
			return nil
		},
	}
}
