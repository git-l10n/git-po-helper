package util

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// workflowReview implements AgentRunWorkflow for agent-run review.
type workflowReview struct {
	agentName             string
	target                *CompareTarget
	useLocalOrchestration bool
	batchSize             int
}

// NewWorkflowReview returns a workflow for review.
func NewWorkflowReview(agentName string, target *CompareTarget, useLocalOrchestration bool, batchSize int) AgentRunWorkflow {
	if batchSize <= 0 {
		batchSize = 50
	}
	return &workflowReview{
		agentName:             agentName,
		target:                target,
		useLocalOrchestration: useLocalOrchestration,
		batchSize:             batchSize,
	}
}

func (w *workflowReview) Name() string { return "review" }

func (w *workflowReview) InitContext(cfg *config.AgentConfig) *AgentRunContext {
	return &AgentRunContext{
		Cfg:                   cfg,
		AgentName:             w.agentName,
		Target:                w.target,
		UseLocalOrchestration: w.useLocalOrchestration,
		BatchSize:             w.batchSize,
		Result:                &AgentRunResult{Score: 0},
		PreCheckResult:        &PreCheckResult{},
		PostCheckResult:       &PostCheckResult{},
	}
}

func (w *workflowReview) PreCheck(ctx *AgentRunContext) error {
	// Review has no programmatic pre-check; agent/orchestration prepares review-input.po.
	return nil
}

func (w *workflowReview) AgentRun(ctx *AgentRunContext) error {
	result, err := runAgentReviewDispatch(ctx.Cfg, ctx.AgentName, ctx.Target, ctx.UseLocalOrchestration, ctx.BatchSize)
	if result != nil {
		*ctx.Result = *result
	}
	return err
}

func (w *workflowReview) PostCheck(ctx *AgentRunContext) error {
	// Verifies review-pending.po is empty or absent after dispatch.
	// Writes to ctx.PostCheckResult and ctx.Result.Score.
	if ctx == nil || ctx.Result == nil {
		return nil
	}
	result := ctx.Result
	if result.ReviewReport.ReviewResult != nil {
		result.Score = result.ReviewReport.Score
	}
	ctx.PostCheckResult.Score = result.Score

	setReviewPostValidation := func(err error) {
		if err == nil {
			return
		}
		if ctx.PostCheckResult.Error == nil {
			ctx.PostCheckResult.Error = err
		}
		ctx.PostCheckResult.Score = 0
		result.Score = 0
		log.Errorf("%v", err)
	}

	ps := GetReviewPathSet()
	totalCount := 0
	if Exist(ps.InputPO) {
		stats, statsErr := GetPoStats(ps.InputPO)
		if statsErr != nil {
			setReviewPostValidation(fmt.Errorf("cannot get PO stats for %s: %w", ps.InputPO, statsErr))
		} else {
			totalCount = stats.Total()
		}
		ctx.PreCheckResult.ReviewTotalEntries = totalCount
		if totalCount == 0 {
			ctx.PreCheckResult.Error = fmt.Errorf("no entries reviewed: input PO %s missing or empty, or pending absent or empty", ps.InputPO)
		}
	} else {
		ctx.PreCheckResult.Error = fmt.Errorf("input PO %s does not exist", ps.InputPO)
	}

	pendingCount := 0
	if Exist(ps.PendingPO) {
		info, statErr := os.Stat(ps.PendingPO)
		if statErr != nil {
			setReviewPostValidation(fmt.Errorf("cannot stat pending review PO %s: %w", ps.PendingPO, statErr))
		} else if info.Size() == 0 {
			pendingCount = 0
		} else {
			stats, statsErr := GetPoStats(ps.PendingPO)
			if statsErr != nil {
				setReviewPostValidation(fmt.Errorf("cannot get PO stats for %s: %w", ps.PendingPO, statsErr))
			} else {
				pendingCount = stats.Total()
			}
		}
	}
	ctx.PostCheckResult.ReviewPendingEntries = pendingCount

	if pendingCount != 0 {
		reviewedCount := totalCount - pendingCount
		if reviewedCount < 0 {
			reviewedCount = 0
		}
		setReviewPostValidation(fmt.Errorf(
			"review incomplete: %d entries still in %s (total in %s: %d, reviewed: %d, not reviewed: %d)",
			pendingCount, ps.PendingPO, ps.InputPO, totalCount, reviewedCount, pendingCount))
	}

	// Local orchestration must have had entries to review; error still set here, display in Report.
	if ctx.PostCheckResult.Error == nil && (!Exist(ps.PendingPO) || pendingCount == 0) {
		if totalCount == 0 {
			setReviewPostValidation(fmt.Errorf(
				"no entries reviewed: input PO %s missing or empty, or pending absent or empty", ps.InputPO))
		}
	}
	return nil
}

func (w *workflowReview) Report(ctx *AgentRunContext) {
	// Show review statistics
	PrintReviewReportResult(ctx.Result, ctx.Result.Error, ctx)

	// PO entry counts (same info formerly logged in PostCheck)
	labelWidth := ReviewStatLabelWidth
	pre, post := ctx.PreCheckResult, ctx.PostCheckResult
	fmt.Println("📋 Summary")
	fmt.Println()
	fmt.Printf("  %-*s %d\n", labelWidth, "Total entries (input):", pre.ReviewTotalEntries)
	fmt.Println()
	fmt.Printf("  %-*s %d\n", labelWidth, "Remaining in pending:", post.ReviewPendingEntries)
	if pre.ReviewTotalEntries > 0 {
		reviewed := pre.ReviewTotalEntries - post.ReviewPendingEntries
		fmt.Printf("  %-*s %d\n", labelWidth, "Reviewed:", reviewed)
	}
	if pre.ReviewTotalEntries > 0 && post.ReviewPendingEntries == 0 && post.Error == nil {
		fmt.Printf("  %-*s %s\n", labelWidth, "Pending cleared:", "all entries reviewed")
	}
	// Print errors if any
	if pre.Error != nil || post.Error != nil || ctx.Result.Error != nil {
		fmt.Println()
		if pre.Error != nil {
			fmt.Printf("  %-*s %s\n", labelWidth, "Pre-validation:", pre.Error.Error())
		}
		if post.Error != nil {
			fmt.Printf("  %-*s %s\n", labelWidth, "Post-validation:", post.Error.Error())
		}
		if ctx.Result.Error != nil {
			fmt.Printf("  %-*s %s\n", labelWidth, "Agent execution:", ctx.Result.Error.Error())
		}
	}
}
