package util

import (
	"fmt"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

type agentTestHooksUpdatePot struct{}

func (agentTestHooksUpdatePot) CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error {
	if err := CleanPoDirectory("po/git.pot"); err != nil {
		log.Warnf("run %d: failed to clean po/ directory: %v", runNum, err)
	}
	return nil
}

func (agentTestHooksUpdatePot) ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PotEntriesBeforeUpdate == nil || *cfg.AgentTest.PotEntriesBeforeUpdate == 0 {
		return nil
	}
	potFile := GetPotFilePath()
	log.Infof("performing pre-validation: checking entry count before update (expected: %d)", *cfg.AgentTest.PotEntriesBeforeUpdate)
	if ctx.PreCheckResult == nil {
		ctx.PreCheckResult = &PreCheckResult{}
	}
	if !Exist(potFile) {
		ctx.PreCheckResult.AllEntries = 0
	} else if stats, e := GetPoStats(potFile); e == nil {
		ctx.PreCheckResult.AllEntries = stats.Total()
	}
	if err := ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesBeforeUpdate, "before update"); err != nil {
		return fmt.Errorf("pre-validation failed: %w\nHint: Ensure po/git.pot exists and has the expected number of entries", err)
	}
	log.Infof("pre-validation passed")
	return nil
}

func (agentTestHooksUpdatePot) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PotEntriesAfterUpdate == nil || *cfg.AgentTest.PotEntriesAfterUpdate == 0 {
		return nil
	}
	potFile := GetPotFilePath()
	log.Infof("performing post-validation: checking entry count after update (expected: %d)", *cfg.AgentTest.PotEntriesAfterUpdate)
	if ctx.PostCheckResult == nil {
		ctx.PostCheckResult = &PostCheckResult{}
	}
	if Exist(potFile) {
		if stats, e := GetPoStats(potFile); e == nil {
			ctx.PostCheckResult.AllEntries = stats.Total()
		}
	}
	if err := ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesAfterUpdate, "after update"); err != nil {
		return fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the POT file correctly", err)
	}
	log.Infof("post-validation passed")
	return nil
}
