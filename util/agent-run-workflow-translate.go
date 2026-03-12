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
	rel, err := GetPoFileRelPath(ctx.Cfg, ctx.PoFile)
	if err != nil {
		return err
	}
	ctx.PoFile = rel
	pre, err := validateTranslatePreResult(ctx.PoFile)
	if err != nil {
		ctx.PreCheckResult = pre
		return err
	}
	ctx.PreCheckResult = pre
	if stats, err := GetPoStats(ctx.PoFile); err == nil {
		w.translateStatsBefore = fmt.Sprintf("Translation statistics: before: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
	}
	return nil
}

func (w *workflowTranslate) AgentRun(ctx *AgentRunContext) error {
	// Dispatch only; stats printing is deferred to Report so agent-test can keep using RunAgentTranslate.
	result, err := runAgentTranslateDispatch(ctx.Cfg, ctx.AgentName, ctx.PoFile, ctx.UseLocalOrchestration, ctx.BatchSize)
	if result == nil {
		result = &AgentRunResult{Score: 0}
	}
	// Single assignment from dispatch; PreCheckResult already set in PreCheck.
	*ctx.Result = *result
	return err
}

func (w *workflowTranslate) PostCheck(ctx *AgentRunContext) error {
	return validateTranslatePostResult(ctx.PoFile, ctx)
}

func (w *workflowTranslate) Report(ctx *AgentRunContext, agentRunErr error) error {
	// Print before/after stats (same behavior as RunAgentTranslate after dispatch)
	if stats, errStats := GetPoStats(ctx.PoFile); errStats != nil {
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
	if ctx.PreValidationError() != nil {
		return fmt.Errorf("pre-validation failed: %w", ctx.PreValidationError())
	}
	if ctx.PostValidationError() != nil {
		return fmt.Errorf("post-validation failed: %w", ctx.PostValidationError())
	}
	if ctx.SyntaxValidationError() != nil {
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", ctx.SyntaxValidationError())
	}
	log.Infof("agent-run translate completed successfully")
	return nil
}
