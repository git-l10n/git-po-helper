// Package util provides business logic for agent-test review command.
package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestReview implements the agent-test review command logic.
// It runs the agent-run review operation multiple times and calculates an average score.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// useLocalOrchestration: if true, use local orchestration (--use-local-orchestration);
// otherwise use agent with po/AGENTS.md (default).
func CmdAgentTestReview(agentName string, target *CompareTarget, runs int, skipConfirmation bool, outputBase string, useLocalOrchestration bool, batchSize int) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return err
	}

	runs = ResolveAgentTestRuns(cfg, runs)

	log.Infof("starting agent-test review with %d runs", runs)

	// Run the test
	_, err = RunAgentTestReview(cfg, agentName, target, runs, outputBase, useLocalOrchestration, batchSize)
	if err != nil {
		log.Errorf("agent-test execution failed: %v", err)
		return fmt.Errorf("agent-test failed: %w", err)
	}

	log.Infof("agent-test review completed successfully")
	return nil
}

// cleanReviewOutputFilesForTest removes all review output files before each test run
// for a fresh start, including InputPO. Backs up core files (ResultJSON, OutputPO, InputPO) first.
func cleanReviewOutputFilesForTest(ps ReviewPathSet) {
	backupFileIfExists(ps.ResultJSON, ps.OutputPO, ps.InputPO)
	for _, p := range []string{
		ps.InputPO,
		ps.PendingPO,
		ps.OutputPO,
		ps.ResultJSON,
		ps.ReviewBatchTxtPath(),
		ps.ReviewTodoJSONPath(),
		ps.ReviewDoneJSONPath(),
	} {
		_ = os.Remove(p)
	}
	globPattern := filepath.Join(filepath.Dir(ps.InputPO), "review-result-*.json")
	if matches, _ := filepath.Glob(globPattern); matches != nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}

// RunAgentTestReview runs the agent-test review operation multiple times.
// It calls RunAgentReview for each run (dispatches to local or prompt orchestration),
// aggregates JSON results (for same msgid takes lowest score), and saves one aggregated
// JSON at the end. No per-run backup.
// Returns scores for each run and error. Aggregated score is computed in PostProcess and shown in ReportSummary.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// useLocalOrchestration: if true, use local orchestration; otherwise use agent with po/AGENTS.md.
func RunAgentTestReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, runs int, outputBase string, useLocalOrchestration bool, batchSize int) ([]TestRunResult, error) {
	if _, err := SelectAgent(cfg, agentName); err != nil {
		return nil, err
	}

	return RunAgentTestWorkflowLoops(func() AgentRunWorkflow {
		return NewWorkflowReview(agentName, target, useLocalOrchestration, batchSize)
	}, agentTestHooksReview{}, cfg, runs)
}
