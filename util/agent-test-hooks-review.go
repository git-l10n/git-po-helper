package util

import (
	"fmt"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// ReviewPostResult is returned by review's PostProcess and holds the aggregated
// score and merged review JSON for display in ReportSummary and for the caller.
type ReviewPostResult struct {
	Score      int
	Aggregated *ReviewResult
}

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

// PostProcess aggregates review JSONs from all runs (same msgid takes lowest score),
// saves the merged JSON, applies it to the output PO, and returns the aggregated score and result.
func (agentTestHooksReview) PostProcess(results []TestRunResult, cfg *config.AgentConfig) (interface{}, error) {
	ps := GetReviewPathSet()
	var reviewJSONs []*ReviewResult

	for i := range results {
		result := &results[i]
		if result.Error != nil {
			log.Errorf("loop %d: agent-run returned error: %v", result.RunNumber, result.Error)
			continue
		}
		if result.Ctx != nil && result.Ctx.PostValidationError() != nil {
			continue
		}
		if result.ReviewResult != nil {
			reviewJSONs = append(reviewJSONs, result.ReviewResult)
			score, _ := result.Score()
			totalEntries, _ := result.ReviewResult.GetTotalEntries()
			log.Infof("loop %d: review score: %d (total_entries=%d, issues=%d)",
				result.RunNumber,
				score,
				totalEntries,
				len(result.ReviewResult.Issues))
		} else {
			log.Warnf("loop %d: no report returned", result.RunNumber)
		}
	}

	aggregatedScore := 0
	aggregated := aggregateReviewJSONResult(reviewJSONs, false)
	if aggregated != nil {
		if Exist(ps.InputPO) {
			aggregated.SetReviewSource(ps.InputPO)
		}
		aggregated.SetReviewPaths(ps.ResultJSON, ps.OutputPO)
		var scoreErr error
		aggregatedScore, scoreErr = aggregated.GetScore()
		if scoreErr != nil {
			log.Warnf("failed to calculate aggregated review score: %v", scoreErr)
		} else {
			log.Infof("aggregated review: score=%d/100 (from %d runs, %d unique issues)",
				aggregatedScore, len(reviewJSONs), len(aggregated.Issues))
			if err := saveReviewJSON(aggregated, ps.ResultJSON); err != nil {
				log.Warnf("failed to save aggregated review JSON: %v", err)
			}
		}
		if Exist(ps.InputPO) {
			if _, err := ApplyReviewFromResultJSON(ps); err != nil {
				log.Warnf("failed to apply aggregated review to %s: %v", ps.OutputPO, err)
			}
		}
	}

	return &ReviewPostResult{Score: aggregatedScore, Aggregated: aggregated}, nil
}

// ReportSummary prints per-run status and the aggregated report (when postResult is *ReviewPostResult).
// Full review report for each run is produced by workflowReview.Report and shown via ReportOutput in the workflow.
func (agentTestHooksReview) ReportSummary(results []TestRunResult, cfg *config.AgentConfig, postResult interface{}) {
	var runScores []string
	w := ReportLabelWidth
	for _, result := range results {
		status := "FAIL"
		if result.Success() {
			status = "PASS"
		}
		s, _ := result.Score()
		fmt.Printf("  %-*s loop %d: %s (Score: %d/100)\n", w, "Run:", result.RunNumber, status, s)
		runScores = append(runScores, fmt.Sprintf("%d", s))
	}
	if len(runScores) > 0 {
		fmt.Printf("  %-*s (%s)\n", w, "Per-run scores:", strings.Join(runScores, ", "))
	}

	// Aggregated report: header + per-run scores, then use the same print as workflow Report
	if postResult != nil {
		if r, ok := postResult.(*ReviewPostResult); ok && r.Aggregated != nil {
			fmt.Println()
			fmt.Println("🧩 Aggregated review report (all issues merged)")
			fmt.Println()
			PrintReviewReportResult(r.Aggregated)
		}
	}
	flushStdout()
}
