// Package util provides business logic for agent-test command.
package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
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

// TestRunResult holds the result of a single test run.
// It embeds AgentRunResult so agent-run fields (Error, ReviewResult, NumTurns, etc.)
// are inherited; RunNumber is test-specific.
// Ctx holds PreCheckResult/PostCheckResult for display.
// ReportOutput holds captured stdout from workflow Report when agent-test runs with a buffer.
type TestRunResult struct {
	AgentRunResult
	RunNumber    int              // Test run index (1-based)
	Ctx          *AgentRunContext // Context after PreCheck/AgentRun/PostCheck
	ReportOutput string           // Captured Report output when run via agent-test loop (optional)
}

// Success returns true when Ctx.Success() (or, if Ctx is nil, when Result.Error is nil).
func (r *TestRunResult) Success() bool {
	if r.Ctx != nil {
		return r.Ctx.Success()
	}
	return r.Error == nil
}

// Score returns 0 when !Success(); when Success(), 100 for update-pot/update-po/translate,
// or ReviewResult.GetScore() for review. Caller may ignore error and treat as 0.
func (r *TestRunResult) Score() (int, error) {
	if !r.Success() {
		return 0, nil
	}
	if r.ReviewResult != nil {
		return r.ReviewResult.GetScore()
	}
	return 100, nil
}

// AverageScoreFromResults returns the average Score of results (0 if empty).
func AverageScoreFromResults(results []TestRunResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0
	for i := range results {
		s, _ := results[i].Score()
		sum += s
	}
	return float64(sum) / float64(len(results))
}

// PrintAgentTestSummaryReport prints the common agent-test summary (Total runs,
// Successful/Failed runs, Average score, pre/post-validation failures, Avg Num turns,
// Avg Execution Time, Total Elapsed Time) using the same format as workflow Report
// (ReportLabelWidth, two-space indent). Call this after all loops in the workflow.
func PrintAgentTestSummaryReport(results []TestRunResult, elapsed time.Duration) {
	runs := len(results)
	successCount := 0
	var totalNumTurns int
	var totalExecutionTime time.Duration
	if runs == 0 {
		log.Warnf("no results for summary")
		return
	}

	var runScores []string
	var numTurnsStrs []string
	var executionTimeStrs []string
	uniqueErrors := make(map[string]struct{}) // dedupe by error string
	for _, result := range results {
		s, _ := result.Score()
		runScores = append(runScores, fmt.Sprintf("%d", s))
		numTurnsStrs = append(numTurnsStrs, fmt.Sprintf("%d", result.NumTurns))
		executionTimeStrs = append(executionTimeStrs, formatDuration(result.ExecutionTime))
		if result.Success() {
			successCount++
		} else {
			// Collect the three error types from this result, dedupe by message
			if result.Error != nil {
				uniqueErrors[result.Error.Error()] = struct{}{}
			}
			if result.Ctx != nil {
				if result.Ctx.PreCheckResult != nil && result.Ctx.PreCheckResult.Error != nil {
					uniqueErrors[result.Ctx.PreCheckResult.Error.Error()] = struct{}{}
				}
				if result.Ctx.PostCheckResult != nil && result.Ctx.PostCheckResult.Error != nil {
					uniqueErrors[result.Ctx.PostCheckResult.Error.Error()] = struct{}{}
				}
			}
		}
		totalNumTurns += result.NumTurns
		totalExecutionTime += result.ExecutionTime
	}

	labelWidth := ReportLabelWidth
	fmt.Printf("  %-*s %d\n", labelWidth, "Total runs:", runs)
	if successCount > 0 {
		fmt.Printf("  %-*s %d ✅\n", labelWidth, "Successful runs:", successCount)
	}
	if runs-successCount > 0 {
		fmt.Printf("  %-*s %d ❌\n", labelWidth, "Failed runs:", runs-successCount)
	}
	fmt.Println()
	fmt.Printf("  %-*s %d (%s)\n", labelWidth, "Avg Num turns:",
		totalNumTurns/runs, strings.Join(numTurnsStrs, ", "))
	fmt.Printf("  %-*s %s (%s)\n", labelWidth, "Avg Execution Time:",
		formatDuration(totalExecutionTime/time.Duration(runs)), strings.Join(executionTimeStrs, ", "))
	fmt.Printf("  %-*s %.0f/100 (%s)\n", labelWidth, "Average score:",
		AverageScoreFromResults(results), strings.Join(runScores, ", "))
	fmt.Println()
	if len(uniqueErrors) > 0 {
		errStrs := make([]string, 0, len(uniqueErrors))
		for s := range uniqueErrors {
			errStrs = append(errStrs, s)
		}
		sort.Strings(errStrs)
		fmt.Println()
		fmt.Printf("  ❌ Error found\n")
		fmt.Println()
		for _, errStr := range errStrs {
			fmt.Printf("  %-*s %s\n", labelWidth, "Error:", errStr)
		}
		fmt.Println()
	}
	fmt.Printf("  %-*s %s\n", labelWidth, "Total Elapsed Time:", formatDuration(elapsed))
	fmt.Println()
	flushStdout()
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

// ResolveAgentTestRuns returns the effective run count for agent-test commands.
// If runs > 0, that value is used (from command line). Otherwise uses cfg.AgentTest.Runs
// when set and positive, or defaults to 3. Logs the source at debug level.
func ResolveAgentTestRuns(cfg *config.AgentConfig, runs int) int {
	if runs != 0 {
		log.Debugf("using runs from command line: %d", runs)
		return runs
	}
	if cfg.AgentTest.Runs != nil && *cfg.AgentTest.Runs > 0 {
		runs = *cfg.AgentTest.Runs
		log.Debugf("using runs from configuration: %d", runs)
		return runs
	}
	runs = 3
	log.Debugf("using default number of runs: %d", runs)
	return runs
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

// CleanPoDirectory restores the po/ directory to its state in HEAD using git restore.
// This is useful for agent-test operations to ensure a clean state before each test run.
// Returns an error if the git restore command fails.
func CleanPoDirectory(paths ...string) error {
	// If no paths provided, do not reset for security
	targetPaths := paths

	log.Debugf("cleaning paths using git restore (paths: %v)", targetPaths)

	// Process each path individually to avoid failures on non-existent paths
	for _, path := range targetPaths {
		log.Debugf("restoring path: %s", path)

		// Backup .po files before restore to protect modified content
		backupFileIfExists(path)

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
