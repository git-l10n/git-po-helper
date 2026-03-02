package util

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

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
