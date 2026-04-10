// Package util provides agent-test direct prompt execution (no workflow).
package util

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// CmdAgentTestDirect runs the configured agent multiple times with the given prompt text
// (no workflow), like agent-run direct mode, and prints an aggregate summary.
func CmdAgentTestDirect(agentName, prompt string, runs int) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt is empty")
	}

	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}
	runs = ResolveAgentTestRuns(cfg, runs)
	log.Infof("starting agent-test direct with %d runs", runs)

	start := time.Now()
	results := make([]TestRunResult, runs)
	for i := 0; i < runs; i++ {
		runNum := i + 1
		log.Infof("⏳ Starting loop %d/%d (direct)", runNum, runs)
		result, runErr := runAgentDirectOnce(cfg, agentName, prompt)
		if runErr != nil {
			return runErr
		}
		PrintAgentDiagnosticsFromResult(result)
		ctx := &AgentRunContext{Cfg: cfg, Result: result}
		PrintAgentRunStatus(ctx)
		results[i] = TestRunResult{
			AgentRunResult: *result,
			RunNumber:      runNum,
			Ctx:            ctx,
		}
	}

	fmt.Println()
	fmt.Printf("========== ✅ Summary of [direct] workflow ==========\n")
	fmt.Println()
	PrintAgentTestSummaryReport(results, time.Since(start))
	return nil
}
