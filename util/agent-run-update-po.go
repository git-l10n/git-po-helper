// Package util provides business logic for agent-run update-po command.
package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/repository"
)

// CmdAgentRunUpdatePo implements the agent-run update-po command logic via AgentRunWorkflow.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	var absPo string
	if poFile != "" {
		var err error
		absPo, err = filepath.Abs(poFile)
		if err != nil {
			return fmt.Errorf("cannot resolve PO file path: %w", err)
		}
		absPo = filepath.Clean(absPo)
	}
	cleanup, err := EnsureInGitProjectRootDir()
	if err != nil {
		return err
	}
	defer cleanup()

	if absPo != "" {
		repoRoot := filepath.Clean(repository.WorkDir())
		rel, err := filepath.Rel(repoRoot, absPo)
		if err != nil {
			return fmt.Errorf("PO file %s vs repository root %s: %w", absPo, repoRoot, err)
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == ".." || strings.HasPrefix(relSlash, "../") || strings.Contains(relSlash, "/../") {
			return fmt.Errorf("PO file %s is not under repository root %s", absPo, repoRoot)
		}
		poFile = relSlash
	}
	return RunAgentRunWorkflow(NewWorkflowUpdatePo(agentName, poFile))
}
