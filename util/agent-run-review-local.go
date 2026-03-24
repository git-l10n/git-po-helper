// Package util provides business logic for agent-run review --use-local-orchestration.
package util

import (
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

// reviewOneBatch implements AGENTS.md Task 4 step 3 (Prepare one batch):
// if PENDING missing or INPUT_PO newer than PENDING, reset (rm batch/todo/done/result-*.json) and cp input→pending;
// then read batch number from review-batch.txt (init 0), increment, extract first NUM entries
// to review-todo.json, move remainder back to review-pending.po.
// Returns (batchNum, entryCount, num, done) where done=true means no entries remain (go to step 8).
func reviewOneBatch(ps ReviewPathSet, minBatchSize int) (batchNum, entryCount, num int, done bool, err error) {
	inputPO := ps.InputPO
	pendingPO := ps.PendingPO
	todoJSON := ps.ReviewTodoJSONPath()
	batchTxt := ps.ReviewBatchTxtPath()

	// AGENTS.md review_one_batch: if PENDING missing or INPUT_PO newer than PENDING, reset and copy
	needReset := !Exist(pendingPO)
	if !needReset && Exist(inputPO) {
		inputStat, e1 := os.Stat(inputPO)
		pendingStat, e2 := os.Stat(pendingPO)
		if e1 == nil && e2 == nil && inputStat.ModTime().After(pendingStat.ModTime()) {
			needReset = true
		}
	}
	if needReset {
		os.Remove(batchTxt)
		os.Remove(todoJSON)
		os.Remove(ps.ReviewDoneJSONPath())
		dir := filepath.Dir(ps.ResultJSON)
		base := strings.TrimSuffix(filepath.Base(ps.ResultJSON), ".json")
		if matches, globErr := filepath.Glob(filepath.Join(dir, base+"-*.json")); globErr == nil {
			for _, m := range matches {
				os.Remove(m)
			}
		}
		if err := copyFile(inputPO, pendingPO); err != nil {
			return 0, 0, 0, false, fmt.Errorf("failed to copy %s to %s: %w", inputPO, pendingPO, err)
		}
		log.Debugf("reset pending from %s to %s", inputPO, pendingPO)
	}

	// Count remaining entries (exclude header)
	total, err := CountMsgidEntries(pendingPO)
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

	num = CalcBatchSize(entryCount, minBatchSize)

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

// loadReviewDoneFromDisk reads the review result from the file the agent wrote (po/review-done.json).
// The prompt instructs the agent to write directly to that path; we do not parse stdout.
// Sets TotalEntries to entryCount for scoring and overwrites the file so the batch has correct total_entries.
func loadReviewDoneFromDisk(donePath string, entryCount int) (*ReviewResult, error) {
	if !Exist(donePath) {
		return nil, fmt.Errorf("agent did not write review result to %s (prompt requires writing to this file)", donePath)
	}
	review, err := loadReviewJSONFromFile(donePath)
	if err != nil {
		return nil, err
	}
	review.TotalEntries = entryCount
	log.Debugf("loaded review from %s: total_entries=%d, issues=%d", donePath, review.TotalEntries, len(review.Issues))
	// Persist total_entries so the batch file (after rename to review-result-N.json) is correct for merge.
	if err := saveReviewJSON(review, donePath); err != nil {
		return nil, fmt.Errorf("failed to save total_entries to %s: %w", donePath, err)
	}
	return review, nil
}

// RunAgentReviewLocalOrchestration executes agent-run review following AGENTS.md Task 4 steps
// with local orchestration. Uses a single state-detection loop (like RunAgentTranslateLocalOrchestration):
// each iteration checks file state and executes exactly one step, then continues.
//
// Step 1 (resume): Check existing files to determine where to resume (order matches AGENTS.md).
//   - If review-input.po does not exist → step 2 (fresh start).
//   - Else if review-result.json exists → step 8.
//   - Else if review-done.json exists → step 6.
//   - Else if review-todo.json exists → step 5.
//   - Else → step 3 (Prepare one batch).
//
// Step 2: Clean up stale intermediate files (when review-input.po absent).
// Step 3: Extract entries to review-input.po.
// Step 4: Prepare one batch (review-todo.json) from review-pending.po (copy from review-input.po if needed).
// Step 5: Run agent on review-todo.json, write review-done.json.
// Step 6: Rename review-done.json to review-result-<N>.json.
// Step 7: Loop back (handled by continue).
// Step 8: Merge all review-result-*.json and display report.
func RunAgentReviewLocalOrchestration(cfg *config.AgentConfig, agentName string, target *CompareTarget, batchSize int) (*AgentRunResult, error) {
	ps := GetReviewPathSet(target.NewFile)
	startTime := time.Now()
	result := &AgentRunResult{}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}
	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	if batchSize <= 0 {
		batchSize = 100
	}

	for {
		// Resume-state detection (re-evaluated each iteration; order matches AGENTS.md step 1)
		inputExists := Exist(ps.InputPO)
		resultExists := Exist(ps.ResultJSON)
		doneExists := Exist(ps.ReviewDoneJSONPath())
		todoExists := Exist(ps.ReviewTodoJSONPath())

		// Step 1: review-input.po does not exist → step 2 (fresh start).
		if !inputExists {
			log.Infof("starting fresh review; cleaning intermediate files")
			cleanReviewIntermediateFiles(ps)
			log.Infof("extracting review entries to %s", ps.InputPO)
			if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, ps.InputPO, false, false, false); err != nil {
				return result, fmt.Errorf("failed to prepare review data: %w", err)
			}
			continue
		}

		// Step 1: review-input.po exists → else if review-result.json exists → step 8.
		if resultExists {
			log.Infof("%s exists; running merge and summary", ps.ResultJSON)
			return runMergeAndSummary(target, startTime, result)
		}

		// Step 1: else if review-done.json exists → step 6 (Rename result).
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
			if total, err := CountMsgidEntries(ps.InputPO); err == nil && total > 0 {
				entryCount = total - 1
			}
			if err := runReviewOneTodo(cfg, selectedAgent, ps, entryCount, result); err != nil {
				return result, err
			}
			continue
		}

		// Step 3: Prepare one batch from review-pending.po → review-todo.json.
		batchNum, entryCount, num, done, err := reviewOneBatch(ps, batchSize)
		if err != nil {
			return result, err
		}
		if done {
			// Step 4: no todo (no entries left) → step 8. Do NOT cleanup (AGENTS.md: keep for inspection/resumption).
			log.Infof("no more entries in %s; running merge and summary (step 8)", ps.PendingPO)
			return runMergeAndSummary(target, startTime, result)
		}
		log.Infof("prepared batch %d (%d entries, %d remaining)", batchNum, num, entryCount-num)
		// review-todo.json now exists; next iteration executes step 5.
	}
}

// runReviewOneTodo runs the agent on the current review-todo.json and writes review-done.json.
// Corresponds to AGENTS.md Task 4 step 5.
// entryCount is the total entries for scoring (passed to TotalEntries).
func runReviewOneTodo(cfg *config.AgentConfig, selectedAgent config.AgentEntry, ps ReviewPathSet, entryCount int, result *AgentRunResult) error {
	// Align with RunAgentReviewPromptOrchestration: mark agent run before invoking (executeReviewAgent also sets this).
	result.AgentExecuted = true

	prompt, err := GetRawPrompt(cfg, "local-orchestration-review")
	if err != nil {
		return err
	}
	batchVars := make(PlaceholderVars)
	batchVars["prompt"] = prompt
	batchVars["source"] = ps.ReviewTodoJSONPath()
	batchVars["dest"] = ps.ReviewDoneJSONPath()
	resolvedPrompt, err := ExecutePromptTemplate(prompt, batchVars)
	if err != nil {
		return fmt.Errorf("failed to resolve prompt template: %w", err)
	}
	batchVars["prompt"] = resolvedPrompt

	_, _, _, _, err = executeReviewAgent(selectedAgent, batchVars, result)
	if err != nil {
		return err
	}
	// Prompt instructs agent to write result to {{.dest}} (review-done.json); read from disk, do not parse stdout.
	_, err = loadReviewDoneFromDisk(ps.ReviewDoneJSONPath(), entryCount)
	if err != nil {
		return err
	}
	log.Infof("read %s (written by agent)", ps.ReviewDoneJSONPath())
	return nil
}

// runMergeAndSummary merges all review-result-*.json and returns the report.
// Corresponds to AGENTS.md Task 4 step 8.
func runMergeAndSummary(target *CompareTarget, startTime time.Time, result *AgentRunResult) (*AgentRunResult, error) {
	pathName := ""
	if target != nil {
		pathName = target.NewFile
	}
	reportResult, err := GetReviewReport(pathName)
	if err != nil {
		return result, err
	}
	result.ReviewResult = reportResult
	result.ExecutionTime = time.Since(startTime)
	// Local orchestration merge path completes the review workflow (agent ran in prior batches or resume).
	result.AgentExecuted = true
	score, err := reportResult.GetScore()
	if err != nil {
		return result, err
	}
	totalEntries, _ := reportResult.GetTotalEntries()
	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d)",
		score, totalEntries, len(reportResult.Issues))
	return result, nil
}
