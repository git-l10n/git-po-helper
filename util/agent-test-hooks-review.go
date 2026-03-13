package util

import (
	"github.com/git-l10n/git-po-helper/config"
)

// agentTestHooksReview cleans review output files before each loop.
// workflowReview PostCheck performs pending/input validation.
type agentTestHooksReview struct{}

func (agentTestHooksReview) CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error {
	cleanReviewOutputFilesForTest(GetReviewPathSet())
	return nil
}

func (agentTestHooksReview) ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	return nil
}

func (agentTestHooksReview) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	return nil
}

// ReportSummary is a no-op here: review uses aggregated JSON score; CmdAgentTestReview
// calls displayReviewTestResults after RunAgentTestReview with aggregatedScore.
func (agentTestHooksReview) ReportSummary(results []TestRunResult, cfg *config.AgentConfig) {
}
