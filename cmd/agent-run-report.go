package cmd

import (
	"github.com/git-l10n/git-po-helper/util"
	"github.com/spf13/cobra"
)

func newAgentRunReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "report <poDir>",
		Short:      "Report aggregated review statistics from batch or single JSON",
		Hidden:     true,
		Deprecated: "use 'agent-run review --report <dir>' instead",
		Long: `Report review statistics for agent-run review output.

Uses ` + util.DefaultReviewBase + ` for paths. Uses review-input.po for total count,
review-result.json for output. If any files match po/review-result-*.json,
they are aggregated; otherwise review-result.json is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			poDir := util.PoDir
			if len(args) > 0 {
				poDir = args[0]
			}
			result, err := util.GetReviewReport(poDir)
			if err != nil {
				return NewStandardErrorF("%v", err)
			}
			util.PrintReviewReportResult(result)
			return nil
		},
	}
}
