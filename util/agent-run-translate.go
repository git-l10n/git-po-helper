// Package util provides business logic for agent-run translate command.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// RunAgentTranslate executes a single agent-run translate operation.
// When preResult is nil, performs pre-validation internally; otherwise uses preResult.
// Executes the agent command. Post-validation is done by validateTranslatePostResult in CmdAgentRunTranslate.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Determine PO file path
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

	// Build agent command with placeholders replaced
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

	// Build agent command with placeholders replaced
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	// Execute agent command
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

	// Log output if verbose
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
// Called from CmdAgentRunTranslate before RunAgentTranslate or RunAgentTranslateLocalOrchestration.
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
// Called from CmdAgentRunTranslate after RunAgentTranslate or RunAgentTranslateLocalOrchestration.
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
// It loads configuration and calls RunAgentTranslate or RunAgentTranslateLocalOrchestration
// based on useAgentMd/useLocalOrchestration flags, then handles errors appropriately.
func CmdAgentRunTranslate(agentName, poFile string, useAgentMd, useLocalOrchestration bool, batchSize int) error {
	var (
		result *AgentRunResult
		err    error
	)
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	// Pre-validation: store PO stats before translation in preResult.
	preResult, err := validateTranslatePreResult(poFile)
	if err != nil {
		return err
	}

	if batchSize <= 0 {
		batchSize = 50
	}
	if useLocalOrchestration {
		result, err = RunAgentTranslateLocalOrchestration(cfg, agentName, poFile, batchSize)
	} else {
		result, err = RunAgentTranslate(cfg, agentName, poFile, false)
	}
	result.PreValidationError = preResult.PreValidationError
	result.BeforeNewCount = preResult.BeforeNewCount
	result.BeforeFuzzyCount = preResult.BeforeFuzzyCount
	elapsed := time.Since(startTime)
	result.ExecutionTime = elapsed
	log.Infof("agent-run translate: execution time: %s", result.ExecutionTime.Round(time.Millisecond))

	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Post-validation
	if err := validateTranslatePostResult(poFile, result); err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
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
