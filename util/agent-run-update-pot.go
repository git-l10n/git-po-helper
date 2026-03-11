// Package util provides business logic for agent-run update-pot command.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// RunAgentUpdatePot executes a single agent-run update-pot operation.
// It executes the agent command and validates POT file syntax.
// Returns a result structure with detailed information.
// Pre-validation and post-validation (entry count checks) are performed by
// agent-test code when running agent-test update-pot.
func RunAgentUpdatePot(cfg *config.AgentConfig, agentName string) (*AgentRunResult, error) {
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Get POT file path
	potFile := GetPotFilePath()
	log.Debugf("POT file path: %s", potFile)

	// Count entries before update (for display in agent-test results)
	if !Exist(potFile) {
		result.EntryCountBeforeUpdate = 0
	} else {
		if stats, err := GetPoStats(potFile); err == nil {
			result.EntryCountBeforeUpdate = stats.Total()
		}
	}

	// Get prompt from configuration
	prompt, err := GetRawPrompt(cfg, "update-pot")
	if err != nil {
		return result, err
	}
	vars := PlaceholderVars{"prompt": prompt}
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

	// Count entries after update and set score (we only reach here when agent succeeded)
	if Exist(potFile) {
		if stats, err := GetPoStats(potFile); err == nil {
			result.EntryCountAfterUpdate = stats.Total()
		}
	}
	result.Score = 100

	// Validate POT file syntax (we only reach here when agent succeeded)
	log.Infof("validating file syntax: %s", potFile)
	if err := ValidatePoFile(potFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		result.SyntaxValidationError = err
	} else {
		log.Infof("file syntax validation passed")
	}

	return result, nil
}

// CmdAgentRunUpdatePot implements the agent-run update-pot command logic.
// It loads configuration and calls RunAgentUpdatePot, then handles errors appropriately.
func CmdAgentRunUpdatePot(agentName string) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()
	result, err := RunAgentUpdatePot(cfg, agentName)
	elapsed := time.Since(startTime)
	result.ExecutionTime = elapsed
	log.Infof("agent-run update-pot: execution time: %s", result.ExecutionTime.Round(time.Millisecond))
	if err != nil {
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
		ext := filepath.Ext(GetPotFilePath())
		if ext == ".pot" {
			return fmt.Errorf("file validation failed: %w\nHint: Check the POT file syntax using 'msgcat --use-first <file> -o /dev/null'", result.SyntaxValidationError)
		}
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	log.Infof("agent-run update-pot completed successfully")
	return nil
}
