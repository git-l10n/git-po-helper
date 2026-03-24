// Package util provides business logic for agent-run translate command.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// runAgentTranslateDispatch runs either local orchestration or prompt orchestration.
func runAgentTranslateDispatch(cfg *config.AgentConfig, agentName, poFile string, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	if poFile != "" {
		agentsMd := filepath.Join(filepath.Dir(poFile), "AGENTS.md")
		if !Exist(agentsMd) {
			log.Infof("no AGENTS.md beside %s, using local orchestration", poFile)
			useLocalOrchestration = true
		}
	}

	if useLocalOrchestration {
		if batchSize <= 0 {
			batchSize = 100
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
	agentsMdPath := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.Clean(poFile)), "AGENTS.md"))

	vars := PlaceholderVars{"prompt": prompt, "source": sourcePath, "agents_md": agentsMdPath}
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
func CmdAgentRunTranslate(agentName, poFile string, useLocalOrchestration bool, batchSize int) error {
	var absPo string
	if poFile != "" {
		var err error
		absPo, err = filepath.Abs(poFile)
		if err != nil {
			return fmt.Errorf("cannot resolve PO file path: %w", err)
		}
		absPo = filepath.Clean(absPo)
	}
	cleanup, err := EnsureInGitProjectRootDir()
	if err == nil {
		defer cleanup()
		if absPo != "" {
			repoRoot := filepath.Clean(repository.WorkDir())
			rel, err := filepath.Rel(repoRoot, absPo)
			if err != nil {
				return fmt.Errorf("PO file %s vs repository root %s: %w", absPo, repoRoot, err)
			}
			relSlash := filepath.ToSlash(rel)
			if relSlash == ".." || strings.HasPrefix(relSlash, "../") || strings.Contains(relSlash, "/../") {
				return fmt.Errorf("PO file %s is not under repository root %s", absPo, repoRoot)
			}
			poFile = relSlash
		}
	}

	return RunAgentRunWorkflow(NewWorkflowTranslate(agentName, poFile, useLocalOrchestration, batchSize))
}
