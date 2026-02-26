// Package util provides business logic for agent-run command.
package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
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
	NumTurns      int           // Number of turns in the conversation
	ExecutionTime time.Duration // Execution time for this run
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

// ExtractJSONFromOutput extracts a JSON object from agent output.
// It searches for JSON object boundaries ({ and }) and handles cases where
// output contains other text before/after JSON.
// Returns the JSON bytes or an error if not found.
func ExtractJSONFromOutput(output []byte) ([]byte, error) {
	if len(output) == 0 {
		log.Debugf("agent output is empty, cannot extract JSON")
		return nil, fmt.Errorf("empty output, no JSON found")
	}

	log.Debugf("extracting JSON from agent output (length: %d bytes)", len(output))

	// Find the first '{' character
	startIdx := -1
	for i, b := range output {
		if b == '{' {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		log.Debugf("no opening brace found in agent output")
		return nil, fmt.Errorf("no JSON object found in output (missing opening brace)")
	}

	log.Debugf("found JSON start at position %d", startIdx)

	// Find the matching closing '}' by counting braces
	braceCount := 0
	endIdx := -1
	for i := startIdx; i < len(output); i++ {
		if output[i] == '{' {
			braceCount++
		} else if output[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i
				break
			}
		}
	}

	if endIdx == -1 {
		log.Debugf("no matching closing brace found (unclosed JSON object)")
		return nil, fmt.Errorf("no complete JSON object found in output (missing closing brace)")
	}

	log.Debugf("found JSON end at position %d (extracted %d bytes)", endIdx, endIdx-startIdx+1)

	// Extract JSON bytes
	jsonBytes := output[startIdx : endIdx+1]
	return jsonBytes, nil
}

// ParseReviewJSON parses JSON output from agent and validates the structure.
// It validates that the JSON matches ReviewJSONResult structure and that
// all score values are in the valid range (0-3).
// Returns parsed result or error.
func ParseReviewJSON(jsonData []byte) (*ReviewJSONResult, error) {
	if len(jsonData) == 0 {
		log.Debugf("JSON data is empty")
		return nil, fmt.Errorf("empty JSON data")
	}

	log.Debugf("parsing JSON data (length: %d bytes)", len(jsonData))

	var review ReviewJSONResult
	if err := json.Unmarshal(jsonData, &review); err != nil {
		log.Debugf("JSON unmarshal failed: %v", err)
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	log.Debugf("JSON parsed successfully: total_entries=%d, issues_count=%d", review.TotalEntries, len(review.Issues))

	// Note: We allow total_entries to be 0 here because it will be recalculated later
	// from the actual file to ensure accuracy.
	// The validation of total_entries > 0 will happen after recalculation if needed.

	// Validate issues array
	if review.Issues == nil {
		// Issues can be an empty array, but not nil
		log.Debugf("issues array is nil, initializing as empty array")
		review.Issues = []ReviewIssue{}
	}

	// Validate each issue
	for i, issue := range review.Issues {
		// Validate score range
		if issue.Score < 0 || issue.Score > 3 {
			log.Debugf("validation failed: issue[%d].score=%d (must be 0-3)", i, issue.Score)
			return nil, fmt.Errorf("invalid issue score %d at index %d: must be between 0 and 3", issue.Score, i)
		}

		// Validate required fields are not empty (msgid and msgstr can be empty, but should be present)
		// Description and suggestion should not be empty for issues
		if issue.Description == "" {
			log.Debugf("validation failed: issue[%d].description is empty", i)
			return nil, fmt.Errorf("invalid issue at index %d: description is required", i)
		}

		log.Debugf("issue[%d]: msgid=%q, score=%d, description=%q", i, issue.MsgID, issue.Score, issue.Description)
	}

	log.Debugf("JSON validation passed: %d total entries, %d issues", review.TotalEntries, len(review.Issues))
	return &review, nil
}

// getRelativePath converts an absolute path to a path relative to the repository root.
// If conversion fails, returns the original absolute path as fallback.
func getRelativePath(absPath string) string {
	if absPath == "" {
		return ""
	}
	relPath, err := filepath.Rel(repository.WorkDir(), absPath)
	if err != nil {
		return absPath // fallback to absolute path
	}
	return relPath
}

// AggregateReviewJSON merges multiple review JSON results. For each msgid that
// appears in multiple runs, the issue with the lowest score (most severe) is kept.
// total_entries is taken from the first non-empty review. Returns nil if no valid input.
// If warnDup is true, logs an error when duplicate msgid issues are found.
func AggregateReviewJSON(reviews []*ReviewJSONResult, warnDup bool) *ReviewJSONResult {
	if len(reviews) == 0 {
		return nil
	}
	// Map msgid -> best issue (lowest score = most severe)
	byMsgID := make(map[string]*ReviewIssue)
	var totalEntries int
	for _, r := range reviews {
		if r == nil {
			continue
		}
		if r.TotalEntries > 0 && totalEntries == 0 {
			totalEntries = r.TotalEntries
		}
		for i := range r.Issues {
			issue := &r.Issues[i]
			key := issue.MsgID
			existing, ok := byMsgID[key]
			if ok && warnDup {
				log.Errorf("duplicate msgid in review issues: %q (runs on overlaped batches)", key)
			}
			if !ok || issue.Score < existing.Score {
				byMsgID[key] = issue
			}
		}
	}
	issues := make([]ReviewIssue, 0, len(byMsgID))
	for _, issue := range byMsgID {
		issues = append(issues, *issue)
	}
	return &ReviewJSONResult{TotalEntries: totalEntries, Issues: issues}
}

// saveReviewJSON saves review JSON result to the given file path.
func saveReviewJSON(review *ReviewJSONResult, jsonFile string) error {
	if review == nil {
		return fmt.Errorf("review result is nil")
	}

	// Marshal JSON with indentation for readability
	jsonData, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Add newline at end of file
	jsonData = append(jsonData, '\n')

	// Write JSON to file
	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", jsonFile, err)
	}

	return nil
}

// ValidatePotEntryCount validates the entry count in a POT file.
// If expectedCount is nil or 0, validation is disabled and the function returns nil.
// Otherwise, it counts entries using CountPotEntries() and compares with expectedCount.
// Returns an error if counts don't match, nil if they match or validation is disabled.
// The stage parameter is used for error messages ("before update" or "after update").
// For "before update" stage, if the file doesn't exist, the entry count is treated as 0.
func ValidatePotEntryCount(potFile string, expectedCount *int, stage string) error {
	// If expectedCount is nil or 0, validation is disabled
	if expectedCount == nil || *expectedCount == 0 {
		return nil
	}

	// Check if file exists
	fileExists := Exist(potFile)
	var actualCount int
	var err error

	if !fileExists {
		// For "before update" stage, treat missing file as 0 entries
		if stage == "before update" {
			actualCount = 0
			log.Debugf("file %s does not exist, treating entry count as 0 for %s validation", potFile, stage)
		} else {
			// For "after update" stage, file should exist
			return fmt.Errorf("file does not exist %s: %s\nHint: The agent should have created the file", stage, potFile)
		}
	} else {
		// Count entries in POT file
		actualCount, err = CountPotEntries(potFile)
		if err != nil {
			return fmt.Errorf("failed to count entries %s in %s: %w", stage, potFile, err)
		}
	}

	// Compare with expected count
	if actualCount != *expectedCount {
		return fmt.Errorf("entry count %s: expected %d, got %d (file: %s)", stage, *expectedCount, actualCount, potFile)
	}

	log.Debugf("entry count %s validation passed: %d entries", stage, actualCount)
	return nil
}

// ValidatePoEntryCount validates the entry count in a PO file.
// If expectedCount is nil or 0, validation is disabled and the function returns nil.
// Otherwise, it counts entries using CountPoEntries() and compares with expectedCount.
// Returns an error if counts don't match, nil if they match or validation is disabled.
// The stage parameter is used for error messages ("before update" or "after update").
// For "before update" stage, if the file doesn't exist, the entry count is treated as 0.
func ValidatePoEntryCount(poFile string, expectedCount *int, stage string) error {
	// If expectedCount is nil or 0, validation is disabled
	if expectedCount == nil || *expectedCount == 0 {
		return nil
	}

	// Check if file exists
	fileExists := Exist(poFile)
	var actualCount int
	var err error

	if !fileExists {
		// For "before update" stage, treat missing file as 0 entries
		if stage == "before update" {
			actualCount = 0
			log.Debugf("file %s does not exist, treating entry count as 0 for %s validation", poFile, stage)
		} else {
			// For "after update" stage, file should exist
			return fmt.Errorf("file does not exist %s: %s\nHint: The agent should have created the file", stage, poFile)
		}
	} else {
		// Count entries in PO file
		actualCount, err = CountPoEntries(poFile)
		if err != nil {
			return fmt.Errorf("failed to count entries %s in %s: %w", stage, poFile, err)
		}
	}

	// Compare with expected count
	if actualCount != *expectedCount {
		return fmt.Errorf("entry count %s: expected %d, got %d (file: %s)", stage, *expectedCount, actualCount, poFile)
	}

	log.Debugf("entry count %s validation passed: %d entries", stage, actualCount)
	return nil
}

// ValidatePoFile validates POT/PO file syntax.
// For .pot files, it uses msgcat --use-first to validate (since POT files have placeholders in headers).
// For .po files, it uses msgfmt to validate.
// Returns an error if the file is invalid, nil if valid.
// If the file path is absolute, it doesn't require repository context.
// If the file path is relative, it uses repository.WorkDir() as the working directory.
func ValidatePoFile(potFile string) error {
	return validatePoFileInternal(potFile, false)
}

// ValidatePoFileFormat validates POT/PO file format syntax only (using --check-format for PO files).
// This is a more lenient check that doesn't require complete headers.
// For .pot files, it uses msgcat --use-first to validate.
// For .po files, it uses msgfmt --check-format to validate (only checks format, not completeness).
// Returns an error if the file format is invalid, nil if valid.
// If the file path is absolute, it doesn't require repository context.
// If the file path is relative, it uses repository.WorkDir() as the working directory.
func ValidatePoFileFormat(potFile string) error {
	return validatePoFileInternal(potFile, true)
}

// validatePoFileInternal is the internal implementation for PO/POT file validation.
// checkFormatOnly: if true, uses --check-format for PO files (more lenient, only checks format).
//
//	if false, uses --check for PO files (stricter, checks format and completeness).
func validatePoFileInternal(potFile string, checkFormatOnly bool) error {
	if !Exist(potFile) {
		return fmt.Errorf("POT file does not exist: %s\nHint: Ensure the file exists or run the agent to create it", potFile)
	}

	// Determine file extension to choose the appropriate validation tool
	ext := filepath.Ext(potFile)
	var cmd *exec.Cmd
	var toolName string

	if ext == ".pot" {
		// For POT files, use msgcat --use-first since POT files have placeholders in headers
		toolName = "msgcat"
		log.Debugf("running msgcat --use-first on %s", potFile)
		cmd = exec.Command("msgcat",
			"--use-first",
			potFile,
			"-o",
			os.DevNull)
	} else {
		// For PO files, use msgfmt
		toolName = "msgfmt"
		if checkFormatOnly {
			log.Debugf("running msgfmt --check-format on %s", potFile)
			cmd = exec.Command("msgfmt",
				"-o",
				os.DevNull,
				"--check-format",
				potFile)
		} else {
			log.Debugf("running msgfmt --check on %s", potFile)
			cmd = exec.Command("msgfmt",
				"-o",
				os.DevNull,
				"--check",
				potFile)
		}
	}

	// Only set working directory if file path is relative
	// For absolute paths, we don't need repository context
	if filepath.IsAbs(potFile) {
		// For absolute paths, use the directory containing the file as working directory
		cmd.Dir = filepath.Dir(potFile)
	} else {
		// For relative paths, use repository working directory
		cmd.Dir = repository.WorkDir()
	}

	// Capture stderr for error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe for %s: %w", toolName, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s command: %w\nHint: Ensure gettext tools (%s) are installed", toolName, err, toolName)
	}

	// Read stderr output
	var stderrOutput strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			stderrOutput.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		errorMsg := stderrOutput.String()
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		return fmt.Errorf("file syntax validation failed: %s\nHint: Check the file syntax and fix any errors reported by %s", errorMsg, toolName)
	}

	log.Debugf("file validation passed: %s", potFile)
	return nil
}

// GetPoFileAbsPath determines the absolute path of a PO file.
// If poFile is empty, it uses cfg.DefaultLangCode to construct the path.
// If poFile is provided but not absolute, it's treated as relative to the repository root.
// Returns the absolute path and an error if default_lang_code is not configured when needed.
func GetPoFileAbsPath(cfg *config.AgentConfig, poFile string) (string, error) {
	workDir := repository.WorkDir()
	if poFile == "" {
		lang := cfg.DefaultLangCode
		if lang == "" {
			log.Errorf("default_lang_code is not configured in agent configuration")
			return "", fmt.Errorf("default_lang_code is not configured\nHint: Provide po/XX.po on the command line or set default_lang_code in git-po-helper.yaml")
		}
		poFile = filepath.Join(workDir, PoDir, fmt.Sprintf("%s.po", lang))
	} else if !filepath.IsAbs(poFile) {
		// Treat poFile as relative to repository root
		poFile = filepath.Join(workDir, poFile)
	}
	return poFile, nil
}

// GetPoFileRelPath determines the relative path of a PO file in "po/XX.po" format.
// If poFile is empty, it uses cfg.DefaultLangCode to construct the path.
// If poFile is an absolute path, it converts it to a relative path.
// If poFile is already a relative path, it normalizes it to "po/XX.po" format.
// Returns the relative path and an error if default_lang_code is not configured when needed.
func GetPoFileRelPath(cfg *config.AgentConfig, poFile string) (string, error) {
	workDir := repository.WorkDir()
	var absPath string
	var err error

	// First get the absolute path
	absPath, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return "", err
	}

	// Convert absolute path to relative path
	relPath, err := filepath.Rel(workDir, absPath)
	if err != nil {
		log.Errorf("failed to convert absolute path to relative path: %v", err)
		return "", fmt.Errorf("failed to convert path to relative: %w", err)
	}

	// Normalize to use forward slashes (for consistency with "po/XX.po" format)
	relPath = filepath.ToSlash(relPath)

	return relPath, nil
}

// parseStreamByKind parses agent stream output based on kind, returns stdout and unified result.
func parseStreamByKind(kind string, reader io.Reader) (stdout []byte, streamResult AgentStreamResult, err error) {
	switch kind {
	case config.AgentKindCodex:
		parsed, res, e := ParseCodexJSONLRealtime(reader)
		if e != nil {
			log.Warnf("failed to parse codex JSONL: %v", e)
		}
		return parsed, res, e
	case config.AgentKindOpencode:
		parsed, res, e := ParseOpenCodeJSONLRealtime(reader)
		if e != nil {
			log.Warnf("failed to parse opencode JSONL: %v", e)
		}
		return parsed, res, e
	case config.AgentKindGemini, config.AgentKindQwen:
		parsed, res, e := ParseGeminiJSONLRealtime(reader)
		if e != nil {
			log.Warnf("failed to parse gemini JSONL: %v", e)
		}
		return parsed, res, e
	default:
		parsed, res, e := ParseClaudeStreamJSONRealtime(reader)
		if e != nil {
			log.Warnf("failed to parse stream JSON: %v", e)
		}
		return parsed, res, e
	}
}

// applyAgentDiagnostics prints diagnostics and extracts NumTurns from streamResult.
func applyAgentDiagnostics(result *AgentRunResult, streamResult AgentStreamResult) {
	if streamResult == nil {
		return
	}
	PrintAgentDiagnostics(streamResult)
	if n := streamResult.GetNumTurns(); n > 0 {
		result.NumTurns = n
	}
}
