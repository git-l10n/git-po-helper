package util

import (
	"fmt"
	"strings"

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
	expected := *cfg.AgentTest.PoEntriesBeforeUpdate
	actual := 0
	if ctx.PreCheckResult != nil {
		actual = ctx.PreCheckResult.AllEntries
	}
	log.Infof("performing pre-validation: checking PO entry count before update (expected: %d)", expected)
	if actual != expected {
		return fmt.Errorf("pre-validation failed: entry count before update: expected %d, got %d (file: %s)\nHint: Ensure %s exists and has the expected number of entries", expected, actual, h.relPoFile, h.relPoFile)
	}
	log.Infof("pre-validation passed")
	return nil
}

func (h agentTestHooksUpdatePo) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	post := ctx.PostCheckResult
	if post == nil {
		return nil
	}
	// Check total entry count when configured
	if cfg.AgentTest.PoEntriesAfterUpdate != nil && *cfg.AgentTest.PoEntriesAfterUpdate != 0 {
		expected := *cfg.AgentTest.PoEntriesAfterUpdate
		actual := post.AllEntries
		log.Infof("performing post-validation: checking PO entry count after update (expected: %d)", expected)
		if actual != expected {
			return fmt.Errorf("post-validation failed: entry count after update: expected %d, got %d (file: %s)\nHint: The agent may not have updated the PO file correctly", expected, actual, h.relPoFile)
		}
	}
	// Check untranslated (new) entries after update when configured
	if cfg.AgentTest.PoNewEntriesAfterUpdate != nil {
		expected := *cfg.AgentTest.PoNewEntriesAfterUpdate
		actual := post.UntranslatePoEntries
		log.Infof("performing post-validation: checking PO untranslated entries after update (expected: %d)", expected)
		if actual != expected {
			return fmt.Errorf("post-validation failed: untranslated entries after update: expected %d, got %d (file: %s)", expected, actual, h.relPoFile)
		}
	}
	// Check fuzzy entries after update when configured
	if cfg.AgentTest.PoFuzzyEntriesAfterUpdate != nil {
		expected := *cfg.AgentTest.PoFuzzyEntriesAfterUpdate
		actual := post.FuzzyPoEntries
		log.Infof("performing post-validation: checking PO fuzzy entries after update (expected: %d)", expected)
		if actual != expected {
			return fmt.Errorf("post-validation failed: fuzzy entries after update: expected %d, got %d (file: %s)", expected, actual, h.relPoFile)
		}
	}
	log.Infof("post-validation passed")
	return nil
}

// ReportSummary prints update-po specific summary: PreCheck/PostCheck translated, fuzzy, untranslated.
// If all runs agree, shows "n -> m"; otherwise "(n1,n2,...) -> (m1,m2,...)".
func (agentTestHooksUpdatePo) ReportSummary(results []TestRunResult, cfg *config.AgentConfig) {
	if len(results) == 0 {
		return
	}
	getPre := func(r TestRunResult) (all, untrans, fuzzy int) {
		if r.Ctx != nil && r.Ctx.PreCheckResult != nil {
			p := r.Ctx.PreCheckResult
			return p.AllEntries, p.UntranslatePoEntries, p.FuzzyPoEntries
		}
		return 0, 0, 0
	}
	getPost := func(r TestRunResult) (all, untrans, fuzzy int) {
		if r.Ctx != nil && r.Ctx.PostCheckResult != nil {
			p := r.Ctx.PostCheckResult
			return p.AllEntries, p.UntranslatePoEntries, p.FuzzyPoEntries
		}
		return 0, 0, 0
	}
	var translatedPre, translatedPost []int
	var fuzzyPre, fuzzyPost []int
	var untranslatedPre, untranslatedPost []int
	for _, r := range results {
		pAll, pUntrans, pFuzzy := getPre(r)
		gAll, gUntrans, gFuzzy := getPost(r)
		translatedPre = append(translatedPre, pAll-pUntrans-pFuzzy)
		translatedPost = append(translatedPost, gAll-gUntrans-gFuzzy)
		fuzzyPre = append(fuzzyPre, pFuzzy)
		fuzzyPost = append(fuzzyPost, gFuzzy)
		untranslatedPre = append(untranslatedPre, pUntrans)
		untranslatedPost = append(untranslatedPost, gUntrans)
	}
	labelWidth := ReportLabelWidth
	printLine := func(label string, preVals, postVals []int) {
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
			fmt.Printf("  %-*s %d -> %d\n", labelWidth, label+":", preVals[0], postVals[0])
		} else {
			preStr := "(" + strings.Join(intsToStrs(preVals), ",") + ")"
			postStr := "(" + strings.Join(intsToStrs(postVals), ",") + ")"
			fmt.Printf("  %-*s %s -> %s\n", labelWidth, label+":", preStr, postStr)
		}
	}
	printLine("translated", translatedPre, translatedPost)
	printLine("untranslated", untranslatedPre, untranslatedPost)
	printLine("fuzzy", fuzzyPre, fuzzyPost)
	fmt.Println()
	flushStdout()
}

func intsToStrs(ns []int) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = fmt.Sprintf("%d", n)
	}
	return out
}
