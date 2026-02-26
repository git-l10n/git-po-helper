package util

import (
	"fmt"
	"math"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// AgentRunResult holds the result of a single agent-run execution.
type AgentRunResult struct {
	PreValidationPass     bool
	PostValidationPass    bool
	AgentExecuted         bool
	AgentSuccess          bool
	PreValidationError    string
	PostValidationError   string
	AgentError            string
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
	NumTurns      int // Number of turns in the conversation
	ExecutionTime time.Duration
}

// ReviewIssue represents a single issue in a review JSON result.
type ReviewIssue struct {
	MsgID       string `json:"msgid"`
	MsgStr      string `json:"msgstr"`
	Score       int    `json:"score"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// ReviewJSONResult represents the overall review JSON format produced by an agent.
type ReviewJSONResult struct {
	TotalEntries int           `json:"total_entries"`
	Issues       []ReviewIssue `json:"issues"`
}

var (
	ReviewDefaultOutputFile = filepath.Join(PoDir, "review.json")
	ReviewDefaultBatchFile  = filepath.Join(PoDir, "review-batch.po")
)

// ReviewOutputPaths returns (poFile, jsonFile) for the given output base path.
// If base is empty, use ReviewDefaultOutputFile.
// Uses DeriveReviewPaths to ensure consistent json/po derivation.
func ReviewOutputPaths(base string) (poFile, jsonFile string) {
	if base == "" {
		base = ReviewDefaultOutputFile
	}
	jsonFile, poFile = DeriveReviewPaths(base)
	return poFile, jsonFile
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
