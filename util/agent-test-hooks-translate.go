package util

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/git-l10n/git-po-helper/config"
)

// agentTestHooksTranslate cleans po/ and l10n intermediates before each loop.
// PreCheck/PostCheck on workflowTranslate already perform validation.
type agentTestHooksTranslate struct {
	relPoFile string
}

func (h agentTestHooksTranslate) CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error {
	if err := CleanPoDirectory(h.relPoFile); err != nil {
		log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
	}
	cleanL10nIntermediateFiles()
	return nil
}

func (agentTestHooksTranslate) ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	return nil
}

func (agentTestHooksTranslate) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	return nil
}

func (agentTestHooksTranslate) ReportSummary(results []TestRunResult, cfg *config.AgentConfig) {
	runs := len(results)
	var sumScore int
	var totalExecution time.Duration
	for _, r := range results {
		sumScore += r.Score
		totalExecution += r.ExecutionTime
	}
	var averageScore float64
	if runs > 0 {
		averageScore = float64(sumScore) / float64(runs)
	}
	displayTranslateTestResults(results, averageScore, runs, totalExecution)
}
