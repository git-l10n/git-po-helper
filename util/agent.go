// Package util provides utility functions for agent execution.
package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// maxDisplayBytes is the maximum number of bytes to display for agent messages (4KB).
	maxDisplayBytes = 4096
	// maxDisplayLines is the maximum number of lines to display for agent messages.
	maxDisplayLines = 10
)

// flushStdout flushes stdout to ensure agent output (🤖 etc.) is visible immediately.
// Without this, stdout may be buffered when not a TTY, causing output to appear only with -v
// (which produces more stderr activity that can trigger flushing in some environments).
func flushStdout() {
	_ = os.Stdout.Sync()
}

// CountPotEntries counts msgid entries in a POT file.
// It excludes the header entry (which has an empty msgid) and counts
// only non-empty msgid entries.
//
// The function:
// - Opens the POT file
// - Scans for lines starting with "msgid " (excluding commented entries)
// - Parses msgid values to identify the header entry (empty msgid)
// - Returns the count of non-empty msgid entries
func CountPotEntries(potFile string) (int, error) {
	f, err := os.Open(potFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open POT file %s: %w", potFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	inMsgid := false
	msgidValue := ""
	headerFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comment lines (obsolete entries, etc.)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// If we were already in a msgid, finish the previous one
			if inMsgid {
				if !headerFound && strings.Trim(msgidValue, `"`) == "" {
					headerFound = true
				} else if strings.Trim(msgidValue, `"`) != "" {
					// Non-empty msgid entry
					count++
				}
			}
			// Start new msgid entry
			inMsgid = true
			// Extract the msgid value (may be on same line or continue on next lines)
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			// Remove quotes if present
			msgidValue = strings.Trim(msgidValue, `"`)
			continue
		}

		// If we're in a msgid entry and this line continues it (starts with quote)
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// Continuation line - append to msgidValue (remove quotes)
			contValue := strings.Trim(trimmed, `"`)
			msgidValue += contValue
			continue
		}

		// If we encounter msgstr, it means we've finished the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// End of msgid entry
			if !headerFound && strings.Trim(msgidValue, `"`) == "" {
				headerFound = true
			} else if strings.Trim(msgidValue, `"`) != "" {
				// Non-empty msgid entry
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}

		// Empty line might indicate end of entry, but we'll rely on msgstr
		// to be more accurate
	}

	// Handle last entry if file doesn't end with newline or msgstr
	if inMsgid {
		if !headerFound && strings.Trim(msgidValue, `"`) == "" {
			headerFound = true
		} else if strings.Trim(msgidValue, `"`) != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read POT file %s: %w", potFile, err)
	}

	return count, nil
}

// CountPoEntries counts msgid entries in a PO file.
// It excludes the header entry (which has an empty msgid) and counts
// only non-empty msgid entries.
//
// The function:
// - Opens the PO file
// - Scans for lines starting with "msgid " (excluding commented entries)
// - Parses msgid values to identify the header entry (empty msgid)
// - Returns the count of non-empty msgid entries
func CountPoEntries(poFile string) (int, error) {
	f, err := os.Open(poFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open PO file %s: %w", poFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	inMsgid := false
	msgidValue := ""
	headerFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comment lines (obsolete entries, etc.)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// If we were already in a msgid, finish the previous one
			if inMsgid {
				if !headerFound && strings.Trim(msgidValue, `"`) == "" {
					headerFound = true
				} else if strings.Trim(msgidValue, `"`) != "" {
					// Non-empty msgid entry
					count++
				}
			}
			// Start new msgid entry
			inMsgid = true
			// Extract the msgid value (may be on same line or continue on next lines)
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			// Remove quotes if present
			msgidValue = strings.Trim(msgidValue, `"`)
			continue
		}

		// If we're in a msgid entry and this line continues it (starts with quote)
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// Continuation line - append to msgidValue (remove quotes)
			contValue := strings.Trim(trimmed, `"`)
			msgidValue += contValue
			continue
		}

		// If we encounter msgstr, it means we've finished the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// End of msgid entry
			if !headerFound && strings.Trim(msgidValue, `"`) == "" {
				headerFound = true
			} else if strings.Trim(msgidValue, `"`) != "" {
				// Non-empty msgid entry
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}

		// Empty line might indicate end of entry, but we'll rely on msgstr
		// to be more accurate
	}

	// Handle last entry if file doesn't end with newline or msgstr
	if inMsgid {
		if !headerFound && strings.Trim(msgidValue, `"`) == "" {
			headerFound = true
		} else if strings.Trim(msgidValue, `"`) != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read PO file %s: %w", poFile, err)
	}

	return count, nil
}

// PlaceholderVars holds key-value pairs for placeholder replacement.
// Keys correspond to placeholder names in template (e.g. {prompt}, {source}).
type PlaceholderVars map[string]string

// ReplacePlaceholders replaces placeholders in a template string with actual values.
// Placeholders in template use {key} format, e.g. {prompt}, {source}, {commit}.
//
// Example:
//
//	ReplacePlaceholders("cmd -p {prompt} -s {source}", PlaceholderVars{
//	    "prompt": "update",
//	    "source": "po/zh_CN.po",
//	})
func ReplacePlaceholders(template string, kv PlaceholderVars) string {
	result := template
	for key, value := range kv {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}

// ExecuteAgentCommand executes an agent command and captures both stdout and stderr.
// The command is executed in the specified working directory.
//
// Parameters:
//   - cmd: Command and arguments as a slice (e.g., []string{"claude", "-p", "{prompt}"})
//   - workDir: Working directory for command execution (empty string uses current working directory).
//     To use repository root, pass repository.WorkDir() explicitly.
//
// Returns:
//   - stdout: Standard output from the command
//   - stderr: Standard error from the command
//   - error: Error if command execution fails (includes non-zero exit codes)
//
// The function:
//   - Replaces placeholders in command arguments using ReplacePlaceholders
//   - Executes the command in the specified working directory
//   - Captures both stdout and stderr separately
//   - Returns an error if the command exits with a non-zero status code
func ExecuteAgentCommand(cmd []string) ([]byte, []byte, error) {
	if len(cmd) == 0 {
		return nil, nil, fmt.Errorf("command cannot be empty")
	}

	cwd, _ := os.Getwd()

	// Replace placeholders in command arguments
	// Note: Placeholders should be replaced before calling this function,
	// but we'll handle it here for safety
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	log.Debugf("executing agent command: %s (workDir: %s)", strings.Join(cmd, " "), cwd)

	// Capture stdout and stderr separately
	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	// Execute the command
	err := execCmd.Run()
	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()

	// Check for execution errors
	if err != nil {
		// If command exited with non-zero status, include stderr in error message
		if exitError, ok := err.(*exec.ExitError); ok {
			return stdout, stderr, fmt.Errorf("agent command failed with exit code %d: %w\nstderr: %s",
				exitError.ExitCode(), err, string(stderr))
		}
		return stdout, stderr, fmt.Errorf("failed to execute agent command: %w\nstderr: %s", err, string(stderr))
	}

	log.Debugf("agent command completed successfully (stdout: %d bytes, stderr: %d bytes)",
		len(stdout), len(stderr))

	return stdout, stderr, nil
}

// ExecuteAgentCommandStream executes an agent command and returns a reader for real-time stdout streaming.
// The command is executed in the specified working directory.
// This function is used for json format (stream-json internally) to process output in real-time.
//
// Parameters:
//   - cmd: Command and arguments as a slice
//   - workDir: Working directory for command execution
//
// Returns:
//   - stdoutReader: io.ReadCloser for reading stdout in real-time
//   - stderr: Standard error from the command (captured after execution)
//   - cmdProcess: *exec.Cmd for waiting on command completion
//   - error: Error if command setup fails
func ExecuteAgentCommandStream(cmd []string) (stdoutReader io.ReadCloser, stderrBuf *bytes.Buffer, cmdProcess *exec.Cmd, err error) {
	if len(cmd) == 0 {
		return nil, nil, nil, fmt.Errorf("command cannot be empty")
	}

	// Create command
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	log.Debugf("executing agent command (streaming): %s", strings.Join(cmd, " "))

	// Get stdout pipe for real-time reading
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr separately
	var stderrBuffer bytes.Buffer
	execCmd.Stderr = &stderrBuffer

	// Start command execution
	if err := execCmd.Start(); err != nil {
		stdoutPipe.Close()
		return nil, nil, nil, fmt.Errorf("failed to start agent command: %w", err)
	}

	return stdoutPipe, &stderrBuffer, execCmd, nil
}

// normalizeOutputFormat normalizes output format by converting underscores to hyphens
// and unifying stream-json/stream_json to json.
// This allows both "stream_json" and "stream-json" to be treated as "json".
func normalizeOutputFormat(format string) string {
	normalized := strings.ReplaceAll(format, "_", "-")
	// Unify stream-json to json (claude uses stream-json internally, but we simplify it to json)
	if normalized == "stream-json" {
		return "json"
	}
	return normalized
}

// SelectAgent selects an agent from the configuration based on the provided agent name.
// If agentName is empty, it auto-selects an agent (only works if exactly one agent is configured).
// Returns the selected agent and an error if selection fails.
// Validates that agent.Kind is one of the known types (claude, gemini, codex, opencode, echo).
func SelectAgent(cfg *config.AgentConfig, agentName string) (config.Agent, error) {
	var agent config.Agent

	if agentName != "" {
		// Use specified agent
		log.Debugf("using specified agent: %s", agentName)
		a, ok := cfg.Agents[agentName]
		if !ok {
			agentList := make([]string, 0, len(cfg.Agents))
			for k := range cfg.Agents {
				agentList = append(agentList, k)
			}
			log.Errorf("agent '%s' not found in configuration. Available agents: %v", agentName, agentList)
			return config.Agent{}, fmt.Errorf("agent '%s' not found in configuration\nAvailable agents: %s\nHint: Check git-po-helper.yaml for configured agents", agentName, strings.Join(agentList, ", "))
		}
		agent = a
	} else {
		// Auto-select agent
		log.Debugf("auto-selecting agent from configuration")
		if len(cfg.Agents) == 0 {
			log.Error("no agents configured")
			return config.Agent{}, fmt.Errorf("no agents configured\nHint: Add at least one agent to git-po-helper.yaml in the 'agents' section")
		}
		if len(cfg.Agents) > 1 {
			agentList := make([]string, 0, len(cfg.Agents))
			for k := range cfg.Agents {
				agentList = append(agentList, k)
			}
			log.Errorf("multiple agents configured (%s), --agent flag required", strings.Join(agentList, ", "))
			return config.Agent{}, fmt.Errorf("multiple agents configured (%s), please specify --agent\nHint: Use --agent flag to select one of the available agents", strings.Join(agentList, ", "))
		}
		for k, v := range cfg.Agents {
			agent, agentName = v, k
			break
		}
	}

	// Set agent.Kind initial value when empty: try agentName then command name
	if agent.Kind == "" {
		// Try agentName (config key) converted to lowercase
		if lower := strings.ToLower(agentName); config.KnownAgentKinds[lower] {
			agent.Kind = lower
		} else {
			// Try first command argument (command name): use basename for paths
			if len(agent.Cmd) > 0 {
				base := strings.ToLower(filepath.Base(agent.Cmd[0]))
				if config.KnownAgentKinds[base] {
					agent.Kind = base
				}
			}
		}
		if agent.Kind == "" {
			return config.Agent{}, fmt.Errorf(
				"agent '%s' has unknown kind (cmd=%v)\n"+
					"Hint: Add 'kind' field (claude, gemini, codex, opencode, echo, qwen) to agent in git-po-helper.yaml",
				agentName, agent.Cmd)
		}
	}

	// Validate agent.Kind is a known type
	if !config.KnownAgentKinds[agent.Kind] {
		return config.Agent{}, fmt.Errorf(
			"agent '%s' has unknown kind '%s' (must be one of: claude, gemini, codex, opencode, echo, qwen)\n"+
				"Hint: Set 'kind' to a valid value in git-po-helper.yaml", agentName, agent.Kind)
	}

	return agent, nil
}

// BuildAgentCommand builds an agent command by replacing placeholders in the agent's command template.
// It replaces placeholders (e.g. {prompt}, {source}, {commit}) with values from vars.
// For claude/codex/opencode/gemini commands, it adds stream-json parameters based on agent.Output.
// Uses agent.Kind for type-safe detection (Kind must be validated by SelectAgent).
func BuildAgentCommand(agent config.Agent, vars PlaceholderVars) []string {
	cmd := make([]string, len(agent.Cmd))
	for i, arg := range agent.Cmd {
		cmd[i] = ReplacePlaceholders(arg, vars)
	}

	// Use agent.Kind for type detection (validated by SelectAgent)
	kind := agent.Kind
	isClaude := kind == config.AgentKindClaude
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode
	isGemini := kind == config.AgentKindGemini || kind == config.AgentKindQwen

	// For claude command, add --output-format parameter if output format is specified
	if isClaude {
		// Check if --output-format parameter already exists in the command
		hasOutputFormat := false
		for i, arg := range cmd {
			if arg == "--output-format" || arg == "-o" {
				hasOutputFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --output-format if it doesn't already exist
		if !hasOutputFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --output-format parameter for json format (claude uses stream-json internally)
			if outputFormat == "json" {
				cmd = append(cmd, "--verbose", "--output-format", "stream-json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For codex command, add --json parameter if output format is json
	if isCodex {
		// Check if --json parameter already exists in the command
		hasJSON := false
		for _, arg := range cmd {
			if arg == "--json" {
				hasJSON = true
				break
			}
		}

		// Only add --json if it doesn't already exist
		if !hasJSON {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --json parameter for json format (codex uses JSONL format)
			if outputFormat == "json" {
				cmd = append(cmd, "--json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For opencode command, add --format json parameter if output format is json
	if isOpencode {
		// Check if --format parameter already exists in the command
		hasFormat := false
		for i, arg := range cmd {
			if arg == "--format" {
				hasFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --format if it doesn't already exist
		if !hasFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --format json parameter for json format (opencode uses JSONL format)
			if outputFormat == "json" {
				cmd = append(cmd, "--format", "json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For gemini/qwen command, add --output-format stream-json parameter if output format is json
	// (Applicable to Claude Code and Gemini-CLI)
	if isGemini {
		// Check if --output-format or -o parameter already exists in the command
		hasOutputFormat := false
		for i, arg := range cmd {
			if arg == "--output-format" || arg == "-o" {
				hasOutputFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --output-format if it doesn't already exist
		if !hasOutputFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --output-format stream-json parameter for json format (gemini uses stream-json)
			if outputFormat == "json" {
				cmd = append(cmd, "--output-format", "stream-json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	return cmd
}

// GetPotFilePath returns the full path to the POT file in the repository.
func GetPotFilePath() string {
	workDir := repository.WorkDir()
	return filepath.Join(workDir, PoDir, GitPot)
}

// GetPrompt returns the prompt for the specified action from configuration, or an error if not configured.
// Supported actions: "update-pot", "update-po", "translate", "review"
// If --prompt flag is provided via viper, it overrides the configuration value.
func GetPrompt(cfg *config.AgentConfig, action string) (string, error) {
	// Check if --prompt flag is provided via viper (from command line)
	// Check both agent-run--prompt and agent-test--prompt
	overridePrompt := viper.GetString("agent-run--prompt")
	if overridePrompt == "" {
		overridePrompt = viper.GetString("agent-test--prompt")
	}

	// If override prompt is provided, use it directly
	if overridePrompt != "" {
		log.Debugf("using override prompt from --prompt flag for action %s: %s", action, overridePrompt)
		return overridePrompt, nil
	}

	var prompt string
	var promptName string

	switch action {
	case "update-pot":
		prompt = cfg.Prompt.UpdatePot
		promptName = "prompt.update_pot"
	case "update-po":
		prompt = cfg.Prompt.UpdatePo
		promptName = "prompt.update_po"
	case "translate":
		prompt = cfg.Prompt.Translate
		promptName = "prompt.translate"
	case "review":
		prompt = cfg.Prompt.Review
		promptName = "prompt.review"
	default:
		return "", fmt.Errorf("unknown action: %s\nHint: Supported actions are: update-pot, update-po, translate, review", action)
	}

	if prompt == "" {
		log.Errorf("%s is not configured", promptName)
		return "", fmt.Errorf("%s is not configured\nHint: Add '%s' to git-po-helper.yaml", promptName, promptName)
	}
	log.Debugf("using %s prompt: %s", action, prompt)
	return prompt, nil
}

// CountNewEntries counts untranslated entries in a PO file.
// It uses `msgattrib --untranslated` to extract untranslated entries,
// then counts the msgid entries excluding the header entry (empty msgid).
//
// The function:
// - Executes `msgattrib --untranslated poFile`
// - Scans output for lines starting with "msgid "
// - Excludes the header entry (msgid "")
// - Returns the count of untranslated msgid entries
func CountNewEntries(poFile string) (int, error) {
	cmd := exec.Command("msgattrib", "--untranslated", poFile)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("msgattrib failed for %s: %w\nstderr: %s",
				poFile, err, string(exitError.Stderr))
		}
		return 0, fmt.Errorf("failed to execute msgattrib for %s: %w", poFile, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	inMsgid := false
	msgidValue := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// Extract msgid value
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			inMsgid = true
			continue
		}

		// If we're in a msgid and encounter a continuation line
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// This is a multi-line msgid, just mark it as non-empty
			msgidValue += "continuation"
			continue
		}

		// If we encounter msgstr, finish the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// Check if msgid is non-empty (not the header)
			if strings.Trim(msgidValue, `"`) != "" {
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan msgattrib output: %w", err)
	}

	return count, nil
}

// CountFuzzyEntries counts fuzzy entries in a PO file.
// It uses `msgattrib --only-fuzzy` to extract fuzzy entries,
// then counts the msgid entries excluding the header entry (empty msgid).
//
// The function:
// - Executes `msgattrib --only-fuzzy poFile`
// - Scans output for lines starting with "msgid "
// - Excludes the header entry (msgid "")
// - Returns the count of fuzzy msgid entries
func CountFuzzyEntries(poFile string) (int, error) {
	cmd := exec.Command("msgattrib", "--only-fuzzy", poFile)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("msgattrib failed for %s: %w\nstderr: %s",
				poFile, err, string(exitError.Stderr))
		}
		return 0, fmt.Errorf("failed to execute msgattrib for %s: %w", poFile, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	count := 0
	inMsgid := false
	msgidValue := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for msgid line
		if strings.HasPrefix(trimmed, "msgid ") {
			// Extract msgid value
			msgidValue = strings.TrimPrefix(trimmed, "msgid ")
			msgidValue = strings.TrimSpace(msgidValue)
			inMsgid = true
			continue
		}

		// If we're in a msgid and encounter a continuation line
		if inMsgid && strings.HasPrefix(trimmed, `"`) {
			// This is a multi-line msgid, just mark it as non-empty
			msgidValue += "continuation"
			continue
		}

		// If we encounter msgstr, finish the msgid
		if inMsgid && strings.HasPrefix(trimmed, "msgstr") {
			// Check if msgid is non-empty (not the header)
			if strings.Trim(msgidValue, `"`) != "" {
				count++
			}
			inMsgid = false
			msgidValue = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan msgattrib output: %w", err)
	}

	return count, nil
}

// ClaudeJSONOutput represents the JSON output format from Claude API.
type ClaudeJSONOutput struct {
	Type          string       `json:"type"`
	Subtype       string       `json:"subtype"`
	NumTurns      int          `json:"num_turns"`
	Result        string       `json:"result"`
	DurationAPIMS int          `json:"duration_api_ms"`
	Usage         *ClaudeUsage `json:"usage,omitempty"`
}

// ClaudeUsage represents usage information in Claude JSON output.
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeSystemMessage represents a system initialization message in json format (stream-json internally).
type ClaudeSystemMessage struct {
	Type              string   `json:"type"`
	Subtype           string   `json:"subtype"`
	CWD               string   `json:"cwd"`
	SessionID         string   `json:"session_id"`
	Model             string   `json:"model"`
	Tools             []string `json:"tools,omitempty"`
	Agents            []string `json:"agents,omitempty"`
	ClaudeCodeVersion string   `json:"claude_code_version,omitempty"`
	UUID              string   `json:"uuid"`
}

// ClaudeTextContent represents type="text" content block.
type ClaudeTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeThinkingContent represents type="thinking" content block.
type ClaudeThinkingContent struct {
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
}

// ClaudeToolUseContent represents type="tool_use" content block.
type ClaudeToolUseContent struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ClaudeMessage represents the message structure in assistant messages.
type ClaudeMessage struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Role    string            `json:"role"`
	Model   string            `json:"model"`
	Content []json.RawMessage `json:"content"`
	Usage   *ClaudeUsage      `json:"usage,omitempty"`
}

// ClaudeAssistantMessage represents an assistant message in json format (stream-json internally).
type ClaudeAssistantMessage struct {
	Type            string        `json:"type"`
	Message         ClaudeMessage `json:"message"`
	ParentToolUseID *string       `json:"parent_tool_use_id"`
	SessionID       string        `json:"session_id"`
	UUID            string        `json:"uuid"`
}

// ClaudeUserMessage represents a user message (e.g. tool result) in json format (stream-json internally).
type ClaudeUserMessage struct {
	Type            string        `json:"type"`
	Message         ClaudeMessage `json:"message"`
	ParentToolUseID *string       `json:"parent_tool_use_id"`
	SessionID       string        `json:"session_id"`
	UUID            string        `json:"uuid"`
}

// CodexUsage represents token usage information in Codex JSON output.
type CodexUsage struct {
	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
}

// CodexThreadStarted represents a thread.started message in Codex JSONL format.
type CodexThreadStarted struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
}

// CodexItem represents an item in Codex item.completed messages (agent_message type).
type CodexItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

// CodexCommandExecution represents command_execution item in Codex JSONL format.
type CodexCommandExecution struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Command          string `json:"command"`
	AggregatedOutput string `json:"aggregated_output"`
	ExitCode         *int   `json:"exit_code"` // null means not yet completed
	Status           string `json:"status"`
}

// CodexItemStarted represents item.started message in Codex JSONL format.
type CodexItemStarted struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item"`
}

// CodexItemCompleted represents item.completed message in Codex JSONL format.
type CodexItemCompleted struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item"`
}

// CodexTurnCompleted represents a turn.completed message in Codex JSONL format.
type CodexTurnCompleted struct {
	Type       string      `json:"type"`
	Usage      *CodexUsage `json:"usage,omitempty"`
	DurationMS int         `json:"duration_ms"`
}

// CodexJSONOutput represents the unified parsed information from Codex JSONL output.
type CodexJSONOutput struct {
	NumTurns      int         `json:"num_turns"`
	Usage         *CodexUsage `json:"usage,omitempty"`
	DurationAPIMS int         `json:"duration_api_ms"`
	Result        string      `json:"result"`
	ThreadID      string      `json:"thread_id"`
}

// OpenCodeUsage represents token usage information in OpenCode JSON output.
type OpenCodeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// OpenCodePart represents the part structure in OpenCode messages.
type OpenCodePart struct {
	ID        string             `json:"id"`
	SessionID string             `json:"sessionID"`
	MessageID string             `json:"messageID"`
	Type      string             `json:"type"`
	Text      string             `json:"text,omitempty"`
	Tool      string             `json:"tool,omitempty"`
	State     *OpenCodeToolState `json:"state,omitempty"`
	Tokens    *OpenCodeTokens    `json:"tokens,omitempty"`
	Reason    string             `json:"reason,omitempty"` // step_finish: "stop" usually means end
}

// OpenCodeTokens represents token usage information in OpenCode step_finish messages.
type OpenCodeTokens struct {
	Total     int `json:"total"`
	Input     int `json:"input"`
	Output    int `json:"output"`
	Reasoning int `json:"reasoning"`
	Cache     struct {
		Read  int `json:"read"`
		Write int `json:"write"`
	} `json:"cache"`
}

// OpenCodeToolState represents the state information for tool_use messages.
type OpenCodeToolState struct {
	Status string                 `json:"status"`
	Input  map[string]interface{} `json:"input"`
	Output string                 `json:"output"`
}

// OpenCodeStepStart represents a step_start message in OpenCode JSONL format.
type OpenCodeStepStart struct {
	Type      string       `json:"type"`
	SessionID string       `json:"sessionID"`
	Part      OpenCodePart `json:"part"`
}

// OpenCodeStepFinish represents a step_finish message in OpenCode JSONL format.
type OpenCodeStepFinish struct {
	Type      string       `json:"type"`
	SessionID string       `json:"sessionID"`
	Part      OpenCodePart `json:"part"`
}

// OpenCodeText represents a text message in OpenCode JSONL format.
type OpenCodeText struct {
	Type      string       `json:"type"`
	SessionID string       `json:"sessionID"`
	Part      OpenCodePart `json:"part"`
}

// OpenCodeToolUse represents a tool_use message in OpenCode JSONL format.
type OpenCodeToolUse struct {
	Type      string       `json:"type"`
	SessionID string       `json:"sessionID"`
	Part      OpenCodePart `json:"part"`
}

// OpenCodeJSONOutput represents the unified parsed information from OpenCode JSONL output.
type OpenCodeJSONOutput struct {
	NumTurns      int            `json:"num_turns"`
	Usage         *OpenCodeUsage `json:"usage,omitempty"`
	DurationAPIMS int            `json:"duration_api_ms"`
	Result        string         `json:"result"`
	SessionID     string         `json:"session_id"`
}

// GeminiUsage represents usage information in Gemini-CLI messages.
type GeminiUsage struct {
	InputTokens          int `json:"input_tokens"`
	OutputTokens         int `json:"output_tokens"`
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
	TotalTokens          int `json:"total_tokens"`
}

// GeminiContent represents content blocks in Gemini-CLI messages.
type GeminiContent struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ToolUseID string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	// For tool_result type
	ToolUseID2 string `json:"tool_use_id,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
	Content    string `json:"content,omitempty"`
}

// GeminiMessage represents the message object in Gemini-CLI output.
type GeminiMessage struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Role       string            `json:"role"`
	Model      string            `json:"model"`
	Content    []json.RawMessage `json:"content"`
	StopReason interface{}       `json:"stop_reason"`
	Usage      *GeminiUsage      `json:"usage,omitempty"`
}

// GeminiSystemMessage represents a system initialization message in Gemini-CLI JSONL format.
type GeminiSystemMessage struct {
	Type      string   `json:"type"`
	Subtype   string   `json:"subtype"`
	UUID      string   `json:"uuid"`
	SessionID string   `json:"session_id"`
	CWD       string   `json:"cwd"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
}

// GeminiAssistantMessage represents an assistant message in Gemini-CLI JSONL format.
type GeminiAssistantMessage struct {
	Type            string        `json:"type"`
	UUID            string        `json:"uuid"`
	SessionID       string        `json:"session_id"`
	ParentToolUseID *string       `json:"parent_tool_use_id"`
	Message         GeminiMessage `json:"message"`
}

// GeminiUserMessage represents a user message (tool result) in Gemini-CLI JSONL format.
type GeminiUserMessage struct {
	Type            string  `json:"type"`
	UUID            string  `json:"uuid"`
	SessionID       string  `json:"session_id"`
	ParentToolUseID *string `json:"parent_tool_use_id"`
	Message         struct {
		Role    string            `json:"role"`
		Content []json.RawMessage `json:"content"`
	} `json:"message"`
}

// GeminiJSONOutput represents the unified parsed information from Gemini-CLI JSONL output.
type GeminiJSONOutput struct {
	NumTurns      int          `json:"num_turns"`
	Usage         *GeminiUsage `json:"usage,omitempty"`
	DurationAPIMS int          `json:"duration_api_ms"`
	Result        string       `json:"result"`
	SessionID     string       `json:"session_id"`
}

// AgentStreamResult is the common interface for agent stream parsing results.
// Implemented by *CodexJSONOutput, *OpenCodeJSONOutput, *GeminiJSONOutput, *ClaudeJSONOutput.
type AgentStreamResult interface {
	GetNumTurns() int
}

// GetNumTurns implements AgentStreamResult for ClaudeJSONOutput.
func (r *ClaudeJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

// GetNumTurns implements AgentStreamResult for CodexJSONOutput.
func (r *CodexJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

// GetNumTurns implements AgentStreamResult for OpenCodeJSONOutput.
func (r *OpenCodeJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

// GetNumTurns implements AgentStreamResult for GeminiJSONOutput.
func (r *GeminiJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

// ParseClaudeAgentOutput parses agent output based on the output format.
// Returns the actual content (result text) and the parsed JSON result.
// For claude, json format is treated as stream-json (JSONL format).
func ParseClaudeAgentOutput(output []byte, outputFormat string) (content []byte, result *ClaudeJSONOutput, err error) {
	// Normalize output format (convert underscores to hyphens and unify stream-json to json)
	outputFormat = normalizeOutputFormat(outputFormat)

	// Default format: return output as-is
	if outputFormat == "" || outputFormat == "default" {
		return output, nil, nil
	}

	// JSON format: parse as stream JSON (JSONL format, one JSON object per line)
	if outputFormat == "json" {
		return parseClaudeStreamJSON(output)
	}

	// Unknown format: return as-is
	log.Warnf("unknown output format: %s, treating as default", outputFormat)
	return output, nil, nil
}

// parseClaudeStreamJSON parses stream JSON format where each line is a JSON object.
func parseClaudeStreamJSON(output []byte) (content []byte, result *ClaudeJSONOutput, err error) {
	var resultBuilder strings.Builder
	var lastResult *ClaudeJSONOutput

	scanner := bufio.NewScanner(bytes.NewReader(output))
	// Increase buffer size to handle long lines (1MB initial, 10MB max)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // Max token size: 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var jsonOutput ClaudeJSONOutput
		if err := json.Unmarshal([]byte(line), &jsonOutput); err != nil {
			// If line is not valid JSON, treat it as plain text
			resultBuilder.WriteString(line)
			resultBuilder.WriteString("\n")
			continue
		}

		// Accumulate result text
		if jsonOutput.Result != "" {
			resultBuilder.WriteString(jsonOutput.Result)
		}

		// Keep the latest JSON output (contains all fields including usage and duration_api_ms)
		lastResult = &jsonOutput
	}

	if err := scanner.Err(); err != nil {
		return output, nil, fmt.Errorf("failed to parse stream JSON: %w", err)
	}

	return []byte(resultBuilder.String()), lastResult, nil
}

// ParseClaudeStreamJSONRealtime parses stream JSON format in real-time, displaying messages as they arrive.
// It reads from the provided reader line by line, parses each JSON object, and displays
// system, assistant, and result messages in real-time.
// Returns the final result message and accumulated result text.
func ParseClaudeStreamJSONRealtime(reader io.Reader) (content []byte, result *ClaudeJSONOutput, err error) {
	var resultBuilder strings.Builder
	var lastResult *ClaudeJSONOutput
	var turnCount int

	scanner := bufio.NewScanner(reader)
	// Increase buffer size to handle long lines (1MB initial, 10MB max)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // Max token size: 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse as JSON to determine message type
		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			// If line is not valid JSON, treat it as plain text
			log.Debugf("stream-json: non-JSON line: %s", line)
			resultBuilder.WriteString(line)
			resultBuilder.WriteString("\n")
			fmt.Println(line)
			continue
		}

		// Parse based on message type
		switch baseMsg.Type {
		case "system":
			var sysMsg ClaudeSystemMessage
			if err := json.Unmarshal([]byte(line), &sysMsg); err == nil {
				printClaudeSystemMessage(&sysMsg)
			} else {
				log.Debugf("stream-json: failed to parse system message: %v", err)
			}
		case "assistant":
			var asstMsg ClaudeAssistantMessage
			if err := json.Unmarshal([]byte(line), &asstMsg); err == nil {
				turnCount++
				log.Debugf("stream-json: turn %d", turnCount)
				printClaudeAssistantMessage(&asstMsg, &resultBuilder)
			} else {
				log.Debugf("stream-json: failed to parse assistant message: %v", err)
			}
		case "result":
			var resultMsg ClaudeJSONOutput
			if err := json.Unmarshal([]byte(line), &resultMsg); err == nil {
				// Print result parsing process
				resultSize := len(resultMsg.Result)
				printClaudeResultParsing(&resultMsg, resultSize)
				// Merge usage information: prefer the result with more complete usage info
				if lastResult == nil {
					lastResult = &resultMsg
				} else {
					// Merge usage information if the new result has it
					if resultMsg.Usage != nil && (resultMsg.Usage.InputTokens > 0 || resultMsg.Usage.OutputTokens > 0) {
						if lastResult.Usage == nil {
							lastResult.Usage = resultMsg.Usage
						} else {
							// Use the values from the new result if they are non-zero
							if resultMsg.Usage.InputTokens > 0 {
								lastResult.Usage.InputTokens = resultMsg.Usage.InputTokens
							}
							if resultMsg.Usage.OutputTokens > 0 {
								lastResult.Usage.OutputTokens = resultMsg.Usage.OutputTokens
							}
						}
					}
					// Always update duration_api_ms with the latest value
					if resultMsg.DurationAPIMS > 0 {
						lastResult.DurationAPIMS = resultMsg.DurationAPIMS
					}
					// Update result text if present
					if resultMsg.Result != "" {
						lastResult.Result = resultMsg.Result
					}
					// Merge NumTurns: use the maximum value
					if resultMsg.NumTurns > lastResult.NumTurns {
						lastResult.NumTurns = resultMsg.NumTurns
					}
				}
				printClaudeResultMessage(&resultMsg, &resultBuilder)
			} else {
				log.Debugf("stream-json: failed to parse result message: %v", err)
			}
		case "user":
			var userMsg ClaudeUserMessage
			if err := json.Unmarshal([]byte(line), &userMsg); err == nil {
				printClaudeUserMessage([]byte(line), &userMsg)
			} else {
				log.Debugf("stream-json: failed to parse user message: %v", err)
			}
		default:
			// Unknown type, log at debug level and output as-is
			log.Debugf("stream-json: unknown message type: %s", baseMsg.Type)
			resultBuilder.WriteString(line)
			resultBuilder.WriteString("\n")
			fmt.Println(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return []byte(resultBuilder.String()), lastResult, fmt.Errorf("failed to parse stream JSON: %w", err)
	}

	return []byte(resultBuilder.String()), lastResult, nil
}

// printClaudeSystemMessage displays system initialization information.
// (Applicable to Claude Code and Gemini-CLI)
func printClaudeSystemMessage(msg *ClaudeSystemMessage) {
	fmt.Println()
	fmt.Println("🤖 System Initialization")
	fmt.Println("==========================================")
	if msg.SessionID != "" {
		fmt.Printf("**Session ID:** %s\n", msg.SessionID)
	}
	if msg.Model != "" {
		fmt.Printf("**Model:** %s\n", msg.Model)
	}
	if msg.CWD != "" {
		fmt.Printf("**Working Dir:** %s\n", msg.CWD)
	}
	if msg.ClaudeCodeVersion != "" {
		fmt.Printf("**Version:** %s\n", msg.ClaudeCodeVersion)
	}
	if len(msg.Tools) > 0 {
		fmt.Printf("**Tools:** %d\n", len(msg.Tools))
	}
	if len(msg.Agents) > 0 {
		fmt.Printf("**Agents:** %d\n", len(msg.Agents))
	}
	fmt.Println("==========================================")
	fmt.Println()
	flushStdout()
}

// parseClaudeContentBlock parses a raw content block and returns (contentType, displayText, resultText, ok).
// displayText is formatted for console output; resultText is the raw text to accumulate (for text type).
func parseClaudeContentBlock(raw json.RawMessage) (contentType, displayText, resultText string, ok bool) {
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &typeOnly); err != nil {
		return "", "", "", false
	}
	switch typeOnly.Type {
	case "text":
		var c ClaudeTextContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", "", false
		}
		return "text", truncateText(c.Text, maxDisplayBytes, maxDisplayLines), c.Text, true
	case "thinking":
		var c ClaudeThinkingContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", "", false
		}
		return "thinking", truncateText(c.Thinking, maxDisplayBytes, maxDisplayLines), "", true
	case "tool_use":
		var c ClaudeToolUseContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", "", false
		}
		var sb strings.Builder
		sb.WriteString(c.Name)
		if len(c.Input) > 0 {
			sb.WriteString(": ")
			var pairs []string
			for k, v := range c.Input {
				pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
			}
			sb.WriteString(strings.Join(pairs, ", "))
		}
		return "tool_use", truncateCommandDisplay(sb.String()), "", true
	default:
		return typeOnly.Type, fmt.Sprintf("... %d bytes ...", len(raw)), "", true
	}
}

// truncateCommandDisplay truncates long command/agent output for display.
// Shows first 128 bytes + "..." + last 32 bytes when len > 160.
func truncateCommandDisplay(s string) string {
	const headLen, tailLen = 128, 32
	if len(s) <= headLen+tailLen {
		return s
	}
	return s[:headLen] + "..." + s[len(s)-tailLen:]
}

// truncateText truncates text to maxBytes bytes and/or maxLines lines, appending "..." if truncated.
// If maxLines > 0, the text is first limited to maxLines lines, then to maxBytes.
// Returns text without trailing newline so callers can use fmt.Println without double newlines.
func truncateText(text string, maxBytes int, maxLines int) string {
	// First, limit by lines if maxLines > 0
	if maxLines > 0 {
		lines := strings.Split(text, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			text = strings.Join(lines, "\n")
		}
	}

	// Then, limit by bytes
	if len(text) <= maxBytes {
		return strings.TrimRight(text, "\n")
	}
	// Truncate to maxBytes - 3 to leave room for "..."
	truncated := text[:maxBytes-3]
	return strings.TrimRight(truncated, "\n") + "..."
}

// indentSubsequentLines prefixes each line after the first with indent, and wraps long lines (>80 chars)
// at word boundaries. Wrapped continuations are also indented.
func indentSubsequentLines(text string) string {
	const indentStr = "   "
	const maxLineWidth = 99
	contentWidth := maxLineWidth - len(indentStr) // width for indented continuation

	wrapAt := func(s string, width int) []string {
		var out []string
		for len(s) > width {
			chunk := s[:width]
			breakAt := width
			for i := width - 1; i >= 0; i-- {
				if chunk[i] == ' ' || chunk[i] == '\t' {
					breakAt = i + 1
					break
				}
			}
			out = append(out, s[:breakAt])
			s = strings.TrimLeft(s[breakAt:], " \t")
		}
		if s != "" {
			out = append(out, s)
		}
		return out
	}

	lines := strings.Split(text, "\n")
	var result []string
	for i, line := range lines {
		parts := wrapAt(line, maxLineWidth)
		for j, p := range parts {
			if i > 0 || j > 0 {
				// Indented: wrap content at contentWidth, then prefix each part
				sub := wrapAt(p, contentWidth)
				for _, s := range sub {
					result = append(result, indentStr+s)
				}
			} else {
				result = append(result, p)
			}
		}
	}
	if len(result) <= 1 {
		return text
	}
	return strings.Join(result, "\n")
}

// printClaudeAssistantMessage displays assistant message content, printing each block with type-specific icons.
// (Applicable to Claude Code and Gemini-CLI)
// Icons: 🤔 thinking, 🔧 tool_use, 🤖 text, ❓ unknown
func printClaudeAssistantMessage(msg *ClaudeAssistantMessage, resultBuilder *strings.Builder) {
	if msg.Message.Content == nil {
		return
	}

	for _, raw := range msg.Message.Content {
		contentType, displayText, resultText, ok := parseClaudeContentBlock(raw)
		if !ok {
			log.Debugf("stream-json: assistant message: content type: %s", contentType)
			continue
		}
		if displayText == "" {
			continue
		}
		var icon string
		switch contentType {
		case "text":
			icon = "🤖 "
		case "thinking":
			icon = "🤔 "
		case "tool_use":
			icon = "🔧 "
		default:
			icon = "❓ "
		}
		fmt.Print(icon)
		fmt.Println(indentSubsequentLines(displayText))
		flushStdout()
		if resultText != "" {
			resultBuilder.WriteString(resultText)
		}
	}
}

// printClaudeResultParsing displays the parsing process of a result message.
func printClaudeResultParsing(msg *ClaudeJSONOutput, resultSize int) {
	fmt.Printf("🤖 return result (%d bytes)\n", resultSize)
	flushStdout()
}

// parseClaudeUserContentType parses user message content to determine content subtype.
// Returns "tool_result" if all content items are tool_result, otherwise the first non-tool_result type.
func parseClaudeUserContentType(msg *ClaudeUserMessage) string {
	if len(msg.Message.Content) == 0 {
		return "tool_result" // default
	}
	var firstOther string
	for _, raw := range msg.Message.Content {
		var typeOnly struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeOnly); err != nil {
			continue
		}
		if typeOnly.Type != "tool_result" {
			if firstOther == "" {
				firstOther = typeOnly.Type
			}
		}
	}
	if firstOther != "" {
		return firstOther
	}
	return "tool_result"
}

// printClaudeUserMessage displays user message (e.g. tool result) with user icon.
// For tool_result: shows "... xxx bytes ...". For other types: "type: ... xxx bytes ...".
func printClaudeUserMessage(rawLine []byte, msg *ClaudeUserMessage) {
	size := len(rawLine)
	contentType := parseClaudeUserContentType(msg)
	var displayText string
	if contentType == "tool_result" {
		displayText = fmt.Sprintf("... %d bytes ...", size)
	} else {
		displayText = fmt.Sprintf("%s: ... %d bytes ...", contentType, size)
	}
	fmt.Print("💬 ")
	fmt.Println(indentSubsequentLines(displayText))
	flushStdout()
}

// printClaudeResultMessage displays the final result message.
func printClaudeResultMessage(msg *ClaudeJSONOutput, resultBuilder *strings.Builder) {
	if msg.Result != "" {
		fmt.Println()
		fmt.Println("✅ Final Result")
		fmt.Println("==========================================")
		// Print result text (may be multi-line); trim trailing empty from split to avoid extra blank
		lines := strings.Split(msg.Result, "\n")
		for len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Println("==========================================")
		resultBuilder.WriteString(msg.Result)
		flushStdout()
	}
}

// PrintAgentDiagnostics prints diagnostic information in a beautiful format.
// It accepts ClaudeJSONOutput, CodexJSONOutput, OpenCodeJSONOutput, or GeminiJSONOutput.
func PrintAgentDiagnostics(result interface{}) {
	var numTurns int
	var inputTokens, outputTokens int
	var durationAPIMS int
	hasInfo := false

	// Extract information based on type
	switch r := result.(type) {
	case *GeminiJSONOutput:
		if r == nil {
			return
		}
		if r.NumTurns > 0 {
			numTurns = r.NumTurns
			hasInfo = true
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				inputTokens = r.Usage.InputTokens
				hasInfo = true
			}
			if r.Usage.OutputTokens > 0 {
				outputTokens = r.Usage.OutputTokens
				hasInfo = true
			}
		}
		if r.DurationAPIMS > 0 {
			durationAPIMS = r.DurationAPIMS
			hasInfo = true
		}
	case *ClaudeJSONOutput:
		if r == nil {
			return
		}
		if r.NumTurns > 0 {
			numTurns = r.NumTurns
			hasInfo = true
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				inputTokens = r.Usage.InputTokens
				hasInfo = true
			}
			if r.Usage.OutputTokens > 0 {
				outputTokens = r.Usage.OutputTokens
				hasInfo = true
			}
		}
		if r.DurationAPIMS > 0 {
			durationAPIMS = r.DurationAPIMS
			hasInfo = true
		}
	case *CodexJSONOutput:
		if r == nil {
			return
		}
		if r.NumTurns > 0 {
			numTurns = r.NumTurns
			hasInfo = true
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				inputTokens = r.Usage.InputTokens
				hasInfo = true
			}
			if r.Usage.OutputTokens > 0 {
				outputTokens = r.Usage.OutputTokens
				hasInfo = true
			}
		}
		if r.DurationAPIMS > 0 {
			durationAPIMS = r.DurationAPIMS
			hasInfo = true
		}
	case *OpenCodeJSONOutput:
		if r == nil {
			return
		}
		if r.NumTurns > 0 {
			numTurns = r.NumTurns
			hasInfo = true
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				inputTokens = r.Usage.InputTokens
				hasInfo = true
			}
			if r.Usage.OutputTokens > 0 {
				outputTokens = r.Usage.OutputTokens
				hasInfo = true
			}
		}
		if r.DurationAPIMS > 0 {
			durationAPIMS = r.DurationAPIMS
			hasInfo = true
		}
	default:
		return
	}

	if !hasInfo {
		return
	}

	fmt.Println()
	fmt.Println("📊 Agent Diagnostics")
	fmt.Println("==========================================")
	if numTurns > 0 {
		fmt.Printf("**Num turns:** %d\n", numTurns)
	}
	if inputTokens > 0 {
		fmt.Printf("**Input tokens:** %d\n", inputTokens)
	}
	if outputTokens > 0 {
		fmt.Printf("**Output tokens:** %d\n", outputTokens)
	}
	if durationAPIMS > 0 {
		durationSec := float64(durationAPIMS) / 1000.0
		fmt.Printf("**API duration:** %.2f s\n", durationSec)
	}
	fmt.Println("==========================================")
	flushStdout()
}

// ParseCodexJSONLRealtime parses Codex JSONL format in real-time, displaying messages as they arrive.
// It reads from the provided reader line by line, parses each JSON object, and displays
// thread.started, item.completed (agent_message), and turn.completed messages in real-time.
// Returns the final result and accumulated result text.
func ParseCodexJSONLRealtime(reader io.Reader) (content []byte, result *CodexJSONOutput, err error) {
	var lastResult *CodexJSONOutput
	var lastAgentMessage string
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	// Increase buffer size to handle long lines (1MB initial, 10MB max)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // Max token size: 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse as JSON to determine message type
		var baseMsg map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			log.Debugf("codex-json: non-JSON lines, error: %s", err)
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			continue
		}

		// Skip JSON with only "type" and no other fields
		if len(baseMsg) <= 1 {
			log.Debugf("codex-json: skipping message with only type field")
			continue
		}

		var typeOnly struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &typeOnly); err != nil {
			continue
		}

		switch typeOnly.Type {
		case "thread.started":
			var threadMsg CodexThreadStarted
			if err := json.Unmarshal([]byte(line), &threadMsg); err == nil {
				printCodexThreadStarted(&threadMsg)
				if lastResult == nil {
					lastResult = &CodexJSONOutput{}
				}
				lastResult.ThreadID = threadMsg.ThreadID
			} else {
				log.Debugf("codex-json: failed to parse thread.started message: %v", err)
			}
		case "item.started":
			var itemMsg CodexItemStarted
			if err := json.Unmarshal([]byte(line), &itemMsg); err == nil {
				lastAgentMessage = printCodexItem(itemMsg.Item, lastResult, lastAgentMessage, false)
			} else {
				log.Debugf("codex-json: failed to parse item.started message: %v", err)
			}
		case "item.completed":
			var itemMsg CodexItemCompleted
			if err := json.Unmarshal([]byte(line), &itemMsg); err == nil {
				lastAgentMessage = printCodexItem(itemMsg.Item, lastResult, lastAgentMessage, true)
			} else {
				log.Debugf("codex-json: failed to parse item.completed message: %v", err)
			}
		case "turn.completed":
			var turnMsg CodexTurnCompleted
			if err := json.Unmarshal([]byte(line), &turnMsg); err == nil {
				if lastResult == nil {
					lastResult = &CodexJSONOutput{}
				}
				if turnMsg.Usage != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &CodexUsage{}
					}
					if turnMsg.Usage.InputTokens > 0 {
						lastResult.Usage.InputTokens = turnMsg.Usage.InputTokens
					}
					if turnMsg.Usage.OutputTokens > 0 {
						lastResult.Usage.OutputTokens = turnMsg.Usage.OutputTokens
					}
					if turnMsg.Usage.CachedInputTokens > 0 {
						lastResult.Usage.CachedInputTokens = turnMsg.Usage.CachedInputTokens
					}
				}
				if turnMsg.DurationMS > 0 {
					lastResult.DurationAPIMS = turnMsg.DurationMS
				} else {
					elapsed := time.Since(startTime)
					lastResult.DurationAPIMS = int(elapsed.Milliseconds())
				}
				printCodexTurnCompleted(turnMsg.Usage)
			} else {
				log.Debugf("codex-json: failed to parse turn.completed message: %v", err)
			}
		default:
			log.Debugf("codex-json: unknown message type: %s", typeOnly.Type)
			fmt.Printf("❓ %s: ... %d bytes ...\n", typeOnly.Type, len(line))
			flushStdout()
		}
	}

	if err := scanner.Err(); err != nil {
		return []byte(lastAgentMessage), lastResult, fmt.Errorf("failed to parse codex JSONL: %w", err)
	}

	// Only return the last agent message
	return []byte(lastAgentMessage), lastResult, nil
}

// printCodexThreadStarted displays thread initialization information.
func printCodexThreadStarted(msg *CodexThreadStarted) {
	fmt.Println()
	fmt.Println("🤖 Session Started")
	fmt.Println("==========================================")
	if msg.ThreadID != "" {
		fmt.Printf("**Thread ID:** %s\n", msg.ThreadID)
	}
	fmt.Println("==========================================")
	fmt.Println()
	flushStdout()
}

// printCodexItem displays item.started or item.completed based on item type.
// When dedup is false (item.started): show command for command_execution.
// When dedup is true (item.completed): show bytes for command_execution, full message for agent_message.
// Returns last agent message text.
func printCodexItem(itemRaw json.RawMessage, lastResult *CodexJSONOutput, lastAgentMessage string, dedup bool) string {
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(itemRaw, &typeOnly); err != nil {
		return lastAgentMessage
	}
	switch typeOnly.Type {
	case "command_execution":
		var cmd CodexCommandExecution
		if err := json.Unmarshal(itemRaw, &cmd); err != nil {
			return lastAgentMessage
		}
		if !dedup {
			fmt.Printf("🔧 %s\n", indentSubsequentLines(truncateCommandDisplay(cmd.Command)))
		} else {
			size := len(cmd.AggregatedOutput)
			icon := "💬 "
			if cmd.ExitCode != nil && *cmd.ExitCode != 0 {
				icon = "❌ "
			}
			fmt.Printf("%s... %d bytes ...\n", icon, size)
		}
		flushStdout()
	case "agent_message":
		if !dedup {
			return lastAgentMessage
		}
		var item CodexItem
		if err := json.Unmarshal(itemRaw, &item); err != nil {
			return lastAgentMessage
		}
		if lastResult != nil {
			lastResult.NumTurns++
			log.Debugf("codex-json: turn %d", lastResult.NumTurns)
		}
		printCodexAgentMessage(&item, nil)
		return item.Text
	default:
		fmt.Printf("❓ %s: ... %d bytes ...\n", typeOnly.Type, len(itemRaw))
		flushStdout()
	}
	return lastAgentMessage
}

// stripThinkTags removes <think>...</think> tags from text, returning the inner content and any text outside.
func stripThinkTags(text string) string {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	thinkStart := strings.Index(lower, "<think>")
	if thinkStart == -1 {
		return text
	}
	thinkEnd := strings.Index(lower[thinkStart:], "</think>")
	if thinkEnd == -1 {
		return text
	}
	// Extract content: before <think>, content inside, after </think>
	before := strings.TrimSpace(text[:thinkStart])
	inner := strings.TrimSpace(text[thinkStart+7 : thinkStart+thinkEnd])
	after := strings.TrimSpace(text[thinkStart+thinkEnd+8:])
	var parts []string
	if before != "" {
		parts = append(parts, before)
	}
	if inner != "" {
		parts = append(parts, inner)
	}
	if after != "" {
		parts = append(parts, after)
	}
	return strings.Join(parts, "\n\n")
}

// hasThinkTags returns true if text contains <think> </think> tags (after trim).
func hasThinkTags(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	return strings.Contains(lower, "<think>") && strings.Contains(lower, "</think>")
}

// printCodexAgentMessage displays agent message content.
// For agent_message: trim text, use 🤔 if wrapped in <think> </think>, else 🤖. Strip think tags from display.
func printCodexAgentMessage(item *CodexItem, resultBuilder *strings.Builder) {
	text := strings.TrimSpace(item.Text)
	if text == "" {
		return
	}
	displayText := stripThinkTags(text)
	if displayText == "" {
		return
	}
	displayText = truncateText(displayText, maxDisplayBytes, maxDisplayLines)
	var icon string
	if hasThinkTags(item.Text) {
		icon = "🤔 "
	} else {
		icon = "🤖 "
	}
	fmt.Print(icon)
	fmt.Println(indentSubsequentLines(displayText))
	flushStdout()
}

// printCodexTurnCompleted displays turn.completed usage when values are non-zero.
func printCodexTurnCompleted(usage *CodexUsage) {
	if usage == nil {
		return
	}
	var parts []string
	if usage.InputTokens > 0 {
		parts = append(parts, fmt.Sprintf("input_tokens=%d", usage.InputTokens))
	}
	if usage.OutputTokens > 0 {
		parts = append(parts, fmt.Sprintf("output_tokens=%d", usage.OutputTokens))
	}
	if usage.CachedInputTokens > 0 {
		parts = append(parts, fmt.Sprintf("cached_input_tokens=%d", usage.CachedInputTokens))
	}
	if len(parts) > 0 {
		fmt.Printf("📊 %s\n", strings.Join(parts, ", "))
		flushStdout()
	}
}

// ParseOpenCodeJSONLRealtime parses OpenCode JSONL format in real-time, displaying messages as they arrive.
// It reads from the provided reader line by line, parses each JSON object, and displays
// step_start, text, tool_use, and step_finish messages in real-time.
// Returns the final result and accumulated result text.
func ParseOpenCodeJSONLRealtime(reader io.Reader) (content []byte, result *OpenCodeJSONOutput, err error) {
	var resultBuilder strings.Builder
	var lastResult *OpenCodeJSONOutput
	var inStep bool
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	// Increase buffer size to handle long lines (1MB initial, 10MB max)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // Max token size: 10MB
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse as JSON to determine message type
		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			// If line is not valid JSON, log at debug level only
			log.Debugf("opencode-json: non-JSON lines, error: %s", err)
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			continue
		}

		// Parse based on message type
		switch baseMsg.Type {
		case "step_start":
			var stepMsg OpenCodeStepStart
			if err := json.Unmarshal([]byte(line), &stepMsg); err == nil {
				// Initialize result if needed
				if lastResult == nil {
					lastResult = &OpenCodeJSONOutput{}
				}
				lastResult.SessionID = stepMsg.SessionID
				// Increment NumTurns when step starts
				lastResult.NumTurns++
				inStep = true
				log.Debugf("opencode-json: turn %d", lastResult.NumTurns)
			} else {
				log.Debugf("opencode-json: failed to parse step_start message: %v", err)
			}
		case "step_finish":
			var stepMsg OpenCodeStepFinish
			if err := json.Unmarshal([]byte(line), &stepMsg); err == nil {
				if lastResult == nil {
					lastResult = &OpenCodeJSONOutput{}
				}
				// Extract token usage information
				if stepMsg.Part.Tokens != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &OpenCodeUsage{}
					}
					// Use total tokens as input tokens, output tokens from the tokens structure
					if stepMsg.Part.Tokens.Total > 0 {
						// For opencode, we use total as a reference, but extract input/output separately
						lastResult.Usage.InputTokens = stepMsg.Part.Tokens.Input
						lastResult.Usage.OutputTokens = stepMsg.Part.Tokens.Output
					}
				}
				// Calculate duration if not provided
				elapsed := time.Since(startTime)
				lastResult.DurationAPIMS = int(elapsed.Milliseconds())
				inStep = false
				// When Reason is "stop", it usually means the session has ended
				if stepMsg.Part.Reason == "stop" {
					fmt.Println("✅ Step complete (reason: stop)")
					flushStdout()
				}
				log.Debugf("opencode-json: received step_finish (reason: %s)", stepMsg.Part.Reason)
			} else {
				log.Debugf("opencode-json: failed to parse step_finish message: %v", err)
			}
		case "text":
			if !inStep {
				log.Debugf("opencode-json: received text message outside of step (suppressed)")
				continue
			}
			var textMsg OpenCodeText
			if err := json.Unmarshal([]byte(line), &textMsg); err == nil {
				printOpenCodeText(&textMsg, &resultBuilder)
			} else {
				log.Debugf("opencode-json: failed to parse text message: %v", err)
			}
		case "tool_use":
			if !inStep {
				log.Debugf("opencode-json: received tool_use message outside of step (suppressed)")
				continue
			}
			var toolMsg OpenCodeToolUse
			if err := json.Unmarshal([]byte(line), &toolMsg); err == nil {
				printOpenCodeToolUse(&toolMsg, &resultBuilder)
			} else {
				log.Debugf("opencode-json: failed to parse tool_use message: %v", err)
			}
		default:
			// Unknown type, only display if in step
			if inStep {
				log.Debugf("opencode-json: unknown message type: %s", baseMsg.Type)
				resultBuilder.WriteString(line)
				resultBuilder.WriteString("\n")
				fmt.Println(line)
			} else {
				log.Debugf("opencode-json: unknown message type outside of step: %s (suppressed)", baseMsg.Type)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return []byte(resultBuilder.String()), lastResult, fmt.Errorf("failed to parse opencode JSONL: %w", err)
	}

	return []byte(resultBuilder.String()), lastResult, nil
}

// printOpenCodeText displays text message content.
func printOpenCodeText(msg *OpenCodeText, resultBuilder *strings.Builder) {
	if msg.Part.Text != "" {
		// Truncate text to 4KB and 10 lines for display
		displayText := truncateText(msg.Part.Text, maxDisplayBytes, maxDisplayLines)
		// Print agent marker with robot emoji at the beginning of agent output
		fmt.Print("🤖 ")
		fmt.Println(indentSubsequentLines(displayText))
		flushStdout()
		resultBuilder.WriteString(msg.Part.Text)
	}
}

// maxInputValueLen is the max length for each input value when displaying (truncate if longer).
const maxInputValueLen = 100

// printOpenCodeToolUse displays tool use message content.
func printOpenCodeToolUse(msg *OpenCodeToolUse, resultBuilder *strings.Builder) {
	if msg.Part.State == nil {
		return
	}

	toolType := msg.Part.Tool
	if toolType == "" {
		toolType = "unknown"
	}

	// Format input as key=value pairs (generalized dict), truncate long values
	var inputParts []string
	if msg.Part.State.Input != nil {
		for k, v := range msg.Part.State.Input {
			valStr := fmt.Sprintf("%v", v)
			if len(valStr) > maxInputValueLen {
				valStr = valStr[:maxInputValueLen-3] + "..."
			}
			inputParts = append(inputParts, fmt.Sprintf("%s=%s", k, valStr))
		}
	}
	// Sort for deterministic output
	if len(inputParts) > 1 {
		// Keep simple order: prefer common keys first
		sort.Slice(inputParts, func(i, j int) bool { return inputParts[i] < inputParts[j] })
	}

	var displayLine string
	if len(inputParts) > 0 {
		displayLine = toolType + ": " + strings.Join(inputParts, ", ")
	} else {
		displayLine = toolType
	}
	fmt.Printf("🔧 %s\n", indentSubsequentLines(truncateCommandDisplay(displayLine)))
	resultBuilder.WriteString(displayLine + "\n")

	// Display output as size only
	if msg.Part.State.Output != "" {
		fmt.Printf("💬 ... %d bytes ...\n", len(msg.Part.State.Output))
		resultBuilder.WriteString(msg.Part.State.Output)
	}
	flushStdout()
}

// printGeminiAssistantMessage displays assistant message content with type-specific icons.
// Uses same content format as Claude (text, thinking, tool_use). Icons: 🤔 thinking, 🔧 tool_use, 🤖 text, ❓ unknown.
func printGeminiAssistantMessage(msg *GeminiAssistantMessage, resultBuilder *strings.Builder) {
	if len(msg.Message.Content) == 0 {
		return
	}
	for _, raw := range msg.Message.Content {
		contentType, displayText, resultText, ok := parseClaudeContentBlock(raw)
		if !ok {
			log.Debugf("gemini-json: assistant message: content type: %s", contentType)
			continue
		}
		if displayText == "" {
			continue
		}
		var icon string
		switch contentType {
		case "text":
			icon = "🤖 "
		case "thinking":
			icon = "🤔 "
		case "tool_use":
			icon = "🔧 "
		default:
			icon = "❓ "
		}
		fmt.Print(icon)
		fmt.Println(indentSubsequentLines(displayText))
		flushStdout()
		if resultText != "" {
			resultBuilder.WriteString(resultText)
		}
	}
}

// parseGeminiUserContentType parses user message content to determine content subtype.
func parseGeminiUserContentType(msg *GeminiUserMessage) string {
	if len(msg.Message.Content) == 0 {
		return "tool_result"
	}
	var firstOther string
	for _, raw := range msg.Message.Content {
		var typeOnly struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeOnly); err != nil {
			continue
		}
		if typeOnly.Type != "tool_result" {
			if firstOther == "" {
				firstOther = typeOnly.Type
			}
		}
	}
	if firstOther != "" {
		return firstOther
	}
	return "tool_result"
}

// printGeminiUserMessage displays user message (e.g. tool result) with conversation icon.
func printGeminiUserMessage(rawLine []byte, msg *GeminiUserMessage) {
	size := len(rawLine)
	contentType := parseGeminiUserContentType(msg)
	var displayText string
	if contentType == "tool_result" {
		displayText = fmt.Sprintf("tool_result: ... %d bytes ...", size)
	} else {
		displayText = fmt.Sprintf("%s: ... %d bytes ...", contentType, size)
	}
	fmt.Print("💬 ")
	fmt.Println(indentSubsequentLines(displayText))
	flushStdout()
}

// ParseGeminiJSONLRealtime parses Gemini-CLI JSONL output in real-time from an io.Reader.
// It displays messages as they arrive and returns the final parsed result.
func ParseGeminiJSONLRealtime(reader io.Reader) (content []byte, result *GeminiJSONOutput, err error) {
	var lastResult *GeminiJSONOutput
	var lastAssistantText string
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	// Increase buffer size to handle long lines (1MB initial, 10MB max)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024) // Max token size: 10MB

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to parse as JSON to determine message type
		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			// If line is not valid JSON, treat it as plain text
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			log.Debugf("gemini-json: non-JSON line: %s", line)
			continue
		}

		// Parse based on message type
		switch baseMsg.Type {
		case "system":
			var sysMsg GeminiSystemMessage
			if err := json.Unmarshal([]byte(line), &sysMsg); err == nil {
				if sysMsg.Subtype == "init" {
					// Print session info (similar to Claude's printSystemMessage)
					fmt.Println()
					fmt.Println("🚀 Session Initialized")
					fmt.Println("==========================================")
					if sysMsg.Model != "" {
						fmt.Printf("**Model:** %s\n", sysMsg.Model)
					}
					if sysMsg.SessionID != "" {
						fmt.Printf("**Session ID:** %s\n", sysMsg.SessionID)
					}
					if sysMsg.CWD != "" {
						fmt.Printf("**Working Directory:** %s\n", sysMsg.CWD)
					}
					if len(sysMsg.Tools) > 0 {
						fmt.Printf("**Tools:** %s\n", strings.Join(sysMsg.Tools, ", "))
					}
					fmt.Println("==========================================")
					fmt.Println()
					flushStdout()

					// Initialize result
					if lastResult == nil {
						lastResult = &GeminiJSONOutput{
							SessionID: sysMsg.SessionID,
						}
					}
				}
			} else {
				log.Debugf("gemini-json: failed to parse system message: %v", err)
			}
		case "assistant":
			var asstMsg GeminiAssistantMessage
			if err := json.Unmarshal([]byte(line), &asstMsg); err == nil {
				// Increment NumTurns
				if lastResult == nil {
					lastResult = &GeminiJSONOutput{
						SessionID: asstMsg.SessionID,
					}
				}
				lastResult.NumTurns++
				log.Debugf("gemini-json: turn %d", lastResult.NumTurns)

				// Display assistant content with type-specific icons (same as Claude)
				var assistantText strings.Builder
				printGeminiAssistantMessage(&asstMsg, &assistantText)
				lastAssistantText = assistantText.String()

				// Extract usage from message.usage and merge into lastResult.Usage
				if asstMsg.Message.Usage != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &GeminiUsage{}
					}
					// Merge usage information (keep maximum values to ensure completeness)
					if asstMsg.Message.Usage.InputTokens > 0 {
						lastResult.Usage.InputTokens += asstMsg.Message.Usage.InputTokens
					}
					if asstMsg.Message.Usage.OutputTokens > 0 {
						lastResult.Usage.OutputTokens += asstMsg.Message.Usage.OutputTokens
					}
					if asstMsg.Message.Usage.TotalTokens > 0 {
						lastResult.Usage.TotalTokens += asstMsg.Message.Usage.TotalTokens
					}
				}
			} else {
				log.Debugf("gemini-json: failed to parse assistant message: %v", err)
			}
		case "user":
			var userMsg GeminiUserMessage
			if err := json.Unmarshal([]byte(line), &userMsg); err == nil {
				printGeminiUserMessage([]byte(line), &userMsg)
			} else {
				log.Debugf("gemini-json: failed to parse user message: %v", err)
			}
		default:
			log.Debugf("gemini-json: unknown message type: %s", baseMsg.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse gemini JSONL: %w", err)
	}

	// Calculate DurationAPIMS from elapsed time
	if lastResult != nil {
		elapsed := time.Since(startTime)
		lastResult.DurationAPIMS = int(elapsed.Milliseconds())
		lastResult.Result = lastAssistantText
	}

	return []byte(lastAssistantText), lastResult, nil
}
