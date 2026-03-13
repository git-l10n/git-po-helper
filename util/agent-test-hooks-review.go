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
		if result.RunError != nil {
			log.Errorf("loop %d: agent-run returned error: %v", result.RunNumber, result.RunError)
			result.Score = 0
			continue
		}
		if result.Ctx != nil && result.Ctx.PostValidationError() != nil {
			result.Score = 0
			continue
		}
		if result.ReviewResult != nil {
			reviewJSONs = append(reviewJSONs, result.ReviewResult)
			totalEntries, _ := result.ReviewResult.GetTotalEntries()
			log.Infof("loop %d: review score: %d (total_entries=%d, issues=%d)",
				result.RunNumber,
				result.Score,
				totalEntries,
				len(result.ReviewResult.Issues))
		} else {
			log.Warnf("loop %d: no report returned", result.RunNumber)
			result.Score = 0
		}
	}

	aggregatedScore := 0
	aggregated := aggregateReviewJSONResult(reviewJSONs, false)
	if aggregated != nil {
		if Exist(ps.InputPO) {
			if stats, err := GetPoStats(ps.InputPO); err != nil {
				log.Warnf("failed to count entries in %s: %v", ps.InputPO, err)
			} else {
				aggregated.TotalEntries = stats.Total()
			}
		}
		var scoreErr error
		aggregatedScore, scoreErr = CalculateReviewScore(aggregated)
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
	w := ReportLabelWidth
	for _, result := range results {
		status := "FAIL"
		if result.Score > 0 {
			status = "PASS"
		}
		fmt.Printf("  %-*s loop %d: %s (Score: %d/100)\n", w, "Run:", result.RunNumber, status, result.Score)
	}
	// Aggregated report (aggregated score from merged JSON; per-run scores in parentheses)
	if postResult != nil {
		if r, ok := postResult.(*ReviewPostResult); ok && r.Aggregated != nil {
			var runScores []string
			for _, res := range results {
				runScores = append(runScores, fmt.Sprintf("%d", res.Score))
			}
			fmt.Println("  --- Aggregated (merged from all runs) ---")
			fmt.Printf("  %-*s %d/100 (%s)\n", w, "Aggregated score:", r.Score, strings.Join(runScores, ", "))
			fmt.Printf("  %-*s %d\n", w, "Total entries:", r.Aggregated.TotalEntries)
			fmt.Printf("  %-*s %d\n", w, "Unique issues:", len(r.Aggregated.Issues))
			fmt.Println()
		}
	}
	flushStdout()
}
