// Package util provides business logic for agent-run translate command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// RunAgentTranslate runs translate using either local batch orchestration or
// prompt orchestration. It prints translation statistics to stderr before and
// after the run (used by agent-test). Dispatches to RunAgentTranslateLocalOrchestration
// or RunAgentTranslatePromptOrchestration.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	result := &AgentRunResult{}
	rel, err := GuessPoFilePath(cfg, poFile)
	if err != nil {
		return result, err
	}
	poFile = rel
	log.Debugf("PO file path: %s", poFile)

	var translateStatsSummary string
	if stats, err := GetPoStats(poFile); err != nil {
		log.Debugf("GetPoStats before agent: %v", err)
	} else {
		translateStatsSummary = fmt.Sprintf("Translation statistics: before: %d translated, %d untranslated, %d fuzzy.",
			stats.Translated, stats.Untranslated, stats.Fuzzy)
	}

	result, agentErr := runAgentTranslateDispatch(cfg, agentName, poFile, useLocalOrchestration, batchSize)
	if result == nil {
		result = &AgentRunResult{}
	}
	// Non-workflow path: print diagnostics here (workflow prints before Report).
	PrintAgentDiagnosticsFromResult(result)

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
// with the full/extracted PO as source and running the agent.
func RunAgentTranslatePromptOrchestration(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	prompt, err := GetRawPrompt(cfg, "translate")
	if err != nil {
		return result, err
	}

	sourcePath := filepath.ToSlash(filepath.Clean(poFile))
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

	GetAgentDiagnostics(result, streamResult)

	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	result.ExecutionTime = time.Since(startTime)
	return result, nil
}

// CmdAgentRunTranslate implements the agent-run translate command logic via AgentRunWorkflow.
func CmdAgentRunTranslate(agentName, poFile string, useAgentMd, useLocalOrchestration bool, batchSize int) error {
	// useAgentMd unused when local orchestration is false (default path uses prompt with source)
	_ = useAgentMd
	return RunAgentRunWorkflow(NewWorkflowTranslate(agentName, poFile, useLocalOrchestration, batchSize))
}
