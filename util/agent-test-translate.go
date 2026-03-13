// Package util provides business logic for agent-test translate command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// SaveTranslateResults saves the translation results to the output directory.
// It creates output/<agent-name>/<run-number>/ directory and copies the PO file
// and execution logs to preserve translation results for later review.
func SaveTranslateResults(agentName string, runNumber int, poFile string, stdout, stderr []byte) error {
	// Determine output directory path (relative to process cwd)
	workDir, _ := os.Getwd()
	if workDir == "" {
		workDir = "."
	}
	outputDir := filepath.Join(workDir, "output", agentName, fmt.Sprintf("%d", runNumber))

	log.Debugf("saving translation results to %s", outputDir)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Copy translated PO file to output directory
	poFileName := filepath.Base(poFile)
	destPoFile := filepath.Join(outputDir, poFileName)

	log.Debugf("copying %s to %s", poFile, destPoFile)

	// Read source PO file
	data, err := os.ReadFile(poFile)
	if err != nil {
		return fmt.Errorf("failed to read PO file %s: %w", poFile, err)
	}

	// Write to destination
	if err := os.WriteFile(destPoFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write PO file to %s: %w", destPoFile, err)
	}

	// Save execution log (stdout + stderr)
	logFile := filepath.Join(outputDir, "translation.log")
	log.Debugf("saving execution log to %s", logFile)

	var logContent strings.Builder
	if len(stdout) > 0 {
		logContent.WriteString("=== STDOUT ===\n")
		logContent.Write(stdout)
		logContent.WriteString("\n")
	}
	if len(stderr) > 0 {
		logContent.WriteString("=== STDERR ===\n")
		logContent.Write(stderr)
		logContent.WriteString("\n")
	}

	if err := os.WriteFile(logFile, []byte(logContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write log file to %s: %w", logFile, err)
	}

	log.Infof("translation results saved to %s", outputDir)
	return nil
}

// displayTranslateTestResults displays the translation test results in a readable format.
func displayTranslateTestResults(results []TestRunResult, averageScore float64, totalRuns int, elapsed time.Duration) {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Agent Test Results (Translate)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	successCount := 0
	failureCount := 0

	// Display individual run results
	for _, result := range results {
		status := "FAIL"
		if result.Score == 100 {
			status = "PASS"
			successCount++
		} else {
			failureCount++
		}

		fmt.Printf("Run %d: %s (Score: %d/100)\n", result.RunNumber, status, result.Score)

		// Show translation counts
		if result.AgentExecuted {
			if result.Ctx != nil {
				fmt.Printf("  New entries:     %d (before) -> %d (after)\n",
					result.Ctx.BeforeNewCount(), result.Ctx.AfterNewCount())
				fmt.Printf("  Fuzzy entries:   %d (before) -> %d (after)\n",
					result.Ctx.BeforeFuzzyCount(), result.Ctx.AfterFuzzyCount())
			}

			if result.RunError == nil {
				fmt.Printf("  Agent execution: PASS\n")
			} else {
				fmt.Printf("  Agent execution: FAIL - %v\n", result.RunError)
			}

			if result.Ctx != nil && result.Ctx.PostValidationError() == nil {
				fmt.Printf("  Validation:      PASS (all entries translated)\n")
			} else if result.Ctx != nil {
				fmt.Printf("  Validation:      FAIL - %s\n", result.Ctx.PostValidationError())
			}
		} else {
			fmt.Printf("  Agent execution: SKIPPED (pre-validation failed)\n")
		}

		fmt.Println()
	}

	// Calculate statistics for NumTurns and execution time
	var numTurnsList []int
	var executionTimes []time.Duration
	totalNumTurns := 0
	totalExecutionTime := time.Duration(0)
	numTurnsCount := 0

	for _, result := range results {
		if result.NumTurns > 0 {
			numTurnsList = append(numTurnsList, result.NumTurns)
			totalNumTurns += result.NumTurns
			numTurnsCount++
		}
		// Always collect execution time (we measure it ourselves in the loop)
		executionTimes = append(executionTimes, result.ExecutionTime)
		totalExecutionTime += result.ExecutionTime
	}

	// Display summary statistics
	const labelWidth = 25
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Summary")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("%-*s %d\n", labelWidth, "Total runs:", totalRuns)
	fmt.Printf("%-*s %d\n", labelWidth, "Successful runs:", successCount)
	fmt.Printf("%-*s %d\n", labelWidth, "Failed runs:", failureCount)
	fmt.Printf("%-*s %.2f/100\n", labelWidth, "Average score:", averageScore)

	// Display NumTurns statistics
	if numTurnsCount > 0 {
		avgNumTurns := totalNumTurns / numTurnsCount
		var numTurnsStrs []string
		for _, nt := range numTurnsList {
			turnsStr := fmt.Sprintf("%d", nt)
			numTurnsStrs = append(numTurnsStrs, turnsStr)
		}
		fmt.Printf("%-*s %d (%s)\n", labelWidth, "Avg Num turns:", avgNumTurns, strings.Join(numTurnsStrs, ", "))
	}

	// Display execution time statistics
	if len(executionTimes) > 0 {
		avgExecutionTime := totalExecutionTime / time.Duration(len(executionTimes))
		var execTimeStrs []string
		avgTimeStr := formatDuration(avgExecutionTime)
		for _, et := range executionTimes {
			timeStr := formatDuration(et)
			execTimeStrs = append(execTimeStrs, timeStr)
		}
		fmt.Printf("%-*s %s (%s)\n", labelWidth, "Avg Execution Time:", avgTimeStr, strings.Join(execTimeStrs, ", "))
	}

	// Always display total elapsed time
	fmt.Printf("%-*s %s\n", labelWidth, "Total Elapsed Time:", formatDuration(elapsed))
	fmt.Println("=" + strings.Repeat("=", 70))
}
