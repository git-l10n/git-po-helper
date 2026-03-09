// Package util provides business logic for agent-run review command.
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

// executeReviewAgent executes the agent command for reviewing the given file.
// vars contains placeholder values (e.g. "prompt", "source" for the file to review).
// Returns stdout (for JSON extraction), stderr, originalStdout (raw before parsing), streamResult.
// Updates result with AgentExecuted, AgentError, AgentStdout, AgentStderr.
func executeReviewAgent(selectedAgent config.AgentEntry, vars PlaceholderVars, result *AgentRunResult) (stdout, stderr, originalStdout []byte, streamResult AgentStreamResult, err error) {
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, vars)
	if err != nil {
		return nil, nil, nil, streamResult, fmt.Errorf("failed to build agent command: %w", err)
	}

	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat,
		outputFormat == config.OutputJSON || outputFormat == config.OutputStreamJSON,
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	stdout, originalStdout, stderr, streamResult, execErr := RunAgentAndParse(agentCmd, outputFormat, kind)
	if execErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
		if len(stdout) > 0 {
			log.Debugf("agent command stdout: %s", string(stdout))
		}
		result.AgentError = execErr
		return nil, stderr, originalStdout, streamResult, execErr
	}
	log.Infof("agent command completed successfully")

	applyAgentDiagnostics(result, streamResult)
	result.AgentStdout = originalStdout
	if len(stderr) > 0 {
		result.AgentStderr = stderr
	}

	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	return stdout, stderr, originalStdout, streamResult, nil
}

// buildReviewUseAgentMdPrompt constructs a dynamic prompt for RunAgentReviewUseAgentMd
func buildReviewUseAgentMdPrompt(target *CompareTarget) string {
	var taskDesc string
	if target.OldFile != target.NewFile {
		taskDesc = fmt.Sprintf("Review %s changes between %s and %s", target.NewFile, target.OldFile, target.NewFile)
	} else if target.NewCommit != "" && (target.OldCommit == target.NewCommit+"~" ||
		target.OldCommit == target.NewCommit+"~1" ||
		target.OldCommit == target.NewCommit+"^") {
		taskDesc = fmt.Sprintf("Review %s changes in commit %s", target.NewFile, target.NewCommit)
	} else if target.NewCommit == "" && target.OldCommit != "" {
		if target.OldCommit == "HEAD" {
			taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
		} else {
			taskDesc = fmt.Sprintf("Review %s changes since commit %s", target.NewFile, target.OldCommit)
		}
	} else if target.OldCommit != "" && target.NewCommit != "" {
		taskDesc = fmt.Sprintf("Review %s changes in range %s..%s", target.NewFile, target.OldCommit, target.NewCommit)
	} else {
		taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
	}

	return taskDesc + " according to @po/AGENTS.md."
}

// RunAgentReview executes review using agent with po/AGENTS.md (default mode).
// No programmatic extraction or batching; the agent does everything and writes review.json.
// Before execution: deletes review.po and review.json. After: expects review.json to exist.
func RunAgentReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	ps := ReviewPathSetFromBase(outputBase)
	workDir := repository.WorkDirOrCwd()
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err
		return result, err
	}

	if Exist(ps.PendingPO) {
		log.Warnf("review PO file already exists: %s", ps.PendingPO)
	}

	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return result, err
	}
	if !Exist(poFile) {
		result.AgentError = fmt.Errorf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	poFileRel := target.NewFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		poFileRel = filepath.ToSlash(rel)
	}
	prompt := buildReviewUseAgentMdPrompt(target)
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": poFileRel})
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	log.Infof("executing agent command (use-agent-md, output=%s): %s", outputFormat, truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	stdout, _, stderr, streamResult, err := RunAgentAndParse(agentCmd, outputFormat, kind)
	if err != nil {
		result.AgentError = err
		return result, err
	}
	applyAgentDiagnostics(result, streamResult)
	log.Infof("agent command completed successfully")

	if len(stdout) > 0 {
		log.Debugf("agent stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent stderr: %s", string(stderr))
	}

	if !Exist(ps.ResultJSON) {
		return result, fmt.Errorf("review JSON not generated at %s\nHint: The agent must write the review result to this file", ps.ResultJSON)
	}

	_, reportResult, err := ReportReviewFromJSON(ps.ResultJSON)
	if err != nil {
		return result, fmt.Errorf("failed to read review JSON: %w", err)
	}

	result.ReviewJSON = reportResult.Review
	result.ReviewJSONPath = ps.ResultJSON
	result.ReviewScore = reportResult.Score
	result.Score = reportResult.Score
	result.ReviewedFilePath = poFile
	result.ExecutionTime = time.Since(startTime)

	log.Infof("review completed (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))

	return result, nil
}

// CmdAgentRunReview implements the agent-run review command logic.
// It loads configuration and calls RunAgentReview or RunAgentReviewUseAgentMd.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// useLocalOrchestration: if true, use local orchestration (--use-local-orchestration);
// otherwise use agent with po/AGENTS.md (default).
func CmdAgentRunReview(agentName string, target *CompareTarget, outputBase string, useLocalOrchestration bool, batchSize int) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	var result *AgentRunResult
	if useLocalOrchestration {
		result, err = RunAgentReviewLocalOrchestration(cfg, agentName, target, false, outputBase, batchSize)
	} else {
		result, err = RunAgentReview(cfg, agentName, target, false, outputBase)
	}
	if err != nil {
		return err
	}

	// For agent-run, we require agent execution to succeed (no error set)
	if result.AgentError != nil {
		return fmt.Errorf("agent execution failed: %w", result.AgentError)
	}

	elapsed := time.Since(startTime)

	// Display review report (same format as agent-run report)
	if result.ReviewJSON != nil && result.ReviewJSONPath != "" {
		critical, minor, major := CountReviewIssueScores(result.ReviewJSON)
		reportResult := &ReviewReportResult{
			Review:        result.ReviewJSON,
			Score:         result.ReviewScore,
			CriticalCount: critical,
			MinorCount:    minor,
			MajorCount:    major,
		}
		PrintReviewReportResult(result.ReviewJSONPath, reportResult)
	}

	fmt.Printf("\nSummary:\n")
	if result.ReviewJSONPath != "" {
		fmt.Printf("  Review JSON: %s\n", getRelativePath(result.ReviewJSONPath))
	}
	if result.NumTurns > 0 {
		fmt.Printf("  Turns: %d\n", result.NumTurns)
	}
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run review completed successfully")
	return nil
}
