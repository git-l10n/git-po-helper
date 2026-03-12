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
// prompt orchestration. It prints translation statistics to stderr before and
// after the run (used by agent-test). Dispatches to RunAgentTranslateLocalOrchestration
// or RunAgentTranslatePromptOrchestration.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	result := &AgentRunResult{Score: 0}
	poFileAbs, err := GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}
	log.Debugf("PO file path: %s", poFileAbs)

	var translateStatsSummary string
	if stats, err := GetPoStats(poFileAbs); err != nil {
		log.Debugf("GetPoStats before agent: %v", err)
	} else {
		translateStatsSummary = fmt.Sprintf("Translation statistics: before: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
	}

	result, agentErr := runAgentTranslateDispatch(cfg, agentName, poFileAbs, useLocalOrchestration, batchSize)
	if result == nil {
		result = &AgentRunResult{Score: 0}
	}

	if stats, errStats := GetPoStats(poFileAbs); errStats != nil {
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
// with the full/extracted PO as source and running the agent.
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
// counts new/fuzzy entries. Returns (PreCheckResult, err).
// Used by agent-test and workflow translate PreCheck.
func validateTranslatePreResult(poFile string) (*PreCheckResult, error) {
	pre := &PreCheckResult{}
	if !Exist(poFile) {
		return pre, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	log.Infof("performing pre-validation: counting new and fuzzy entries")
	statsBefore, err := GetPoStats(poFile)
	if err != nil {
		return pre, fmt.Errorf("failed to count PO stats: %w", err)
	}
	pre.UntranslatePoEntries = statsBefore.Untranslated
	pre.FuzzyPoEntries = statsBefore.Fuzzy
	log.Infof("new (untranslated) entries before translation: %d", statsBefore.Untranslated)
	log.Infof("fuzzy entries before translation: %d", statsBefore.Fuzzy)

	if statsBefore.Untranslated == 0 && statsBefore.Fuzzy == 0 {
		pre.Error = fmt.Errorf("no new or fuzzy entries to translate, PO file is ready for use")
		return pre, pre.Error
	}
	return pre, nil
}

// validateTranslatePostResult performs post-validation after translation.
// Writes to ctx.PostCheckResult and ctx.Result.Score.
// Used by agent-test and workflow translate PostCheck.
func validateTranslatePostResult(poFile string, ctx *AgentRunContext) error {
	if ctx.PostCheckResult == nil {
		ctx.PostCheckResult = &PostCheckResult{}
	}
	log.Infof("performing post-validation: counting new and fuzzy entries")

	statsAfter, err := GetPoStats(poFile)
	if err != nil {
		return fmt.Errorf("failed to count PO stats after translation: %w", err)
	}
	ctx.PostCheckResult.UntranslatePoEntries = statsAfter.Untranslated
	ctx.PostCheckResult.FuzzyPoEntries = statsAfter.Fuzzy
	log.Infof("new (untranslated) entries after translation: %d", statsAfter.Untranslated)
	log.Infof("fuzzy entries after translation: %d", statsAfter.Fuzzy)

	if statsAfter.Untranslated != 0 || statsAfter.Fuzzy != 0 {
		ctx.PostCheckResult.Error = fmt.Errorf("translation incomplete: %d new entries and %d fuzzy entries remaining", statsAfter.Untranslated, statsAfter.Fuzzy)
		ctx.PostCheckResult.Score = 0
		ctx.Result.Score = 0
		return fmt.Errorf("post-validation failed: %w\nHint: The agent should translate all new entries and resolve all fuzzy entries", ctx.PostCheckResult.Error)
	}

	ctx.PostCheckResult.Score = 100
	ctx.Result.Score = 100
	log.Infof("post-validation passed: all entries translated")

	log.Infof("validating file syntax: %s", poFile)
	if err := ValidatePoFile(poFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		ctx.PostCheckResult.SyntaxValidationError = err
		ctx.Result.Score = 0
	} else {
		log.Infof("file syntax validation passed")
	}
	return nil
}

// CmdAgentRunTranslate implements the agent-run translate command logic via AgentRunWorkflow.
func CmdAgentRunTranslate(agentName, poFile string, useAgentMd, useLocalOrchestration bool, batchSize int) error {
	// useAgentMd unused when local orchestration is false (default path uses prompt with source)
	_ = useAgentMd
	return RunAgentRunWorkflow(NewWorkflowTranslate(agentName, poFile, useLocalOrchestration, batchSize))
}
