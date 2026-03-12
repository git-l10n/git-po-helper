// Package util provides agent-test loop orchestration aligned with AgentRunWorkflow.
// Each loop: Cleanup → PreCheck → agent-test pre validation → AgentRun → PostCheck →
// agent-test post validation → Report; then aggregate stats.
package util

import (
	"fmt"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// AgentTestLoopHooks is implemented per command (update-pot, update-po, translate, review).
// Cleanup runs before every iteration. Validators run after workflow PreCheck/PostCheck
// and may set ctx.PreCheckResult.Error / ctx.PostCheckResult.Error and zero score.
type AgentTestLoopHooks interface {
	// CleanupBeforeLoop resets filesystem/state so each run starts clean.
	// Errors are logged; loop continues unless hooks choose to return fatal err.
	CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error
	// ValidateAfterPreCheck runs after wf.PreCheck; use cfg.AgentTest expectations.
	ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error
	// ValidateAfterPostCheck runs after wf.PostCheck; validates post state (entry counts, etc.).
	ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error
}

// runAgentTestSingleLoop executes one iteration: cleanup, PreCheck, pre validation,
// AgentRun (sets ctx.Result.Error), PostCheck, post validation.
// Report is not called here; RunAgentTestWorkflowLoops calls Workflow.Report(Ctx) after all loops.
// Returns TestRunResult with Ctx and Workflow filled for deferred Report and aggregation.
func runAgentTestSingleLoop(wf AgentRunWorkflow, hooks AgentTestLoopHooks, cfg *config.AgentConfig, runNum, totalRuns int) TestRunResult {
	var (
		ctx    = wf.InitContext(cfg)
		runErr error
		tr     = TestRunResult{
			AgentRunResult: *ctx.Result,
			RunNumber:      runNum,
			RunError:       runErr,
			Ctx:            ctx,
		}
	)

	iterStart := time.Now()
	if ctx.Result == nil {
		ctx.Result = &AgentRunResult{Score: 0}
	}

	if err := hooks.CleanupBeforeLoop(ctx, runNum, totalRuns); err != nil {
		log.Warnf("run %d: cleanup: %v", runNum, err)
	}

	// PreCheck — workflow prepares paths and fills PreCheckResult where applicable
	preErr := wf.PreCheck(ctx)
	if preErr != nil {
		if ctx.PreCheckResult == nil {
			ctx.PreCheckResult = &PreCheckResult{}
		}
		if ctx.PreCheckResult.Error == nil {
			ctx.PreCheckResult.Error = preErr
		}
		ctx.Result.Score = 0
	}

	// Agent-test validation based on precheck (config-driven)
	if valErr := hooks.ValidateAfterPreCheck(ctx, cfg); valErr != nil {
		ctx.Result.Score = 0
		if ctx.PreCheckResult == nil {
			ctx.PreCheckResult = &PreCheckResult{}
		}
		if ctx.PreCheckResult.Error == nil {
			ctx.PreCheckResult.Error = valErr
		}
	}

	var agentErr error
	if ctx.PreValidationError() == nil {
		agentErr = wf.AgentRun(ctx)
		if ctx.Result != nil {
			ctx.Result.Error = agentErr
		}
		if agentErr != nil {
			ctx.Result.Score = 0
		}
	} else {
		agentErr = ctx.PreValidationError()
	}

	// PostCheck always (matches RunAgentRunWorkflow); may fix score on success
	_ = wf.PostCheck(ctx)
	if ctx.PostValidationError() != nil {
		ctx.Result.Score = 0
	}

	if valErr := hooks.ValidateAfterPostCheck(ctx, cfg); valErr != nil {
		if ctx.PostCheckResult == nil {
			ctx.PostCheckResult = &PostCheckResult{}
		}
		if ctx.PostCheckResult.Error == nil {
			ctx.PostCheckResult.Error = valErr
		}
		ctx.PostCheckResult.Score = 0
		ctx.Result.Score = 0
	}

	// Per-loop report (same output as agent-run single shot)
	fmt.Printf("\n--- Run %d/%d ---\n", runNum, totalRuns)
	wf.Report(ctx)

	// Terminal error for this run: agent failure takes precedence for RunError
	runErr = agentErr
	if runErr == nil && ctx.PreValidationError() != nil {
		runErr = ctx.PreValidationError()
	}
	if runErr == nil && ctx.PostValidationError() != nil {
		runErr = ctx.PostValidationError()
	}

	// Sync embedded result from ctx (Score etc. updated on ctx.Result during loop)
	if ctx.Result != nil {
		tr.AgentRunResult = *ctx.Result
	}
	tr.ExecutionTime = time.Since(iterStart)
	tr.RunError = runErr
	return tr
}

// RunAgentTestWorkflowLoops runs newWorkflow() once per iteration so each loop gets a
// fresh workflow instance (no shared mutable state across runs).
func RunAgentTestWorkflowLoops(newWorkflow func() AgentRunWorkflow, hooks AgentTestLoopHooks, cfg *config.AgentConfig, runs int) ([]TestRunResult, float64, error) {
	if runs <= 0 {
		return nil, 0, fmt.Errorf("runs must be positive")
	}
	if newWorkflow == nil {
		return nil, 0, fmt.Errorf("newWorkflow must not be nil")
	}
	results := make([]TestRunResult, runs)
	totalScore := 0
	for i := 0; i < runs; i++ {
		runNum := i + 1
		wf := newWorkflow()
		log.Infof("run %d/%d (%s)", runNum, runs, wf.Name())
		results[i] = runAgentTestSingleLoop(wf, hooks, cfg, runNum, runs)
		totalScore += results[i].Score
	}

	// After all loops, print each run's Report using stored Workflow + Ctx
	fmt.Println()
	fmt.Println("========== Reports for each run ==========")
	for i := range results {
		if results[i].Workflow != nil && results[i].Ctx != nil {
			fmt.Printf("\n--- Run %d/%d ---\n", results[i].RunNumber, runs)
			results[i].Workflow.Report(results[i].Ctx)
		}
	}

	avg := float64(totalScore) / float64(runs)
	log.Infof("all runs completed. Total score: %d/%d, Average: %.2f/100", totalScore, runs*100, avg)
	return results, avg, nil
}
