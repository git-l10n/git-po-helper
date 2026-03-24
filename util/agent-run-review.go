// Package util provides business logic for agent-run review command.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

// executeReviewAgent executes the agent command for reviewing the given file.
// vars contains placeholder values (e.g. "prompt", "source" for the file to review).
// Returns stdout (for JSON extraction), stderr, originalStdout (raw before parsing), streamResult.
// Updates result with AgentExecuted, AgentStdout, AgentStderr.
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
		return nil, stderr, originalStdout, streamResult, execErr
	}
	log.Infof("agent command completed successfully")

	GetAgentDiagnostics(result, streamResult)
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

	return taskDesc + " according to @{{.agents_md}}."
}

// runAgentReviewDispatch runs either local orchestration or prompt orchestration.
func runAgentReviewDispatch(cfg *config.AgentConfig, agentName string, target *CompareTarget, useLocalOrchestration bool, batchSize int) (*AgentRunResult, error) {
	if target.NewFile != "" && target.OldFile == target.NewFile {
		agentsMd := filepath.Join(filepath.Dir(target.NewFile), "AGENTS.md")
		if !Exist(agentsMd) {
			log.Infof("no AGENTS.md beside %s, using local orchestration", target.NewFile)
			useLocalOrchestration = true
		}
	}
	if useLocalOrchestration {
		if batchSize <= 0 {
			batchSize = 100
		}
		return RunAgentReviewLocalOrchestration(cfg, agentName, target, batchSize)
	}
	return RunAgentReviewPromptOrchestration(cfg, agentName, target)
}

// RunAgentReviewPromptOrchestration executes review using agent with po/AGENTS.md (default mode).
// No programmatic extraction or batching; the agent does everything and writes review.json.
// Before execution: deletes review.po and review.json. After: expects review.json to exist.
func RunAgentReviewPromptOrchestration(cfg *config.AgentConfig, agentName string, target *CompareTarget) (*AgentRunResult, error) {
	ps := GetReviewPathSet()
	workDir, _ := os.Getwd()
	if workDir == "" {
		workDir = "."
	}
	startTime := time.Now()
	result := &AgentRunResult{}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
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
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	poFileRel := target.NewFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		poFileRel = filepath.ToSlash(rel)
	}
	agentsMdPath := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.Clean(poFileRel)), "AGENTS.md"))
	prompt := buildReviewUseAgentMdPrompt(target)
	agentCmd, outputFormat, err := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": poFileRel, "agents_md": agentsMdPath})
	if err != nil {
		return result, fmt.Errorf("failed to build agent command: %w", err)
	}

	log.Infof("executing agent command (prompt orchestration, output=%s): %s", outputFormat, truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	stdout, _, stderr, streamResult, err := RunAgentAndParse(agentCmd, outputFormat, kind)
	if err != nil {
		return result, err
	}
	GetAgentDiagnostics(result, streamResult)
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

	reportResult, err := GetReviewReport()
	if err != nil {
		return result, fmt.Errorf("failed to read review JSON: %w", err)
	}

	result.ReviewResult = reportResult
	result.ExecutionTime = time.Since(startTime)

	score, err := reportResult.GetScore()
	if err != nil {
		return result, fmt.Errorf("review score: %w", err)
	}
	totalEntries, _ := reportResult.GetTotalEntries()
	log.Infof("review completed (score: %d/100, total entries: %d, issues: %d)",
		score, totalEntries, len(reportResult.Issues))

	return result, nil
}

// CmdAgentRunReview implements the agent-run review command logic via AgentRunWorkflow.
func CmdAgentRunReview(agentName string, target *CompareTarget, useLocalOrchestration bool, batchSize int) error {
	if target.NewFile != "" && target.OldFile == target.NewFile {
		absPo, err := filepath.Abs(target.NewFile)
		if err != nil {
			return fmt.Errorf("cannot resolve PO file path: %w", err)
		}
		absPo = filepath.Clean(absPo)
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
				target.OldFile = relSlash
				target.NewFile = relSlash
			}
		}
	}

	return RunAgentRunWorkflow(NewWorkflowReview(agentName, target, useLocalOrchestration, batchSize))
}
