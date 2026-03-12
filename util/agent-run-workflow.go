// Package util provides agent-run workflow abstraction (pre-check, agent-run,
// post-check, report) for update-pot, update-po, translate, and review.
package util

import (
	"time"

	"github.com/git-l10n/git-po-helper/config"
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

	// Pre/post check results; workflows write here; synced to Result before Report.
	PreCheckResult  *PreCheckResult
	PostCheckResult *PostCheckResult

	// Set by workflows during execution
	potFile string // update-pot
}

// PreValidationError returns the pre-check error from ctx; nil when ctx or PreCheckResult is nil.
func (ctx *AgentRunContext) PreValidationError() error {
	if ctx != nil && ctx.PreCheckResult != nil {
		return ctx.PreCheckResult.Error
	}
	return nil
}

// PostValidationError returns the post-check error from ctx; nil when ctx or PostCheckResult is nil.
func (ctx *AgentRunContext) PostValidationError() error {
	if ctx != nil && ctx.PostCheckResult != nil {
		return ctx.PostCheckResult.Error
	}
	return nil
}

// EntryCountBeforeUpdate returns PO/POT entry count before update from ctx.
func (ctx *AgentRunContext) EntryCountBeforeUpdate() int {
	if ctx != nil && ctx.PreCheckResult != nil {
		return ctx.PreCheckResult.AllEntries
	}
	return 0
}

// EntryCountAfterUpdate returns PO/POT entry count after update from ctx.
func (ctx *AgentRunContext) EntryCountAfterUpdate() int {
	if ctx != nil && ctx.PostCheckResult != nil {
		return ctx.PostCheckResult.AllEntries
	}
	return 0
}

// BeforeNewCount returns translate new (untranslated) entries before from ctx.
func (ctx *AgentRunContext) BeforeNewCount() int {
	if ctx != nil && ctx.PreCheckResult != nil {
		return ctx.PreCheckResult.UntranslatePoEntries
	}
	return 0
}

// AfterNewCount returns translate new (untranslated) entries after from ctx.
func (ctx *AgentRunContext) AfterNewCount() int {
	if ctx != nil && ctx.PostCheckResult != nil {
		return ctx.PostCheckResult.UntranslatePoEntries
	}
	return 0
}

// BeforeFuzzyCount returns translate fuzzy entries before from ctx.
func (ctx *AgentRunContext) BeforeFuzzyCount() int {
	if ctx != nil && ctx.PreCheckResult != nil {
		return ctx.PreCheckResult.FuzzyPoEntries
	}
	return 0
}

// AfterFuzzyCount returns translate fuzzy entries after from ctx.
func (ctx *AgentRunContext) AfterFuzzyCount() int {
	if ctx != nil && ctx.PostCheckResult != nil {
		return ctx.PostCheckResult.FuzzyPoEntries
	}
	return 0
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
	// PostCheck runs after the agent; may set PostCheckResult.Error.
	PostCheck(ctx *AgentRunContext) error
	// Report prints command-specific output only; terminal status comes from ctx.Result.Error
	// and ctx pre/post validation (returned by RunAgentRunWorkflow after Report).
	Report(ctx *AgentRunContext)
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
	err = wf.AgentRun(ctx)
	if ctx.Result == nil {
		ctx.Result = &AgentRunResult{Score: 0}
	}
	ctx.Result.Error = err
	ctx.Result.ExecutionTime = time.Since(start)
	// Print agent diagnostics from ctx.Result before workflow Report (fields filled by GetAgentDiagnostics during AgentRun).
	PrintAgentDiagnosticsFromResult(ctx.Result)
	// Report runs even when agent or post-check failed (e.g. translate prints after-stats then returns error).
	_ = wf.PostCheck(ctx)
	wf.Report(ctx)
	if ctx.Result.Error != nil {
		return ctx.Result.Error
	} else if ctx.PreValidationError() != nil {
		return ctx.PreValidationError()
	} else if ctx.PostValidationError() != nil {
		return ctx.PostValidationError()
	} else {
		return nil
	}
}
