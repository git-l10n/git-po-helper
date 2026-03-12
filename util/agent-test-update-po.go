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
// It reuses RunAgentUpdatePo for each run and accumulates scores.
// Pre-validation and post-validation (entry count checks) are performed here
// when configured in cfg.AgentTest.
// Returns scores for each run, average score, and error.
func RunAgentTestUpdatePo(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]TestRunResult, float64, error) {
	// Resolve PO file to relative path once (cwd at repo root)
	relPoFile, err := GetPoFileRelPath(cfg, poFile)
	if err != nil {
		log.Warnf("failed to get relative path of poFile: %v", err)
	}

	// Run the test multiple times
	results := make([]TestRunResult, runs)
	totalScore := 0

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

		// Pre-validation: check entry count before update (when configured)
		var agentResult *AgentRunResult
		var runCtx *AgentRunContext
		var runErr error
		if cfg.AgentTest.PoEntriesBeforeUpdate != nil && *cfg.AgentTest.PoEntriesBeforeUpdate != 0 {
			log.Infof("performing pre-validation: checking PO entry count before update (expected: %d)", *cfg.AgentTest.PoEntriesBeforeUpdate)
			agentResult = &AgentRunResult{Score: 0}
			runCtx = &AgentRunContext{Result: agentResult}
			runCtx.PreCheckResult = &PreCheckResult{}
			if !Exist(relPoFile) {
				runCtx.PreCheckResult.AllEntries = 0
			} else if stats, e := GetPoStats(relPoFile); e == nil {
				runCtx.PreCheckResult.AllEntries = stats.Total()
			}
			if runErr = ValidatePoEntryCount(relPoFile, cfg.AgentTest.PoEntriesBeforeUpdate, "before update"); runErr != nil {
				runCtx.PreCheckResult.Error = fmt.Errorf("pre-validation failed: %w\nHint: Ensure %s exists and has the expected number of entries", runErr, relPoFile)
				agentResult.Score = 0
				// Skip agent run when pre-validation fails
			} else {
				log.Infof("pre-validation passed")
				agentResult, runCtx, runErr = RunAgentUpdatePo(cfg, agentName, poFile)
			}
		} else {
			agentResult, runCtx, runErr = RunAgentUpdatePo(cfg, agentName, poFile)
		}

		// Post-validation: check entry count after update (when configured)
		if runErr == nil && agentResult != nil && runCtx != nil && cfg.AgentTest.PoEntriesAfterUpdate != nil && *cfg.AgentTest.PoEntriesAfterUpdate != 0 {
			log.Infof("performing post-validation: checking PO entry count after update (expected: %d)", *cfg.AgentTest.PoEntriesAfterUpdate)
			if runCtx.PostCheckResult == nil {
				runCtx.PostCheckResult = &PostCheckResult{}
			}
			if Exist(relPoFile) {
				if stats, e := GetPoStats(relPoFile); e == nil {
					runCtx.PostCheckResult.AllEntries = stats.Total()
				}
			}
			if postErr := ValidatePoEntryCount(relPoFile, cfg.AgentTest.PoEntriesAfterUpdate, "after update"); postErr != nil {
				runCtx.PostCheckResult.Error = fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the PO file correctly", postErr)
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
			RunError:       runErr,
			Ctx:            runCtx,
		}
		result.ExecutionTime = iterExecutionTime

		// If there was an error, log it but continue (for agent-test, we want to collect all results)
		if runErr != nil {
			log.Debugf("run %d: agent-run returned error: %v", runNum, runErr)
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
