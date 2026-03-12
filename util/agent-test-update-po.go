// Package util provides business logic for agent-test update-po command.
package util

import (
	"fmt"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestUpdatePo implements the agent-test update-po command logic.
// It runs the agent-run update-po operation multiple times and calculates an average score.
func CmdAgentTestUpdatePo(agentName, poFile string, runs int, skipConfirmation bool) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}

	runs = ResolveAgentTestRuns(cfg, runs)

	log.Infof("starting agent-test update-po with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, averageScore, err := RunAgentTestUpdatePo(agentName, poFile, runs, cfg)
	if err != nil {
		return fmt.Errorf("agent-test failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Debugf("displaying test results (average score: %.2f)", averageScore)
	displayTestResults(results, averageScore, runs, elapsed, cfg.AgentTest.PoEntriesBeforeUpdate, cfg.AgentTest.PoEntriesAfterUpdate)

	log.Infof("agent-test update-po completed successfully (average score: %.2f/100)", averageScore)
	return nil
}

// RunAgentTestUpdatePo runs the agent-test update-po operation multiple times.
// Each loop uses workflow PreCheck/AgentRun/PostCheck/Report plus agent-test hooks.
func RunAgentTestUpdatePo(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]TestRunResult, float64, error) {
	relPoFile, err := GuessPoFilePath(cfg, poFile)
	if err != nil {
		log.Warnf("failed to get relative path of poFile: %v", err)
		relPoFile = poFile
	}
	hooks := agentTestHooksUpdatePo{relPoFile: relPoFile}
	return RunAgentTestWorkflowLoops(func() AgentRunWorkflow {
		return NewWorkflowUpdatePo(agentName, poFile)
	}, hooks, cfg, runs)
}
