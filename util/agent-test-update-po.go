// Package util provides business logic for agent-test update-po command.
package util

import (
	"fmt"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestUpdatePo implements the agent-test update-po command logic.
// It runs the agent-run update-po operation multiple times and calculates an average score.
func CmdAgentTestUpdatePo(agentName, poFile string, runs int, skipConfirmation bool) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	// Determine number of runs
	if runs == 0 {
		if cfg.AgentTest.Runs != nil && *cfg.AgentTest.Runs > 0 {
			runs = *cfg.AgentTest.Runs
			log.Debugf("using runs from configuration: %d", runs)
		} else {
			runs = 5 // Default
			log.Debugf("using default number of runs: %d", runs)
		}
	} else {
		log.Debugf("using runs from command line: %d", runs)
	}

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
	displayTestResults(results, averageScore, runs, elapsed)

	log.Infof("agent-test update-po completed successfully (average score: %.2f/100)", averageScore)
	return nil
}

// RunAgentTestUpdatePo runs the agent-test update-po operation multiple times.
// It reuses RunAgentUpdatePo for each run and accumulates scores.
// Returns scores for each run, average score, and error.
func RunAgentTestUpdatePo(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]RunResult, float64, error) {
	// Run the test multiple times
	results := make([]RunResult, runs)
	totalScore := 0
	relPoFile, err := GetPoFileRelPath(cfg, poFile)
	if err != nil {
		log.Warnf("failed to get relative path of poFile: %v", err)
	}

	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("run %d/%d", runNum, runs)

		// Start timing for this iteration
		iterStartTime := time.Now()

		// Clean po/ directory before each run to ensure a clean state
		if err := CleanPoDirectory(relPoFile, "po/git.pot"); err != nil {
			log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
			// Continue with the run even if cleanup fails, but log the warning
		}

		// Reuse RunAgentUpdatePo for each run
		agentResult, err := RunAgentUpdatePo(cfg, agentName, poFile, true)

		// Calculate execution time for this iteration
		iterExecutionTime := time.Since(iterStartTime)

		// Convert AgentRunResult to RunResult
		// agentResult is never nil (always returns a result structure)
		result := RunResult{
			RunNumber:           runNum,
			Score:               agentResult.Score,
			PreValidationPass:   agentResult.PreValidationPass,
			PostValidationPass:  agentResult.PostValidationPass,
			AgentExecuted:       agentResult.AgentExecuted,
			PreValidationError:  agentResult.PreValidationError,
			PostValidationError: agentResult.PostValidationError,
			AgentError:          agentResult.AgentError,
			BeforeCount:         agentResult.BeforeCount,
			AfterCount:          agentResult.AfterCount,
			ExpectedBefore:      cfg.AgentTest.PoEntriesBeforeUpdate,
			ExpectedAfter:       cfg.AgentTest.PoEntriesAfterUpdate,
			NumTurns:            agentResult.NumTurns,
			ExecutionTime:       iterExecutionTime,
		}

		// If there was an error, log it but continue (for agent-test, we want to collect all results)
		if err != nil {
			log.Debugf("run %d: agent-run returned error: %v", runNum, err)
			// Error details are already in the result structure
		}

		results[i] = result
		totalScore += result.Score
		log.Debugf("run %d: completed with score %d/100", runNum, result.Score)
	}

	// Calculate average score
	averageScore := float64(totalScore) / float64(runs)
	log.Infof("all runs completed. Total score: %d/%d, Average: %.2f/100", totalScore, runs*100, averageScore)

	return results, averageScore, nil
}
