// Package util provides business logic for agent-test update-pot command.
package util

import (
	"fmt"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestUpdatePot implements the agent-test update-pot command logic.
// It runs the agent-run update-pot operation multiple times and calculates an average score.
func CmdAgentTestUpdatePot(agentName string, runs int, skipConfirmation bool) error {
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
// It reuses RunAgentUpdatePot for each run and accumulates scores.
// Pre-validation and post-validation (entry count checks) are performed here
// when configured in cfg.AgentTest.
// Returns scores for each run, average score, and error.
func RunAgentTestUpdatePot(agentName string, runs int, cfg *config.AgentConfig) ([]TestRunResult, float64, error) {
	potFile := GetPotFilePath()

	// Run the test multiple times
	results := make([]TestRunResult, runs)
	totalScore := 0

	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("run %d/%d", runNum, runs)

		// Start timing for this iteration
		iterStartTime := time.Now()

		// Clean po/ directory before each run to ensure a clean state
		if err := CleanPoDirectory("po/git.pot"); err != nil {
			log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
			// Continue with the run even if cleanup fails, but log the warning
		}

		// Pre-validation: check entry count before update (when configured)
		var agentResult *AgentRunResult
		var err error
		if cfg.AgentTest.PotEntriesBeforeUpdate != nil && *cfg.AgentTest.PotEntriesBeforeUpdate != 0 {
			log.Infof("performing pre-validation: checking entry count before update (expected: %d)", *cfg.AgentTest.PotEntriesBeforeUpdate)
			agentResult = &AgentRunResult{Score: 0}
			if !Exist(potFile) {
				agentResult.EntryCountBeforeUpdate = 0
			} else if stats, e := GetPoStats(potFile); e == nil {
				agentResult.EntryCountBeforeUpdate = stats.Total()
			}
			if err = ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesBeforeUpdate, "before update"); err != nil {
				agentResult.PreValidationError = fmt.Errorf("pre-validation failed: %w\nHint: Ensure po/git.pot exists and has the expected number of entries", err)
				agentResult.Score = 0
				// Skip agent run when pre-validation fails
			} else {
				log.Infof("pre-validation passed")
				agentResult, err = RunAgentUpdatePot(cfg, agentName)
			}
		} else {
			agentResult, err = RunAgentUpdatePot(cfg, agentName)
		}

		// Post-validation: check entry count after update (when configured)
		if err == nil && agentResult != nil && cfg.AgentTest.PotEntriesAfterUpdate != nil && *cfg.AgentTest.PotEntriesAfterUpdate != 0 {
			log.Infof("performing post-validation: checking entry count after update (expected: %d)", *cfg.AgentTest.PotEntriesAfterUpdate)
			if Exist(potFile) {
				if stats, e := GetPoStats(potFile); e == nil {
					agentResult.EntryCountAfterUpdate = stats.Total()
				}
			}
			if postErr := ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesAfterUpdate, "after update"); postErr != nil {
				agentResult.PostValidationError = fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the POT file correctly", postErr)
				agentResult.Score = 0
			} else {
				log.Infof("post-validation passed")
			}
		}

		// Calculate execution time for this iteration
		iterExecutionTime := time.Since(iterStartTime)

		// Convert AgentRunResult to TestRunResult (embedding avoids field duplication)
		result := TestRunResult{
			AgentRunResult: *agentResult,
			RunNumber:      runNum,
			RunError:       err,
		}
		result.ExecutionTime = iterExecutionTime

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
