// Package util provides business logic for agent-run command.
package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

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

// ValidatePotEntryCount validates the entry count in a POT file.
// If expectedCount is nil or 0, validation is disabled and the function returns nil.
// Otherwise, it counts entries using CountPoReportStats() and compares with expectedCount.
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
		var stats *PoReportStats
		stats, err = CountPoReportStats(potFile)
		if err != nil {
			return fmt.Errorf("failed to count entries %s in %s: %w", stage, potFile, err)
		}
		actualCount = stats.Total()
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
// Otherwise, it counts entries using CountPoReportStats() and compares with expectedCount.
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
		var stats *PoReportStats
		stats, err = CountPoReportStats(poFile)
		if err != nil {
			return fmt.Errorf("failed to count entries %s in %s: %w", stage, poFile, err)
		}
		actualCount = stats.Total()
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
