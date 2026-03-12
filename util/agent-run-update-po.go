// Package util provides business logic for agent-run update-po command.
package util

import (
	"github.com/git-l10n/git-po-helper/config"
)

// RunAgentUpdatePo executes a single agent-run update-po operation.
// It uses the same PreCheck → AgentRun → PostCheck pipeline as the workflow.
// Returns (result, ctx, error); ctx holds PreCheckResult/PostCheckResult for display.
func RunAgentUpdatePo(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, *AgentRunContext, error) {
	wf := &workflowUpdatePo{agentName: agentName, poFile: poFile}
	ctx := wf.InitContext(cfg)
	if err := wf.PreCheck(ctx); err != nil {
		return ctx.Result, ctx, err
	}
	if err := wf.AgentRun(ctx); err != nil {
		return ctx.Result, ctx, err
	}
	_ = wf.PostCheck(ctx)
	return ctx.Result, ctx, nil
}

// CmdAgentRunUpdatePo implements the agent-run update-po command logic via AgentRunWorkflow.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	return RunAgentRunWorkflow(NewWorkflowUpdatePo(agentName, poFile))
}
