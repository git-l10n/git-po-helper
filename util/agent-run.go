// Package util provides business logic for agent-run command.
package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

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
func AggregateReviewJSON(reviews []*ReviewJSONResult) *ReviewJSONResult {
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

// RunAgentUpdatePot executes a single agent-run update-pot operation.
// It performs pre-validation, executes the agent command, performs post-validation,
// and validates POT file syntax. Returns a result structure with detailed information.
// The agentTest parameter controls whether AgentTest configuration should be used.
// When agentTest is false (for agent-run), AgentTest configuration is ignored.
func RunAgentUpdatePot(cfg *config.AgentConfig, agentName string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Get POT file path
	potFile := GetPotFilePath()
	log.Debugf("POT file path: %s", potFile)

	// Pre-validation: Check entry count before update (only for agent-test)
	if agentTest && cfg.AgentTest.PotEntriesBeforeUpdate != nil && *cfg.AgentTest.PotEntriesBeforeUpdate != 0 {
		log.Infof("performing pre-validation: checking entry count before update (expected: %d)", *cfg.AgentTest.PotEntriesBeforeUpdate)

		// Get before count for result
		if !Exist(potFile) {
			result.BeforeCount = 0
		} else {
			result.BeforeCount, _ = CountPotEntries(potFile)
		}

		if err := ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesBeforeUpdate, "before update"); err != nil {
			log.Errorf("pre-validation failed: %v", err)
			result.PreValidationError = err.Error()
			return result, fmt.Errorf("pre-validation failed: %w\nHint: Ensure po/git.pot exists and has the expected number of entries", err)
		}
		result.PreValidationPass = true
		log.Infof("pre-validation passed")
	} else {
		// No pre-validation configured, count entries for display purposes
		if !Exist(potFile) {
			result.BeforeCount = 0
		} else {
			result.BeforeCount, _ = CountPotEntries(potFile)
		}
		result.PreValidationPass = true // Consider it passed if not configured
	}

	// Get prompt from configuration
	prompt, err := GetPrompt(cfg, "update-pot")
	if err != nil {
		return result, err
	}

	// Build agent command with placeholders replaced
	agentCmd := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt})

	// Determine output format
	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	// Normalize output format (convert underscores to hyphens)
	outputFormat = normalizeOutputFormat(outputFormat)

	// Execute agent command
	workDir := repository.WorkDir()
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json", truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	var stdout []byte
	var stderr []byte
	var streamResult AgentStreamResult

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	// Use streaming execution for json format (treated as stream-json)
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd, workDir)
		if err != nil {
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		defer stdoutReader.Close()

		stdout, streamResult, _ = parseStreamByKind(kind, stdoutReader)

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd, workDir)
		if err != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", err)
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, err := ParseClaudeAgentOutput(stdout, outputFormat)
			if err != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", err)
				parsedStdout = stdout
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

	applyAgentDiagnostics(result, streamResult)

	// Log output if verbose
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	// Post-validation: Check entry count after update (only for agent-test)
	if agentTest && cfg.AgentTest.PotEntriesAfterUpdate != nil && *cfg.AgentTest.PotEntriesAfterUpdate != 0 {
		log.Infof("performing post-validation: checking entry count after update (expected: %d)", *cfg.AgentTest.PotEntriesAfterUpdate)

		// Get after count for result
		if Exist(potFile) {
			result.AfterCount, _ = CountPotEntries(potFile)
		}

		if err := ValidatePotEntryCount(potFile, cfg.AgentTest.PotEntriesAfterUpdate, "after update"); err != nil {
			log.Errorf("post-validation failed: %v", err)
			result.PostValidationError = err.Error()
			result.Score = 0
			return result, fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the POT file correctly", err)
		}
		result.PostValidationPass = true
		result.Score = 100
		log.Infof("post-validation passed")
	} else {
		// No post-validation configured, score based on agent exit code
		if Exist(potFile) {
			result.AfterCount, _ = CountPotEntries(potFile)
		}
		if result.AgentSuccess {
			result.Score = 100
			result.PostValidationPass = true // Consider it passed if agent succeeded
		} else {
			result.Score = 0
		}
	}

	// Validate POT file syntax (only if agent succeeded)
	if result.AgentSuccess {
		log.Infof("validating file syntax: %s", potFile)
		if err := ValidatePoFile(potFile); err != nil {
			log.Errorf("file syntax validation failed: %v", err)
			result.SyntaxValidationError = err.Error()
			// Don't fail the run for syntax errors in agent-run, but log it
			// In agent-test, this might affect the score
		} else {
			result.SyntaxValidationPass = true
			log.Infof("file syntax validation passed")
		}
	}

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// RunAgentUpdatePo executes a single agent-run update-po operation.
// It performs pre-validation, executes the agent command, performs post-validation,
// and validates PO file syntax. Returns a result structure with detailed information.
// The agentTest parameter controls whether AgentTest configuration should be used.
// When agentTest is false (for agent-run), AgentTest configuration is ignored.
func RunAgentUpdatePo(cfg *config.AgentConfig, agentName, poFile string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Determine PO file path
	poFile, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}

	log.Debugf("PO file path: %s", poFile)

	// Pre-validation: Check entry count before update (only for agent-test)
	if agentTest && cfg.AgentTest.PoEntriesBeforeUpdate != nil && *cfg.AgentTest.PoEntriesBeforeUpdate != 0 {
		log.Infof("performing pre-validation: checking PO entry count before update (expected: %d)", *cfg.AgentTest.PoEntriesBeforeUpdate)

		// Get before count for result
		if !Exist(poFile) {
			result.BeforeCount = 0
		} else {
			result.BeforeCount, _ = CountPoEntries(poFile)
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesBeforeUpdate, "before update"); err != nil {
			log.Errorf("pre-validation failed: %v", err)
			result.PreValidationError = err.Error()
			return result, fmt.Errorf("pre-validation failed: %w\nHint: Ensure %s exists and has the expected number of entries", err, poFile)
		}
		result.PreValidationPass = true
		log.Infof("pre-validation passed")
	} else {
		// No pre-validation configured, count entries for display purposes
		if !Exist(poFile) {
			result.BeforeCount = 0
		} else {
			result.BeforeCount, _ = CountPoEntries(poFile)
		}
		result.PreValidationPass = true // Consider it passed if not configured
	}

	// Get prompt for update-po from configuration
	prompt, err := GetPrompt(cfg, "update-po")
	if err != nil {
		return result, err
	}

	// Build agent command with placeholders replaced
	workDir := repository.WorkDir()
	sourcePath := poFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		sourcePath = filepath.ToSlash(rel)
	}
	agentCmd := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": sourcePath})

	// Determine output format
	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	// Normalize output format (convert underscores to hyphens)
	outputFormat = normalizeOutputFormat(outputFormat)

	// Execute agent command
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json",
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	var stdout []byte
	var stderr []byte
	var streamResult AgentStreamResult

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	// Use streaming execution for json format (treated as stream-json)
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd, workDir)
		if err != nil {
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		defer stdoutReader.Close()

		stdout, streamResult, _ = parseStreamByKind(kind, stdoutReader)

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd, workDir)
		if err != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", err)
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, err := ParseClaudeAgentOutput(stdout, outputFormat)
			if err != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", err)
				parsedStdout = stdout
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

	applyAgentDiagnostics(result, streamResult)

	// Log output if verbose
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	// Post-validation: Check entry count after update (only for agent-test)
	if agentTest && cfg.AgentTest.PoEntriesAfterUpdate != nil && *cfg.AgentTest.PoEntriesAfterUpdate != 0 {
		log.Infof("performing post-validation: checking PO entry count after update (expected: %d)", *cfg.AgentTest.PoEntriesAfterUpdate)

		// Get after count for result
		if Exist(poFile) {
			result.AfterCount, _ = CountPoEntries(poFile)
		}

		if err := ValidatePoEntryCount(poFile, cfg.AgentTest.PoEntriesAfterUpdate, "after update"); err != nil {
			log.Errorf("post-validation failed: %v", err)
			result.PostValidationError = err.Error()
			result.Score = 0
			return result, fmt.Errorf("post-validation failed: %w\nHint: The agent may not have updated the PO file correctly", err)
		}
		result.PostValidationPass = true
		result.Score = 100
		log.Infof("post-validation passed")
	} else {
		// No post-validation configured, score based on agent exit code
		if Exist(poFile) {
			result.AfterCount, _ = CountPoEntries(poFile)
		}
		if result.AgentSuccess {
			result.Score = 100
			result.PostValidationPass = true // Consider it passed if agent succeeded
		} else {
			result.Score = 0
		}
	}

	// Validate PO file syntax (only if agent succeeded)
	if result.AgentSuccess {
		log.Infof("validating file syntax: %s", poFile)
		if err := ValidatePoFile(poFile); err != nil {
			log.Errorf("file syntax validation failed: %v", err)
			result.SyntaxValidationError = err.Error()
			// Don't fail the run for syntax errors in agent-run, but log it
			// In agent-test, this might affect the score
		} else {
			result.SyntaxValidationPass = true
			log.Infof("file syntax validation passed")
		}
	}

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// CmdAgentRunUpdatePot implements the agent-run update-pot command logic.
// It loads configuration and calls RunAgentUpdatePot, then handles errors appropriately.
func CmdAgentRunUpdatePot(agentName string) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	result, err := RunAgentUpdatePot(cfg, agentName, false)
	if err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
	if !result.PreValidationPass {
		return fmt.Errorf("pre-validation failed: %s", result.PreValidationError)
	}
	if !result.AgentSuccess {
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}
	if !result.PostValidationPass {
		return fmt.Errorf("post-validation failed: %s", result.PostValidationError)
	}
	if result.SyntaxValidationError != "" {
		ext := filepath.Ext(GetPotFilePath())
		if ext == ".pot" {
			return fmt.Errorf("file validation failed: %s\nHint: Check the POT file syntax using 'msgcat --use-first <file> -o /dev/null'", result.SyntaxValidationError)
		}
		return fmt.Errorf("file validation failed: %s\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run update-pot completed successfully")
	return nil
}

// CmdAgentRunUpdatePo implements the agent-run update-po command logic.
// It loads configuration and calls RunAgentUpdatePo, then handles errors appropriately.
func CmdAgentRunUpdatePo(agentName, poFile string) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	result, err := RunAgentUpdatePo(cfg, agentName, poFile, false)
	if err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
	if !result.PreValidationPass {
		return fmt.Errorf("pre-validation failed: %s", result.PreValidationError)
	}
	if !result.AgentSuccess {
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}
	if !result.PostValidationPass {
		return fmt.Errorf("post-validation failed: %s", result.PostValidationError)
	}
	if result.SyntaxValidationError != "" {
		return fmt.Errorf("file validation failed: %s\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run update-po completed successfully")
	return nil
}

// CmdAgentRunShowConfig displays the current agent configuration in YAML format.
func CmdAgentRunShowConfig() error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w", err)
	}

	// Marshal configuration to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		log.Errorf("failed to marshal configuration to YAML: %v", err)
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Display the configuration
	fmt.Println("# Agent Configuration")
	fmt.Println("# This is the merged configuration from:")
	fmt.Println("# - User home directory: ~/.git-po-helper.yaml (lower priority)")
	fmt.Println("# - Repository root: <repo-root>/git-po-helper.yaml (higher priority)")
	fmt.Println()
	os.Stdout.Write(yamlData)

	return nil
}

// CmdAgentRunParseLog parses an agent JSONL log file and displays formatted output.
// Auto-detects format: Claude (claude_code_version) vs Qwen/Gemini (qwen_code_version or Gemini-style).
// Each line in the file should be a JSON object. Supports system, assistant (with text,
// thinking, tool_use content types), user (tool_result), and result messages.
func CmdAgentRunParseLog(logFile string) error {
	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	firstLine, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read log file: %w", err)
	}
	parseReader := io.MultiReader(strings.NewReader(firstLine), reader)

	if strings.Contains(firstLine, "claude_code_version") {
		_, _, err = ParseClaudeStreamJSONRealtime(parseReader)
	} else if strings.Contains(firstLine, `"type":"step_start"`) || strings.Contains(firstLine, `"type": "step_start"`) {
		// OpenCode format
		_, _, err = ParseOpenCodeJSONLRealtime(parseReader)
	} else if strings.Contains(firstLine, "thread.started") {
		// Codex format
		_, _, err = ParseCodexJSONLRealtime(parseReader)
	} else {
		// Qwen/Gemini format (qwen_code_version or Gemini-style system init)
		_, _, err = ParseGeminiJSONLRealtime(parseReader)
	}
	if err != nil {
		return fmt.Errorf("failed to parse log file: %w", err)
	}
	return nil
}

// RunAgentTranslate executes a single agent-run translate operation.
// It performs pre-validation (count new/fuzzy entries), executes the agent command,
// performs post-validation (verify new=0 and fuzzy=0), and validates PO file syntax.
// Returns a result structure with detailed information.
// The agentTest parameter is provided for consistency, though this method
// does not use AgentTest configuration.
func RunAgentTranslate(cfg *config.AgentConfig, agentName, poFile string, agentTest bool) (*AgentRunResult, error) {
	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Determine PO file path
	poFile, err = GetPoFileAbsPath(cfg, poFile)
	if err != nil {
		return result, err
	}

	log.Debugf("PO file path: %s", poFile)

	// Check if PO file exists
	if !Exist(poFile) {
		log.Errorf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running translate", poFile)
	}

	// Pre-validation: Count new and fuzzy entries before translation
	log.Infof("performing pre-validation: counting new and fuzzy entries")

	// Count new entries
	newCountBefore, err := CountNewEntries(poFile)
	if err != nil {
		log.Errorf("failed to count new entries: %v", err)
		return result, fmt.Errorf("failed to count new entries: %w", err)
	}
	result.BeforeNewCount = newCountBefore
	log.Infof("new (untranslated) entries before translation: %d", newCountBefore)

	// Count fuzzy entries
	fuzzyCountBefore, err := CountFuzzyEntries(poFile)
	if err != nil {
		log.Errorf("failed to count fuzzy entries: %v", err)
		return result, fmt.Errorf("failed to count fuzzy entries: %w", err)
	}
	result.BeforeFuzzyCount = fuzzyCountBefore
	log.Infof("fuzzy entries before translation: %d", fuzzyCountBefore)

	// Check if there's anything to translate
	if newCountBefore == 0 && fuzzyCountBefore == 0 {
		log.Infof("no new or fuzzy entries to translate, PO file is already complete")
		result.PreValidationPass = true
		result.PostValidationPass = true
		result.Score = 100
		return result, nil
	}

	result.PreValidationPass = true

	// We can extract new entries and fuzzy entries from the PO file using
	// "msgattrib --untranslated --only-fuzzy poFile", and saved to a
	// temporary file, then pass it to the agent as a source file.
	// This way, we can translate the new entries and fuzzy entries in one
	// round of translation. Later, we can use msgcat to merge the translations
	// back to the PO file like "msgcat --use-first new.po original.po -o merged.po".
	//
	// But we can document this in the po/README.md, and let the code agent
	// decide whether to use this feature.
	//
	// Now, load the simple prompt for translate the file.
	prompt, err := GetPrompt(cfg, "translate")
	if err != nil {
		return result, err
	}

	// Build agent command with placeholders replaced
	workDir := repository.WorkDir()
	sourcePath := poFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		sourcePath = filepath.ToSlash(rel)
	}
	agentCmd := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": sourcePath})

	// Determine output format
	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	// Normalize output format (convert underscores to hyphens)
	outputFormat = normalizeOutputFormat(outputFormat)

	// Execute agent command
	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json",
		truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	var stdout []byte
	var stderr []byte
	var streamResult AgentStreamResult

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	// Use streaming execution for json format (treated as stream-json)
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd, workDir)
		if err != nil {
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		defer stdoutReader.Close()

		stdout, streamResult, _ = parseStreamByKind(kind, stdoutReader)

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd, workDir)
		if err != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", err)
			log.Errorf("agent command execution failed: %v", err)
			return result, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", err)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, err := ParseClaudeAgentOutput(stdout, outputFormat)
			if err != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", err)
				parsedStdout = stdout
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

	applyAgentDiagnostics(result, streamResult)

	// Log output if verbose
	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	// Post-validation: Count new and fuzzy entries after translation
	log.Infof("performing post-validation: counting new and fuzzy entries")

	// Count new entries
	newCountAfter, err := CountNewEntries(poFile)
	if err != nil {
		log.Errorf("failed to count new entries after translation: %v", err)
		return result, fmt.Errorf("failed to count new entries after translation: %w", err)
	}
	result.AfterNewCount = newCountAfter
	log.Infof("new (untranslated) entries after translation: %d", newCountAfter)

	// Count fuzzy entries
	fuzzyCountAfter, err := CountFuzzyEntries(poFile)
	if err != nil {
		log.Errorf("failed to count fuzzy entries after translation: %v", err)
		return result, fmt.Errorf("failed to count fuzzy entries after translation: %w", err)
	}
	result.AfterFuzzyCount = fuzzyCountAfter
	log.Infof("fuzzy entries after translation: %d", fuzzyCountAfter)

	// Validate translation success: both new and fuzzy entries must be 0
	if newCountAfter != 0 || fuzzyCountAfter != 0 {
		log.Errorf("post-validation failed: translation incomplete (new: %d, fuzzy: %d)", newCountAfter, fuzzyCountAfter)
		result.PostValidationError = fmt.Sprintf("translation incomplete: %d new entries and %d fuzzy entries remaining", newCountAfter, fuzzyCountAfter)
		result.Score = 0
		return result, fmt.Errorf("post-validation failed: %s\nHint: The agent should translate all new entries and resolve all fuzzy entries", result.PostValidationError)
	}

	result.PostValidationPass = true
	result.Score = 100
	log.Infof("post-validation passed: all entries translated")

	// Validate PO file syntax (only if agent succeeded)
	if result.AgentSuccess {
		log.Infof("validating file syntax: %s", poFile)
		if err := ValidatePoFile(poFile); err != nil {
			log.Errorf("file syntax validation failed: %v", err)
			result.SyntaxValidationError = err.Error()
			// Don't fail the run for syntax errors in agent-run, but log it
			// In agent-test, this might affect the score
		} else {
			result.SyntaxValidationPass = true
			log.Infof("file syntax validation passed")
		}
	}

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// CmdAgentRunTranslate implements the agent-run translate command logic.
// It loads configuration and calls RunAgentTranslate, then handles errors appropriately.
func CmdAgentRunTranslate(agentName, poFile string) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	result, err := RunAgentTranslate(cfg, agentName, poFile, false)
	if err != nil {
		return err
	}

	// For agent-run, we require all validations to pass
	if !result.PreValidationPass {
		return fmt.Errorf("pre-validation failed: %s", result.PreValidationError)
	}
	if !result.AgentSuccess {
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}
	if !result.PostValidationPass {
		return fmt.Errorf("post-validation failed: %s", result.PostValidationError)
	}
	if result.SyntaxValidationError != "" {
		return fmt.Errorf("file validation failed: %s\nHint: Check the PO file syntax using 'msgfmt --check-format'", result.SyntaxValidationError)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run translate completed successfully")
	return nil
}

// executeReviewAgent executes the agent command for reviewing the given file.
// vars contains placeholder values (e.g. "prompt", "source" for the file to review).
// Returns stdout (for JSON extraction), stderr, originalStdout (raw before parsing), streamResult.
// Updates result with AgentExecuted, AgentSuccess, AgentError, AgentStdout, AgentStderr.
func executeReviewAgent(selectedAgent config.Agent, vars PlaceholderVars, workDir string, result *AgentRunResult) (stdout, stderr, originalStdout []byte, streamResult AgentStreamResult, err error) {
	agentCmd := BuildAgentCommand(selectedAgent, vars)

	outputFormat := selectedAgent.Output
	if outputFormat == "" {
		outputFormat = "default"
	}
	outputFormat = normalizeOutputFormat(outputFormat)

	log.Infof("executing agent command (output=%s, streaming=%v): %s", outputFormat, outputFormat == "json", truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(agentCmd, workDir)
		if execErr != nil {
			log.Errorf("agent command execution failed: %v", execErr)
			return nil, nil, nil, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
		defer stdoutReader.Close()

		var stdoutBuf bytes.Buffer
		teeReader := io.TeeReader(stdoutReader, &stdoutBuf)

		stdout, streamResult, _ = parseStreamByKind(kind, teeReader)
		originalStdout = stdoutBuf.Bytes()

		waitErr := cmdProcess.Wait()
		stderr = stderrBuf.Bytes()

		if waitErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", waitErr)
			log.Errorf("agent command execution failed: %v", waitErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")
	} else {
		var execErr error
		stdout, stderr, execErr = ExecuteAgentCommand(agentCmd, workDir)
		originalStdout = stdout
		result.AgentStdout = stdout
		result.AgentStderr = stderr

		if execErr != nil {
			if len(stderr) > 0 {
				log.Debugf("agent command stderr: %s", string(stderr))
			}
			if len(stdout) > 0 {
				log.Debugf("agent command stdout: %s", string(stdout))
			}
			result.AgentError = fmt.Sprintf("agent command failed: %v (see logs for agent stderr output)", execErr)
			log.Errorf("agent command execution failed: %v", execErr)
			return nil, stderr, originalStdout, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
		}
		result.AgentSuccess = true
		log.Infof("agent command completed successfully")

		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, parseErr := ParseClaudeAgentOutput(stdout, outputFormat)
			if parseErr != nil {
				log.Warnf("failed to parse agent output: %v, using raw output", parseErr)
			} else {
				stdout = parsedStdout
				streamResult = parsedResult
			}
		}
	}

	applyAgentDiagnostics(result, streamResult)
	result.AgentStdout = originalStdout
	if len(stderr) > 0 {
		result.AgentStderr = stderr
	}

	if len(stdout) > 0 {
		log.Debugf("agent command stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent command stderr: %s", string(stderr))
	}

	return stdout, stderr, originalStdout, streamResult, nil
}

// runReviewSingleBatch runs review on the full file (single batch).
func runReviewSingleBatch(selectedAgent config.Agent, vars PlaceholderVars, workDir string, result *AgentRunResult, entryCount int) (*ReviewJSONResult, error) {
	stdout, _, _, _, err := executeReviewAgent(selectedAgent, vars, workDir, result)
	if err != nil {
		return nil, err
	}
	return parseAndAccumulateReviewJSON(stdout, entryCount)
}

// runReviewBatched runs review in batches using msg-select when entry count > 100.
func runReviewBatched(selectedAgent config.Agent, vars PlaceholderVars, workDir string, result *AgentRunResult, entryCount int) (*ReviewJSONResult, error) {
	reviewPOFile := vars["source"]
	num := 50
	if entryCount > 500 {
		num = 100
	} else if entryCount > 200 {
		num = 75
	}

	batchFile := filepath.Join(PoDir, "review-batch.po")

	var allIssues []ReviewIssue
	for batchNum := 1; ; batchNum++ {
		start := (batchNum-1)*num + 1
		end := batchNum * num
		if end > entryCount {
			end = entryCount
		}

		rangeSpec := formatMsgSelectRange(batchNum, start, end, entryCount, num)
		log.Infof("reviewing batch %d: entries %d-%d (of %d)", batchNum, start, end, entryCount)

		// Extract batch with msg-select
		f, err := os.Create(batchFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create batch file: %w", err)
		}
		if err := MsgSelect(reviewPOFile, rangeSpec, f, false); err != nil {
			f.Close()
			os.Remove(batchFile)
			return nil, fmt.Errorf("msg-select failed: %w", err)
		}
		f.Close()

		// Run agent on batch
		batchVars := make(PlaceholderVars)
		for k, v := range vars {
			batchVars[k] = v
		}
		batchVars["source"] = batchFile
		stdout, _, _, _, err := executeReviewAgent(selectedAgent, batchVars, workDir, result)
		os.Remove(batchFile) // Clean up batch file
		if err != nil {
			return nil, err
		}

		// Parse JSON and accumulate issues
		batchJSON, err := parseAndAccumulateReviewJSON(stdout, entryCount)
		if err != nil {
			return nil, err
		}
		if batchJSON != nil {
			allIssues = append(allIssues, batchJSON.Issues...)
		}

		if end >= entryCount {
			break
		}
	}

	return &ReviewJSONResult{TotalEntries: entryCount, Issues: allIssues}, nil
}

// formatMsgSelectRange returns the range spec for msg-select (e.g. "-50", "51-100", "101-").
func formatMsgSelectRange(batchNum, start, end, entryCount, num int) string {
	if batchNum == 1 {
		return fmt.Sprintf("-%d", num)
	}
	if end >= entryCount {
		return fmt.Sprintf("%d-", start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

// parseAndAccumulateReviewJSON extracts and parses JSON from stdout, updates total_entries.
func parseAndAccumulateReviewJSON(stdout []byte, entryCount int) (*ReviewJSONResult, error) {
	jsonBytes, err := ExtractJSONFromOutput(stdout)
	if err != nil {
		log.Errorf("failed to extract JSON from agent output: %v", err)
		return nil, fmt.Errorf("failed to extract JSON: %w", err)
	}

	reviewJSON, err := ParseReviewJSON(jsonBytes)
	if err != nil {
		log.Errorf("failed to parse review JSON: %v", err)
		return nil, fmt.Errorf("failed to parse review JSON: %w", err)
	}

	reviewJSON.TotalEntries = entryCount
	log.Debugf("parsed review JSON: total_entries=%d, issues=%d", reviewJSON.TotalEntries, len(reviewJSON.Issues))
	return reviewJSON, nil
}

// buildReviewAllWithLLMPrompt constructs a dynamic prompt for RunAgentReviewAllWithLLM
func buildReviewAllWithLLMPrompt(target *CompareTarget) string {
	var taskDesc string
	if target.OldFile != target.NewFile {
		taskDesc = fmt.Sprintf("Review %s changes between %s and %s", target.NewFile, target.OldFile, target.NewFile)
	} else if target.NewCommit != "" && (target.OldCommit == target.NewCommit+"~" ||
		target.OldCommit == target.NewCommit+"~1" ||
		target.OldCommit == target.NewCommit+"^") {
		taskDesc = fmt.Sprintf("Review %s changes in commit %s", target.NewFile, target.NewCommit)
	} else if target.NewCommit == "" && target.OldCommit != "" {
		if target.OldCommit == "HEAD" {
			taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
		} else {
			taskDesc = fmt.Sprintf("Review %s changes since commit %s", target.NewFile, target.OldCommit)
		}
	} else if target.OldCommit != "" && target.NewCommit != "" {
		taskDesc = fmt.Sprintf("Review %s changes in range %s..%s", target.NewFile, target.OldCommit, target.NewCommit)
	} else {
		taskDesc = fmt.Sprintf("Review %s local changes", target.NewFile)
	}

	return taskDesc + " according to @po/AGENTS.md."
}

// RunAgentReviewAllWithLLM executes review using a pure LLM approach (--all-with-llm).
// No programmatic extraction or batching; the LLM does everything and writes review.json.
// Before execution: deletes review.po and review.json. After: expects review.json to exist.
func RunAgentReviewAllWithLLM(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	reviewPOFile, reviewJSONFile := ReviewOutputPaths(outputBase)
	workDir := repository.WorkDir()
	startTime := time.Now()
	result := &AgentRunResult{Score: 0}

	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return result, err
	}
	if !Exist(poFile) {
		result.AgentError = fmt.Sprintf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	// Delete existing review output files
	os.Remove(reviewPOFile)
	os.Remove(reviewJSONFile)
	log.Infof("removed existing %s and %s", reviewPOFile, reviewJSONFile)

	poFileRel := target.NewFile
	if rel, err := filepath.Rel(workDir, poFile); err == nil && rel != "" && rel != "." {
		poFileRel = filepath.ToSlash(rel)
	}
	prompt := buildReviewAllWithLLMPrompt(target)
	agentCmd := BuildAgentCommand(selectedAgent, PlaceholderVars{"prompt": prompt, "source": poFileRel})

	outputFormat := normalizeOutputFormat(selectedAgent.Output)
	if outputFormat == "" {
		outputFormat = "default"
	}

	log.Infof("executing agent command (all-with-llm, output=%s): %s", outputFormat, truncateCommandDisplay(strings.Join(agentCmd, " ")))
	result.AgentExecuted = true

	kind := selectedAgent.Kind
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode

	var stdout, stderr []byte
	if outputFormat == "json" {
		stdoutReader, stderrBuf, cmdProcess, err := ExecuteAgentCommandStream(agentCmd, workDir)
		if err != nil {
			result.AgentError = err.Error()
			return result, fmt.Errorf("agent command failed: %w", err)
		}
		defer stdoutReader.Close()
		_, streamResult, _ := parseStreamByKind(kind, stdoutReader)
		applyAgentDiagnostics(result, streamResult)
		if waitErr := cmdProcess.Wait(); waitErr != nil {
			result.AgentError = waitErr.Error()
			return result, fmt.Errorf("agent command failed: %w", waitErr)
		}
		stderr = stderrBuf.Bytes()
	} else {
		var err error
		stdout, stderr, err = ExecuteAgentCommand(agentCmd, workDir)
		if err != nil {
			result.AgentError = err.Error()
			return result, err
		}
		if !isCodex && !isOpencode {
			parsedStdout, parsedResult, _ := ParseClaudeAgentOutput(stdout, outputFormat)
			stdout = parsedStdout
			applyAgentDiagnostics(result, parsedResult)
		}
	}

	result.AgentSuccess = true
	log.Infof("agent command completed successfully")

	if len(stdout) > 0 {
		log.Debugf("agent stdout: %s", string(stdout))
	}
	if len(stderr) > 0 {
		log.Debugf("agent stderr: %s", string(stderr))
	}

	if !Exist(reviewJSONFile) {
		return result, fmt.Errorf("review JSON not generated at %s\nHint: The agent must write the review result to this file", reviewJSONFile)
	}

	reportResult, err := ReportReviewFromJSON(reviewJSONFile)
	if err != nil {
		return result, fmt.Errorf("failed to read review JSON: %w", err)
	}

	result.ReviewJSON = reportResult.Review
	result.ReviewJSONPath = reviewJSONFile
	result.ReviewScore = reportResult.Score
	result.Score = reportResult.Score
	result.ReviewedFilePath = poFile
	result.ExecutionTime = time.Since(startTime)

	log.Infof("review completed (score: %d/100, total entries: %d, issues: %d)",
		reportResult.Score, reportResult.Review.TotalEntries, len(reportResult.Review.Issues))

	return result, nil
}

// RunAgentReview executes a single agent-run review operation with the new workflow:
// 1. Prepare review data (orig.po, new.po, review-input.po)
// 2. Copy review-input.po to review-output.po
// 3. Execute agent to review and modify review-output.po
// 4. Merge review-output.po with new.po using msgcat
// 5. Parse JSON from agent output and calculate score
// Returns a result structure with detailed information.
// The agentTest parameter is provided for consistency, though this method
// does not use AgentTest configuration.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
func RunAgentReview(cfg *config.AgentConfig, agentName string, target *CompareTarget, agentTest bool, outputBase string) (*AgentRunResult, error) {
	reviewPOFile, reviewJSONFile := ReviewOutputPaths(outputBase)
	var (
		workDir    = repository.WorkDir()
		reviewJSON *ReviewJSONResult
	)

	startTime := time.Now()
	result := &AgentRunResult{
		Score: 0,
	}

	// Determine agent to use
	selectedAgent, err := SelectAgent(cfg, agentName)
	if err != nil {
		result.AgentError = err.Error()
		return result, err
	}

	log.Debugf("using agent: %s (%s)", agentName, selectedAgent.Kind)

	// Determine PO file path (convert to absolute) - use newFile as the file being reviewed
	poFile, err := GetPoFileAbsPath(cfg, target.NewFile)
	if err != nil {
		return result, err
	}

	log.Debugf("PO file path: %s", poFile)

	// Check if PO file exists
	if !Exist(poFile) {
		log.Errorf("PO file does not exist: %s", poFile)
		return result, fmt.Errorf("PO file does not exist: %s\nHint: Ensure the PO file exists before running review", poFile)
	}

	// Step 1: Prepare review data
	log.Infof("preparing review data: %s", reviewPOFile)
	if err := PrepareReviewData(target.OldCommit, target.OldFile, target.NewCommit, target.NewFile, reviewPOFile); err != nil {
		return result, fmt.Errorf("failed to prepare review data: %w", err)
	}

	// Step 2: Get prompt.review
	prompt, err := GetPrompt(cfg, "review")
	if err != nil {
		return result, err
	}
	log.Debugf("using review prompt: %s", prompt)

	totalEntries, err := countMsgidEntries(reviewPOFile)
	if err != nil {
		log.Errorf("failed to count msgid entries in review input file: %v", err)
		return result, fmt.Errorf("failed to count entries: %w", err)
	}
	entryCount := totalEntries
	if entryCount > 0 {
		entryCount-- // Exclude header
	}

	reviewVars := PlaceholderVars{
		"prompt": prompt,
		"source": reviewPOFile,
		"dest":   reviewPOFile,
		"json":   reviewJSONFile,
	}
	if entryCount <= 100 {
		// Single run: review entire file
		reviewJSON, err = runReviewSingleBatch(selectedAgent, reviewVars, workDir, result, entryCount)
		if err != nil {
			return result, err
		}
	} else {
		// Batch mode: iterate with msg-select
		reviewJSON, err = runReviewBatched(selectedAgent, reviewVars, workDir, result, entryCount)
		if err != nil {
			return result, err
		}
	}

	// Save JSON to file
	log.Infof("saving review JSON to %s", reviewJSONFile)
	if err := saveReviewJSON(reviewJSON, reviewJSONFile); err != nil {
		log.Errorf("failed to save review JSON: %v", err)
		log.Debugf("PO file path: %s", poFile)
		return result, fmt.Errorf("failed to save review JSON: %w", err)
	}
	result.ReviewJSON = reviewJSON
	result.ReviewJSONPath = reviewJSONFile

	// Calculate review score
	log.Infof("calculating review score")
	reviewScore, err := CalculateReviewScore(reviewJSON)
	if err != nil {
		log.Errorf("failed to calculate review score: %v", err)
		log.Debugf("review JSON: total_entries=%d, issues=%d", reviewJSON.TotalEntries, len(reviewJSON.Issues))
		return result, fmt.Errorf("failed to calculate review score: %w", err)
	}
	result.ReviewScore = reviewScore
	result.Score = reviewScore
	result.ReviewedFilePath = reviewPOFile

	log.Infof("review completed successfully (score: %d/100, total entries: %d, issues: %d, reviewed file: %s)",
		reviewScore, reviewJSON.TotalEntries, len(reviewJSON.Issues), reviewPOFile)

	// Record execution time
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// CmdAgentRunReview implements the agent-run review command logic.
// It loads configuration and calls RunAgentReview or RunAgentReviewAllWithLLM.
// outputBase: base path for review output files (e.g. "po/review"); empty uses default.
// allWithLLM: if true, use pure LLM approach (--all-with-llm).
func CmdAgentRunReview(agentName string, target *CompareTarget, outputBase string, allWithLLM bool) error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}

	startTime := time.Now()

	var result *AgentRunResult
	if allWithLLM {
		result, err = RunAgentReviewAllWithLLM(cfg, agentName, target, false, outputBase)
	} else {
		result, err = RunAgentReview(cfg, agentName, target, false, outputBase)
	}
	if err != nil {
		log.Errorf("failed to run agent review: %v", err)
		return err
	}

	// For agent-run, we require agent execution to succeed
	if !result.AgentSuccess {
		log.Errorf("agent execution failed: %s", result.AgentError)
		return fmt.Errorf("agent execution failed: %s", result.AgentError)
	}

	elapsed := time.Since(startTime)

	// Display review results
	if result.ReviewJSON != nil {
		fmt.Printf("\nReview Results:\n")
		fmt.Printf("  Total entries: %d\n", result.ReviewJSON.TotalEntries)
		fmt.Printf("  Issues found: %d\n", len(result.ReviewJSON.Issues))
		fmt.Printf("  Review score: %d/100\n", result.ReviewScore)

		// Count issues by severity
		criticalCount := 0
		majorCount := 0
		minorCount := 0
		for _, issue := range result.ReviewJSON.Issues {
			switch issue.Score {
			case 0:
				criticalCount++
			case 1:
				majorCount++
			case 2:
				minorCount++
			}
		}

		fmt.Printf("\n  Issue breakdown:\n")
		if len(result.ReviewJSON.Issues) > 0 {
			if criticalCount > 0 {
				fmt.Printf("    Critical (must fix immediately): %d\n", criticalCount)
			}
			if majorCount > 0 {
				fmt.Printf("    Major (should fix): %d\n", majorCount)
			}
			if minorCount > 0 {
				fmt.Printf("    Minor (recommended to improve): %d\n", minorCount)
			}
		}
		fmt.Printf("    Perfect entries: %d\n",
			result.ReviewJSON.TotalEntries-criticalCount-minorCount)

		if result.ReviewJSONPath != "" {
			fmt.Printf("\n  JSON saved to: %s\n", getRelativePath(result.ReviewJSONPath))
		}
		if result.ReviewedFilePath != "" {
			fmt.Printf("  Reviewed file: %s\n", getRelativePath(result.ReviewedFilePath))
		}
	}

	fmt.Printf("\nSummary:\n")
	if result.NumTurns > 0 {
		fmt.Printf("  Turns: %d\n", result.NumTurns)
	}
	fmt.Printf("  Execution time: %s\n", elapsed.Round(time.Millisecond))

	log.Infof("agent-run review completed successfully")
	return nil
}

// countMsgidEntries counts the number of msgid entries in a PO file by counting lines that start with "msgid "
func countMsgidEntries(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "msgid ") {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return count, nil
}
