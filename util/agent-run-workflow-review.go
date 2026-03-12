package util

import (
	"fmt"
	"time"

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
	reviewAgentRunPostCheck(ctx)
	return nil
}

func (w *workflowReview) Report(ctx *AgentRunContext) {
	PrintReviewReportResult(ctx.Result, ctx.Result.Error, ctx)
	fmt.Printf("\nSummary:\n")
	if ctx.Result.ReviewReport.ReportFile != "" {
		fmt.Printf("  Review JSON: %s\n", getRelativePath(ctx.Result.ReviewReport.ReportFile))
	}
	if ctx.Result.NumTurns > 0 {
		fmt.Printf("  Turns: %d\n", ctx.Result.NumTurns)
	}
	fmt.Printf("  Execution time: %s\n", ctx.Result.ExecutionTime.Round(time.Millisecond))
	log.Infof("agent-run review completed successfully")
}
