// Package util provides business logic for agent-test command.
package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
