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
// It performs pre-validation (count new/fuzzy entries), executes the agent command,
// performs post-validation (verify new=0 and fuzzy=0), and validates PO file syntax.
// Returns a result structure with detailed information.
// The agentTest parameter is provided for consistency, though this method
// does not use AgentTest configuration.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Determine PO file path
	poFile, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}

	log.Debugf("PO file path: %s", poFile)

	// Check if PO file exists
	if !Exist(poFile) {
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	// Pre-validation: Count new and fuzzy entries before translation
	log.Infof("performing pre-validation: counting new and fuzzy entries")

	statsBefore, err := CountReportStats(poFile)
	if err != nil {
		return result, fmt.Errorf("failed to count PO stats: %w", err)
	}
	result.BeforeNewCount = statsBefore.Untranslated
	result.BeforeFuzzyCount = statsBefore.Fuzzy
	log.Infof("new (untranslated) entries before translation: %d", statsBefore.Untranslated)
	log.Infof("fuzzy entries before translation: %d", statsBefore.Fuzzy)

	// Check if there's anything to translate
	if statsBefore.Untranslated == 0 && statsBefore.Fuzzy == 0 {
		log.Infof("no new or fuzzy entries to translate, PO file is already complete")
		result.PreValidationPass = true
		result.PostValidationPass = true
		result.Score = 100
		return result, nil
	}

	result.PreValidationPass = true

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
			result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
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
			result.AgentError = fmt.Errorf("agent command failed: %v (see logs for agent stderr output)", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
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

	// Post-validation: Count new and fuzzy entries after translation
	log.Infof("performing post-validation: counting new and fuzzy entries")

	statsAfter, err := CountReportStats(poFile)
	if err != nil {
		return result, fmt.Errorf("failed to count PO stats after translation: %w", err)
	}
	result.AfterNewCount = statsAfter.Untranslated
	result.AfterFuzzyCount = statsAfter.Fuzzy
	log.Infof("new (untranslated) entries after translation: %d", statsAfter.Untranslated)
	log.Infof("fuzzy entries after translation: %d", statsAfter.Fuzzy)

	// Validate translation success: both new and fuzzy entries must be 0
	if statsAfter.Untranslated != 0 || statsAfter.Fuzzy != 0 {
		result.PostValidationError = fmt.Sprintf("translation incomplete: %d new entries and %d fuzzy entries remaining", statsAfter.Untranslated, statsAfter.Fuzzy)
		result.Score = 0
		return result, fmt.Errorf("post-validation failed: %s\nHint: The agent should translate all new entries and resolve all fuzzy entries", result.PostValidationError)
	}

	result.PostValidationPass = true
	result.Score = 100
	log.Infof("post-validation passed: all entries translated")

	// Validate PO file syntax (only if agent succeeded)
	if result.AgentError == nil {
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

// CmdAgentRunTranslate implements the agent-run translate command logic.
// It loads configuration and calls RunAgentTranslate or RunAgentTranslateLocalOrchestration
// based on useAgentMd/useLocalOrchestration flags, then handles errors appropriately.
func CmdAgentRunTranslate(agentName, poFile string, useAgentMd, useLocalOrchestration bool, batchSize int) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	if useLocalOrchestration {
		if batchSize <= 0 {
			batchSize = 50
		}
		result, err := RunAgentTranslateLocalOrchestration(cfg, agentName, poFile, batchSize)
		if err != nil {
			log.Errorf("failed to run agent translate local orchestration: %v", err)
			return err
		}
		if result.AgentError != nil {
			return fmt.Errorf("agent execution failed: %w", result.AgentError)
		}
		elapsed := time.Since(startTime)
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))
		log.Infof("agent-run translate (local orchestration) completed successfully")
		return nil
	}

	result, err := RunAgentTranslate(cfg, agentName, poFile, false)
	if err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
	if !result.PreValidationPass {
		return fmt.Errorf("pre-validation failed: %s", result.PreValidationError)
	}
	if result.AgentError != nil {
		return fmt.Errorf("agent execution failed: %w", result.AgentError)
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

	log.Infof("agent-run translate completed successfully")
	return nil
}
