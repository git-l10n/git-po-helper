package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// workflowUpdatePot implements AgentRunWorkflow for agent-run update-pot.
type workflowUpdatePot struct {
	agentName string
}

// NewWorkflowUpdatePot returns a workflow for update-pot.
func NewWorkflowUpdatePot(agentName string) AgentRunWorkflow {
	return &workflowUpdatePot{agentName: agentName}
}

func (w *workflowUpdatePot) Name() string { return "update-pot" }

func (w *workflowUpdatePot) InitContext(cfg *config.AgentConfig) *AgentRunContext {
	return &AgentRunContext{
		Cfg:       cfg,
		AgentName: w.agentName,
		Result:    &AgentRunResult{Score: 0},
	}
}

func (w *workflowUpdatePot) PreCheck(ctx *AgentRunContext) error {
	_, err := SelectAgent(ctx.Cfg, ctx.AgentName)
	if err != nil {
		return err
	}
	ctx.potFile = GetPotFilePath()
	log.Debugf("POT file path: %s", ctx.potFile)
	if !Exist(ctx.potFile) {
		ctx.Result.EntryCountBeforeUpdate = 0
	} else if stats, err := GetPoStats(ctx.potFile); err == nil {
		ctx.Result.EntryCountBeforeUpdate = stats.Total()
	}
	return nil
}

func (w *workflowUpdatePot) AgentRun(ctx *AgentRunContext) error {
	return agentRunUpdatePotExecute(ctx)
}

func (w *workflowUpdatePot) PostCheck(ctx *AgentRunContext) error {
	if Exist(ctx.potFile) {
		if stats, err := GetPoStats(ctx.potFile); err == nil {
			ctx.Result.EntryCountAfterUpdate = stats.Total()
		}
	}
	if ctx.Result.EntryCountAfterUpdate > 0 {
		ctx.Result.Score = 100
	}
	log.Infof("validating file syntax: %s", ctx.potFile)
	if err := ValidatePoFile(ctx.potFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		ctx.Result.SyntaxValidationError = err
		ctx.Result.Score = 0
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

func (w *workflowUpdatePot) Report(ctx *AgentRunContext, agentRunErr error) error {
	if agentRunErr != nil {
		return agentRunErr
	}
	if ctx.Result.PreValidationError != nil {
		return fmt.Errorf("pre-validation failed: %w", ctx.Result.PreValidationError)
	}
	if ctx.Result.PostValidationError != nil {
		return fmt.Errorf("post-validation failed: %w", ctx.Result.PostValidationError)
	}
	if ctx.Result.SyntaxValidationError != nil {
		ext := filepath.Ext(ctx.potFile)
		if ext == ".pot" {
			return fmt.Errorf("file validation failed: %w\nHint: Check the POT file syntax using 'msgcat --use-first <file> -o /dev/null'", ctx.Result.SyntaxValidationError)
		}
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", ctx.Result.SyntaxValidationError)
	}
	log.Infof("agent-run update-pot completed successfully")
	return nil
}

// agentRunUpdatePotExecute runs the agent for update-pot (prompt build through RunAgentAndParse).
func agentRunUpdatePotExecute(ctx *AgentRunContext) error {
	selectedAgent, err := SelectAgent(ctx.Cfg, ctx.AgentName)
	if err != nil {
		return err
	}
	log.Debugf("using agent: %s (%s)", ctx.AgentName, selectedAgent.Kind)
	prompt, err := GetRawPrompt(ctx.Cfg, "update-pot")
	if err != nil {
		return err
	}
	vars := PlaceholderVars{"prompt": prompt}
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
