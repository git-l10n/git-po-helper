// Package util provides business logic for agent-run review --use-local-orchestration.
package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
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

// calcBatchNum reads po/review-batch.txt, increments by 1, writes back, and returns the new batch number.
func calcBatchNum(batchTxtPath string) (int, error) {
	batch := 0
	if data, err := os.ReadFile(batchTxtPath); err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		batch = n
	}
	batch++
	if err := os.WriteFile(batchTxtPath, []byte(strconv.Itoa(batch)), 0644); err != nil {
		return 0, fmt.Errorf("failed to write %s: %w", batchTxtPath, err)
	}
	return batch, nil
}

// calcNumForBatch returns the batch size (NUM) for the given entryCount and minBatchSize,
// following the AGENTS.md step 4 formula exactly.
func calcNumForBatch(entryCount, minBatchSize int) int {
	if entryCount > minBatchSize*8 {
		return minBatchSize * 2
	} else if entryCount > minBatchSize*4 {
		return minBatchSize + minBatchSize/2
	} else if entryCount > minBatchSize {
		return minBatchSize
	}
	return entryCount
}

// cleanReviewIntermediateFiles removes stale intermediate files before a fresh review run.
// Corresponds to AGENTS.md Task 4 step 2.
// Does NOT remove review-input.po (source of truth for entry count and OUTPUT_PO template).
func cleanReviewIntermediateFiles(ps ReviewPathSet) {
	for _, f := range []string{
		ps.ReviewBatchTxtPath(),
		ps.ReviewTodoJSONPath(),
		ps.ReviewDoneJSONPath(),
		ps.PendingPO,
		ps.ResultJSON,
	} {
		os.Remove(f)
	}
	// Remove review-result-*.json
	dir := filepath.Dir(ps.ResultJSON)
	base := strings.TrimSuffix(filepath.Base(ps.ResultJSON), ".json")
	if matches, err := filepath.Glob(filepath.Join(dir, base+"-*.json")); err == nil {
		for _, m := range matches {
			os.Remove(m)
		}
	}
}

// reviewOneBatch implements AGENTS.md Task 4 step 4:
// reads batch number from review-batch.txt (init 0), increments, extracts first NUM entries
// to review-todo.json, moves remainder back to review-pending.po.
// If review-pending.po does not exist, copies from review-input.po first.
// Returns (batchNum, entryCount, num, done) where done=true means no entries remain.
func reviewOneBatch(ps ReviewPathSet, minBatchSize int) (batchNum, entryCount, num int, done bool, err error) {
	inputPO := ps.InputPO
	pendingPO := ps.PendingPO
	todoJSON := ps.ReviewTodoJSONPath()
	batchTxt := ps.ReviewBatchTxtPath()

	// If pendingPO does not exist, copy from inputPO (AGENTS.md step 4)
	if !Exist(pendingPO) {
		if err := copyFile(inputPO, pendingPO); err != nil {
			return 0, 0, 0, false, fmt.Errorf("failed to copy %s to %s: %w", inputPO, pendingPO, err)
		}
		log.Debugf("copied %s to %s", inputPO, pendingPO)
	}

	// Count remaining entries (exclude header)
	total, err := countMsgidEntries(pendingPO)
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("failed to count entries in %s: %w", pendingPO, err)
	}
	entryCount = total
	if entryCount > 0 {
		entryCount-- // exclude header
	}
	if entryCount == 0 {
		return 0, 0, 0, true, nil
	}

	num = calcNumForBatch(entryCount, minBatchSize)

	batchNum, err = calcBatchNum(batchTxt)
	if err != nil {
		return 0, 0, 0, false, err
	}

	// Extract first NUM entries to review-todo.json
	tmpPO := pendingPO + ".tmp"
	os.Remove(todoJSON)
	os.Remove(tmpPO)

	todoFile, err := os.Create(todoJSON)
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("failed to create %s: %w", todoJSON, err)
	}
	if err := WriteGettextJSONFromPOFile(pendingPO, fmt.Sprintf("-%d", num), todoFile, nil); err != nil {
		todoFile.Close()
		os.Remove(todoJSON)
		return 0, 0, 0, false, fmt.Errorf("msg-select --json failed: %w", err)
	}
	todoFile.Close()

	// Move remaining entries (NUM+1 to end) back to review-pending.po via temp file
	if entryCount > num {
		tmpFile, err := os.Create(tmpPO)
		if err != nil {
			return 0, 0, 0, false, fmt.Errorf("failed to create %s: %w", tmpPO, err)
		}
		remainRange := fmt.Sprintf("%d-", num+1)
		if err := MsgSelect(pendingPO, remainRange, tmpFile, false, nil); err != nil {
			tmpFile.Close()
			os.Remove(tmpPO)
			return 0, 0, 0, false, fmt.Errorf("msg-select for remainder failed: %w", err)
		}
		tmpFile.Close()
		if err := os.Rename(tmpPO, pendingPO); err != nil {
			return 0, 0, 0, false, fmt.Errorf("failed to rename %s to %s: %w", tmpPO, pendingPO, err)
		}
	} else {
		// All entries consumed; write empty PO (header only) to review-pending.po
		if err := os.WriteFile(pendingPO, []byte{}, 0644); err != nil {
			return 0, 0, 0, false, fmt.Errorf("failed to clear %s: %w", pendingPO, err)
		}
	}

	log.Infof("prepared review batch %d (%d entries, %d remaining)", batchNum, num, entryCount-num)
	return batchNum, entryCount, num, false, nil
}

// renameReviewDone implements AGENTS.md Task 4 step 6:
// renames review-done.json to review-result-<N>.json where N is from review-batch.txt.
func renameReviewDone(ps ReviewPathSet) error {
	doneJSON := ps.ReviewDoneJSONPath()
	batchTxt := ps.ReviewBatchTxtPath()
	todoJSON := ps.ReviewTodoJSONPath()

	if !Exist(doneJSON) {
		return nil
	}
	data, err := os.ReadFile(batchTxt)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", batchTxt, err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n <= 0 {
		return fmt.Errorf("invalid batch number in %s: %q", batchTxt, string(data))
	}
	resultPath := ps.ReviewResultJSONPath(n)
	if err := os.Rename(doneJSON, resultPath); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", doneJSON, resultPath, err)
	}
	log.Infof("renamed %s to %s", filepath.Base(doneJSON), filepath.Base(resultPath))
	if Exist(todoJSON) {
		os.Remove(todoJSON)
		log.Infof("removed %s", filepath.Base(todoJSON))
	}
	return nil
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

// RunAgentReviewLocalOrchestration executes agent-run review following AGENTS.md Task 4 steps
// with local orchestration. Uses a single state-detection loop (like RunAgentTranslateLocalOrchestration):
// each iteration checks file state and executes exactly one step, then continues.
//
// Step 1 (resume): Check existing files to determine where to resume.
//   - If review-result.json exists → step 8.
//   - If review-input.po does not exist → step 2 (fresh start).
//   - If review-input.po exists:
//   - If review-done.json exists → step 6.
//   - Else if review-todo.json exists → step 5.
//   - Else → step 4.
//
// Step 2: Clean up stale intermediate files (when review-input.po absent).
// Step 3: Extract entries to review-input.po.
// Step 4: Prepare one batch (review-todo.json) from review-pending.po (copy from review-input.po if needed).
// Step 5: Run agent on review-todo.json, write review-done.json.
// Step 6: Rename review-done.json to review-result-<N>.json.
// Step 7: Loop back (handled by continue).
// Step 8: Merge all review-result-*.json and display report.
//
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
func RunAgentReviewLocalOrchestration(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string, batchSize int) (*AgentRunResult, error) {
	ps := ReviewPathSetFromBase(outputBase)
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}
	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	if batchSize <= 0 {
		batchSize = 100
	}

	for {
		// Resume-state detection (re-evaluated each iteration)
		resultExists := Exist(ps.ResultJSON)
		inputExists := Exist(ps.InputPO)
		doneExists := Exist(ps.ReviewDoneJSONPath())
		todoExists := Exist(ps.ReviewTodoJSONPath())

		// Step 1 / Step 8: review-result.json exists → merge and summary.
		if resultExists {
			log.Infof("%s exists; running merge and summary", ps.ResultJSON)
			return runMergeAndSummary(ps, outputBase, startTime, result)
		}

		// Step 1: Check for existing review (resume support)
		if !inputExists {
			// Step 2: Clean up stale intermediate files.
			log.Infof("starting fresh review; cleaning intermediate files")
			cleanReviewIntermediateFiles(ps)
			// Step 3: Extract entries to review-input.po.
			log.Infof("extracting review entries to %s", ps.InputPO)
			if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, ps.InputPO, false, false, false); err != nil {
				return result, fmt.Errorf("failed to prepare review data: %w", err)
			}
			continue
		}

		// Step 1: review-input.po exists - check for resume conditions
		// Step 6: review-done.json exists → rename to review-result-<N>.json.
		if doneExists {
			log.Infof("%s exists; renaming to result (step 6)", ps.ReviewDoneJSONPath())
			if err := renameReviewDone(ps); err != nil {
				return result, err
			}
			continue
		}

		// Step 5: review-todo.json exists → run agent, write review-done.json.
		if todoExists {
			log.Infof("%s exists; running agent (step 5)", ps.ReviewTodoJSONPath())
			// Use total entry count from review-input.po for scoring.
			entryCount := 0
			if total, err := countMsgidEntries(ps.InputPO); err == nil && total > 0 {
				entryCount = total - 1
			}
			if err := runReviewOneTodo(cfg, selectedAgent, ps, entryCount, result); err != nil {
				return result, err
			}
			continue
		}

		// Step 4: Prepare one batch from review-pending.po → review-todo.json.
		batchNum, entryCount, num, done, err := reviewOneBatch(ps, batchSize)
		if err != nil {
			return result, err
		}
		if done {
			// review-pending.po is empty → proceed to step 8.
			log.Infof("no more entries in %s; proceeding to merge", ps.PendingPO)
			// Clean intermediate files but preserve review-input.po (step 8 requirement)
			cleanReviewIntermediateFiles(ps)
			continue
		}
		log.Infof("prepared batch %d (%d entries, %d remaining)", batchNum, num, entryCount-num)
		// review-todo.json now exists; next iteration executes step 5.
	}
}

// runReviewOneTodo runs the agent on the current review-todo.json and writes review-done.json.
// Corresponds to AGENTS.md Task 4 step 5.
// entryCount is the total entries for scoring (passed to TotalEntries).
func runReviewOneTodo(cfg *config.AgentConfig, selectedAgent config.AgentEntry, ps ReviewPathSet, entryCount int, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "review")
	if err != nil {
		return err
	}
	batchVars := make(PlaceholderVars)
	batchVars["prompt"] = prompt
	batchVars["source"] = ps.ReviewTodoJSONPath()
	batchVars["dest"] = ps.OutputPO
	batchVars["json"] = ps.ReviewDoneJSONPath()
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
		if err := saveReviewJSON(batchJSON, ps.ReviewDoneJSONPath()); err != nil {
			return fmt.Errorf("failed to save review-done.json to %s: %w", ps.ReviewDoneJSONPath(), err)
		}
	}
	log.Infof("wrote %s", ps.ReviewDoneJSONPath())
	return nil
}

// runMergeAndSummary merges all review-result-*.json and returns the report.
// Corresponds to AGENTS.md Task 4 step 8.
func runMergeAndSummary(ps ReviewPathSet, outputBase string, startTime time.Time, result *AgentRunResult) (*AgentRunResult, error) {
	jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
	if err != nil {
		return result, err
	}
	result.ReviewJSON = reportResult.Review
	result.ReviewJSONPath = jsonFile
	result.ReviewScore = reportResult.Score
	result.Score = reportResult.Score
	result.ReviewedFilePath = ps.PendingPO
	result.ExecutionTime = time.Since(startTime)
	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))
	return result, nil
}
