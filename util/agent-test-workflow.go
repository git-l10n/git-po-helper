// Package util provides agent-test loop orchestration aligned with AgentRunWorkflow.
// Each loop: Cleanup → PreCheck → agent-test pre validation → AgentRun → PostCheck →
// agent-test post validation → Report (to stdout or writer); then aggregate stats.
package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// reportStdoutMu serializes os.Stdout replacement so concurrent agent-test is safe.
var reportStdoutMu sync.Mutex

// runReportWithWriter runs wf.Report(ctx). If writer is nil, prints run header to stdout
// then calls Report (same as agent-run). If writer is non-nil, captures Report stdout
// and stderr into writer (including run header) so the caller can store it in
// tr.ReportOutput and display it when running agent-test.
func runReportWithWriter(wf AgentRunWorkflow, ctx *AgentRunContext, writer io.Writer, runNum, totalRuns int) {
	header := fmt.Sprintf("\n--- Report for loop %d/%d ---\n", runNum, totalRuns)
	if writer == nil {
		fmt.Print(header)
		wf.Report(ctx)
		return
	}
	// Capture stdout and stderr from Report by temporarily redirecting os.Stdout and os.Stderr.
	reportStdoutMu.Lock()
	defer reportStdoutMu.Unlock()
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Print(header)
		wf.Report(ctx)
		return
	}
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	done := make(chan struct{})
	var copyErr error
	go func() {
		_, copyErr = io.Copy(writer, r)
		_ = r.Close()
		close(done)
	}()
	_, _ = io.WriteString(writer, header)
	wf.Report(ctx)
	_ = w.Sync() // flush so all Report output is in the pipe before close
	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	<-done
	if copyErr != nil {
		log.Debugf("report capture copy: %v", copyErr)
	}
}

// AgentTestLoopHooks is implemented per command (update-pot, update-po, translate, review).
// Cleanup runs before every iteration. Validators run after workflow PreCheck/PostCheck
// and may set ctx.PreCheckResult.Error / ctx.PostCheckResult.Error and zero score.
// PostProcess runs once after all loops; it may aggregate results and write output (e.g. review).
// Its return value is passed to ReportSummary and to the runner's caller (e.g. aggregated score).
// ReportSummary runs after PostProcess to print workflow-specific stats and optionally the aggregated report.
type AgentTestLoopHooks interface {
	CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error
	ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error
	ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error
	PostProcess(results []TestRunResult, cfg *config.AgentConfig) (interface{}, error)
	ReportSummary(results []TestRunResult, cfg *config.AgentConfig, postResult interface{})
}

// runAgentTestSingleLoop executes one iteration through PostCheck; then calls Report
// with reportWriter (nil => stdout). Returns TestRunResult with Ctx and optional
// ReportOutput filled when reportWriter is a *bytes.Buffer (caller reads .String()).
func runAgentTestSingleLoop(wf AgentRunWorkflow, hooks AgentTestLoopHooks, cfg *config.AgentConfig, runNum, totalRuns int, reportWriter io.Writer) TestRunResult {
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

	runReportWithWriter(wf, ctx, reportWriter, runNum, totalRuns)

	runErr = agentErr
	if runErr == nil && ctx.PreValidationError() != nil {
		runErr = ctx.PreValidationError()
	}
	if runErr == nil && ctx.PostValidationError() != nil {
		runErr = ctx.PostValidationError()
	}

	if ctx.Result != nil {
		tr.AgentRunResult = *ctx.Result
	}
	tr.ExecutionTime = time.Since(iterStart)
	tr.RunError = runErr
	if buf, ok := reportWriter.(*bytes.Buffer); ok {
		tr.ReportOutput = buf.String()
	}
	return tr
}

// RunAgentTestWorkflowLoops runs newWorkflow() once per iteration. Each run captures
// Report into ReportOutput, then prints all captured reports again after the loop.
// PostProcess runs next (e.g. review aggregates and applies to output); its return value
// is passed to ReportSummary. ReportSummary prints workflow-specific stats.
func RunAgentTestWorkflowLoops(newWorkflow func() AgentRunWorkflow, hooks AgentTestLoopHooks, cfg *config.AgentConfig, runs int) ([]TestRunResult, error) {
	if runs <= 0 {
		return nil, fmt.Errorf("runs must be positive")
	}
	if newWorkflow == nil {
		return nil, fmt.Errorf("newWorkflow must not be nil")
	}
	workflowName := ""
	startTime := time.Now()
	results := make([]TestRunResult, runs)
	for i := 0; i < runs; i++ {
		runNum := i + 1
		wf := newWorkflow()
		workflowName = wf.Name()
		log.Infof("⏳ Starting loop %d/%d (%s)", runNum, runs, wf.Name())
		buf := &bytes.Buffer{}
		results[i] = runAgentTestSingleLoop(wf, hooks, cfg, runNum, runs, buf)
		// First print the report for the loop
		fmt.Println(results[i].ReportOutput)
	}

	fmt.Println()
	fmt.Printf("========== 📝 Reports for each run ==========\n")
	fmt.Println()
	for i := range results {
		// Second print the report for the loop
		if results[i].ReportOutput != "" {
			fmt.Println(results[i].ReportOutput)
		}
	}

	fmt.Println()
	fmt.Printf("========== ✅ Summary of [%s] workflow ==========\n", workflowName)
	fmt.Println()
	elapsed := time.Since(startTime)
	PrintAgentTestSummaryReport(results, elapsed)
	postResult, postErr := hooks.PostProcess(results, cfg)
	if postErr != nil {
		return nil, postErr
	}
	hooks.ReportSummary(results, cfg, postResult)
	return results, nil
}
