// Package util provides business logic for agent-test translate command.
package util

import (
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// cleanL10nIntermediateFiles removes untracked l10n intermediate files that are never
// tracked by git. Corresponds to AGENTS.md Task 3 Step 8 (po_cleanup): these files must
// be removed before each test run to avoid stale state from a previous run.
func cleanL10nIntermediateFiles() {
	files := []string{
		"po/l10n-pending.po",
		"po/l10n-todo.json",
		"po/l10n-done.json",
		"po/l10n-done.po",
		"po/l10n-done.merged",
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			log.Debugf("failed to remove %s: %v", f, err)
		}
	}
	log.Debugf("l10n intermediate files cleaned")
}

// CmdAgentTestTranslate implements the agent-test translate command logic.
// It runs the agent-run translate operation multiple times and calculates an average score.
func CmdAgentTestTranslate(agentName, poFile string, runs int, skipConfirmation bool, useLocalOrchestration bool, batchSize int) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}

	runs = ResolveAgentTestRuns(cfg, runs)

	log.Infof("starting agent-test translate with %d runs", runs)

	_, err = RunAgentTestTranslate(agentName, poFile, runs, cfg, useLocalOrchestration, batchSize)
	if err != nil {
		return fmt.Errorf("agent-test failed: %w", err)
	}
	return nil
}

// RunAgentTestTranslate runs the agent-test translate operation multiple times.
// Uses AgentRunWorkflow (translate) + hooks: cleanup before each loop; PreCheck/PostCheck
// on the workflow perform validation; Report runs per loop; then aggregate scores.
func RunAgentTestTranslate(agentName, poFile string, runs int, cfg *config.AgentConfig, useLocalOrchestration bool, batchSize int) ([]TestRunResult, error) {
	if _, err := SelectAgent(cfg, agentName); err != nil {
		return nil, err
	}
	resolvedPo, err := GuessPoFilePath(cfg, poFile)
	if err != nil {
		return nil, err
	}
	hooks := agentTestHooksTranslate{relPoFile: resolvedPo}
	results, err := RunAgentTestWorkflowLoops(func() AgentRunWorkflow {
		return NewWorkflowTranslate(agentName, poFile, useLocalOrchestration, batchSize)
	}, hooks, cfg, runs)
	if err != nil {
		return nil, err
	}
	return results, nil
}
