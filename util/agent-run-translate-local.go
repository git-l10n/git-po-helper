// Package util provides business logic for agent-run translate --use-local-orchestration.
// Aligns with AGENTS.md Task 3: one batch per iteration, l10n-pending.po → l10n-todo.json
// → l10n-done.json → l10n-done.po → l10n-done.merged → merge into po/XX.po.
package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

const (
	l10nPendingBase = "l10n-pending"
	l10nTodoBase    = "l10n-todo"
	l10nDoneBase    = "l10n-done"
	l10nMergedExt   = "l10n-done.merged"
)

// RunAgentTranslateLocalOrchestration executes the local orchestration flow for translate.
// Aligns with AGENTS.md Task 3:
//
//	Step 1: extract po/XX.po → l10n-pending.po (untranslated+fuzzy)
//	Step 2: slice first batch from l10n-pending.po → l10n-todo.json
//	Step 3a: agent translates l10n-todo.json → l10n-done.json
//	Step 4: validate (msgid consistency + msgfmt), convert → l10n-done.po
//	Step 5: msgcat l10n-done.po + po/XX.po → l10n-done.merged → replace po/XX.po
//	Step 6: repeat from Step 1 until l10n-pending.po is empty
func RunAgentTranslateLocalOrchestration(cfg *config.AgentConfig, agentName, poFile string, batchSize int) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}

	rel, err := GuessPoFilePath(cfg, poFile)
	if err != nil {
		return result, err
	}
	poFile = rel

	if !Exist(poFile) {
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	poDir := filepath.Dir(poFile)
	pendingPO := filepath.Join(poDir, l10nPendingBase+".po")
	todoJSON := filepath.Join(poDir, l10nTodoBase+".json")
	doneJSON := filepath.Join(poDir, l10nDoneBase+".json")
	donePO := filepath.Join(poDir, l10nDoneBase+".po")
	mergedFile := filepath.Join(poDir, l10nMergedExt)

	filter := &EntryStateFilter{Untranslated: true, Fuzzy: true, NoObsolete: true}

	for {
		// Resume-state detection
		pendingExists := Exist(pendingPO)
		todoJSONExists := Exist(todoJSON)
		doneJSONExists := Exist(doneJSON)

		if !pendingExists {
			// Step 1: Extract untranslated+fuzzy entries from po/XX.po → l10n-pending.po
			if err := generatePendingPO(poFile, pendingPO, filter); err != nil {
				return result, err
			}
			entryCount, err := countContentEntries(pendingPO)
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
			// Step 3a: Agent translates l10n-todo.json → l10n-done.json
			if err := translateOneBatch(cfg, selectedAgent, todoJSON, doneJSON, result); err != nil {
				return result, err
			}
			continue
		}

		if doneJSONExists {
			// Step 4 & 5: Validate then merge into po/XX.po
			if err := mergeAndComplete(cfg, selectedAgent, doneJSON, donePO, pendingPO, poFile, mergedFile, result); err != nil {
				return result, err
			}
			// Step 6: delete l10n-pending.po so next iteration re-extracts from updated po/XX.po
			cleanupAfterMerge(pendingPO, doneJSON, donePO, mergedFile)
			continue
		}

		// Step 2: Slice first batch from l10n-pending.po → l10n-todo.json
		if err := generateOneBatchJSON(pendingPO, todoJSON, batchSize); err != nil {
			return result, err
		}
	}
}

// generatePendingPO extracts untranslated+fuzzy entries from poFile into pendingPO.
// Corresponds to AGENTS.md Task 3 Step 1 (po_extract_pending).
func generatePendingPO(poFile, pendingPO string, filter *EntryStateFilter) error {
	poDir := filepath.Dir(pendingPO)
	// Clean up any stale batch files before starting fresh
	os.Remove(filepath.Join(poDir, l10nTodoBase+".json"))
	os.Remove(filepath.Join(poDir, l10nDoneBase+".json"))

	f, err := os.Create(pendingPO)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", pendingPO, err)
	}
	defer f.Close()

	if err := MsgSelect(poFile, "1-", f, false, filter); err != nil {
		os.Remove(pendingPO)
		return fmt.Errorf("msg-select failed: %w", err)
	}
	log.Infof("generated %s", pendingPO)
	return nil
}

func countContentEntries(poFile string) (int, error) {
	total, err := CountMsgidEntries(poFile)
	if err != nil {
		return 0, err
	}
	if total > 0 {
		total-- // exclude header
	}
	return total, nil
}

// generateOneBatchJSON slices the first batch from pendingPO and writes it as l10n-todo.json.
// Corresponds to AGENTS.md Task 3 Step 2 (l10n_one_batch, git-po-helper path).
func generateOneBatchJSON(pendingPO, todoJSON string, minBatchSize int) error {
	entryCount, err := countContentEntries(pendingPO)
	if err != nil {
		return err
	}
	if entryCount <= 0 {
		return nil
	}

	num := CalcBatchSize(entryCount, minBatchSize)
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

	if err := WriteGettextJSONFromPOFile(pendingPO, rangeSpec, f, nil); err != nil {
		os.Remove(todoJSON)
		return fmt.Errorf("msg-select --json failed: %w", err)
	}
	log.Infof("prepared %s", todoJSON)
	return nil
}

func translateOneBatch(cfg *config.AgentConfig, selectedAgent config.AgentEntry, todoJSON, doneJSON string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "local-orchestration-translation")
	if err != nil {
		return err
	}

	workDir, _ := os.Getwd()
	if workDir == "" {
		workDir = "."
	}
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

	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, batchVars)
	if err != nil {
		return fmt.Errorf("failed to build agent command: %w", err)
	}
	// Align with RunAgentTranslate prompt path: mark executed before RunAgentAndParse.
	result.AgentExecuted = true
	log.Infof("translating: %s -> %s (output=%s, streaming=%v)", sourceRel, destRel, outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON)

	_, _, stderr, streamResult, execErr := RunAgentAndParse(agentCmd, outputFormat, selectedAgent.Kind)
	if execErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent stderr: %s", string(stderr))
		}
		return fmt.Errorf("agent failed: %w", execErr)
	}
	applyAgentDiagnostics(result, streamResult)

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

// mergeAndComplete implements AGENTS.md Task 3 Step 4 (validate) and Step 5 (merge):
//  1. Convert l10n-done.json → l10n-done.po (--unset-fuzzy)
//  2. Validate msgid consistency: compare pendingPO vs donePO with msgidOnly=true;
//     any Added entries mean msgid was altered during translation → error
//  3. Validate PO format with msgfmt --check
//  4. Merge: msgcat --use-first donePO targetPO → mergedFile → replace targetPO
func mergeAndComplete(cfg *config.AgentConfig, selectedAgent config.AgentEntry, doneJSON, donePO, pendingPO, targetPO, mergedFile string, result *AgentRunResult) error {
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

	// Check 1 (AGENTS.md Step 4): msgid consistency — detect AI-altered msgid
	if err := validateMsgIDConsistency(pendingPO, donePO); err != nil {
		os.Remove(donePO)
		return err
	}

	// Check 2 (AGENTS.md Step 4): PO format
	cmd := exec.Command("msgfmt", "--check", "-o", os.DevNull, donePO)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(donePO)
		return fmt.Errorf("msgfmt validation failed: %w\n%s", err, string(out))
	}

	// Step 5: msgcat --use-first donePO targetPO → mergedFile
	cmd = exec.Command("msgcat", "--use-first", donePO, targetPO, "-o", mergedFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("msgcat failed: %w\n%s", err, string(out))
	}

	cmd = exec.Command("msgfmt", "--check", "-o", os.DevNull, mergedFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("msgfmt validation failed, invoking agent to fix: %s", string(out))
		if fixErr := fixPoWithAgent(cfg, selectedAgent, mergedFile, string(out), result); fixErr != nil {
			os.Remove(mergedFile)
			return fmt.Errorf("msgfmt validation failed and agent fix failed: %w\nOriginal error: %s", fixErr, string(out))
		}
		cmd = exec.Command("msgfmt", "--check", "-o", os.DevNull, mergedFile)
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			os.Remove(mergedFile)
			return fmt.Errorf("msgfmt still fails after agent fix: %w\n%s", err2, string(out2))
		}
	}

	if err := os.Rename(mergedFile, targetPO); err != nil {
		os.Remove(mergedFile)
		return fmt.Errorf("failed to replace %s: %w", targetPO, err)
	}
	log.Infof("merged translations into %s", targetPO)
	return nil
}

// validateMsgIDConsistency checks that no msgid was altered during translation.
// Corresponds to AGENTS.md Task 3 Step 4, Check 1:
//
//	git-po-helper compare -q --msgid --assert-no-changes pendingPO donePO
//
// Any entry present in donePO but not in pendingPO (Added > 0) means the AI
// changed a msgid, making the entry appear as a new addition instead of a replacement.
func validateMsgIDConsistency(pendingPO, donePO string) error {
	pendingData, err := os.ReadFile(pendingPO)
	if err != nil {
		return fmt.Errorf("read %s: %w", pendingPO, err)
	}
	doneData, err := os.ReadFile(donePO)
	if err != nil {
		return fmt.Errorf("read %s: %w", donePO, err)
	}

	pendingJ, err := LoadFileToGettextJSON(pendingData, pendingPO)
	if err != nil {
		return fmt.Errorf("parse %s: %w", pendingPO, err)
	}
	doneJ, err := LoadFileToGettextJSON(doneData, donePO)
	if err != nil {
		return fmt.Errorf("parse %s: %w", donePO, err)
	}

	stat, addedEntries := CompareGettextEntries(pendingJ, doneJ, true)
	if stat.Added > 0 {
		var msgs []string
		for _, e := range addedEntries {
			msgs = append(msgs, fmt.Sprintf("  msgid %q", e.MsgID))
		}
		return fmt.Errorf("ERROR [msgid modified]: %d entr(ies) appeared after translation because msgid was altered.\n"+
			"Fix in %s:\n%s", stat.Added, donePO, strings.Join(msgs, "\n"))
	}
	return nil
}

// fixPoWithAgent invokes the agent to fix PO file syntax errors. The agent modifies the file in place.
func fixPoWithAgent(cfg *config.AgentConfig, selectedAgent config.AgentEntry, poFile, msgfmtError string, result *AgentRunResult) error {
	prompt, err := GetRawPrompt(cfg, "fix-po")
	if err != nil {
		return fmt.Errorf("fix-po prompt not configured: %w", err)
	}

	workDir, _ := os.Getwd()
	if workDir == "" {
		workDir = "."
	}
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

	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return fmt.Errorf("failed to build agent command: %w", err)
	}
	log.Infof("invoking agent to fix PO file: %s (output=%s, streaming=%v)", sourceRel, outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON)
	result.AgentExecuted = true

	_, _, stderr, streamResult, execErr := RunAgentAndParse(agentCmd, outputFormat, selectedAgent.Kind)
	if execErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent stderr: %s", string(stderr))
		}
		return fmt.Errorf("agent fix failed: %w", execErr)
	}
	applyAgentDiagnostics(result, streamResult)
	log.Infof("agent completed fix, re-validating with msgfmt")
	return nil
}

// cleanupAfterMerge removes intermediate files after a successful batch merge.
// pendingPO (l10n-pending.po) is deleted so the next iteration re-extracts from
// the updated po/XX.po (AGENTS.md Task 3 Step 6: repeat from Step 1).
func cleanupAfterMerge(pendingPO, doneJSON, donePO, mergedFile string) {
	os.Remove(pendingPO)
	os.Remove(doneJSON)
	os.Remove(donePO)
	os.Remove(mergedFile)
}

// cleanupIntermediateFiles removes all l10n intermediate files.
// Corresponds to AGENTS.md Task 3 Step 8 (po_cleanup).
func cleanupIntermediateFiles(poDir string) {
	removeGlob(poDir, l10nPendingBase+".po")
	removeGlob(poDir, l10nTodoBase+".json")
	removeGlob(poDir, l10nDoneBase+".json")
	removeGlob(poDir, l10nDoneBase+".po")
	removeGlob(poDir, l10nMergedExt)
}

func removeGlob(dir, pattern string) {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	for _, m := range matches {
		os.Remove(m)
	}
}
