package util

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// AgentRunResult holds the result of a single agent-run execution.
type AgentRunResult struct {
	PreValidationPass     bool
	PostValidationPass    bool
	AgentExecuted         bool
	PreValidationError    string
	PostValidationError   string
	BeforeCount           int
	AfterCount            int
	BeforeNewCount        int // For translate: new (untranslated) entries before
	AfterNewCount         int // For translate: new (untranslated) entries after
	BeforeFuzzyCount      int // For translate: fuzzy entries before
	AfterFuzzyCount       int // For translate: fuzzy entries after
	SyntaxValidationPass  bool
	SyntaxValidationError string
	Score                 int // 0-100, calculated based on validations

	// Review-specific fields
	ReviewJSON       *ReviewJSONResult `json:"review_json,omitempty"`
	ReviewScore      int               `json:"review_score,omitempty"`
	ReviewJSONPath   string            `json:"review_json_path,omitempty"`
	ReviewedFilePath string            `json:"reviewed_file_path,omitempty"` // Final reviewed PO file path

	// Agent output (for saving logs in agent-test)
	AgentStdout []byte `json:"-"`
	AgentStderr []byte `json:"-"`

	// Agent diagnostics
	AgentError    error
	NumTurns      int // Number of turns in the conversation
	ExecutionTime time.Duration
}

// ReviewIssue represents a single issue in a review JSON result.
type ReviewIssue struct {
	MsgID               string   `json:"msgid"`                   // original msgid (singular)
	MsgStr              string   `json:"msgstr"`                  // original translation (singular)
	MsgIDPlural         string   `json:"msgid_plural,omitempty"`  // original msgid (plural)
	MsgStrPlural        []string `json:"msgstr_plural,omitempty"` // original translation (plural)
	Score               int      `json:"score"`                   // issue score (0-3)
	Description         string   `json:"description"`             // issue description
	SuggestMsgstr       string   `json:"suggest_msgstr"`          // corrected translation (singular)
	SuggestMsgstrPlural []string `json:"suggest_msgstr_plural"`   // corrected translation (plural)
}

// ReviewJSONResult represents the overall review JSON format produced by an agent.
type ReviewJSONResult struct {
	TotalEntries int           `json:"total_entries"`
	Issues       []ReviewIssue `json:"issues"`
}

// IssueCount returns the number of issues that count as problems (score < 3).
// Issues with score 3 are not counted as problems.
func (r *ReviewJSONResult) IssueCount() int {
	if r == nil {
		return 0
	}
	n := 0
	for _, issue := range r.Issues {
		if issue.Score < 3 {
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

// ReviewPathSetFromBase returns paths for the given base (e.g. "po/review").
// If base is empty, uses ReviewDefaultBase.
func ReviewPathSetFromBase(base string) ReviewPathSet {
	if base == "" {
		base = ReviewDefaultBase
	}
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
// The scoring model treats each entry as having a maximum of 3 points.
// For each reported issue, the score is reduced by (3 - issue.Score).
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

	totalPossible := review.TotalEntries * 3
	totalScore := totalPossible

	log.Debugf("calculating review score: total_entries=%d, total_possible=%d, issues_count=%d",
		review.TotalEntries, totalPossible, len(review.Issues))

	for i, issue := range review.Issues {
		if issue.Score < 0 || issue.Score > 3 {
			log.Debugf("calculate score failed: issue[%d].score=%d (must be 0-3)", i, issue.Score)
			return 0, fmt.Errorf("invalid issue score %d: must be between 0 and 3", issue.Score)
		}
		deduction := 3 - issue.Score
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
