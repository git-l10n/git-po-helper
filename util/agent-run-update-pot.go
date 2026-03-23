// Package util provides business logic for agent-run update-pot command.
package util

// CmdAgentRunUpdatePot implements the agent-run update-pot command logic via AgentRunWorkflow.
func CmdAgentRunUpdatePot(agentName string) error {
	cleanup, err := EnsureInGitProjectRootDir()
	if err != nil {
		return err
	}
	defer cleanup()
	return RunAgentRunWorkflow(NewWorkflowUpdatePot(agentName))
}
