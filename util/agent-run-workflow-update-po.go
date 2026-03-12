package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
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
		Cfg:       cfg,
		AgentName: w.agentName,
		PoFile:    w.poFile,
		Result:    &AgentRunResult{Score: 0},
	}
}

func (w *workflowUpdatePo) PreCheck(ctx *AgentRunContext) error {
	_, err := SelectAgent(ctx.Cfg, ctx.AgentName)
	if err != nil {
		return err
	}
	poFile, err := GetPoFileAbsPath(ctx.Cfg, ctx.PoFile)
	if err != nil {
		return err
	}
	ctx.poFileAbs = poFile
	log.Debugf("PO file path: %s", ctx.poFileAbs)
	ctx.PreCheckResult = &PreCheckResult{}
	if !Exist(ctx.poFileAbs) {
		ctx.PreCheckResult.AllEntries = 0
	} else if stats, err := GetPoStats(ctx.poFileAbs); err == nil {
		ctx.PreCheckResult.AllEntries = stats.Total()
	}
	return nil
}

func (w *workflowUpdatePo) AgentRun(ctx *AgentRunContext) error {
	return agentRunUpdatePoExecute(ctx)
}

func (w *workflowUpdatePo) PostCheck(ctx *AgentRunContext) error {
	ctx.PostCheckResult = &PostCheckResult{Score: 100}
	if Exist(ctx.poFileAbs) {
		if stats, err := GetPoStats(ctx.poFileAbs); err == nil {
			ctx.PostCheckResult.AllEntries = stats.Total()
		}
	}
	ctx.Result.Score = 100
	log.Infof("validating file syntax: %s", ctx.poFileAbs)
	if err := ValidatePoFile(ctx.poFileAbs); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		ctx.PostCheckResult.SyntaxValidationError = err
		ctx.Result.Score = 0
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

func (w *workflowUpdatePo) Report(ctx *AgentRunContext, agentRunErr error) error {
	if agentRunErr != nil {
		return agentRunErr
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
	log.Infof("agent-run update-po completed successfully")
	return nil
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
	workDir := repository.WorkDirOrCwd()
	sourcePath := ctx.poFileAbs
	if rel, err := filepath.Rel(workDir, ctx.poFileAbs); err == nil && rel != "" && rel != "." {
		sourcePath = filepath.ToSlash(rel)
	}
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
	applyAgentDiagnostics(ctx.Result, streamResult)
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}
	return nil
}
