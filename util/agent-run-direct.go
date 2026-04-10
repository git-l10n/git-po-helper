// Package util provides business logic for agent-run direct prompt execution.
package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// runAgentDirectOnce runs the configured agent once with the given prompt (no workflow).
// cfg must be loaded; agentName is optional when exactly one agent is configured.
func runAgentDirectOnce(cfg *config.AgentConfig, agentName, prompt string) (*AgentRunResult, error) {
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return nil, err
	}
	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	vars := PlaceholderVars{"prompt": prompt}
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent command: %w", err)
	}

	start := time.Now()
	result := &AgentRunResult{AgentExecuted: true}
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON,
		truncateCommandDisplay(strings.Join(agentCmd, " ")))

	stdout, _, stderr, streamResult, execErr := RunAgentAndParse(agentCmd, outputFormat, selectedAgent.Kind)
	result.ExecutionTime = time.Since(start)
	result.Error = execErr
	if execErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
		if len(stdout) > 0 {
			log.Debugf("agent command stdout: %s", string(stdout))
		}
	} else {
		log.Infof("agent command completed successfully")
		if len(stdout) > 0 {
			log.Debugf("agent command stdout: %s", string(stdout))
		}
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
	}

	GetAgentDiagnostics(result, streamResult)
	return result, nil
}

// CmdAgentRunDirect runs the configured agent once with the given prompt text (no workflow).
// agentName is optional when exactly one agent is configured.
func CmdAgentRunDirect(agentName, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt is empty")
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}
	result, err := runAgentDirectOnce(cfg, agentName, prompt)
	if err != nil {
		return err
	}
	PrintAgentDiagnosticsFromResult(result)
	ctx := &AgentRunContext{Cfg: cfg, Result: result}
	PrintAgentRunStatus(ctx)

	if result.Error != nil {
		return result.Error
	}
	if ctx.PreValidationError() != nil {
		return ctx.PreValidationError()
	}
	if ctx.PostValidationError() != nil {
		return ctx.PostValidationError()
	}
	return nil
}
