// Package util provides business logic for agent-run review command.
package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// executeReviewAgent executes the agent command for reviewing the given file.
// vars contains placeholder values (e.g. "prompt", "source" for the file to review).
// Returns stdout (for JSON extraction), stderr, originalStdout (raw before parsing), streamResult.
// Updates result with AgentExecuted, AgentSuccess, AgentError, AgentStdout, AgentStderr.
func executeReviewAgent(selectedAgent config.Agent, vars PlaceholderVars, result *AgentRunResult) (stdout, stderr, originalStdout []byte, streamResult AgentStreamResult, err error) {
	agentCmd, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return nil, nil, nil, streamResult, fmt.Errorf("failed to build agent command: %w", err)
	}

	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	outputFormat = normalizeOutputFormat(outputFormat)

	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json", truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(agentCmd)
		if execErr != nil {
			log.Errorf("agent command execution failed: %v", execErr)
			return nil, nil, nil, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
		defer stdoutReader.Close()

		var stdoutBuf bytes.Buffer
		teeReader := io.TeeReader(stdoutReader, &stdoutBuf)

		stdout, streamResult, _ = parseStreamByKind(kind, teeReader)
		originalStdout = stdoutBuf.Bytes()

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var execErr error
		stdout, stderr, execErr = ExecuteAgentCommand(agentCmd)
		originalStdout = stdout
		result.AgentStdout = stdout
		result.AgentStderr = stderr

		if execErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", execErr)
			log.Errorf("agent command execution failed: %v", execErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, parseErr := ParseClaudeAgentOutput(stdout, outputFormat)
			if parseErr != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", parseErr)
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

	applyAgentDiagnostics(result, streamResult)
	result.AgentStdout = originalStdout
	if len(stderr) > 0 {
		result.AgentStderr = stderr
	}

	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	return stdout, stderr, originalStdout, streamResult, nil
}

// runReviewSingleBatch runs review on the full file (single batch).
func runReviewSingleBatch(selectedAgent config.Agent, vars PlaceholderVars, result *AgentRunResult, entryCount int) (*ReviewJSONResult, error) {
	stdout, _, _, _, err := executeReviewAgent(selectedAgent, vars, result)
	if err != nil {
		return nil, err
	}
	return parseAndAccumulateReviewJSON(stdout, entryCount)
}

// runReviewBatched runs review in batches using msg-select when entry count > 100.
func runReviewBatched(selectedAgent config.Agent, vars PlaceholderVars, result *AgentRunResult, entryCount int) (*ReviewJSONResult, error) {
	reviewPOFile := vars["source"]
	num := 50
	if entryCount > 500 {
		num = 100
	} else if entryCount > 200 {
		num = 75
	}

	batchFile := ReviewDefaultBatchFile

	var batchReviews []*ReviewJSONResult
	for batchNum := 1; ; batchNum++ {
		start := (batchNum-1)*num + 1
		end := batchNum * num
		if end > entryCount {
			end = entryCount
		}

		rangeSpec := formatMsgSelectRange(batchNum, start, end, entryCount, num)
		log.Infof("reviewing batch %d: entries %d-%d (of %d)", batchNum, start, end, entryCount)

		// Extract batch with msg-select
		f, err := os.Create(batchFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create batch file: %w", err)
		}
		if err := MsgSelect(reviewPOFile, rangeSpec, f, false); err != nil {
			f.Close()
			os.Remove(batchFile)
			return nil, fmt.Errorf("msg-select failed: %w", err)
		}
		f.Close()

		// Run agent on batch
		batchVars := make(PlaceholderVars)
		for k, v := range vars {
			batchVars[k] = v
		}
		batchVars["source"] = batchFile
		stdout, _, _, _, err := executeReviewAgent(selectedAgent, batchVars, result)
		os.Remove(batchFile) // Clean up batch file
		if err != nil {
			return nil, err
		}

		// Parse JSON and accumulate batch results
		batchJSON, err := parseAndAccumulateReviewJSON(stdout, entryCount)
		if err != nil {
			return nil, err
		}
		if batchJSON != nil {
			batchReviews = append(batchReviews, batchJSON)
		}

		if end >= entryCount {
			break
		}
	}

	// Merge issues using same logic as AggregateReviewJSON: keep the most severe
	// (lowest score) per msgid. We do not use simple array append because the
	// model may not follow instructions when executing; deduplication by msgid
	// with severity preference is required for consistent results.
	merged := AggregateReviewJSON(batchReviews, true)
	if merged == nil {
		return &ReviewJSONResult{TotalEntries: entryCount, Issues: []ReviewIssue{}}, nil
	}
	return merged, nil
}

// formatMsgSelectRange returns the range spec for msg-select (e.g. "-50", "51-100", "101-").
func formatMsgSelectRange(batchNum, start, end, entryCount, num int) string {
	if batchNum == 1 {
		return fmt.Sprintf("-%d", num)
	}
	if end >= entryCount {
		return fmt.Sprintf("%d-", start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

// parseAndAccumulateReviewJSON extracts and parses JSON from stdout, updates total_entries.
func parseAndAccumulateReviewJSON(stdout []byte, entryCount int) (*ReviewJSONResult, error) {
	jsonBytes, err := ExtractJSONFromOutput(stdout)
	if err != nil {
		log.Errorf("failed to extract JSON from agent output: %v", err)
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	reviewJSON, err := ParseReviewJSON(jsonBytes)
	if err != nil {
		log.Errorf("failed to parse review JSON: %v", err)
		return nil, fmt.Errorf("failed to parse review JSON: %w", err)
	}

	reviewJSON.TotalEntries = entryCount
	log.Debugf("parsed review JSON: total_entries=%d, issues=%d", reviewJSON.TotalEntries, len(reviewJSON.Issues))
	return reviewJSON, nil
}

// buildReviewAllWithLLMPrompt constructs a dynamic prompt for RunAgentReviewAllWithLLM
func buildReviewAllWithLLMPrompt(target *CompareTarget) string {
	var taskDesc string
	if target.OldFile != target.NewFile {
		taskDesc = fmt.Sprintf("Review %s changes between %s and %s", target.NewFile, target.OldFile, target.NewFile)
	} else if target.NewCommit != "" && (target.OldCommit == target.NewCommit+"~" ||
		target.OldCommit == target.NewCommit+"~1" ||
		target.OldCommit == target.NewCommit+"^") {
		taskDesc = fmt.Sprintf("Review %s changes in commit %s", target.NewFile, target.NewCommit)
	} else if target.NewCommit == "" && target.OldCommit != "" {
		if target.OldCommit == "HEAD" {
			taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
		} else {
			taskDesc = fmt.Sprintf("Review %s changes since commit %s", target.NewFile, target.OldCommit)
		}
	} else if target.OldCommit != "" && target.NewCommit != "" {
		taskDesc = fmt.Sprintf("Review %s changes in range %s..%s", target.NewFile, target.OldCommit, target.NewCommit)
	} else {
		taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
	}

	return taskDesc + " according to @po/AGENTS.md."
}

// RunAgentReviewAllWithLLM executes review using a pure LLM approach (--all-with-llm).
// No programmatic extraction or batching; the LLM does everything and writes review.json.
// Before execution: deletes review.po and review.json. After: expects review.json to exist.
func RunAgentReviewAllWithLLM(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	reviewPOFile, reviewJSONFile := ReviewOutputPaths(outputBase)
	workDir := repository.WorkDir()
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return result, err
	}
	if !Exist(poFile) {
		result.AgentError = fmt.Sprintf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	// Delete existing review output files
	os.Remove(reviewPOFile)
	os.Remove(reviewJSONFile)
	log.Infof("removed existing %s and %s", reviewPOFile, reviewJSONFile)

	poFileRel := target.NewFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		poFileRel = filepath.ToSlash(rel)
	}
	prompt := buildReviewAllWithLLMPrompt(target)
	agentCmd, err := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": poFileRel})
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	outputFormat := normalizeOutputFormat(selectedAgent.Output)
	if outputFormat == "" {
		outputFormat = "default"
	}

	log.Infof("executing agent command (all-with-llm, output=%s): %s", outputFormat, truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	var stdout, stderr []byte
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd)
		if err != nil {
			result.AgentError = err.Error()
			return result, fmt.Errorf("agent command failed: %w", err)
		}
		defer stdoutReader.Close()
		_, streamResult, _ := parseStreamByKind(kind, stdoutReader)
		applyAgentDiagnostics(result, streamResult)
		if waitErr := cmdProcess.Wait(); waitErr != nil {
			result.AgentError = waitErr.Error()
			return result, fmt.Errorf("agent command failed: %w", waitErr)
		}
		stderr = stderrBuf.Bytes()
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd)
		if err != nil {
			result.AgentError = err.Error()
			return result, err
		}
		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, _ := ParseClaudeAgentOutput(stdout, outputFormat)
			stdout = parsedStdout
			applyAgentDiagnostics(result, parsedResult)
		}
	}

	result.AgentSuccess = true
	log.Infof("agent command completed successfully")

	if len(stdout) > 0 {
		log.Debugf("agent stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent stderr: %s", string(stderr))
	}

	if !Exist(reviewJSONFile) {
		return result, fmt.Errorf("review JSON not generated at %s\nHint: The agent must write the review result to this file", reviewJSONFile)
	}

	reportResult, err := ReportReviewFromJSON(reviewJSONFile)
	if err != nil {
		return result, fmt.Errorf("failed to read review JSON: %w", err)
	}

	result.ReviewJSON = reportResult.Review
	result.ReviewJSONPath = reviewJSONFile
	result.ReviewScore = reportResult.Score
	result.Score = reportResult.Score
	result.ReviewedFilePath = poFile
	result.ExecutionTime = time.Since(startTime)

	log.Infof("review completed (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))

	return result, nil
}

// RunAgentReview executes a single agent-run review operation with the new workflow:
// 1. Prepare review data (orig.po, new.po, review-input.po)
// 2. Copy review-input.po to review-output.po
// 3. Execute agent to review and modify review-output.po
// 4. Merge review-output.po with new.po using msgcat
// 5. Parse JSON from agent output and calculate score
// Returns a result structure with detailed information.
// The agentTest parameter is provided for consistency, though this method
// does not use AgentTest configuration.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
func RunAgentReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	var (
		reviewPOFile, reviewJSONFile = ReviewOutputPaths(outputBase)
		reviewJSON                   *ReviewJSONResult
	)

	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Prepare review data
	log.Infof("preparing review data: %s", reviewPOFile)
	if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, reviewPOFile); err != nil {
		return result, fmt.Errorf("failed to prepare review data: %w", err)
	}

	// Get prompt.review
	prompt, err := GetRawPrompt(cfg, "review")
	if err != nil {
		return result, err
	}
	log.Debugf("using review prompt: %s", prompt)

	totalEntries, err := countMsgidEntries(reviewPOFile)
	if err != nil {
		log.Errorf("failed to count msgid entries in review input file: %v", err)
		return result, fmt.Errorf("failed to count entries: %w", err)
	}
	entryCount := totalEntries
	if entryCount > 0 {
		entryCount-- // Exclude header
	}

	if entryCount <= 100 {
		reviewVars := PlaceholderVars{
			"prompt": prompt,
			"source": reviewPOFile,
			"dest":   reviewPOFile,
			"json":   reviewJSONFile,
		}
		resolvedPrompt, err := ExecutePromptTemplate(prompt, reviewVars)
		if err != nil {
			return result, fmt.Errorf("failed to resolve prompt template: %w", err)
		}
		reviewVars["prompt"] = resolvedPrompt

		// Single run: review entire file
		reviewJSON, err = runReviewSingleBatch(selectedAgent, reviewVars, result, entryCount)
		if err != nil {
			return result, err
		}
	} else {
		reviewVars := PlaceholderVars{
			"prompt": prompt,
			"source": ReviewDefaultBatchFile,
			"dest":   reviewPOFile,
			"json":   reviewJSONFile,
		}
		resolvedPrompt, err := ExecutePromptTemplate(prompt, reviewVars)
		if err != nil {
			return result, fmt.Errorf("failed to resolve prompt template: %w", err)
		}
		reviewVars["prompt"] = resolvedPrompt

		// Batch mode: iterate with msg-select
		reviewJSON, err = runReviewBatched(selectedAgent, reviewVars, result, entryCount)
		if err != nil {
			return result, err
		}
	}

	// Save JSON to file
	log.Infof("saving review JSON to %s", reviewJSONFile)
	if err := saveReviewJSON(reviewJSON, reviewJSONFile); err != nil {
		log.Errorf("failed to save review JSON: %v", err)
		return result, fmt.Errorf("failed to save review JSON: %w", err)
	}
	result.ReviewJSON = reviewJSON
	result.ReviewJSONPath = reviewJSONFile

	// Calculate review score
	log.Infof("calculating review score")
	reviewScore, err := CalculateReviewScore(reviewJSON)
	if err != nil {
		log.Errorf("failed to calculate review score: %v", err)
		log.Debugf("review JSON: total_entries=%d, issues=%d", reviewJSON.TotalEntries, len(reviewJSON.Issues))
		return result, fmt.Errorf("failed to calculate review score: %w", err)
	}
	result.ReviewScore = reviewScore
	result.Score = reviewScore
	result.ReviewedFilePath = reviewPOFile

	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d, reviewed file: %s)",
		reviewScore, reviewJSON.TotalEntries, len(reviewJSON.Issues), reviewPOFile)

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// CmdAgentRunReview implements the agent-run review command logic.
// It loads configuration and calls RunAgentReview or RunAgentReviewAllWithLLM.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// allWithLLM: if true, use pure LLM approach (--all-with-llm).
func CmdAgentRunReview(agentName string, target *CompareTarget, outputBase string, allWithLLM bool) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	var result *AgentRunResult
	if allWithLLM {
		result, err = RunAgentReviewAllWithLLM(cfg, agentName, target, false, outputBase)
	} else {
		result, err = RunAgentReview(cfg, agentName, target, false, outputBase)
	}
	if err != nil {
		log.Errorf("failed to run agent review: %v", err)
		return err
	}

	// For agent-run, we require agent execution to succeed
	if !result.AgentSuccess {
		log.Errorf("agent execution failed: %s", result.AgentError)
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}

	elapsed := time.Since(startTime)

	// Display review results
	if result.ReviewJSON != nil {
		fmt.Printf("\nReview Results:\n")
		fmt.Printf("  Total entries: %d\n", result.ReviewJSON.TotalEntries)
		fmt.Printf("  Issues found: %d\n", len(result.ReviewJSON.Issues))
		fmt.Printf("  Review score: %d/100\n", result.ReviewScore)

		// Count issues by severity
		criticalCount := 0
		majorCount := 0
		minorCount := 0
		for _, issue := range result.ReviewJSON.Issues {
			switch issue.Score {
			case 0:
				criticalCount++
			case 1:
				majorCount++
			case 2:
				minorCount++
			}
		}

		fmt.Printf("\n  Issue breakdown:\n")
		if len(result.ReviewJSON.Issues) > 0 {
			if criticalCount > 0 {
				fmt.Printf("    Critical (must fix immediately): %d\n", criticalCount)
			}
			if majorCount > 0 {
				fmt.Printf("    Major (should fix): %d\n", majorCount)
			}
			if minorCount > 0 {
				fmt.Printf("    Minor (recommended to improve): %d\n", minorCount)
			}
		}
		fmt.Printf("    Perfect entries: %d\n",
			result.ReviewJSON.TotalEntries-criticalCount-minorCount)

		if result.ReviewJSONPath != "" {
			fmt.Printf("\n  JSON saved to: %s\n", getRelativePath(result.ReviewJSONPath))
		}
		if result.ReviewedFilePath != "" {
			fmt.Printf("  Reviewed file: %s\n", getRelativePath(result.ReviewedFilePath))
		}
	}

	fmt.Printf("\nSummary:\n")
	if result.NumTurns > 0 {
		fmt.Printf("  Turns: %d\n", result.NumTurns)
	}
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run review completed successfully")
	return nil
}

// countMsgidEntries counts the number of msgid entries in a PO file by counting lines that start with "msgid "
func countMsgidEntries(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "msgid ") {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return count, nil
}
