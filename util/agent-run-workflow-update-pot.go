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
		Cfg:             cfg,
		AgentName:       w.agentName,
		Result:          &AgentRunResult{Score: 0},
		PreCheckResult:  &PreCheckResult{},
		PostCheckResult: &PostCheckResult{},
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
		ctx.PreCheckResult.AllEntries = 0
	} else if stats, err := GetPoStats(ctx.potFile); err == nil {
		ctx.PreCheckResult.AllEntries = stats.Total()
	} else {
		ctx.PreCheckResult.Error = fmt.Errorf("failed to get POT file stats: %w", err)
		return ctx.PreCheckResult.Error
	}
	return nil
}

func (w *workflowUpdatePot) AgentRun(ctx *AgentRunContext) error {
	// PreCheckResult already set in PreCheck; mutates ctx.Result in place.
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
	GetAgentDiagnostics(ctx.Result, streamResult)
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}
	return nil
}

func (w *workflowUpdatePot) PostCheck(ctx *AgentRunContext) error {
	ctx.PostCheckResult = &PostCheckResult{}
	if Exist(ctx.potFile) {
		if stats, err := GetPoStats(ctx.potFile); err == nil {
			ctx.PostCheckResult.AllEntries = stats.Total()
		}
	}
	if ctx.PostCheckResult.AllEntries > 0 {
		ctx.PostCheckResult.Score = 100
	}
	ctx.Result.Score = ctx.PostCheckResult.Score
	log.Infof("validating file syntax: %s", ctx.potFile)
	if err := ValidatePoFile(ctx.potFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		ext := filepath.Ext(ctx.potFile)
		if ext == ".pot" {
			ctx.PostCheckResult.Error = fmt.Errorf("file syntax validation failed: %w\nHint: Check the POT file syntax using 'msgcat --use-first <file> -o /dev/null'", err)
		} else {
			ctx.PostCheckResult.Error = fmt.Errorf("file syntax validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", err)
		}
		ctx.PostCheckResult.Score = 0
		ctx.Result.Score = 0
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

func (w *workflowUpdatePot) Report(ctx *AgentRunContext) {
	if ctx == nil {
		return
	}

	labelWidth := ReviewStatLabelWidth
	pre, post := ctx.PreCheckResult, ctx.PostCheckResult
	fmt.Println()
	fmt.Println("🔍 Update POT Report")
	fmt.Println()
	fmt.Printf("  %-*s %d\n", labelWidth, "Before AllEntries:", pre.AllEntries)
	fmt.Printf("  %-*s %d\n", labelWidth, "After AllEntries:", post.AllEntries)
	if pre.Error != nil || post.Error != nil || ctx.Result.Error != nil {
		fmt.Println()
	}
	if ctx.Result != nil && ctx.Result.Error != nil {
		fmt.Printf("  %-*s %s\n", labelWidth, "Agent execution:", ctx.Result.Error)
	}
	if pre.Error != nil {
		fmt.Printf("  %-*s %s\n", labelWidth, "Pre-validation:", pre.Error.Error())
	}
	if post.Error != nil {
		fmt.Printf("  %-*s %s\n", labelWidth, "Post-validation:", post.Error.Error())
	}
	fmt.Println()
	flushStdout()
}
