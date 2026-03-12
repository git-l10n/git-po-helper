package util

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// workflowTranslate implements AgentRunWorkflow for agent-run translate.
type workflowTranslate struct {
	agentName             string
	poFile                string
	useLocalOrchestration bool
	batchSize             int
	translateStatsBefore  string // full "before" line printed in Report
}

// NewWorkflowTranslate returns a workflow for translate.
func NewWorkflowTranslate(agentName, poFile string, useLocalOrchestration bool, batchSize int) AgentRunWorkflow {
	if batchSize <= 0 {
		batchSize = 50
	}
	return &workflowTranslate{
		agentName:             agentName,
		poFile:                poFile,
		useLocalOrchestration: useLocalOrchestration,
		batchSize:             batchSize,
	}
}

func (w *workflowTranslate) Name() string { return "translate" }

func (w *workflowTranslate) InitContext(cfg *config.AgentConfig) *AgentRunContext {
	return &AgentRunContext{
		Cfg:                   cfg,
		AgentName:             w.agentName,
		PoFile:                w.poFile,
		UseLocalOrchestration: w.useLocalOrchestration,
		BatchSize:             w.batchSize,
		Result:                &AgentRunResult{Score: 0},
	}
}

func (w *workflowTranslate) PreCheck(ctx *AgentRunContext) error {
	poFile, err := GetPoFileAbsPath(ctx.Cfg, ctx.PoFile)
	if err != nil {
		return err
	}
	ctx.poFileAbs = poFile
	preResult, err := validateTranslatePreResult(ctx.poFileAbs)
	if err != nil {
		*ctx.Result = *preResult
		return err
	}
	ctx.Result.BeforeNewCount = preResult.BeforeNewCount
	ctx.Result.BeforeFuzzyCount = preResult.BeforeFuzzyCount
	if stats, err := GetPoStats(ctx.poFileAbs); err == nil {
		w.translateStatsBefore = fmt.Sprintf("Translation statistics: before: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
	}
	return nil
}

func (w *workflowTranslate) AgentRun(ctx *AgentRunContext) error {
	// Dispatch only; stats printing is deferred to Report so agent-test can keep using RunAgentTranslate.
	result, err := runAgentTranslateDispatch(ctx.Cfg, ctx.AgentName, ctx.poFileAbs, ctx.UseLocalOrchestration, ctx.BatchSize)
	if result != nil {
		// Merge dispatch result into context result (turns, AgentExecuted, etc.)
		ctx.Result.AgentExecuted = result.AgentExecuted
		ctx.Result.AgentStdout = result.AgentStdout
		ctx.Result.AgentStderr = result.AgentStderr
		ctx.Result.NumTurns = result.NumTurns
		if result.PreValidationError != nil {
			ctx.Result.PreValidationError = result.PreValidationError
		}
	}
	return err
}

func (w *workflowTranslate) PostCheck(ctx *AgentRunContext) error {
	return validateTranslatePostResult(ctx.poFileAbs, ctx.Result)
}

func (w *workflowTranslate) Report(ctx *AgentRunContext, agentRunErr error) error {
	// Print before/after stats (same behavior as RunAgentTranslate after dispatch)
	if stats, errStats := GetPoStats(ctx.poFileAbs); errStats != nil {
		log.Errorf("GetPoStats after agent: %v", errStats)
		if w.translateStatsBefore != "" {
			fmt.Fprintln(os.Stderr, w.translateStatsBefore)
		}
	} else {
		afterSummary := fmt.Sprintf("Translation statistics: after: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
		if w.translateStatsBefore != "" {
			fmt.Fprintf(os.Stderr, "%s\n", w.translateStatsBefore)
		}
		fmt.Fprintf(os.Stderr, "%s\n", afterSummary)
	}
	if agentRunErr != nil {
		return fmt.Errorf("agent execution failed: %w", agentRunErr)
	}
	if ctx.Result.PreValidationError != nil {
		return fmt.Errorf("pre-validation failed: %w", ctx.Result.PreValidationError)
	}
	if ctx.Result.PostValidationError != nil {
		return fmt.Errorf("post-validation failed: %w", ctx.Result.PostValidationError)
	}
	if ctx.Result.SyntaxValidationError != nil {
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", ctx.Result.SyntaxValidationError)
	}
	log.Infof("agent-run translate completed successfully")
	return nil
}
