package util

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// ReviewReport holds the result of reporting from a review JSON file.
// Issue scores: ReviewIssueScoreCritical, ReviewIssueScoreMajor, ReviewIssueScoreMinor. Perfect = ReviewIssueScorePerfect.
type ReviewReport struct {
	ReviewResult  *ReviewJSONResult
	Score         int
	CriticalCount int    // ReviewIssueScoreCritical
	MajorCount    int    // ReviewIssueScoreMajor
	MinorCount    int    // ReviewIssueScoreMinor
	ReportFile    string // Review JSON file path, if it exists
	AppliedFile   string // Output PO file path (after applying suggestions), if it exists
}

// PerfectCount returns the number of entries with no reported issue.
func (r *ReviewReport) PerfectCount() int {
	if r == nil || r.ReviewResult == nil {
		return 0
	}
	n := r.ReviewResult.TotalEntries - (r.CriticalCount + r.MinorCount + r.MajorCount)
	if n < 0 {
		return 0
	}
	return n
}

// PoUpdateResult holds entry counts for update-po/update-pot operations.
// Embedded in AgentRunResult; zero values when not from update-po/update-pot.
type PoUpdateResult struct {
	EntryCountBeforeUpdate int // PO/POT msgid count before agent update
	EntryCountAfterUpdate  int // PO/POT msgid count after agent update
}

// TranslateResult holds new/fuzzy entry counts for translate operations.
// Embedded in AgentRunResult; zero values when not from translate.
type TranslateResult struct {
	BeforeNewCount   int // New (untranslated) entries before
	AfterNewCount    int // New (untranslated) entries after
	BeforeFuzzyCount int // Fuzzy entries before
	AfterFuzzyCount  int // Fuzzy entries after
}

// AgentRunResult holds the result of a single agent-run execution.
type AgentRunResult struct {
	PreValidationPass     bool
	PostValidationPass    bool
	AgentExecuted         bool
	PreValidationError    string
	PostValidationError   string
	SyntaxValidationPass  bool
	SyntaxValidationError string
	Score                 int // 0-100, calculated based on validations

	// Update-po/update-pot: embedded when from update-po/update-pot
	PoUpdateResult
	// Translate: embedded when from translate
	TranslateResult
	// Review: embedded when from review; ReviewResult==nil when not from review
	ReviewReport

	// Agent output (for saving logs in agent-test)
	AgentStdout []byte `json:"-"`
	AgentStderr []byte `json:"-"`

	// Agent diagnostics
	AgentError    error
	NumTurns      int // Number of turns in the conversation
	ExecutionTime time.Duration
}

// ReviewIssue score constants (0-3). Lower = more severe.
// Used for JSON "score" field in ReviewIssue.
const (
	ReviewIssueScoreCritical = 0 // critical issue (most severe)
	ReviewIssueScoreMajor    = 1 // major issue (serious)
	ReviewIssueScoreMinor    = 2 // minor issue (small)
	ReviewIssueScorePerfect  = 3 // no issue (perfect)
	ReviewIssueScoreMax      = 3 // maximum valid score

	// ReviewIssuePointsPerEntry is max points per entry for score calculation.
	ReviewIssuePointsPerEntry = ReviewIssueScoreMax
)

// ReviewIssue represents a single issue in a review JSON result.
type ReviewIssue struct {
	MsgID               string   `json:"msgid"`                   // original msgid (singular)
	MsgStr              string   `json:"msgstr"`                  // original translation (singular)
	MsgIDPlural         string   `json:"msgid_plural,omitempty"`  // original msgid (plural)
	MsgStrPlural        []string `json:"msgstr_plural,omitempty"` // original translation (plural)
	Score               int      `json:"score"`                   // issue score (ReviewIssueScoreCritical..ReviewIssueScorePerfect)
	Description         string   `json:"description"`             // issue description
	SuggestMsgstr       string   `json:"suggest_msgstr"`          // corrected translation (singular)
	SuggestMsgstrPlural []string `json:"suggest_msgstr_plural"`   // corrected translation (plural)
}

// ReviewJSONResult represents the overall review JSON format produced by an agent.
type ReviewJSONResult struct {
	TotalEntries int           `json:"total_entries"`
	Issues       []ReviewIssue `json:"issues"`
}

// IssueCount returns the number of issues that count as problems (score < ReviewIssueScorePerfect).
// Issues with score ReviewIssueScorePerfect are not counted as problems.
func (r *ReviewJSONResult) IssueCount() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, issue := range r.Issues {
		if issue.Score < ReviewIssueScorePerfect {
			n++
		}
	}
	return n
}

var (
	ReviewDefaultOutputFile = filepath.Join(PoDir, "review.json")
	// ReviewDefaultBase is the default base for Task 4 review paths (po/review).
	ReviewDefaultBase = filepath.Join(PoDir, "review")
)

// ReviewPathSet holds paths for Task 4 review workflow (AGENTS.md).
// Naming: review-input.po, review-pending.po, review-result.json, review-output.po,
// review-todo.json, review-done.json, review-batch.txt,
// review-result-<N>.json.
type ReviewPathSet struct {
	BaseDir    string // directory containing all review files (e.g. "po")
	BaseName   string // base name prefix (e.g. "review")
	InputPO    string // po/review-input.po (original extracted PO file, immutable)
	PendingPO  string // po/review-pending.po (remaining entries to review)
	ResultJSON string // po/review-result.json
	OutputPO   string // po/review-output.po
}

// GetReviewPathSet returns paths using ReviewDefaultBase (po/review).
func GetReviewPathSet() ReviewPathSet {
	base := ReviewDefaultBase
	dir := filepath.Dir(base)
	name := filepath.Base(base)
	if name == "" || name == "." {
		name = "review"
		dir = PoDir
	}
	return ReviewPathSet{
		BaseDir:    dir,
		BaseName:   name,
		InputPO:    filepath.Join(dir, name+"-input.po"),
		PendingPO:  filepath.Join(dir, name+"-pending.po"),
		ResultJSON: filepath.Join(dir, name+"-result.json"),
		OutputPO:   filepath.Join(dir, name+"-output.po"),
	}
}

// ReviewTodoJSONPath returns po/review-todo.json (AGENTS.md step 4).
func (p ReviewPathSet) ReviewTodoJSONPath() string {
	return filepath.Join(p.BaseDir, p.BaseName+"-todo.json")
}

// ReviewDoneJSONPath returns po/review-done.json (AGENTS.md step 5).
func (p ReviewPathSet) ReviewDoneJSONPath() string {
	return filepath.Join(p.BaseDir, p.BaseName+"-done.json")
}

// ReviewBatchTxtPath returns po/review-batch.txt (AGENTS.md step 4).
func (p ReviewPathSet) ReviewBatchTxtPath() string {
	return filepath.Join(p.BaseDir, p.BaseName+"-batch.txt")
}

// ReviewResultJSONPath returns po/review-result-<N>.json.
func (p ReviewPathSet) ReviewResultJSONPath(n int) string {
	return filepath.Join(p.BaseDir, p.BaseName+"-result-"+strconv.Itoa(n)+".json")
}

// CalculateReviewScore calculates a 0-100 score from a ReviewJSONResult.
// The scoring model treats each entry as having a maximum of ReviewIssueScoreMax points.
// For each reported issue, the score is reduced by (ReviewIssueScoreMax - issue.Score).
// The final score is normalized to 0-100.
func CalculateReviewScore(review *ReviewJSONResult) (int, error) {
	// If total_entries is 0, we can't calculate a meaningful score
	// This might happen if the calculation hasn't been performed yet
	if review.TotalEntries <= 0 {
		// If there are no entries, and no issues, we can consider it as perfect
		if len(review.Issues) == 0 {
			log.Debugf("no entries and no issues, returning perfect score of 100")
			return 100, nil
		}
		// If there are issues but no entries, this is an inconsistent state
		log.Debugf("calculate score failed: total_entries=%d but has %d issues", review.TotalEntries, len(review.Issues))
		return 0, fmt.Errorf("invalid review result: total_entries must be greater than 0, got %d", review.TotalEntries)
	}

	totalPossible := review.TotalEntries * ReviewIssuePointsPerEntry
	totalScore := totalPossible

	log.Debugf("calculating review score: total_entries=%d, total_possible=%d, issues_count=%d",
		review.TotalEntries, totalPossible, len(review.Issues))

	for i, issue := range review.Issues {
		if issue.Score < ReviewIssueScoreCritical || issue.Score > ReviewIssueScoreMax {
			log.Debugf("calculate score failed: issue[%d].score=%d (must be %d-%d)", i, issue.Score, ReviewIssueScoreCritical, ReviewIssueScoreMax)
			return 0, fmt.Errorf("invalid issue score %d: must be between %d and %d", issue.Score, ReviewIssueScoreCritical, ReviewIssueScoreMax)
		}
		deduction := ReviewIssueScoreMax - issue.Score
		totalScore -= deduction
		log.Debugf("issue[%d]: score=%d, deduction=%d, remaining=%d", i, issue.Score, deduction, totalScore)
	}

	if totalScore < 0 {
		log.Debugf("total score is negative (%d), clamping to 0", totalScore)
		totalScore = 0
	}

	scorePercent := int(math.Round(float64(totalScore) * 100.0 / float64(totalPossible)))
	if scorePercent < 0 {
		scorePercent = 0
	} else if scorePercent > 100 {
		scorePercent = 100
	}

	log.Debugf("review score calculated: %d/100 (total_score=%d, total_possible=%d)",
		scorePercent, totalScore, totalPossible)

	return scorePercent, nil
}
