package util

import (
	"fmt"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// workflowTranslate implements AgentRunWorkflow for agent-run translate.
type workflowTranslate struct {
	agentName             string
	poFile                string
	useLocalOrchestration bool
	batchSize             int
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
		Result:                &AgentRunResult{},
		PreCheckResult:        &PreCheckResult{},
		PostCheckResult:       &PostCheckResult{},
	}
}

func (w *workflowTranslate) PreCheck(ctx *AgentRunContext) error {
	poFile, err := GuessPoFilePath(ctx.Cfg, ctx.PoFile)
	if err != nil {
		return err
	}
	ctx.PoFile = poFile

	ctx.PreCheckResult = &PreCheckResult{}
	if !Exist(ctx.PoFile) {
		ctx.PreCheckResult.Error = fmt.Errorf("pre-validation: PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", ctx.PoFile)
		return ctx.PreCheckResult.Error
	}

	log.Infof("performing pre-validation: counting new and fuzzy entries")
	statsBefore, err := GetPoStats(ctx.PoFile)
	if err != nil {
		ctx.PreCheckResult.Error = fmt.Errorf("pre-validation: failed to count PO stats: %w", err)
		return ctx.PreCheckResult.Error
	}
	ctx.PreCheckResult.AllEntries = statsBefore.Total()
	ctx.PreCheckResult.UntranslatePoEntries = statsBefore.Untranslated
	ctx.PreCheckResult.FuzzyPoEntries = statsBefore.Fuzzy

	if statsBefore.Untranslated == 0 && statsBefore.Fuzzy == 0 {
		ctx.PreCheckResult.Error = fmt.Errorf("pre-validation: no new or fuzzy entries to translate, PO file is ready for use")
		return ctx.PreCheckResult.Error
	}

	return nil
}

func (w *workflowTranslate) AgentRun(ctx *AgentRunContext) error {
	// Dispatch only; stats printing is deferred to Report so agent-test can keep using RunAgentTranslate.
	result, err := runAgentTranslateDispatch(ctx.Cfg, ctx.AgentName, ctx.PoFile, ctx.UseLocalOrchestration, ctx.BatchSize)
	if result == nil {
		result = &AgentRunResult{}
	}
	// Single assignment from dispatch; PreCheckResult already set in PreCheck.
	*ctx.Result = *result
	return err
}

func (w *workflowTranslate) PostCheck(ctx *AgentRunContext) error {
	poFile := ctx.PoFile
	if ctx.PostCheckResult == nil {
		ctx.PostCheckResult = &PostCheckResult{}
	}
	log.Infof("performing post-validation: counting new and fuzzy entries")

	statsAfter, err := GetPoStats(poFile)
	if err != nil {
		ctx.PostCheckResult.Error = fmt.Errorf("failed to count PO stats after translation: %w", err)
		return ctx.PostCheckResult.Error
	}
	ctx.PostCheckResult.AllEntries = statsAfter.Total()
	ctx.PostCheckResult.UntranslatePoEntries = statsAfter.Untranslated
	ctx.PostCheckResult.FuzzyPoEntries = statsAfter.Fuzzy
	log.Infof("new (untranslated) entries after translation: %d", statsAfter.Untranslated)
	log.Infof("fuzzy entries after translation: %d", statsAfter.Fuzzy)

	if statsAfter.Untranslated != 0 || statsAfter.Fuzzy != 0 {
		ctx.PostCheckResult.Error = fmt.Errorf("post-validation: translation incomplete: %d new entries and %d fuzzy entries remaining", statsAfter.Untranslated, statsAfter.Fuzzy)
		return ctx.PostCheckResult.Error
	}

	log.Infof("post-validation passed: all entries translated")

	log.Infof("validating file syntax: %s", poFile)
	if err := ValidatePoFile(poFile); err != nil {
		ctx.PostCheckResult.Error = fmt.Errorf("post-validation: file syntax validation failed: %v", err)
		return ctx.PostCheckResult.Error
	} else {
		log.Infof("post-validation: file syntax validation passed")
	}
	return nil
}

func (w *workflowTranslate) Report(ctx *AgentRunContext) {
	if ctx == nil {
		return
	}

	labelWidth := ReportLabelWidth
	pre, post := ctx.PreCheckResult, ctx.PostCheckResult
	// Print report when we have any pre/post context or errors to show.
	fmt.Println()
	fmt.Println("🔍 Translation Report")
	fmt.Println()
	if pre != nil {
		beforeTranslated := pre.AllEntries - pre.UntranslatePoEntries - pre.FuzzyPoEntries
		if beforeTranslated < 0 {
			beforeTranslated = 0
		}
		fmt.Printf("  %-*s %d\n", labelWidth, "Before translated:", beforeTranslated)
		fmt.Printf("  %-*s %d\n", labelWidth, "Before untranslated:", pre.UntranslatePoEntries)
		fmt.Printf("  %-*s %d\n", labelWidth, "Before fuzzy:", pre.FuzzyPoEntries)
	}
	if post != nil {
		fmt.Println()
		afterTranslated := post.AllEntries - post.UntranslatePoEntries - post.FuzzyPoEntries
		if afterTranslated < 0 {
			afterTranslated = 0
		}
		fmt.Printf("  %-*s %d\n", labelWidth, "After translated:", afterTranslated)
		fmt.Printf("  %-*s %d\n", labelWidth, "After untranslated:", post.UntranslatePoEntries)
		fmt.Printf("  %-*s %d\n", labelWidth, "After fuzzy:", post.FuzzyPoEntries)
	}
	PrintAgentRunStatus(ctx)
	flushStdout()
}
