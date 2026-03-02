// Package util provides business logic for agent-run review command.
package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// executeReviewAgent executes the agent command for reviewing the given file.
// vars contains placeholder values (e.g. "prompt", "source" for the file to review).
// Returns stdout (for JSON extraction), stderr, originalStdout (raw before parsing), streamResult.
// Updates result with AgentExecuted, AgentError, AgentStdout, AgentStderr.
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
			result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
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
			result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", execErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
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

// runReviewBatched runs review for each batch in batchPOPaths, saves to po/review-batch-<N>.json (step 7),
// deletes each po/review-batch-<N>.po (step 8), and does not merge; caller runs step 9.
// reviewPOFile is the base path (e.g. po/review.po) used to derive batch JSON filenames.
func runReviewBatched(cfg *config.AgentConfig, selectedAgent config.Agent, entryCount int, reviewPOFile string, batchPOPaths []string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "review")
	if err != nil {
		return err
	}
	for i, batchPOPath := range batchPOPaths {
		batchNum := i + 1
		batchJSONPath := reviewBatchJSONPath(reviewPOFile, batchNum)
		batchVars := make(PlaceholderVars)
		batchVars["prompt"] = prompt
		batchVars["source"] = batchPOPath
		batchVars["dest"] = reviewPOFile
		batchVars["json"] = batchJSONPath
		resolvedPrompt, err := ExecutePromptTemplate(prompt, batchVars)
		if err != nil {
			return fmt.Errorf("failed to resolve prompt template: %w", err)
		}
		batchVars["prompt"] = resolvedPrompt

		stdout, _, _, _, err := executeReviewAgent(selectedAgent, batchVars, result)
		if err != nil {
			return err
		}
		batchJSON, err := parseAndAccumulateReviewJSON(stdout, entryCount)
		if err != nil {
			return err
		}
		if batchJSON != nil {
			if err := saveReviewJSON(batchJSON, batchJSONPath); err != nil {
				return fmt.Errorf("failed to save batch JSON to %s: %w", batchJSONPath, err)
			}
		}
		os.Remove(batchPOPath)
		log.Infof("saved review batch %d JSON and removed batch PO", batchNum)
	}
	return nil
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

// reviewBatchPath returns the path for the N-th batch PO file (e.g. po/review-batch-1.po).
func reviewBatchPath(reviewPOFile string, n int) string {
	dir := filepath.Dir(reviewPOFile)
	base := strings.TrimSuffix(filepath.Base(reviewPOFile), ".po")
	return filepath.Join(dir, base+"-batch-"+strconv.Itoa(n)+".po")
}

// reviewBatchJSONPath returns the path for the N-th batch JSON file (e.g. po/review-batch-1.json).
func reviewBatchJSONPath(reviewPOFile string, n int) string {
	dir := filepath.Dir(reviewPOFile)
	base := strings.TrimSuffix(filepath.Base(reviewPOFile), ".po")
	return filepath.Join(dir, base+"-batch-"+strconv.Itoa(n)+".json")
}

// listReviewBatchPOPaths returns existing po/review-batch-*.po paths sorted by batch number (for resume).
func listReviewBatchPOPaths(reviewPOFile string) ([]string, error) {
	dir := filepath.Dir(reviewPOFile)
	base := strings.TrimSuffix(filepath.Base(reviewPOFile), ".po")
	pattern := filepath.Join(dir, base+"-batch-*.po")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Slice(matches, func(i, j int) bool {
		ni, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[i]), base+"-batch-"), ".po"))
		nj, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[j]), base+"-batch-"), ".po"))
		return ni < nj
	})
	return matches, nil
}

// prepareReviewBatches creates po/review-batch-<N>.po files per AGENTS.md step 3 and returns their paths.
// Removes any existing po/review-batch-*.po and po/review-batch-*.json. Uses minBatchSize (e.g. 50)
// and AGENTS.md formula for NUM. Returns batch POPaths and content entry count (excluding header).
func prepareReviewBatches(reviewPOFile string, minBatchSize int) (batchPOPaths []string, entryCount int, err error) {
	dir := filepath.Dir(reviewPOFile)
	base := strings.TrimSuffix(filepath.Base(reviewPOFile), ".po")
	poPattern := filepath.Join(dir, base+"-batch-*.po")
	jsonPattern := filepath.Join(dir, base+"-batch-*.json")
	aggregateJSONFile := filepath.Join(dir, base+".json")
	for _, pattern := range []string{poPattern, jsonPattern} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			os.Remove(m)
		}
	}
	os.Remove(aggregateJSONFile)

	totalEntries, err := countMsgidEntries(reviewPOFile)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count entries in %s: %w", reviewPOFile, err)
	}
	entryCount = totalEntries
	if entryCount > 0 {
		entryCount-- // Exclude header
	}
	if entryCount <= 0 {
		return nil, entryCount, nil
	}

	var num int
	if entryCount <= minBatchSize*2 {
		num = entryCount
	} else {
		if entryCount > minBatchSize*8 {
			num = minBatchSize * 2
		} else if entryCount > minBatchSize*4 {
			num = minBatchSize + minBatchSize/2
		} else {
			num = minBatchSize
		}
	}

	batchCount := (entryCount + num - 1) / num
	for i := 1; i <= batchCount; i++ {
		start := (i-1)*num + 1
		end := i * num
		if end > entryCount {
			end = entryCount
		}
		rangeSpec := formatMsgSelectRange(i, start, end, entryCount, num)
		batchPath := reviewBatchPath(reviewPOFile, i)
		f, err := os.Create(batchPath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create batch file %s: %w", batchPath, err)
		}
		if err := MsgSelect(reviewPOFile, rangeSpec, f, false, nil); err != nil {
			f.Close()
			os.Remove(batchPath)
			return nil, 0, fmt.Errorf("msg-select failed for batch %d: %w", i, err)
		}
		f.Close()
		batchPOPaths = append(batchPOPaths, batchPath)
		log.Infof("prepared review batch %d: entries %d-%d (of %d)", i, start, end, entryCount)
	}
	return batchPOPaths, entryCount, nil
}

// parseAndAccumulateReviewJSON extracts and parses JSON from stdout, updates total_entries.
func parseAndAccumulateReviewJSON(stdout []byte, entryCount int) (*ReviewJSONResult, error) {
	jsonBytes, err := ExtractJSONFromOutput(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	reviewJSON, err := ParseReviewJSON(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review JSON: %w", err)
	}

	reviewJSON.TotalEntries = entryCount
	log.Debugf("parsed review JSON: total_entries=%d, issues=%d", reviewJSON.TotalEntries, len(reviewJSON.Issues))
	return reviewJSON, nil
}

// buildReviewUseAgentMdPrompt constructs a dynamic prompt for RunAgentReviewUseAgentMd
func buildReviewUseAgentMdPrompt(target *CompareTarget) string {
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

// RunAgentReviewUseAgentMd executes review using agent with po/AGENTS.md (--use-agent-md).
// No programmatic extraction or batching; the agent does everything and writes review.json.
// Before execution: deletes review.po and review.json. After: expects review.json to exist.
func RunAgentReviewUseAgentMd(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	reviewPOFile, reviewJSONFile := ReviewOutputPaths(outputBase)
	workDir := repository.WorkDirOrCwd()
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}

	if Exist(reviewPOFile) {
		log.Warnf("review PO file already exists: %s", reviewPOFile)
	}

	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return result, err
	}
	if !Exist(poFile) {
		result.AgentError = fmt.Errorf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	poFileRel := target.NewFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		poFileRel = filepath.ToSlash(rel)
	}
	prompt := buildReviewUseAgentMdPrompt(target)
	agentCmd, err := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": poFileRel})
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	outputFormat := normalizeOutputFormat(selectedAgent.Output)
	if outputFormat == "" {
		outputFormat = "default"
	}

	log.Infof("executing agent command (use-agent-md, output=%s): %s", outputFormat, truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	var stdout, stderr []byte
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd)
		if err != nil {
			result.AgentError = err
			return result, fmt.Errorf("agent command failed: %w", err)
		}
		defer stdoutReader.Close()
		_, streamResult, _ := parseStreamByKind(kind, stdoutReader)
		applyAgentDiagnostics(result, streamResult)
		if waitErr := cmdProcess.Wait(); waitErr != nil {
			result.AgentError = waitErr
			return result, fmt.Errorf("agent command failed: %w", waitErr)
		}
		stderr = stderrBuf.Bytes()
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd)
		if err != nil {
			result.AgentError = err
			return result, err
		}
		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, _ := ParseClaudeAgentOutput(stdout, outputFormat)
			stdout = parsedStdout
			applyAgentDiagnostics(result, parsedResult)
		}
	}

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

	_, reportResult, err := ReportReviewFromJSON(reviewJSONFile)
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

// RunAgentReview executes agent-run review following AGENTS.md Task 4 steps.
// Step 1: Check existing review. Step 2: Extract entries (PrepareReviewData). Step 3: Prepare batches.
// Steps 4–8: Run agent per batch, save JSON, delete batch PO. Step 9: Merge and summary.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
func RunAgentReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string, batchSize int) (*AgentRunResult, error) {
	var (
		batchPOPaths []string
		entryCount   int
		err          error
	)

	reviewPOFile, reviewJSONFile := ReviewOutputPaths(outputBase)
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}
	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Step 1: Check for existing review
	if Exist(reviewPOFile) && Exist(reviewJSONFile) {
		// Merge and summary only (step 9)
		log.Infof("both %s and %s exist; running merge and summary only", reviewPOFile, reviewJSONFile)
		jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
		if err != nil {
			return result, err
		}
		result.ReviewJSON = reportResult.Review
		result.ReviewJSONPath = jsonFile
		result.ReviewScore = reportResult.Score
		result.Score = reportResult.Score
		result.ReviewedFilePath = reviewPOFile
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}
	if Exist(reviewPOFile) && !Exist(reviewJSONFile) {
		// Resume: continue from step 4 (remaining batch PO files)
		batchPOPaths, err = listReviewBatchPOPaths(reviewPOFile)
		if err != nil {
			return result, fmt.Errorf("failed to list batch files: %w", err)
		}
		if len(batchPOPaths) == 0 {
			// No batch files left; merge any existing batch JSONs (step 9)
			log.Infof("no batch PO files remaining; running merge and summary")
			jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
			if err != nil {
				return result, err
			}
			result.ReviewJSON = reportResult.Review
			result.ReviewJSONPath = jsonFile
			result.ReviewScore = reportResult.Score
			result.Score = reportResult.Score
			result.ReviewedFilePath = reviewPOFile
			result.ExecutionTime = time.Since(startTime)
			return result, nil
		}
		entryCount = 0
		if total, err := countMsgidEntries(reviewPOFile); err == nil && total > 0 {
			entryCount = total - 1
		}
		// Continue to run remaining batches (steps 4–8) then step 9
	} else {
		// Step 2: Extract entries
		log.Infof("preparing review data: %s", reviewPOFile)
		if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, reviewPOFile, false, false); err != nil {
			return result, fmt.Errorf("failed to prepare review data: %w", err)
		}

		// Step 3: Prepare review batches
		if batchSize <= 0 {
			batchSize = 50
		}
		batchPOPaths, entryCount, err = prepareReviewBatches(reviewPOFile, batchSize)
		if err != nil {
			return result, err
		}
		if len(batchPOPaths) == 0 {
			// Empty or no entries; go to step 9 (merge will use po/review.po for total count)
			log.Infof("no review batches; running merge and summary")
			jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
			if err != nil {
				return result, err
			}
			result.ReviewJSON = reportResult.Review
			result.ReviewJSONPath = jsonFile
			result.ReviewScore = reportResult.Score
			result.Score = reportResult.Score
			result.ReviewedFilePath = reviewPOFile
			result.ExecutionTime = time.Since(startTime)
			return result, nil
		}
	}

	// Steps 4–8: Run agent per batch, save JSON, delete batch PO
	if err := runReviewBatched(cfg, selectedAgent, entryCount, reviewPOFile, batchPOPaths, result); err != nil {
		return result, err
	}

	// Step 9: Merge and summary
	jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
	if err != nil {
		return result, err
	}
	result.ReviewJSON = reportResult.Review
	result.ReviewJSONPath = jsonFile
	result.ReviewScore = reportResult.Score
	result.Score = reportResult.Score
	result.ReviewedFilePath = reviewPOFile
	result.ExecutionTime = time.Since(startTime)
	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))
	return result, nil
}

// CmdAgentRunReview implements the agent-run review command logic.
// It loads configuration and calls RunAgentReview or RunAgentReviewUseAgentMd.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// useAgentMd: if true, use agent with po/AGENTS.md (--use-agent-md).
func CmdAgentRunReview(agentName string, target *CompareTarget, outputBase string, useAgentMd bool, batchSize int) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	var result *AgentRunResult
	if useAgentMd {
		result, err = RunAgentReviewUseAgentMd(cfg, agentName, target, false, outputBase)
	} else {
		result, err = RunAgentReview(cfg, agentName, target, false, outputBase, batchSize)
	}
	if err != nil {
		return err
	}

	// For agent-run, we require agent execution to succeed (no error set)
	if result.AgentError != nil {
		return fmt.Errorf("agent execution failed: %w", result.AgentError)
	}

	elapsed := time.Since(startTime)

	// Display review report (same format as agent-run report)
	if result.ReviewJSON != nil && result.ReviewJSONPath != "" {
		critical, minor, major := CountReviewIssueScores(result.ReviewJSON)
		reportResult := &ReviewReportResult{
			Review:        result.ReviewJSON,
			Score:         result.ReviewScore,
			CriticalCount: critical,
			MinorCount:    minor,
			MajorCount:    major,
		}
		PrintReviewReportResult(result.ReviewJSONPath, reportResult)
	}

	fmt.Printf("\nSummary:\n")
	if result.ReviewJSONPath != "" {
		fmt.Printf("  Review JSON: %s\n", getRelativePath(result.ReviewJSONPath))
	}
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
