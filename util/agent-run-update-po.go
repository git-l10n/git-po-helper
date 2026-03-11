// Package util provides business logic for agent-run update-po command.
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

// RunAgentUpdatePo executes a single agent-run update-po operation.
// It performs pre-validation, executes the agent command, performs post-validation,
// and validates PO file syntax. Returns a result structure with detailed information.
// The agentTest parameter controls whether AgentTest configuration should be used.
// When agentTest is false (for agent-run), AgentTest configuration is ignored.
func RunAgentUpdatePo(cfg *config.AgentConfig, agentName, poFile string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
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

	// Pre-validation: Check entry count before update (only for agent-test)
	if agentTest && cfg.AgentTest.PoEntriesBeforeUpdate != nil && *cfg.AgentTest.PoEntriesBeforeUpdate != 0 {
		log.Infof("performing pre-validation: checking PO entry count before update (expected: %d)", *cfg.AgentTest.PoEntriesBeforeUpdate)

		// Get before count for result
		if !Exist(poFile) {
			result.EntryCountBeforeUpdate = 0
		} else {
			if stats, err := GetPoStats(poFile); err == nil {
				result.EntryCountBeforeUpdate = stats.Total()
			}
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesBeforeUpdate, "before update"); err != nil {
			result.PreValidationError = fmt.Errorf("pre-validation failed: %w\nHint: Ensure %s exists and has the expected number of entries", err, poFile)
			return result, result.PreValidationError
		}
		log.Infof("pre-validation passed")
	} else {
		// No pre-validation configured, count entries for display purposes
		if !Exist(poFile) {
			result.EntryCountBeforeUpdate = 0
		} else {
			if stats, err := GetPoStats(poFile); err == nil {
				result.EntryCountBeforeUpdate = stats.Total()
			}
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

	// Post-validation: Check entry count after update (only for agent-test)
	if agentTest && cfg.AgentTest.PoEntriesAfterUpdate != nil && *cfg.AgentTest.PoEntriesAfterUpdate != 0 {
		log.Infof("performing post-validation: checking PO entry count after update (expected: %d)", *cfg.AgentTest.PoEntriesAfterUpdate)

		// Get after count for result
		if Exist(poFile) {
			if stats, err := GetPoStats(poFile); err == nil {
				result.EntryCountAfterUpdate = stats.Total()
			}
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesAfterUpdate, "after update"); err != nil {
			result.PostValidationError = fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the PO file correctly", err)
			result.Score = 0
			return result, result.PostValidationError
		}
		result.Score = 100
		log.Infof("post-validation passed")
	} else {
		// No post-validation configured, score based on agent exit code (we only reach here when agent succeeded)
		if Exist(poFile) {
			if stats, err := GetPoStats(poFile); err == nil {
				result.EntryCountAfterUpdate = stats.Total()
			}
		}
		result.Score = 100
	}

	// Validate PO file syntax (we only reach here when agent succeeded)
	log.Infof("validating file syntax: %s", poFile)
	if err := ValidatePoFile(poFile); err != nil {
		log.Errorf("file syntax validation failed: %v", err)
		result.SyntaxValidationError = err
	} else {
		log.Infof("file syntax validation passed")
	}

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// CmdAgentRunUpdatePo implements the agent-run update-po command logic.
// It loads configuration and calls RunAgentUpdatePo, then handles errors appropriately.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	result, err := RunAgentUpdatePo(cfg, agentName, poFile, false)
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

	elapsed := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run update-po completed successfully")
	return nil
}
