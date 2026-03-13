package util

import (
	"fmt"
	"strings"

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
	if ctx.PreCheckResult == nil {
		return nil
	}
	untrans := ctx.PreCheckResult.UntranslatePoEntries
	fuzzy := ctx.PreCheckResult.FuzzyPoEntries
	if untrans == 0 && fuzzy == 0 {
		return fmt.Errorf("pre-validation failed: untranslated and fuzzy cannot both be 0 (no entries to translate)")
	}
	log.Infof("pre-validation passed (untranslated=%d, fuzzy=%d)", untrans, fuzzy)
	return nil
}

func (agentTestHooksTranslate) ValidateAfterPostCheck(ctx *AgentRunContext, cfg *config.AgentConfig) error {
	if ctx.PostCheckResult == nil {
		return nil
	}
	untrans := ctx.PostCheckResult.UntranslatePoEntries
	fuzzy := ctx.PostCheckResult.FuzzyPoEntries
	if untrans != 0 || fuzzy != 0 {
		return fmt.Errorf("post-validation failed: after translation untranslated and fuzzy must be 0, got untranslated=%d fuzzy=%d",
			untrans, fuzzy)
	}
	log.Infof("post-validation passed (untranslated=0, fuzzy=0)")
	return nil
}

// ReportSummary prints translate-specific summary: PreCheck/PostCheck new (untranslated)
// and fuzzy entries. If all runs agree, shows "n -> m"; otherwise "(n1,n2,...) -> (m1,m2,...)".
func (agentTestHooksTranslate) ReportSummary(results []TestRunResult, cfg *config.AgentConfig) {
	if len(results) == 0 {
		return
	}
	var newPre, newPost []int
	var fuzzyPre, fuzzyPost []int
	for _, r := range results {
		preNew, preFuzzy := 0, 0
		postNew, postFuzzy := 0, 0
		if r.Ctx != nil {
			if r.Ctx.PreCheckResult != nil {
				preNew = r.Ctx.PreCheckResult.UntranslatePoEntries
				preFuzzy = r.Ctx.PreCheckResult.FuzzyPoEntries
			}
			if r.Ctx.PostCheckResult != nil {
				postNew = r.Ctx.PostCheckResult.UntranslatePoEntries
				postFuzzy = r.Ctx.PostCheckResult.FuzzyPoEntries
			}
		}
		newPre = append(newPre, preNew)
		newPost = append(newPost, postNew)
		fuzzyPre = append(fuzzyPre, preFuzzy)
		fuzzyPost = append(fuzzyPost, postFuzzy)
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
	printLine("Translating New", newPre, newPost)
	printLine("Translating Fuzzy", fuzzyPre, fuzzyPost)
	fmt.Println()
	flushStdout()
}
