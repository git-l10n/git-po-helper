// Package util provides business logic for agent-run update-po command.
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

// RunAgentUpdatePo executes a single agent-run update-po operation.
// It executes the agent command and validates PO file syntax.
// Returns a result structure with detailed information.
// Pre-validation and post-validation (entry count checks) are performed by
// agent-test code when running agent-test update-po.
func RunAgentUpdatePo(cfg *config.AgentConfig, agentName, poFile string) (*AgentRunResult, error) {
	result := &AgentRunResult{
		Score: 0,
	}

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

	// Count entries before update (for display in agent-test results)
	if !Exist(poFile) {
		result.EntryCountBeforeUpdate = 0
	} else {
		if stats, err := GetPoStats(poFile); err == nil {
			result.EntryCountBeforeUpdate = stats.Total()
		}
	}

	// Get prompt for update-po from configuration
	prompt, err := GetRawPrompt(cfg, "update-po")
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
	if Exist(poFile) {
		if stats, err := GetPoStats(poFile); err == nil {
			result.EntryCountAfterUpdate = stats.Total()
		}
	}
	result.Score = 100

	// Validate PO file syntax (we only reach here when agent succeeded)
	log.Infof("validating file syntax: %s", poFile)
	if err := ValidatePoFile(poFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		result.SyntaxValidationError = err
	} else {
		log.Infof("file syntax validation passed")
	}

	return result, nil
}

// CmdAgentRunUpdatePo implements the agent-run update-po command logic.
// It loads configuration and calls RunAgentUpdatePo, then handles errors appropriately.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}

	startTime := time.Now()

	result, err := RunAgentUpdatePo(cfg, agentName, poFile)
	result.ExecutionTime = time.Since(startTime)
	log.Infof("agent-run update-po: execution time: %s", result.ExecutionTime.Round(time.Millisecond))
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
		return fmt.Errorf("file validation failed: %w\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	log.Infof("agent-run update-po completed successfully")
	return nil
}
