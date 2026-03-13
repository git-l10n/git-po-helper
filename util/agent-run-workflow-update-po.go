package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// workflowUpdatePo implements AgentRunWorkflow for agent-run update-po.
type workflowUpdatePo struct {
	agentName string
	poFile    string
}

// NewWorkflowUpdatePo returns a workflow for update-po.
func NewWorkflowUpdatePo(agentName, poFile string) AgentRunWorkflow {
	return &workflowUpdatePo{agentName: agentName, poFile: poFile}
}

func (w *workflowUpdatePo) Name() string { return "update-po" }

func (w *workflowUpdatePo) InitContext(cfg *config.AgentConfig) *AgentRunContext {
	return &AgentRunContext{
		Cfg:             cfg,
		AgentName:       w.agentName,
		PoFile:          w.poFile,
		Result:          &AgentRunResult{},
		PreCheckResult:  &PreCheckResult{},
		PostCheckResult: &PostCheckResult{},
	}
}

func (w *workflowUpdatePo) PreCheck(ctx *AgentRunContext) error {
	_, err := SelectAgent(ctx.Cfg, ctx.AgentName)
	if err != nil {
		return err
	}
	rel, err := GuessPoFilePath(ctx.Cfg, ctx.PoFile)
	if err != nil {
		return err
	}
	ctx.PoFile = rel
	log.Debugf("PO file path: %s", ctx.PoFile)
	ctx.PreCheckResult = &PreCheckResult{}
	if !Exist(ctx.PoFile) {
		ctx.PreCheckResult.AllEntries = 0
	} else if stats, err := GetPoStats(ctx.PoFile); err == nil {
		ctx.PreCheckResult.AllEntries = stats.Total()
		ctx.PreCheckResult.UntranslatePoEntries = stats.Untranslated
		ctx.PreCheckResult.FuzzyPoEntries = stats.Fuzzy
	}
	return nil
}

func (w *workflowUpdatePo) AgentRun(ctx *AgentRunContext) error {
	return agentRunUpdatePoExecute(ctx)
}

func (w *workflowUpdatePo) PostCheck(ctx *AgentRunContext) error {
	ctx.PostCheckResult = &PostCheckResult{}
	if Exist(ctx.PoFile) {
		if stats, err := GetPoStats(ctx.PoFile); err == nil {
			ctx.PostCheckResult.AllEntries = stats.Total()
			ctx.PostCheckResult.UntranslatePoEntries = stats.Untranslated
			ctx.PostCheckResult.FuzzyPoEntries = stats.Fuzzy
		} else {
			ctx.PostCheckResult.Error = fmt.Errorf("failed to get PO stats: %w", err)
			return ctx.PostCheckResult.Error
		}
	} else {
		ctx.PostCheckResult.Error = fmt.Errorf("PO file does not exist: %s", ctx.PoFile)
		return ctx.PostCheckResult.Error
	}
	log.Infof("validating file syntax: %s", ctx.PoFile)
	if err := ValidatePoFile(ctx.PoFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		ctx.PostCheckResult.Error = fmt.Errorf("file syntax validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", err)
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

func (w *workflowUpdatePo) Report(ctx *AgentRunContext) {
	if ctx == nil {
		return
	}

	labelWidth := ReportLabelWidth
	pre, post := ctx.PreCheckResult, ctx.PostCheckResult
	fmt.Println()
	fmt.Println("🔍 Update PO Report")
	fmt.Println()
	fmt.Printf("  %-*s %d\n", labelWidth, "Before AllEntries:", pre.AllEntries)
	fmt.Printf("  %-*s %d\n", labelWidth, "Before untranslated:", pre.UntranslatePoEntries)
	fmt.Printf("  %-*s %d\n", labelWidth, "Before fuzzy:", pre.FuzzyPoEntries)
	fmt.Println()
	fmt.Printf("  %-*s %d\n", labelWidth, "After AllEntries:", post.AllEntries)
	fmt.Printf("  %-*s %d\n", labelWidth, "After untranslated:", post.UntranslatePoEntries)
	fmt.Printf("  %-*s %d\n", labelWidth, "After fuzzy:", post.FuzzyPoEntries)
	PrintAgentRunStatus(ctx)
	flushStdout()
}

// agentRunUpdatePoExecute runs the agent for update-po.
func agentRunUpdatePoExecute(ctx *AgentRunContext) error {
	selectedAgent, err := SelectAgent(ctx.Cfg, ctx.AgentName)
	if err != nil {
		return err
	}
	log.Debugf("using agent: %s (%s)", ctx.AgentName, selectedAgent.Kind)
	prompt, err := GetRawPrompt(ctx.Cfg, "update-po")
	if err != nil {
		return err
	}
	sourcePath := filepath.ToSlash(ctx.PoFile)
	vars := PlaceholderVars{"prompt": prompt, "source": sourcePath}
	resolvedPrompt, err := ExecutePromptTemplate(prompt, vars)
	if err != nil {
		return fmt.Errorf("failed to resolve prompt template: %w", err)
	}
	vars["prompt"] = resolvedPrompt
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return fmt.Errorf("failed to build agent command: %w", err)
	}
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON,
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	ctx.Result.AgentExecuted = true
	stdout, _, stderr, streamResult, err := RunAgentAndParse(agentCmd, outputFormat, selectedAgent.Kind)
	if err != nil {
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
		if len(stdout) > 0 {
			log.Debugf("agent command stdout: %s", string(stdout))
		}
		return err
	}
	log.Infof("agent command completed successfully")
	GetAgentDiagnostics(ctx.Result, streamResult)
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}
	return nil
}
