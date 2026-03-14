// Package util provides business logic for agent-run update-po command.
package util

// CmdAgentRunUpdatePo implements the agent-run update-po command logic via AgentRunWorkflow.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	return RunAgentRunWorkflow(NewWorkflowUpdatePo(agentName, poFile))
}
