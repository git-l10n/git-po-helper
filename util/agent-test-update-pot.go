// Package util provides business logic for agent-test update-pot command.
package util

import (
	"fmt"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestUpdatePot implements the agent-test update-pot command logic.
// It runs the agent-run update-pot operation multiple times and calculates an average score.
func CmdAgentTestUpdatePot(agentName string, runs int, skipConfirmation bool) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}

	runs = ResolveAgentTestRuns(cfg, runs)

	log.Infof("starting agent-test update-pot with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, averageScore, err := RunAgentTestUpdatePot(agentName, runs, cfg)
	if err != nil {
		return fmt.Errorf("agent-test failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Debugf("displaying test results (average score: %.2f)", averageScore)
	displayTestResults(results, averageScore, runs, elapsed, cfg.AgentTest.PotEntriesBeforeUpdate, cfg.AgentTest.PotEntriesAfterUpdate)

	log.Infof("agent-test update-pot completed successfully (average score: %.2f/100)", averageScore)
	return nil
}

// RunAgentTestUpdatePot runs the agent-test update-pot operation multiple times.
// Each loop: Cleanup → PreCheck → agent-test pre validation → AgentRun → PostCheck →
// agent-test post validation → Report; then aggregate scores.
func RunAgentTestUpdatePot(agentName string, runs int, cfg *config.AgentConfig) ([]TestRunResult, float64, error) {
	return RunAgentTestWorkflowLoops(func() AgentRunWorkflow {
		return NewWorkflowUpdatePot(agentName)
	}, agentTestHooksUpdatePot{}, cfg, runs)
}
