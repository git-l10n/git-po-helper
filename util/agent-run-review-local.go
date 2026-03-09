// Package util provides business logic for agent-run review --use-local-orchestration.
package util

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

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

// listReviewInputJSONPaths returns existing po/review-input-*.json paths sorted by batch number (for resume).
func listReviewInputJSONPaths(ps ReviewPathSet) ([]string, error) {
	dir := filepath.Dir(ps.InputPO)
	base := strings.TrimSuffix(filepath.Base(ps.InputPO), ".po")
	pattern := filepath.Join(dir, base+"-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Slice(matches, func(i, j int) bool {
		ni, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[i]), base+"-"), ".json"))
		nj, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[j]), base+"-"), ".json"))
		return ni < nj
	})
	return matches, nil
}

// prepareReviewInputBatches creates po/review-input-<N>.json files per AGENTS.md step 3 and returns their paths.
// Removes any existing review-input-*.json, review-result-*.json, review-result.json, review-output.po.
// Creates review-output.po as copy of review-input.po (per AGENTS.md step 3).
// Uses minBatchSize and AGENTS.md formula for NUM. Returns batch input JSON paths and content entry count.
func prepareReviewInputBatches(ps ReviewPathSet, minBatchSize int) (batchInputJSONPaths []string, entryCount int, err error) {
	dir := filepath.Dir(ps.InputPO)
	inputBase := strings.TrimSuffix(filepath.Base(ps.InputPO), ".po")
	resultBase := strings.TrimSuffix(filepath.Base(ps.ResultJSON), ".json")
	for _, p := range []string{
		filepath.Join(dir, inputBase+"-*.json"),
		filepath.Join(dir, resultBase+"-*.json"),
	} {
		matches, _ := filepath.Glob(p)
		for _, m := range matches {
			os.Remove(m)
		}
	}
	os.Remove(ps.ResultJSON)
	os.Remove(ps.OutputPO)

	// Create review-output.po as copy of review-input.po (per AGENTS.md step 3)
	if Exist(ps.InputPO) {
		data, err := os.ReadFile(ps.InputPO)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read %s: %w", ps.InputPO, err)
		}
		if err := os.WriteFile(ps.OutputPO, data, 0644); err != nil {
			return nil, 0, fmt.Errorf("failed to create %s: %w", ps.OutputPO, err)
		}
	}

	totalEntries, err := countMsgidEntries(ps.InputPO)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count entries in %s: %w", ps.InputPO, err)
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
		batchPath := ps.ReviewInputJSONPath(i)
		f, err := os.Create(batchPath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create batch file %s: %w", batchPath, err)
		}
		if err := WriteGettextJSONFromPOFile(ps.InputPO, rangeSpec, f, nil); err != nil {
			f.Close()
			os.Remove(batchPath)
			return nil, 0, fmt.Errorf("msg-select --json failed for batch %d: %w", i, err)
		}
		f.Close()
		batchInputJSONPaths = append(batchInputJSONPaths, batchPath)
		log.Infof("prepared review batch %d: entries %d-%d (of %d)", i, start, end, entryCount)
	}
	return batchInputJSONPaths, entryCount, nil
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

// runReviewBatched runs review for each batch in batchInputJSONPaths, saves to po/review-result-<N>.json (step 7),
// deletes each po/review-input-<N>.json (step 8), and does not merge; caller runs step 9.
func runReviewBatched(cfg *config.AgentConfig, selectedAgent config.AgentEntry, entryCount int, ps ReviewPathSet, batchInputJSONPaths []string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "review")
	if err != nil {
		return err
	}
	for i, inputJSONPath := range batchInputJSONPaths {
		batchNum := i + 1
		resultJSONPath := ps.ReviewResultJSONPath(batchNum)
		batchVars := make(PlaceholderVars)
		batchVars["prompt"] = prompt
		batchVars["source"] = inputJSONPath
		batchVars["dest"] = ps.OutputPO
		batchVars["json"] = resultJSONPath
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
			if err := saveReviewJSON(batchJSON, resultJSONPath); err != nil {
				return fmt.Errorf("failed to save batch JSON to %s: %w", resultJSONPath, err)
			}
		}
		os.Remove(inputJSONPath)
		log.Infof("saved review batch %d JSON and removed batch input", batchNum)
	}
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

// RunAgentReviewLocalOrchestration executes agent-run review following AGENTS.md Task 4 steps with local orchestration.
// Step 1: Check existing review. Step 2: Extract entries (PrepareReviewData). Step 3: Prepare batches.
// Steps 4–8: Run agent per batch, save JSON, delete batch input. Step 9: Merge and summary.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
func RunAgentReviewLocalOrchestration(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string, batchSize int) (*AgentRunResult, error) {
	var (
		batchInputPaths []string
		entryCount      int
		err             error
	)

	ps := ReviewPathSetFromBase(outputBase)
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}
	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Step 1: Check for existing review
	if Exist(ps.InputPO) && Exist(ps.ResultJSON) {
		// Merge and summary only (step 9)
		log.Infof("both %s and %s exist; running merge and summary only", ps.InputPO, ps.ResultJSON)
		jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
		if err != nil {
			return result, err
		}
		result.ReviewJSON = reportResult.Review
		result.ReviewJSONPath = jsonFile
		result.ReviewScore = reportResult.Score
		result.Score = reportResult.Score
		result.ReviewedFilePath = ps.InputPO
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}
	if Exist(ps.InputPO) && !Exist(ps.ResultJSON) {
		// Resume: continue from step 4 (remaining batch input JSON files)
		batchInputPaths, err = listReviewInputJSONPaths(ps)
		if err != nil {
			return result, fmt.Errorf("failed to list batch files: %w", err)
		}
		if len(batchInputPaths) == 0 {
			// No batch files left; merge any existing batch JSONs (step 9)
			log.Infof("no batch input files remaining; running merge and summary")
			jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
			if err != nil {
				return result, err
			}
			result.ReviewJSON = reportResult.Review
			result.ReviewJSONPath = jsonFile
			result.ReviewScore = reportResult.Score
			result.Score = reportResult.Score
			result.ReviewedFilePath = ps.InputPO
			result.ExecutionTime = time.Since(startTime)
			return result, nil
		}
		entryCount = 0
		if total, err := countMsgidEntries(ps.InputPO); err == nil && total > 0 {
			entryCount = total - 1
		}
		// Continue to run remaining batches (steps 4–8) then step 9
	} else {
		// Step 2: Extract entries
		log.Infof("preparing review data: %s", ps.InputPO)
		if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, ps.InputPO, false, false, false); err != nil {
			return result, fmt.Errorf("failed to prepare review data: %w", err)
		}

		// Step 3: Prepare review batches
		if batchSize <= 0 {
			batchSize = 50
		}
		batchInputPaths, entryCount, err = prepareReviewInputBatches(ps, batchSize)
		if err != nil {
			return result, err
		}
		if len(batchInputPaths) == 0 {
			// Empty or no entries; go to step 9
			log.Infof("no review batches; running merge and summary")
			jsonFile, reportResult, err := ReportReviewFromPathWithBatches(outputBase)
			if err != nil {
				return result, err
			}
			result.ReviewJSON = reportResult.Review
			result.ReviewJSONPath = jsonFile
			result.ReviewScore = reportResult.Score
			result.Score = reportResult.Score
			result.ReviewedFilePath = ps.InputPO
			result.ExecutionTime = time.Since(startTime)
			return result, nil
		}
	}

	// Steps 4–8: Run agent per batch, save JSON, delete batch input
	if err := runReviewBatched(cfg, selectedAgent, entryCount, ps, batchInputPaths, result); err != nil {
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
	result.ReviewedFilePath = ps.InputPO
	result.ExecutionTime = time.Since(startTime)
	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))
	return result, nil
}
