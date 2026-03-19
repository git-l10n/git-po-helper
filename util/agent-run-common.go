// Package util provides business logic for agent-run command.
package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
)

// LoadAgentConfigForCmd loads agent configuration from the standard location
// (flag.AgentConfigFile()). On failure returns an error wrapped with a consistent
// hint for missing or invalid git-po-helper.yaml.
func LoadAgentConfigForCmd() (*config.AgentConfig, error) {
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.GetConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("failed to load agent configuration: %w\nHint: Ensure git-po-helper.yaml exists in repository root or user home directory", err)
	}
	return cfg, nil
}

// CountMsgidEntries counts the number of msgid entries in a PO file by counting
// lines that start with "msgid "
func CountMsgidEntries(filePath string) (int, error) {
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

// CalcBatchSize returns the batch size for the given entryCount and minBatchSize,
// following the AGENTS.md formula exactly.
func CalcBatchSize(entryCount, minBatchSize int) int {
	if entryCount > minBatchSize*8 {
		return minBatchSize * 2
	}
	if entryCount > minBatchSize*4 {
		return minBatchSize + minBatchSize/2
	}
	if entryCount > minBatchSize {
		return minBatchSize
	}
	return entryCount
}

// validateEntryCount is the internal implementation for POT/PO entry count validation.
// filePath is used in error messages. stage is "before update" or "after update".
func validateEntryCount(filePath string, expectedCount *int, stage string) error {
	if expectedCount == nil || *expectedCount == 0 {
		return nil
	}

	fileExists := Exist(filePath)
	var actualCount int
	var err error

	if !fileExists {
		if stage == "before update" {
			actualCount = 0
			log.Debugf("file %s does not exist, treating entry count as 0 for %s validation", filePath, stage)
		} else {
			return fmt.Errorf("file does not exist %s: %s\nHint: The agent should have created the file", stage, filePath)
		}
	} else {
		var stats *PoStats
		stats, err = GetPoStats(filePath)
		if err != nil {
			return fmt.Errorf("failed to count entries %s in %s: %w", stage, filePath, err)
		}
		actualCount = stats.Total()
	}

	if actualCount != *expectedCount {
		return fmt.Errorf("entry count %s: expected %d, got %d (file: %s)", stage, *expectedCount, actualCount, filePath)
	}

	log.Debugf("entry count %s validation passed: %d entries", stage, actualCount)
	return nil
}

// ValidatePotEntryCount validates the entry count in a POT file.
// If expectedCount is nil or 0, validation is disabled and the function returns nil.
// Otherwise, it counts entries using GetPoStats() and compares with expectedCount.
// Returns an error if counts don't match, nil if they match or validation is disabled.
// The stage parameter is used for error messages ("before update" or "after update").
// For "before update" stage, if the file doesn't exist, the entry count is treated as 0.
func ValidatePotEntryCount(potFile string, expectedCount *int, stage string) error {
	return validateEntryCount(potFile, expectedCount, stage)
}

// ValidatePoEntryCount validates the entry count in a PO file.
// If expectedCount is nil or 0, validation is disabled and the function returns nil.
// Otherwise, it counts entries using GetPoStats() and compares with expectedCount.
// Returns an error if counts don't match, nil if they match or validation is disabled.
// The stage parameter is used for error messages ("before update" or "after update").
// For "before update" stage, if the file doesn't exist, the entry count is treated as 0.
func ValidatePoEntryCount(poFile string, expectedCount *int, stage string) error {
	return validateEntryCount(poFile, expectedCount, stage)
}

// ValidatePoFile validates POT/PO file syntax.
// For .pot files, it uses msgcat --use-first to validate (since POT files have placeholders in headers).
// For .po files, it uses msgfmt to validate.
// Returns an error if the file is invalid, nil if valid.
// If the file path is absolute, it doesn't require repository context.
// If the file path is relative, the subprocess uses the process current working directory (cmd.Dir unset).
func ValidatePoFile(potFile string) error {
	return validatePoFileInternal(potFile, false)
}

// ValidatePoFileFormat validates POT/PO file format syntax only (using --check-format for PO files).
// This is a more lenient check that doesn't require complete headers.
// For .pot files, it uses msgcat --use-first to validate.
// For .po files, it uses msgfmt --check-format to validate (only checks format, not completeness).
// Returns an error if the file format is invalid, nil if valid.
// If the file path is absolute, it doesn't require repository context.
// If the file path is relative, the subprocess uses the process current working directory (cmd.Dir unset).
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

	// For absolute paths, use the directory containing the file as working directory.
	// For relative paths, cmd.Dir is not set and the command uses process CWD.
	if filepath.IsAbs(potFile) {
		cmd.Dir = filepath.Dir(potFile)
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
// If poFile is empty, it uses the effective default_lang_code (config or system locale) to construct the path.
// If poFile is provided but not absolute, it's treated as relative to the repository root.
func GetPoFileAbsPath(cfg *config.AgentConfig, poFile string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if poFile == "" {
		lang := cfg.DefaultLangCode
		if lang == "" {
			return "", fmt.Errorf("default_lang_code is not configured\nHint: Provide po/XX.po on the command line or set default_lang_code in git-po-helper.yaml")
		}
		poFile = filepath.Join(workDir, PoDir, fmt.Sprintf("%s.po", lang))
	} else if !filepath.IsAbs(poFile) {
		// Treat poFile as relative to repository root
		poFile = filepath.Join(workDir, poFile)
	}
	return poFile, nil
}

// GuessPoFilePath determines the relative path of a PO file in "po/XX.po" format.
// If poFile is empty, it uses default_lang_code to construct PoDir/lang.po (relative).
// If poFile is absolute, it converts via filepath.Rel to repository root.
// If poFile is already relative, it normalizes with filepath.Clean and ToSlash.
// Does not join with workDir—callers run with cwd at repo root so relative paths suffice.
func GuessPoFilePath(cfg *config.AgentConfig, poFile string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if poFile == "" {
		lang := cfg.DefaultLangCode
		if lang == "" {
			return "", fmt.Errorf("default_lang_code is not configured\nHint: Provide po/XX.po on the command line or set default_lang_code in git-po-helper.yaml")
		}
		return filepath.ToSlash(filepath.Join(PoDir, fmt.Sprintf("%s.po", lang))), nil
	}
	if filepath.IsAbs(poFile) {
		relPath, err := filepath.Rel(workDir, poFile)
		if err != nil {
			return "", fmt.Errorf("failed to convert path to relative: %w", err)
		}
		return filepath.ToSlash(relPath), nil
	}
	return filepath.ToSlash(filepath.Clean(poFile)), nil
}

// detectAgentOutputFormat inspects the first non-empty line and returns the detected
// agent format (e.g. config.AgentKindClaude) or "" if output appears to be plain text.
func detectAgentOutputFormat(raw []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "{") {
			return ""
		}
		if strings.Contains(line, "claude_code_version") {
			return config.AgentKindClaude
		}
		if strings.Contains(line, `"type":"step_start"`) || strings.Contains(line, `"type": "step_start"`) {
			return config.AgentKindOpencode
		}
		if strings.Contains(line, "thread.started") {
			return config.AgentKindCodex
		}
		if strings.Contains(line, `"provider":"qoder"`) || strings.Contains(line, `"provider": "qoder"`) {
			return config.AgentKindQoder
		}
		if strings.Contains(line, `"type":"result"`) && strings.Contains(line, `"subtype":"success"`) {
			return config.AgentKindQoder
		}
		if strings.Contains(line, `"type":"system"`) {
			return config.AgentKindGemini
		}
		return ""

	}
	return ""
}

// parseBatchOutput parses buffered agent output. It auto-detects format from content
// and uses the appropriate parser. Returns (content, streamResult, nil) on success.
// If output is plain text (no JSONL detected), returns (raw, nil, nil).
func parseBatchOutput(raw []byte) (content []byte, streamResult AgentStreamResult, err error) {
	detected := detectAgentOutputFormat(raw)
	if detected == "" {
		return raw, nil, nil
	}
	content, streamResult, err = parseStreamByKind(detected, bytes.NewReader(raw))
	if err != nil {
		log.Warnf("failed to parse agent output as %s: %v, using raw output", detected, err)
		return raw, nil, nil
	}
	return content, streamResult, nil
}

// RunAgentAndParse executes the agent command and parses output.
// It always uses ExecuteAgentCommandStream internally.
//
// Returns:
//   - stdout: Parsed or raw stdout content (for downstream use)
//   - originalStdout: Raw stdout bytes before parsing (for result.AgentStdout)
//   - stderr: Stderr bytes
//   - streamResult: AgentStreamResult for diagnostics (NumTurns, Usage, etc.)
//   - err: Execution or parse error
func RunAgentAndParse(cmd []string, outputFormat, kind string) (
	stdout, originalStdout, stderr []byte,
	streamResult AgentStreamResult,
	err error,
) {
	stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(cmd)
	if execErr != nil {
		return nil, nil, nil, nil, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", execErr)
	}
	defer stdoutReader.Close()

	switch outputFormat {
	case config.OutputJSON:
		fallthrough
	case config.OutputStreamJSON:
		// Stream parsing for JSONL output
		var rawBuf bytes.Buffer
		teeReader := io.TeeReader(stdoutReader, &rawBuf)
		stdout, streamResult, _ = parseStreamByKind(kind, teeReader)
		originalStdout = rawBuf.Bytes()
	default:
		// OutputText or unknown: stream read and print to stdout
		var rawBuf bytes.Buffer
		teeReader := io.TeeReader(stdoutReader, &rawBuf)
		if _, err := io.Copy(os.Stdout, teeReader); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to read agent output: %w", err)
		}
		originalStdout = rawBuf.Bytes()
		stdout = originalStdout
	}

	waitErr := cmdProcess.Wait()
	stderr = stderrBuf.Bytes()
	if waitErr != nil {
		if len(stderr) > 0 {
			log.Debugf("agent command stderr: %s", string(stderr))
		}
		return stdout, originalStdout, stderr, streamResult, fmt.Errorf("agent command failed: %w\nHint: Check that the agent command is correct and executable", waitErr)
	}

	return stdout, originalStdout, stderr, streamResult, nil
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
	case config.AgentKindQoder:
		parsed, res, e := ParseQoderJSONLRealtime(reader)
		if e != nil {
			log.Warnf("failed to parse qoder JSONL: %v", e)
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
