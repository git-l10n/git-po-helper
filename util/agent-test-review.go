// Package util provides business logic for agent-test review command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
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

	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
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

	log.Infof("starting agent-test review with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, aggregatedScore, err := RunAgentTestReview(cfg, agentName, target, runs, outputBase, useLocalOrchestration, batchSize)
	if err != nil {
		log.Errorf("agent-test execution failed: %v", err)
		return fmt.Errorf("agent-test failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Debugf("displaying test results (aggregated score: %d)", aggregatedScore)
	displayReviewTestResults(results, aggregatedScore, runs, elapsed)

	log.Infof("agent-test review completed successfully (aggregated score: %d/100)", aggregatedScore)
	return nil
}

// backupFileIfExists backs up path to path.<MM-DD-HH-MM-SS> if it exists and is a regular file.
func backupFileIfExists(paths ...string) {
	timeSuffix := time.Now().Format("01-02-15-04-05")
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			backupPath := path + "." + timeSuffix
			data, err := os.ReadFile(path)
			if err != nil {
				log.Debugf("failed to read %s for backup: %v", path, err)
			} else if err := os.WriteFile(backupPath, data, 0644); err != nil {
				log.Debugf("failed to write backup %s: %v", backupPath, err)
			} else {
				log.Debugf("backed up %s to %s", path, filepath.Base(backupPath))
			}
		}
	}
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
// It reuses RunAgentReview (or RunAgentReviewUseAgentMd when useLocalOrchestration is false) for each run,
// aggregates JSON results (for same msgid takes lowest score), and saves one aggregated
// JSON at the end. No per-run backup.
// Returns scores for each run, aggregated score (from merged JSON), and error.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// useLocalOrchestration: if true, use local orchestration; otherwise use agent with po/AGENTS.md.
func RunAgentTestReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, runs int, outputBase string, useLocalOrchestration bool, batchSize int) ([]TestRunResult, int, error) {
	ps := GetReviewPathSet()
	// Determine the agent to use
	_, err := SelectAgent(cfg, agentName)
	if err != nil {
		return nil, 0, err
	}

	// Run the test multiple times
	results := make([]TestRunResult, runs)
	var reviewJSONs []*ReviewJSONResult

	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("run %d/%d", runNum, runs)

		// Start timing for this iteration
		iterStartTime := time.Now()

		cleanReviewOutputFilesForTest(ps)

		// Reuse RunAgentReview or RunAgentReviewUseAgentMd for each run
		var agentResult *AgentRunResult
		if useLocalOrchestration {
			agentResult, err = RunAgentReviewLocalOrchestration(cfg, agentName, target, true, batchSize)
		} else {
			agentResult, err = RunAgentReview(cfg, agentName, target, true)
		}

		// Calculate execution time for this iteration
		iterExecutionTime := time.Since(iterStartTime)

		// Convert AgentRunResult to TestRunResult (embedding avoids field duplication)
		result := TestRunResult{
			AgentRunResult: *agentResult,
			RunNumber:      runNum,
		}
		result.ExecutionTime = iterExecutionTime

		// Record per-run score and collect JSON for aggregation
		if agentResult.ReviewReport.ReviewResult != nil {
			result.Score = agentResult.ReviewReport.Score
			reviewJSONs = append(reviewJSONs, agentResult.ReviewReport.ReviewResult)
			log.Debugf("run %d: review score from JSON: %d (total_entries=%d, issues=%d)",
				runNum, agentResult.ReviewReport.Score, agentResult.ReviewReport.ReviewResult.TotalEntries, len(agentResult.ReviewReport.ReviewResult.Issues))
		} else if agentResult.AgentError == nil {
			log.Debugf("run %d: agent succeeded but no review JSON found, score=0", runNum)
			result.Score = 0
		} else {
			log.Debugf("run %d: agent execution failed, score=0", runNum)
			result.Score = 0
		}

		if err != nil {
			log.Debugf("run %d: agent-run returned error: %v", runNum, err)
		}

		results[i] = result
		log.Debugf("run %d: completed with score %d", runNum, result.Score)
	}

	// Aggregate JSONs (for same msgid, take lowest score) and save
	aggregatedScore := 0
	aggregated := aggregateReviewJSONResult(reviewJSONs, false)
	if aggregated != nil {
		// Update TotalEntries from ps.InputPO (same as ReportReviewFromPathWithBatches)
		if Exist(ps.InputPO) {
			if stats, err := CountReportStats(ps.InputPO); err != nil {
				log.Warnf("failed to count entries in %s: %v", ps.InputPO, err)
			} else {
				aggregated.TotalEntries = stats.Total()
			}
		}
		var scoreErr error
		aggregatedScore, scoreErr = CalculateReviewScore(aggregated)
		if scoreErr != nil {
			log.Warnf("failed to calculate aggregated review score: %v", scoreErr)
		} else {
			log.Infof("aggregated review: score=%d/100 (from %d runs, %d unique issues)",
				aggregatedScore, len(reviewJSONs), len(aggregated.Issues))
			if err := saveReviewJSON(aggregated, ps.ResultJSON); err != nil {
				log.Warnf("failed to save aggregated review JSON: %v", err)
			}
		}
		// Apply aggregated review to generate ps.OutputPO
		if Exist(ps.InputPO) {
			if _, err := ApplyReviewFromResultJSON(ps); err != nil {
				log.Warnf("failed to apply aggregated review to %s: %v", ps.OutputPO, err)
			}
		}
	}

	log.Infof("all runs completed. Aggregated score: %d/100", aggregatedScore)
	return results, aggregatedScore, nil
}

// SaveReviewResults saves review results to output directory for agent-test review.
// It creates the output directory structure, copies the PO file and JSON file,
// and saves the execution log. Files are overwritten if the directory exists.
// Returns error if any operation fails.
func SaveReviewResults(agentName string, runNumber int, poFile string, jsonFile string, stdout, stderr []byte) error {
	// Determine output directory path
	workDir := repository.WorkDirOrCwd()
	outputDir := filepath.Join(workDir, "output", agentName, fmt.Sprintf("%d", runNumber))

	log.Debugf("saving review results to %s", outputDir)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Copy PO file to output directory as XX-reviewed.po
	poFileName := filepath.Base(poFile)
	langCode := strings.TrimSuffix(poFileName, ".po")
	if langCode == "" || langCode == poFileName {
		return fmt.Errorf("invalid PO file path: %s (expected format: po/XX.po)", poFile)
	}
	destPoFile := filepath.Join(outputDir, fmt.Sprintf("%s-reviewed.po", langCode))

	log.Debugf("copying %s to %s", poFile, destPoFile)

	// Read source PO file
	poData, err := os.ReadFile(poFile)
	if err != nil {
		return fmt.Errorf("failed to read PO file %s: %w", poFile, err)
	}

	// Write to destination
	if err := os.WriteFile(destPoFile, poData, 0644); err != nil {
		return fmt.Errorf("failed to write PO file to %s: %w", destPoFile, err)
	}

	// Copy JSON file to output directory as XX-reviewed.json
	if jsonFile != "" {
		destJSONFile := filepath.Join(outputDir, fmt.Sprintf("%s-reviewed.json", langCode))

		log.Debugf("copying %s to %s", jsonFile, destJSONFile)

		// Read source JSON file
		jsonData, err := os.ReadFile(jsonFile)
		if err != nil {
			return fmt.Errorf("failed to read JSON file %s: %w", jsonFile, err)
		}

		// Write to destination
		if err := os.WriteFile(destJSONFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write JSON file to %s: %w", destJSONFile, err)
		}
	}

	// Save execution log (stdout + stderr) to review.log
	logFile := filepath.Join(outputDir, "review.log")
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

	log.Infof("review results saved to %s", outputDir)
	return nil
}

// displayReviewTestResults displays the review test results in a readable format.
// Output format matches other agent-test subcommands. Shows per-run scores and
// aggregated score (from merged JSON, same msgid takes lowest score) in summary.
func displayReviewTestResults(results []TestRunResult, aggregatedScore int, totalRuns int, elapsed time.Duration) {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Agent Test Results (Review)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	successCount := 0
	failureCount := 0

	// Display individual run results (same format as translate/update-po)
	for _, result := range results {
		status := "FAIL"
		if result.Score > 0 {
			status = "PASS"
			successCount++
		} else {
			failureCount++
		}

		fmt.Printf("Run %d: %s (Score: %d/100)\n", result.RunNumber, status, result.Score)

		if result.AgentExecuted {
			if result.AgentError == nil {
				fmt.Printf("  Agent execution: PASS\n")
			} else {
				fmt.Printf("  Agent execution: FAIL - %v\n", result.AgentError)
			}

			if result.Score > 0 {
				fmt.Printf("  Review status:   PASS (valid JSON with score %d/100)\n", result.Score)
			} else {
				fmt.Printf("  Review status:   FAIL (no valid JSON or agent failed)\n")
			}
		} else {
			fmt.Printf("  Agent execution: SKIPPED\n")
		}

		fmt.Println()
	}

	// Calculate statistics for NumTurns and execution time
	var numTurnsList []int
	var executionTimes []time.Duration
	var runScores []string
	totalNumTurns := 0
	totalExecutionTime := time.Duration(0)
	numTurnsCount := 0

	for _, result := range results {
		runScores = append(runScores, fmt.Sprintf("%d", result.Score))
		if result.NumTurns > 0 {
			numTurnsList = append(numTurnsList, result.NumTurns)
			totalNumTurns += result.NumTurns
			numTurnsCount++
		}
		executionTimes = append(executionTimes, result.ExecutionTime)
		totalExecutionTime += result.ExecutionTime
	}

	// Display summary statistics (aggregated score as total, per-run in parentheses)
	const labelWidth = 25
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Summary")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("%-*s %d\n", labelWidth, "Total runs:", totalRuns)
	fmt.Printf("%-*s %d\n", labelWidth, "Successful runs:", successCount)
	fmt.Printf("%-*s %d\n", labelWidth, "Failed runs:", failureCount)
	fmt.Printf("%-*s %d/100 (%s)\n", labelWidth, "Aggregated score:", aggregatedScore, strings.Join(runScores, ", "))

	if numTurnsCount > 0 {
		avgNumTurns := totalNumTurns / numTurnsCount
		var numTurnsStrs []string
		for _, nt := range numTurnsList {
			numTurnsStrs = append(numTurnsStrs, fmt.Sprintf("%d", nt))
		}
		fmt.Printf("%-*s %d (%s)\n", labelWidth, "Avg Num turns:", avgNumTurns, strings.Join(numTurnsStrs, ", "))
	}

	if len(executionTimes) > 0 {
		avgExecutionTime := totalExecutionTime / time.Duration(len(executionTimes))
		var execTimeStrs []string
		for _, et := range executionTimes {
			execTimeStrs = append(execTimeStrs, formatDuration(et))
		}
		fmt.Printf("%-*s %s (%s)\n", labelWidth, "Avg Execution Time:", formatDuration(avgExecutionTime), strings.Join(execTimeStrs, ", "))
	}

	fmt.Printf("%-*s %s\n", labelWidth, "Total Elapsed Time:", formatDuration(elapsed))
	fmt.Println("=" + strings.Repeat("=", 70))
}
