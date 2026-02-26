// Package util provides business logic for agent-test review command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// CmdAgentTestReview implements the agent-test review command logic.
// It runs the agent-run review operation multiple times and calculates an average score.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// allWithLLM: if true, use pure LLM approach (--all-with-llm).
func CmdAgentTestReview(agentName string, target *CompareTarget, runs int, skipConfirmation bool, outputBase string, allWithLLM bool) error {
	// Require user confirmation before proceeding
	if err := ConfirmAgentTestExecution(skipConfirmation); err != nil {
		return err
	}

	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
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
	results, aggregatedScore, err := RunAgentTestReview(cfg, agentName, target, runs, outputBase, allWithLLM)
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

// RunAgentTestReview runs the agent-test review operation multiple times.
// It reuses RunAgentReview (or RunAgentReviewAllWithLLM when allWithLLM) for each run,
// aggregates JSON results (for same msgid takes lowest score), and saves one aggregated
// JSON at the end. No per-run backup.
// Returns scores for each run, aggregated score (from merged JSON), and error.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// allWithLLM: if true, use pure LLM approach (--all-with-llm).
func RunAgentTestReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, runs int, outputBase string, allWithLLM bool) ([]RunResult, int, error) {
	_, reviewJSONFile := ReviewOutputPaths(outputBase)
	// Determine the agent to use
	_, err := SelectAgent(cfg, agentName)
	if err != nil {
		return nil, 0, err
	}

	// Determine PO file path (use newFile as the file being reviewed)
	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return nil, 0, err
	}

	// Will clean poFile using relative path
	relPoFile, err := GetPoFileRelPath(cfg, poFile)
	if err != nil {
		log.Warnf("failed to get relative path of poFile: %v", err)
	}

	// Run the test multiple times
	results := make([]RunResult, runs)
	var reviewJSONs []*ReviewJSONResult

	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("run %d/%d", runNum, runs)

		// Start timing for this iteration
		iterStartTime := time.Now()

		if err := CleanPoDirectory(relPoFile); err != nil {
			log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
		}

		// Reuse RunAgentReview or RunAgentReviewAllWithLLM for each run
		var agentResult *AgentRunResult
		if allWithLLM {
			agentResult, err = RunAgentReviewAllWithLLM(cfg, agentName, target, true, outputBase)
		} else {
			agentResult, err = RunAgentReview(cfg, agentName, target, true, outputBase)
		}

		// Calculate execution time for this iteration
		iterExecutionTime := time.Since(iterStartTime)

		// Convert AgentRunResult to RunResult
		result := RunResult{
			RunNumber:           runNum,
			PreValidationPass:   agentResult.PreValidationPass,
			PostValidationPass:  agentResult.PostValidationPass,
			AgentExecuted:       agentResult.AgentExecuted,
			AgentSuccess:        agentResult.AgentSuccess,
			PreValidationError:  agentResult.PreValidationError,
			PostValidationError: agentResult.PostValidationError,
			AgentError:          agentResult.AgentError,
			BeforeCount:         agentResult.BeforeCount,
			AfterCount:          agentResult.AfterCount,
			BeforeNewCount:      agentResult.BeforeNewCount,
			AfterNewCount:       agentResult.AfterNewCount,
			BeforeFuzzyCount:    agentResult.BeforeFuzzyCount,
			AfterFuzzyCount:     agentResult.AfterFuzzyCount,
			ExpectedBefore:      nil,
			ExpectedAfter:       nil,
			NumTurns:            agentResult.NumTurns,
			ExecutionTime:       iterExecutionTime,
		}

		// Record per-run score and collect JSON for aggregation
		if agentResult.ReviewJSON != nil {
			result.Score = agentResult.ReviewScore
			reviewJSONs = append(reviewJSONs, agentResult.ReviewJSON)
			log.Debugf("run %d: review score from JSON: %d (total_entries=%d, issues=%d)",
				runNum, agentResult.ReviewScore, agentResult.ReviewJSON.TotalEntries, len(agentResult.ReviewJSON.Issues))
		} else if agentResult.AgentSuccess {
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
	aggregated := AggregateReviewJSON(reviewJSONs)
	if aggregated != nil {
		var scoreErr error
		aggregatedScore, scoreErr = CalculateReviewScore(aggregated)
		if scoreErr != nil {
			log.Warnf("failed to calculate aggregated review score: %v", scoreErr)
		} else {
			log.Infof("aggregated review: score=%d/100 (from %d runs, %d unique issues)",
				aggregatedScore, len(reviewJSONs), len(aggregated.Issues))
			if err := saveReviewJSON(aggregated, reviewJSONFile); err != nil {
				log.Warnf("failed to save aggregated review JSON: %v", err)
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
	workDir := repository.WorkDir()
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
func displayReviewTestResults(results []RunResult, aggregatedScore int, totalRuns int, elapsed time.Duration) {
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
			if result.AgentSuccess {
				fmt.Printf("  Agent execution: PASS\n")
			} else {
				fmt.Printf("  Agent execution: FAIL - %s\n", result.AgentError)
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
