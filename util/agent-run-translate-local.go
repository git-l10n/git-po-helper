// Package util provides business logic for agent-run translate --use-local-orchestration.
// Aligns with AGENTS.md Task 3: one batch per iteration, l10n-todo.json/l10n-done.json.
package util

import (
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

const (
	l10nTodoBase   = "l10n-todo"
	l10nDoneBase   = "l10n-done"
	l10nMergedBase = "l10n-merged"
)

// RunAgentTranslateLocalOrchestration executes the local orchestration flow for translate.
// One batch per iteration (AGENTS.md Task 3): msg-select todo → l10n-todo.json → agent → l10n-done.json
// → msg-cat --unset-fuzzy → l10n-done.po → msgcat merge → repeat.
func RunAgentTranslateLocalOrchestration(cfg *config.AgentConfig, agentName, poFile string, batchSize int) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}

	poFile, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}

	if !Exist(poFile) {
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	poDir := filepath.Dir(poFile)
	todoPO := filepath.Join(poDir, l10nTodoBase+".po")
	todoJSON := filepath.Join(poDir, l10nTodoBase+".json")
	doneJSON := filepath.Join(poDir, l10nDoneBase+".json")
	donePO := filepath.Join(poDir, l10nDoneBase+".po")
	mergedPO := filepath.Join(poDir, l10nMergedBase+".po")

	filter := &EntryStateFilter{Untranslated: true, Fuzzy: true, NoObsolete: true}

	for {
		// Step 1: Condition check
		todoPOExists := Exist(todoPO)
		todoJSONExists := Exist(todoJSON)
		doneJSONExists := Exist(doneJSON)

		if !todoPOExists {
			// Step 2: Generate pending file
			if err := generateTodoPO(poFile, todoPO, filter); err != nil {
				return result, err
			}
			entryCount, err := countContentEntries(todoPO)
			if err != nil {
				return result, err
			}
			if entryCount == 0 {
				log.Infof("no untranslated or fuzzy entries, translation complete")
				cleanupIntermediateFiles(poDir)
				result.Score = 100
				result.ExecutionTime = time.Since(startTime)
				return result, nil
			}
			continue
		}

		if todoJSONExists {
			// Step 4: Translate one batch
			if err := translateOneBatch(cfg, selectedAgent, todoJSON, doneJSON, result); err != nil {
				return result, err
			}
			continue
		}

		if doneJSONExists {
			// Step 5 & 6: Merge (JSON→PO with --unset-fuzzy) and complete
			if err := mergeAndComplete(cfg, selectedAgent, doneJSON, donePO, poFile, mergedPO, result); err != nil {
				return result, err
			}
			cleanupAfterMerge(poDir, todoPO, doneJSON, donePO, mergedPO)
			continue
		}

		// Step 3: Generate one batch JSON
		if err := generateOneBatchJSON(todoPO, todoJSON, batchSize); err != nil {
			return result, err
		}
	}
}

func generateTodoPO(poFile, todoPO string, filter *EntryStateFilter) error {
	poDir := filepath.Dir(todoPO)
	os.Remove(filepath.Join(poDir, l10nTodoBase+".json"))
	os.Remove(filepath.Join(poDir, l10nDoneBase+".json"))

	f, err := os.Create(todoPO)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", todoPO, err)
	}
	defer f.Close()

	if err := MsgSelect(poFile, "1-", f, false, filter); err != nil {
		os.Remove(todoPO)
		return fmt.Errorf("msg-select failed: %w", err)
	}
	log.Infof("generated %s", todoPO)
	return nil
}

func countContentEntries(poFile string) (int, error) {
	total, err := countMsgidEntriesInFile(poFile)
	if err != nil {
		return 0, err
	}
	if total > 0 {
		total-- // exclude header
	}
	return total, nil
}

func countMsgidEntriesInFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "msgid ") {
			count++
		}
	}
	return count, nil
}

func batchSizeFromFormula(entryCount, minBatchSize int) int {
	if entryCount <= minBatchSize*2 {
		return entryCount
	}
	if entryCount > minBatchSize*8 {
		return minBatchSize * 2
	}
	if entryCount > minBatchSize*4 {
		return minBatchSize + minBatchSize/2
	}
	return minBatchSize
}

// generateOneBatchJSON creates a single l10n-todo.json (first batch or all entries).
func generateOneBatchJSON(todoPO, todoJSON string, minBatchSize int) error {
	entryCount, err := countContentEntries(todoPO)
	if err != nil {
		return err
	}
	if entryCount <= 0 {
		return nil
	}

	num := batchSizeFromFormula(entryCount, minBatchSize)
	var rangeSpec string
	if num >= entryCount {
		rangeSpec = "1-"
		log.Infof("processing all %d entries at once", entryCount)
	} else {
		rangeSpec = fmt.Sprintf("-%d", num)
		log.Infof("processing batch of %d entries (out of %d remaining)", num, entryCount)
	}

	f, err := os.Create(todoJSON)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", todoJSON, err)
	}
	defer f.Close()

	if err := WriteGettextJSONFromPOFile(todoPO, rangeSpec, f, nil); err != nil {
		os.Remove(todoJSON)
		return fmt.Errorf("msg-select --json failed: %w", err)
	}
	log.Infof("prepared %s", todoJSON)
	return nil
}

func translateOneBatch(cfg *config.AgentConfig, selectedAgent config.Agent, todoJSON, doneJSON string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "local-orchestration-translation")
	if err != nil {
		return err
	}

	workDir := repository.WorkDirOrCwd()
	sourceRel, _ := filepath.Rel(workDir, todoJSON)
	destRel, _ := filepath.Rel(workDir, doneJSON)
	if sourceRel == "" || sourceRel == "." {
		sourceRel = todoJSON
	}
	if destRel == "" || destRel == "." {
		destRel = doneJSON
	}
	sourceRel = filepath.ToSlash(sourceRel)
	destRel = filepath.ToSlash(destRel)

	batchVars := PlaceholderVars{
		"prompt": prompt,
		"source": sourceRel,
		"dest":   destRel,
	}
	resolvedPrompt, err := ExecutePromptTemplate(prompt, batchVars)
	if err != nil {
		return fmt.Errorf("failed to resolve prompt template: %w", err)
	}
	batchVars["prompt"] = resolvedPrompt

	agentCmd, err := BuildAgentCommand(selectedAgent, batchVars)
	if err != nil {
		return fmt.Errorf("failed to build agent command: %w", err)
	}

	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	outputFormat = normalizeOutputFormat(outputFormat)

	log.Infof("translating: %s -> %s (output=%s, streaming=%v)", sourceRel, destRel, outputFormat, outputFormat == "json")
	result.AgentExecuted = true

	stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(agentCmd)
	if execErr != nil {
		return fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
	}
	defer stdoutReader.Close()
	_, streamResult, _ := parseStreamByKind(selectedAgent.Kind, stdoutReader)
	applyAgentDiagnostics(result, streamResult)
	waitErr := cmdProcess.Wait()
	stderr := stderrBuf.Bytes()
	if waitErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent stderr: %s", string(stderr))
		}
		result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", waitErr)
		return fmt.Errorf("agent failed: %w", waitErr)
	}

	if !Exist(doneJSON) {
		return fmt.Errorf("agent did not create output file %s\nHint: The agent must write the translated JSON to {{.dest}}", destRel)
	}

	if _, err := ReadFileToGettextJSON(doneJSON); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", destRel, err)
	}

	os.Remove(todoJSON)
	log.Infof("translated batch")
	return nil
}

// mergeAndComplete converts done JSON to PO (with --unset-fuzzy), validates, and merges into target.
func mergeAndComplete(cfg *config.AgentConfig, selectedAgent config.Agent, doneJSON, donePO, targetPO, mergedPO string, result *AgentRunResult) error {
	j, err := ReadFileToGettextJSON(doneJSON)
	if err != nil {
		return fmt.Errorf("read %s: %w", doneJSON, err)
	}
	ClearFuzzyTagFromGettextJSON(j)

	f, err := os.Create(donePO)
	if err != nil {
		return fmt.Errorf("create %s: %w", donePO, err)
	}
	if err := WriteGettextJSONToPO(j, f, false, false); err != nil {
		f.Close()
		os.Remove(donePO)
		return fmt.Errorf("write %s: %w", donePO, err)
	}
	f.Close()

	cmd := exec.Command("msgfmt", "--check", "-o", os.DevNull, donePO)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(donePO)
		return fmt.Errorf("msgfmt validation failed: %w\n%s", err, string(out))
	}

	cmd = exec.Command("msgcat", "--use-first", donePO, targetPO, "-o", mergedPO)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("msgcat failed: %w\n%s", err, string(out))
	}

	cmd = exec.Command("msgfmt", "--check", "-o", os.DevNull, mergedPO)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("msgfmt validation failed, invoking agent to fix: %s", string(out))
		if fixErr := fixPoWithAgent(cfg, selectedAgent, mergedPO, string(out), result); fixErr != nil {
			os.Remove(mergedPO)
			return fmt.Errorf("msgfmt validation failed and agent fix failed: %w\nOriginal error: %s", fixErr, string(out))
		}
		cmd = exec.Command("msgfmt", "--check", "-o", os.DevNull, mergedPO)
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			os.Remove(mergedPO)
			return fmt.Errorf("msgfmt still fails after agent fix: %w\n%s", err2, string(out2))
		}
	}

	if err := os.Rename(mergedPO, targetPO); err != nil {
		os.Remove(mergedPO)
		return fmt.Errorf("failed to replace %s: %w", targetPO, err)
	}
	log.Infof("merged translations into %s", targetPO)
	return nil
}

// fixPoWithAgent invokes the agent to fix PO file syntax errors. The agent modifies the file in place.
func fixPoWithAgent(cfg *config.AgentConfig, selectedAgent config.Agent, poFile, msgfmtError string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "fix-po")
	if err != nil {
		return fmt.Errorf("fix-po prompt not configured: %w", err)
	}

	workDir := repository.WorkDirOrCwd()
	sourceRel, _ := filepath.Rel(workDir, poFile)
	if sourceRel == "" || sourceRel == "." {
		sourceRel = poFile
	}
	sourceRel = filepath.ToSlash(sourceRel)

	vars := PlaceholderVars{
		"prompt": prompt,
		"source": sourceRel,
		"dest":   sourceRel,
		"error":  msgfmtError,
	}
	resolvedPrompt, err := ExecutePromptTemplate(prompt, vars)
	if err != nil {
		return fmt.Errorf("failed to resolve prompt: %w", err)
	}
	vars["prompt"] = resolvedPrompt

	agentCmd, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return fmt.Errorf("failed to build agent command: %w", err)
	}

	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	outputFormat = normalizeOutputFormat(outputFormat)

	log.Infof("invoking agent to fix PO file: %s (output=%s, streaming=%v)", sourceRel, outputFormat, outputFormat == "json")
	result.AgentExecuted = true

	var stderr []byte
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(agentCmd)
		if execErr != nil {
			return fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
		defer stdoutReader.Close()
		_, streamResult, _ := parseStreamByKind(selectedAgent.Kind, stdoutReader)
		applyAgentDiagnostics(result, streamResult)
		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()
		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			return fmt.Errorf("agent fix failed: %w", waitErr)
		}
	} else {
		_, stderr, err = ExecuteAgentCommand(agentCmd)
		if err != nil {
			if len(stderr) > 0 {
				log.Debugf("agent stderr: %s", string(stderr))
			}
			result.AgentError = err
			return fmt.Errorf("agent fix failed: %w", err)
		}
	}
	log.Infof("agent completed fix, re-validating with msgfmt")
	return nil
}

func cleanupAfterMerge(poDir, todoPO, doneJSON, donePO, mergedPO string) {
	os.Remove(todoPO)
	os.Remove(doneJSON)
	os.Remove(donePO)
	os.Remove(mergedPO)
}

func cleanupIntermediateFiles(poDir string) {
	removeGlob(poDir, l10nTodoBase+".po")
	removeGlob(poDir, l10nTodoBase+".json")
	removeGlob(poDir, l10nDoneBase+".json")
	removeGlob(poDir, l10nDoneBase+".po")
	removeGlob(poDir, l10nMergedBase+".po")
}

func removeGlob(dir, pattern string) {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	for _, m := range matches {
		os.Remove(m)
	}
}
