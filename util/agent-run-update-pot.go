// Package util provides business logic for agent-run update-pot command.
package util

import (
	"github.com/git-l10n/git-po-helper/config"
)

// RunAgentUpdatePot executes a single agent-run update-pot operation.
// It uses the same PreCheck → AgentRun → PostCheck pipeline as the workflow
// (agent-test calls this; agent-run Cmd uses RunAgentRunWorkflow).
func RunAgentUpdatePot(cfg *config.AgentConfig, agentName string) (*AgentRunResult, error) {
	wf := &workflowUpdatePot{agentName: agentName}
	ctx := wf.InitContext(cfg)
	if err := wf.PreCheck(ctx); err != nil {
		return ctx.Result, err
	}
	if err := wf.AgentRun(ctx); err != nil {
		return ctx.Result, err
	}
	_ = wf.PostCheck(ctx)
	return ctx.Result, nil
}

// CmdAgentRunUpdatePot implements the agent-run update-pot command logic via AgentRunWorkflow.
func CmdAgentRunUpdatePot(agentName string) error {
	return RunAgentRunWorkflow(NewWorkflowUpdatePot(agentName))
}
