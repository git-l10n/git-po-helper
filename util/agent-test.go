// Package util provides business logic for agent-test command.
package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// formatDuration formats a duration in a user-friendly way.
// For durations >= 60 seconds, uses format like "1h4m50s".
// For durations < 60 seconds, uses format like "45s".
// Seconds are rounded to integers (no decimal precision).
func formatDuration(d time.Duration) string {
	// Round to nearest second
	d = d.Round(time.Second)

	// If less than 60 seconds, just show seconds
	if d < 60*time.Second {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	// For >= 60 seconds, use hours, minutes, seconds format
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, "")
}

// RunResult holds the result of a single test run.
type RunResult struct {
	RunNumber           int
	Score               int
	PreValidationPass   bool
	PostValidationPass  bool
	AgentExecuted       bool
	AgentSuccess        bool
	PreValidationError  string
	PostValidationError string
	AgentError          string
	BeforeCount         int
	AfterCount          int
	BeforeNewCount      int // For translate: new (untranslated) entries before
	AfterNewCount       int // For translate: new (untranslated) entries after
	BeforeFuzzyCount    int // For translate: fuzzy entries before
	AfterFuzzyCount     int // For translate: fuzzy entries after
	ExpectedBefore      *int
	ExpectedAfter       *int
	NumTurns            int           // Number of turns in the conversation
	ExecutionTime       time.Duration // Execution time for this run
}

// ConfirmAgentTestExecution displays a warning and requires user confirmation before proceeding.
// The user must explicitly type "yes" to continue, otherwise the function returns an error.
// This is used to prevent accidental data loss when agent-test commands modify po/ directory.
// If skipConfirmation is true, the confirmation prompt is skipped.
func ConfirmAgentTestExecution(skipConfirmation bool) error {
	if skipConfirmation {
		log.Debugf("skipping confirmation prompt (--dangerously-remove-po-directory flag set)")
		return nil
	}

	fmt.Fprintln(os.Stderr, "WARNING: This command will modify files under po/ and may cause data loss.")
	fmt.Fprint(os.Stderr, "Are you sure you want to continue? Type 'yes' to proceed: ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "yes" {
		return fmt.Errorf("operation cancelled by user")
	}

	return nil
}

// CleanPoDirectory restores the po/ directory to its state in HEAD using git restore.
// This is useful for agent-test operations to ensure a clean state before each test run.
// Returns an error if the git restore command fails.
func CleanPoDirectory(paths ...string) error {
	workDir := repository.WorkDir()

	// If no paths provided, use default "po/"
	targetPaths := paths
	if len(targetPaths) == 0 {
		targetPaths = []string{"po/"}
	}

	log.Debugf("cleaning paths using git restore (workDir: %s, paths: %v)", workDir, targetPaths)

	// Process each path individually to avoid failures on non-existent paths
	for _, path := range targetPaths {
		log.Debugf("restoring path: %s", path)

		// Build git restore command for this path
		args := []string{
			"restore",
			"--staged",
			"--worktree",
			"--source", "HEAD",
			"--",
			path,
		}

		cmd := exec.Command("git", args...)
		cmd.Dir = workDir

		// Capture stderr for error messages
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Debugf("failed to create stderr pipe for git restore on path %s: %v", path, err)
			continue // Skip this path and continue with next
		}

		if err := cmd.Start(); err != nil {
			log.Debugf("failed to start git restore command for path %s: %v", path, err)
			continue // Skip this path and continue with next
		}

		// Read stderr output
		var stderrOutput strings.Builder
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				stderrOutput.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}

		if err := cmd.Wait(); err != nil {
			// Ignore errors for individual paths (path might not exist in repository)
			errorMsg := stderrOutput.String()
			if errorMsg == "" {
				errorMsg = err.Error()
			}
			log.Debugf("git restore failed for path %s (ignored): %s", path, errorMsg)
		} else {
			log.Debugf("path %s restored successfully", path)
		}
	}

	log.Debugf("all paths processed")

	// Clean untracked po/git.pot file that might not be in git repository
	// Only clean po/git.pot if default path "po/" is being used or explicitly specified
	shouldCleanPot := len(paths) == 0 || containsPath(paths, "po/") || containsPath(paths, "po/git.pot")
	if shouldCleanPot {
		log.Debugf("cleaning untracked po/git.pot file using git clean")
		cleanCmd := exec.Command("git",
			"clean",
			"-fx",
			"--",
			"po/git.pot")
		cleanCmd.Dir = workDir

		// Capture stderr for error messages
		cleanStderr, err := cleanCmd.StderrPipe()
		if err != nil {
			log.Warnf("failed to create stderr pipe for git clean: %v", err)
			// Continue even if we can't capture stderr
		} else {
			if err := cleanCmd.Start(); err != nil {
				log.Warnf("failed to start git clean command: %v", err)
				// Continue even if git clean fails
			} else {
				// Read stderr output
				var cleanStderrOutput strings.Builder
				buf := make([]byte, 1024)
				for {
					n, err := cleanStderr.Read(buf)
					if n > 0 {
						cleanStderrOutput.Write(buf[:n])
					}
					if err != nil {
						break
					}
				}

				if err := cleanCmd.Wait(); err != nil {
					// git clean may fail if there's nothing to clean, which is fine
					errorMsg := cleanStderrOutput.String()
					if errorMsg != "" {
						log.Debugf("git clean output: %s", errorMsg)
					}
					log.Debugf("git clean completed (exit code may be non-zero if nothing to clean)")
				} else {
					log.Debugf("untracked po/git.pot file cleaned successfully")
				}
			}
		}
	} else {
		log.Debugf("skipping po/git.pot cleanup (not in specified paths)")
	}

	log.Debugf("paths cleaned successfully")
	return nil
}

// containsPath checks if a path exists in the paths slice (exact match or prefix match).
func containsPath(paths []string, target string) bool {
	for _, p := range paths {
		if p == target || strings.HasPrefix(target, p) || strings.HasPrefix(p, target) {
			return true
		}
	}
	return false
}

// CmdAgentTestUpdatePot implements the agent-test update-pot command logic.
// It runs the agent-run update-pot operation multiple times and calculates an average score.
func CmdAgentTestUpdatePot(agentName string, runs int, skipConfirmation bool) error {
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

	log.Infof("starting agent-test update-pot with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, averageScore, err := RunAgentTestUpdatePot(agentName, runs, cfg)
	if err != nil {
		log.Errorf("agent-test execution failed: %v", err)
		return fmt.Errorf("agent-test failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Debugf("displaying test results (average score: %.2f)", averageScore)
	displayTestResults(results, averageScore, runs, elapsed)

	log.Infof("agent-test update-pot completed successfully (average score: %.2f/100)", averageScore)
	return nil
}

// RunAgentTestUpdatePot runs the agent-test update-pot operation multiple times.
// It reuses RunAgentUpdatePot for each run and accumulates scores.
// Returns scores for each run, average score, and error.
func RunAgentTestUpdatePot(agentName string, runs int, cfg *config.AgentConfig) ([]RunResult, float64, error) {
	// Run the test multiple times
	results := make([]RunResult, runs)
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

		// Reuse RunAgentUpdatePot for each run
		agentResult, err := RunAgentUpdatePot(cfg, agentName, true)

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
			AgentSuccess:        agentResult.AgentSuccess,
			PreValidationError:  agentResult.PreValidationError,
			PostValidationError: agentResult.PostValidationError,
			AgentError:          agentResult.AgentError,
			BeforeCount:         agentResult.BeforeCount,
			AfterCount:          agentResult.AfterCount,
			ExpectedBefore:      cfg.AgentTest.PotEntriesBeforeUpdate,
			ExpectedAfter:       cfg.AgentTest.PotEntriesAfterUpdate,
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

// CmdAgentTestUpdatePo implements the agent-test update-po command logic.
// It runs the agent-run update-po operation multiple times and calculates an average score.
func CmdAgentTestUpdatePo(agentName, poFile string, runs int, skipConfirmation bool) error {
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

	log.Infof("starting agent-test update-po with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, averageScore, err := RunAgentTestUpdatePo(agentName, poFile, runs, cfg)
	if err != nil {
		log.Errorf("agent-test execution failed: %v", err)
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
			AgentSuccess:        agentResult.AgentSuccess,
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

// displayTestResults displays the test results in a readable format.
func displayTestResults(results []RunResult, averageScore float64, totalRuns int, elapsed time.Duration) {
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Agent Test Results")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	successCount := 0
	failureCount := 0
	preValidationFailures := 0
	postValidationFailures := 0

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

		// Show validation status
		if result.ExpectedBefore != nil && *result.ExpectedBefore != 0 {
			if result.PreValidationPass {
				fmt.Printf("  Pre-validation:  PASS (expected: %d, actual: %d)\n",
					*result.ExpectedBefore, result.BeforeCount)
			} else {
				fmt.Printf("  Pre-validation:  FAIL - %s\n", result.PreValidationError)
				preValidationFailures++
			}
		}

		if result.AgentExecuted {
			if result.AgentSuccess {
				fmt.Printf("  Agent execution: PASS\n")
			} else {
				fmt.Printf("  Agent execution: FAIL - %s\n", result.AgentError)
			}
		} else {
			fmt.Printf("  Agent execution: SKIPPED (pre-validation failed)\n")
		}

		if result.ExpectedAfter != nil && *result.ExpectedAfter != 0 {
			if result.PostValidationPass {
				fmt.Printf("  Post-validation: PASS (expected: %d, actual: %d)\n",
					*result.ExpectedAfter, result.AfterCount)
			} else {
				fmt.Printf("  Post-validation: FAIL - %s\n", result.PostValidationError)
				postValidationFailures++
			}
		} else if result.AgentExecuted {
			// Show entry counts even if validation is not configured
			fmt.Printf("  Entry count:     %d (before) -> %d (after)\n",
				result.BeforeCount, result.AfterCount)
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
	if preValidationFailures > 0 {
		fmt.Printf("%-*s %d\n", labelWidth, "Pre-validation failures:", preValidationFailures)
	}
	if postValidationFailures > 0 {
		fmt.Printf("%-*s %d\n", labelWidth, "Post-validation failures:", postValidationFailures)
	}
	fmt.Printf("%-*s %.2f/100\n", labelWidth, "Average score:", averageScore)

	// Display NumTurns statistics
	if numTurnsCount > 0 {
		avgNumTurns := totalNumTurns / numTurnsCount
		var numTurnsStrs []string
		for _, turns := range numTurnsList {
			turnsStr := fmt.Sprintf("%d", turns)
			numTurnsStrs = append(numTurnsStrs, turnsStr)
		}
		fmt.Printf("%-*s %d (%s)\n", labelWidth, "Avg Num turns:", avgNumTurns, strings.Join(numTurnsStrs, ", "))
	}

	// Display execution time statistics
	if len(executionTimes) > 0 {
		avgExecutionTime := totalExecutionTime / time.Duration(len(executionTimes))
		var execTimeStrs []string
		avgTimeStr := formatDuration(avgExecutionTime)
		for _, execTime := range executionTimes {
			timeStr := formatDuration(execTime)
			execTimeStrs = append(execTimeStrs, timeStr)
		}
		fmt.Printf("%-*s %s (%s)\n", labelWidth, "Avg Execution Time:", avgTimeStr, strings.Join(execTimeStrs, ", "))
	}

	// Always display total elapsed time
	fmt.Printf("%-*s %s\n", labelWidth, "Total Elapsed Time:", formatDuration(elapsed))
	fmt.Println("=" + strings.Repeat("=", 70))
}

// CmdAgentTestShowConfig displays the current agent configuration in YAML format.
// It reuses CmdAgentRunShowConfig from agent-run.
func CmdAgentTestShowConfig() error {
	return CmdAgentRunShowConfig()
}

// SaveTranslateResults saves the translation results to the output directory.
// It creates output/<agent-name>/<run-number>/ directory and copies the PO file
// and execution logs to preserve translation results for later review.
func SaveTranslateResults(agentName string, runNumber int, poFile string, stdout, stderr []byte) error {
	// Determine output directory path
	workDir := repository.WorkDir()
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

// CmdAgentTestTranslate implements the agent-test translate command logic.
// It runs the agent-run translate operation multiple times and calculates an average score.
func CmdAgentTestTranslate(agentName, poFile string, runs int, skipConfirmation bool) error {
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

	log.Infof("starting agent-test translate with %d runs", runs)

	startTime := time.Now()

	// Run the test
	results, averageScore, err := RunAgentTestTranslate(agentName, poFile, runs, cfg)
	if err != nil {
		log.Errorf("agent-test execution failed: %v", err)
		return fmt.Errorf("agent-test failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Debugf("displaying test results (average score: %.2f)", averageScore)
	displayTranslateTestResults(results, averageScore, runs, elapsed)

	log.Infof("agent-test translate completed successfully (average score: %.2f/100)", averageScore)
	return nil
}

// RunAgentTestTranslate runs the agent-test translate operation multiple times.
// It reuses RunAgentTranslate for each run and accumulates scores.
// Returns scores for each run, average score, and error.
func RunAgentTestTranslate(agentName, poFile string, runs int, cfg *config.AgentConfig) ([]RunResult, float64, error) {
	// Determine the agent to use (for saving results)
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return nil, 0, err
	}
	_ = selectedAgent // Avoid unused variable warning

	// Determine PO file path
	poFile, err = GetPoFileAbsPath(cfg, poFile)
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
	totalScore := 0
	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("run %d/%d", runNum, runs)

		// Start timing for this iteration
		iterStartTime := time.Now()

		if err := CleanPoDirectory(relPoFile); err != nil {
			log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
			// Continue with the run even if cleanup fails, but log the warning
		}

		// Reuse RunAgentTranslate for each run
		agentResult, err := RunAgentTranslate(cfg, agentName, poFile, true)

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
			ExpectedBefore:      nil, // Not used for translate
			ExpectedAfter:       nil, // Not used for translate
			NumTurns:            agentResult.NumTurns,
			ExecutionTime:       iterExecutionTime,
		}

		// If there was an error, log it but continue (for agent-test, we want to collect all results)
		if err != nil {
			log.Debugf("run %d: agent-run returned error: %v", runNum, err)
			// Error details are already in the result structure
		}

		// Save translation results to output directory (ignore errors)
		if err := SaveTranslateResults(agentName, runNum, poFile, nil, nil); err != nil {
			log.Warnf("run %d: failed to save translation results: %v", runNum, err)
			// Continue even if saving results fails
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

// displayTranslateTestResults displays the translation test results in a readable format.
func displayTranslateTestResults(results []RunResult, averageScore float64, totalRuns int, elapsed time.Duration) {
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
			fmt.Printf("  New entries:     %d (before) -> %d (after)\n",
				result.BeforeNewCount, result.AfterNewCount)
			fmt.Printf("  Fuzzy entries:   %d (before) -> %d (after)\n",
				result.BeforeFuzzyCount, result.AfterFuzzyCount)

			if result.AgentSuccess {
				fmt.Printf("  Agent execution: PASS\n")
			} else {
				fmt.Printf("  Agent execution: FAIL - %s\n", result.AgentError)
			}

			if result.PostValidationPass {
				fmt.Printf("  Validation:      PASS (all entries translated)\n")
			} else {
				fmt.Printf("  Validation:      FAIL - %s\n", result.PostValidationError)
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
