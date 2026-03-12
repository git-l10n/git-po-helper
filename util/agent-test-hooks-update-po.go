package util

import (
	"fmt"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// agentTestHooksUpdatePo holds resolved PO path for cleanup/validation.
type agentTestHooksUpdatePo struct {
	relPoFile string
}

func (h agentTestHooksUpdatePo) CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error {
	if err := CleanPoDirectory(h.relPoFile, "po/git.pot"); err != nil {
		log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
	}
	return nil
}

func (h agentTestHooksUpdatePo) ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PoEntriesBeforeUpdate == nil || *cfg.AgentTest.PoEntriesBeforeUpdate == 0 {
		return nil
	}
	log.Infof("performing pre-validation: checking PO entry count before update (expected: %d)", *cfg.AgentTest.PoEntriesBeforeUpdate)
	if ctx.PreCheckResult == nil {
		ctx.PreCheckResult = &PreCheckResult{}
	}
	if !Exist(h.relPoFile) {
		ctx.PreCheckResult.AllEntries = 0
	} else if stats, e := GetPoStats(h.relPoFile); e == nil {
		ctx.PreCheckResult.AllEntries = stats.Total()
	}
	if err := ValidatePoEntryCount(h.relPoFile, cfg.AgentTest.PoEntriesBeforeUpdate, "before update"); err != nil {
		return fmt.Errorf("pre-validation failed: %w\nHint: Ensure %s exists and has the expected number of entries", err, h.relPoFile)
	}
	log.Infof("pre-validation passed")
	return nil
}

func (h agentTestHooksUpdatePo) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PoEntriesAfterUpdate == nil || *cfg.AgentTest.PoEntriesAfterUpdate == 0 {
		return nil
	}
	log.Infof("performing post-validation: checking PO entry count after update (expected: %d)", *cfg.AgentTest.PoEntriesAfterUpdate)
	if ctx.PostCheckResult == nil {
		ctx.PostCheckResult = &PostCheckResult{}
	}
	if Exist(h.relPoFile) {
		if stats, e := GetPoStats(h.relPoFile); e == nil {
			ctx.PostCheckResult.AllEntries = stats.Total()
		}
	}
	if err := ValidatePoEntryCount(h.relPoFile, cfg.AgentTest.PoEntriesAfterUpdate, "after update"); err != nil {
		return fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the PO file correctly", err)
	}
	log.Infof("post-validation passed")
	return nil
}
