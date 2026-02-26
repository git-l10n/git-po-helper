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
		result.AgentError = err.Error()
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
			result.BeforeCount = 0
		} else {
			if stats, err := CountPoReportStats(poFile); err == nil {
				result.BeforeCount = stats.Total()
			}
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesBeforeUpdate, "before update"); err != nil {
			log.Errorf("pre-validation failed: %v", err)
			result.PreValidationError = err.Error()
			return result, fmt.Errorf("pre-validation failed: %w\nHint: Ensure %s exists and has the expected number of entries", err, poFile)
		}
		result.PreValidationPass = true
		log.Infof("pre-validation passed")
	} else {
		// No pre-validation configured, count entries for display purposes
		if !Exist(poFile) {
			result.BeforeCount = 0
		} else {
			if stats, err := CountPoReportStats(poFile); err == nil {
				result.BeforeCount = stats.Total()
			}
		}
		result.PreValidationPass = true // Consider it passed if not configured
	}

	// Get prompt for update-po from configuration
	prompt, err := GetRawPrompt(cfg, "update-po")
	if err != nil {
		return result, err
	}

	workDir := repository.WorkDir()
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
	agentCmd, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	// Determine output format
	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	// Normalize output format (convert underscores to hyphens)
	outputFormat = normalizeOutputFormat(outputFormat)

	// Execute agent command
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json",
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	var stdout []byte
	var stderr []byte
	var streamResult AgentStreamResult

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	// Use streaming execution for json format (treated as stream-json)
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd)
		if err != nil {
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		defer stdoutReader.Close()

		stdout, streamResult, _ = parseStreamByKind(kind, stdoutReader)

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd)
		if err != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", err)
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, err := ParseClaudeAgentOutput(stdout, outputFormat)
			if err != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", err)
				parsedStdout = stdout
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

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
			if stats, err := CountPoReportStats(poFile); err == nil {
				result.AfterCount = stats.Total()
			}
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesAfterUpdate, "after update"); err != nil {
			log.Errorf("post-validation failed: %v", err)
			result.PostValidationError = err.Error()
			result.Score = 0
			return result, fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the PO file correctly", err)
		}
		result.PostValidationPass = true
		result.Score = 100
		log.Infof("post-validation passed")
	} else {
		// No post-validation configured, score based on agent exit code
		if Exist(poFile) {
			if stats, err := CountPoReportStats(poFile); err == nil {
				result.AfterCount = stats.Total()
			}
		}
		if result.AgentSuccess {
			result.Score = 100
			result.PostValidationPass = true // Consider it passed if agent succeeded
		} else {
			result.Score = 0
		}
	}

	// Validate PO file syntax (only if agent succeeded)
	if result.AgentSuccess {
		log.Infof("validating file syntax: %s", poFile)
		if err := ValidatePoFile(poFile); err != nil {
			log.Errorf("file syntax validation failed: %v", err)
			result.SyntaxValidationError = err.Error()
			// Don't fail the run for syntax errors in agent-run, but log it
			// In agent-test, this might affect the score
		} else {
			result.SyntaxValidationPass = true
			log.Infof("file syntax validation passed")
		}
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
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	result, err := RunAgentUpdatePo(cfg, agentName, poFile, false)
	if err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
	if !result.PreValidationPass {
		return fmt.Errorf("pre-validation failed: %s", result.PreValidationError)
	}
	if !result.AgentSuccess {
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}
	if !result.PostValidationPass {
		return fmt.Errorf("post-validation failed: %s", result.PostValidationError)
	}
	if result.SyntaxValidationError != "" {
		return fmt.Errorf("file validation failed: %s\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run update-po completed successfully")
	return nil
}
