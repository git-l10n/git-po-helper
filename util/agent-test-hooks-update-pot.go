package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

type agentTestHooksUpdatePot struct{}

func (agentTestHooksUpdatePot) CleanupBeforeLoop(ctx *AgentRunContext, runNum, totalRuns int) error {
	if err := CleanPoDirectory(filepath.Join(PoDir, GitPot)); err != nil {
		log.Warnf("run %d: failed to clean po/git.pot: %v", runNum, err)
	}
	return nil
}

func (agentTestHooksUpdatePot) ValidateAfterPreCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PotEntriesBeforeUpdate == nil || *cfg.AgentTest.PotEntriesBeforeUpdate == 0 {
		return nil
	}
	expected := *cfg.AgentTest.PotEntriesBeforeUpdate
	actual := 0
	if ctx.PreCheckResult != nil {
		actual = ctx.PreCheckResult.AllEntries
	}
	log.Infof("performing pre-validation: checking entry count before update (expected: %d)", expected)
	if actual != expected {
		potFile := GetPotFilePath()
		return fmt.Errorf("pre-validation failed: entry count before update: expected %d, got %d (file: %s)\nHint: Ensure po/git.pot exists and has the expected number of entries", expected, actual, potFile)
	}
	log.Infof("pre-validation passed")
	return nil
}

func (agentTestHooksUpdatePot) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if cfg.AgentTest.PotEntriesAfterUpdate == nil || *cfg.AgentTest.PotEntriesAfterUpdate == 0 {
		return nil
	}
	expected := *cfg.AgentTest.PotEntriesAfterUpdate
	actual := 0
	if ctx.PostCheckResult != nil {
		actual = ctx.PostCheckResult.AllEntries
	}
	log.Infof("performing post-validation: checking entry count after update (expected: %d)", expected)
	if actual != expected {
		potFile := GetPotFilePath()
		return fmt.Errorf("post-validation failed: entry count after update: expected %d, got %d (file: %s)\nHint: The agent may not have updated the POT file correctly", expected, actual, potFile)
	}
	log.Infof("post-validation passed")
	return nil
}

// ReportSummary prints update-pot specific summary: PreCheck/PostCheck AllEntries (before -> after).
// If all runs agree, shows "n -> m"; otherwise "(n1,n2,...) -> (m1,m2,...)".
func (agentTestHooksUpdatePot) ReportSummary(results []TestRunResult, cfg *config.AgentConfig) {
	if len(results) == 0 {
		return
	}
	var preVals, postVals []int
	for _, r := range results {
		pre, post := 0, 0
		if r.Ctx != nil {
			if r.Ctx.PreCheckResult != nil {
				pre = r.Ctx.PreCheckResult.AllEntries
			}
			if r.Ctx.PostCheckResult != nil {
				post = r.Ctx.PostCheckResult.AllEntries
			}
		}
		preVals = append(preVals, pre)
		postVals = append(postVals, post)
	}
	labelWidth := ReviewStatLabelWidth
	allSamePre := true
	for _, v := range preVals[1:] {
		if v != preVals[0] {
			allSamePre = false
			break
		}
	}
	allSamePost := true
	for _, v := range postVals[1:] {
		if v != postVals[0] {
			allSamePost = false
			break
		}
	}
	if allSamePre && allSamePost {
		fmt.Printf("  %-*s %d -> %d\n", labelWidth, "Number of POT entries:", preVals[0], postVals[0])
	} else {
		preStr := "(" + strings.Join(intsToStrs(preVals), ",") + ")"
		postStr := "(" + strings.Join(intsToStrs(postVals), ",") + ")"
		fmt.Printf("  %-*s %s -> %s\n", labelWidth, "Number of POT entries:", preStr, postStr)
	}
	fmt.Println()
	flushStdout()
}
