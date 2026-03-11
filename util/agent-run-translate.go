// Package util provides business logic for agent-run translate command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// RunAgentTranslate runs translate using either local batch orchestration or
// prompt orchestration (full/extracted PO to agent). It prints translation
// statistics to stderr before and after the run. Dispatches to
// RunAgentTranslateLocalOrchestration or RunAgentTranslatePromptOrchestration.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	var agentErr error

	result := &AgentRunResult{Score: 0}

	poFile, err := GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}
	log.Debugf("PO file path: %s", poFile)

	var translateStatsSummary string
	if stats, err := GetPoStats(poFile); err != nil {
		log.Debugf("GetPoStats before agent: %v", err)
	} else {
		translateStatsSummary = fmt.Sprintf("Translation statistics: before: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
	}

	result, agentErr = runAgentTranslateDispatch(cfg, agentName, poFile, useLocalOrchestration, batchSize)

	// After stats: print once whether dispatch succeeded or failed (PO may be partially updated).
	if stats, errStats := GetPoStats(poFile); errStats != nil {
		log.Errorf("GetPoStats after agent: %v", errStats)
		if translateStatsSummary != "" {
			fmt.Fprintln(os.Stderr, translateStatsSummary)
		}
	} else {
		afterSummary := fmt.Sprintf("Translation statistics: after: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
		if translateStatsSummary != "" {
			fmt.Fprintf(os.Stderr, "%s\n", translateStatsSummary)
		}
		fmt.Fprintf(os.Stderr, "%s\n", afterSummary)
	}

	if agentErr != nil {
		return result, agentErr
	}
	return result, nil
}

// runAgentTranslateDispatch runs either local orchestration or prompt orchestration.
func runAgentTranslateDispatch(cfg *config.AgentConfig, agentName, poFile string, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	if useLocalOrchestration {
		if batchSize <= 0 {
			batchSize = 50
		}
		return RunAgentTranslateLocalOrchestration(cfg, agentName, poFile, batchSize)
	}
	return RunAgentTranslatePromptOrchestration(cfg, agentName, poFile)
}

// RunAgentTranslatePromptOrchestration executes translate by building a prompt
// with the full/extracted PO as source and running the agent once (or as the
// agent handles it). Post-validation is done by validateTranslatePostResult
// in CmdAgentRunTranslate.
func RunAgentTranslatePromptOrchestration(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	poFile, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}

	log.Debugf("PO file path: %s", poFile)

	// We can extract new entries and fuzzy entries from the PO file using
	// "msgattrib --untranslated --only-fuzzy poFile", and saved to a
	// temporary file, then pass it to the agent as a source file.
	// This way, we can translate the new entries and fuzzy entries in one
	// round of translation. Later, we can use msgcat to merge the translations
	// back to the PO file like "msgcat --use-first new.po original.po -o merged.po".
	//
	// But we can document this in the po/README.md, and let the code agent
	// decide whether to use this feature.
	//
	// Now, load the simple prompt for translate the file.
	prompt, err := GetRawPrompt(cfg, "translate")
	if err != nil {
		return result, err
	}

	workDir := repository.WorkDirOrCwd()
	sourcePath := poFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		sourcePath = filepath.ToSlash(rel)
	}
	vars := PlaceholderVars{"prompt": prompt, "source": sourcePath}
	resolvedPrompt, err := ExecutePromptTemplate(prompt, vars)
	if err != nil {
		return result, fmt.Errorf("failed to resolve prompt template: %w", err)
	}
	vars["prompt"] = resolvedPrompt

	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON,
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	stdout, _, stderr, streamResult, err := RunAgentAndParse(agentCmd, outputFormat, kind)
	if err != nil {
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
		if len(stdout) > 0 {
			log.Debugf("agent command stdout: %s", string(stdout))
		}
		return result, err
	}
	log.Infof("agent command completed successfully")

	applyAgentDiagnostics(result, streamResult)

	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// validateTranslatePreResult performs pre-validation before translation:
// counts new/fuzzy entries. Returns (result, needRun, err).
// needRun is false when nothing to translate (file already complete).
// Called from CmdAgentRunTranslate before RunAgentTranslate.
func validateTranslatePreResult(poFile string) (*AgentRunResult, error) {
	result := &AgentRunResult{Score: 0}
	if !Exist(poFile) {
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	log.Infof("performing pre-validation: counting new and fuzzy entries")
	statsBefore, err := GetPoStats(poFile)
	if err != nil {
		return result, fmt.Errorf("failed to count PO stats: %w", err)
	}
	result.BeforeNewCount = statsBefore.Untranslated
	result.BeforeFuzzyCount = statsBefore.Fuzzy
	log.Infof("new (untranslated) entries before translation: %d", statsBefore.Untranslated)
	log.Infof("fuzzy entries before translation: %d", statsBefore.Fuzzy)

	if statsBefore.Untranslated == 0 && statsBefore.Fuzzy == 0 {
		result.Score = 0
		result.PreValidationError = fmt.Errorf("no new or fuzzy entries to translate, PO file is ready for use")
		return result, result.PreValidationError
	}
	return result, nil
}

// validateTranslatePostResult performs post-validation after translation:
// counts new/fuzzy entries, verifies both are 0, and validates PO syntax.
// Called from CmdAgentRunTranslate after RunAgentTranslate.
func validateTranslatePostResult(poFile string, result *AgentRunResult) error {
	log.Infof("performing post-validation: counting new and fuzzy entries")

	statsAfter, err := GetPoStats(poFile)
	if err != nil {
		return fmt.Errorf("failed to count PO stats after translation: %w", err)
	}
	result.AfterNewCount = statsAfter.Untranslated
	result.AfterFuzzyCount = statsAfter.Fuzzy
	log.Infof("new (untranslated) entries after translation: %d", statsAfter.Untranslated)
	log.Infof("fuzzy entries after translation: %d", statsAfter.Fuzzy)

	if statsAfter.Untranslated != 0 || statsAfter.Fuzzy != 0 {
		result.PostValidationError = fmt.Errorf("translation incomplete: %d new entries and %d fuzzy entries remaining", statsAfter.Untranslated, statsAfter.Fuzzy)
		result.Score = 0
		return fmt.Errorf("post-validation failed: %w\nHint: The agent should translate all new entries and resolve all fuzzy entries", result.PostValidationError)
	}

	result.Score = 100
	log.Infof("post-validation passed: all entries translated")

	log.Infof("validating file syntax: %s", poFile)
	if err := ValidatePoFile(poFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		result.SyntaxValidationError = err
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

// CmdAgentRunTranslate implements the agent-run translate command logic.
// It loads configuration and calls RunAgentTranslate (which dispatches to
// local or prompt orchestration), then handles errors appropriately.
func CmdAgentRunTranslate(agentName, poFile string, useAgentMd, useLocalOrchestration bool, batchSize int) error {
	var (
		result *AgentRunResult
		err    error
	)
	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return err
	}

	startTime := time.Now()

	preResult, err := validateTranslatePreResult(poFile)
	if err != nil {
		return err
	}

	if batchSize <= 0 {
		batchSize = 50
	}
	result, err = RunAgentTranslate(cfg, agentName, poFile, false, useLocalOrchestration, batchSize)
	result.PreValidationError = preResult.PreValidationError
	result.BeforeNewCount = preResult.BeforeNewCount
	result.BeforeFuzzyCount = preResult.BeforeFuzzyCount
	elapsed := time.Since(startTime)
	result.ExecutionTime = elapsed
	log.Infof("agent-run translate: execution time: %s", result.ExecutionTime.Round(time.Millisecond))

	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	if err := validateTranslatePostResult(poFile, result); err != nil {
		return err
	}

	if result.PreValidationError != nil {
		return fmt.Errorf("pre-validation failed: %w", result.PreValidationError)
	}
	if result.PostValidationError != nil {
		return fmt.Errorf("post-validation failed: %w", result.PostValidationError)
	}
	if result.SyntaxValidationError != nil {
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}
	log.Infof("agent-run translate completed successfully")
	return nil
}
