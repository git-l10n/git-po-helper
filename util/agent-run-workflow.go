// Package util provides agent-run workflow abstraction (pre-check, agent-run,
// post-check, report) for update-pot, update-po, translate, and review.
package util

import (
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// AgentRunContext holds shared state for one agent-run execution through
// PreCheck → AgentRun → PostCheck → Report.
type AgentRunContext struct {
	Cfg                   *config.AgentConfig
	AgentName             string
	Result                *AgentRunResult
	PoFile                string // relative or empty; resolved by workflows
	Target                *CompareTarget
	UseLocalOrchestration bool
	BatchSize             int

	// Set by workflows during execution
	poFileAbs string // absolute PO path when applicable
	potFile   string // update-pot
}

// AgentRunWorkflow is the interface for agent-run subcommands. Each command
// implements the four phases; agent-test continues to call RunAgent* directly.
type AgentRunWorkflow interface {
	// Name returns the subcommand name for logging (e.g. "update-pot").
	Name() string
	// InitContext builds a new context with Result initialized; wf stores
	// command-specific parameters on the workflow struct.
	InitContext(cfg *config.AgentConfig) *AgentRunContext
	// PreCheck runs before the agent; on error the agent must not run.
	PreCheck(ctx *AgentRunContext) error
	// AgentRun executes the agent; agentRunErr is non-nil if the process failed.
	AgentRun(ctx *AgentRunContext) (agentRunErr error)
	// PostCheck runs after the agent; may set PostValidationError / SyntaxValidationError.
	PostCheck(ctx *AgentRunContext) error
	// Report prints command-specific output and returns a terminal error for the CLI.
	Report(ctx *AgentRunContext, agentRunErr error) error
}

// RunAgentRunWorkflow loads config, runs PreCheck → AgentRun → PostCheck → Report,
// and sets ExecutionTime on result after AgentRun.
func RunAgentRunWorkflow(wf AgentRunWorkflow) error {
	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		return err
	}
	ctx := wf.InitContext(cfg)
	if ctx.Result == nil {
		ctx.Result = &AgentRunResult{Score: 0}
	}
	start := time.Now()
	if err := wf.PreCheck(ctx); err != nil {
		return err
	}
	agentErr := wf.AgentRun(ctx)
	ctx.Result.ExecutionTime = time.Since(start)
	log.Infof("agent-run %s: execution time: %s", wf.Name(), ctx.Result.ExecutionTime.Round(time.Millisecond))
	postErr := wf.PostCheck(ctx)
	// Report runs even when agent or post-check failed (e.g. translate prints after-stats then returns error).
	if reportErr := wf.Report(ctx, agentErr); reportErr != nil {
		return reportErr
	}
	return postErr
}
