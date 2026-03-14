// Package util provides business logic for agent-run translate command.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

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
