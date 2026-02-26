// Package util provides agent JSONL parsing and display utilities.
package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// maxDisplayBytes is the maximum number of bytes to display for agent messages (4KB).
	maxDisplayBytes = 4096
	// maxDisplayLines is the maximum number of lines to display for agent messages.
	maxDisplayLines = 10
)

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
func ParseClaudeStreamJSONRealtime(reader io.Reader) (content []byte, result *ClaudeJSONOutput, err error) {
	var resultBuilder strings.Builder
	var lastResult *ClaudeJSONOutput
	var turnCount int

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			log.Debugf("stream-json: non-JSON line: %s", line)
			resultBuilder.WriteString(line)
			resultBuilder.WriteString("\n")
			fmt.Println(line)
			continue
		}

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
				resultSize := len(resultMsg.Result)
				printClaudeResultParsing(&resultMsg, resultSize)
				if lastResult == nil {
					lastResult = &resultMsg
				} else {
					if resultMsg.Usage != nil && (resultMsg.Usage.InputTokens > 0 || resultMsg.Usage.OutputTokens > 0) {
						if lastResult.Usage == nil {
							lastResult.Usage = resultMsg.Usage
						} else {
							if resultMsg.Usage.InputTokens > 0 {
								lastResult.Usage.InputTokens = resultMsg.Usage.InputTokens
							}
							if resultMsg.Usage.OutputTokens > 0 {
								lastResult.Usage.OutputTokens = resultMsg.Usage.OutputTokens
							}
						}
					}
					if resultMsg.DurationAPIMS > 0 {
						lastResult.DurationAPIMS = resultMsg.DurationAPIMS
					}
					if resultMsg.Result != "" {
						lastResult.Result = resultMsg.Result
					}
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

func truncateCommandDisplay(s string) string {
	const headLen, tailLen = 128, 32
	if len(s) <= headLen+tailLen {
		return s
	}
	return s[:headLen] + "..." + s[len(s)-tailLen:]
}

func truncateText(text string, maxBytes int, maxLines int) string {
	if maxLines > 0 {
		lines := strings.Split(text, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			text = strings.Join(lines, "\n")
		}
	}
	if len(text) <= maxBytes {
		return strings.TrimRight(text, "\n")
	}
	truncated := text[:maxBytes-3]
	return strings.TrimRight(truncated, "\n") + "..."
}

func indentSubsequentLines(text string) string {
	const indentStr = "   "
	const maxLineWidth = 99
	contentWidth := maxLineWidth - len(indentStr)

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

func printClaudeResultParsing(msg *ClaudeJSONOutput, resultSize int) {
	fmt.Printf("🤖 return result (%d bytes)\n", resultSize)
	flushStdout()
}

func parseClaudeUserContentType(msg *ClaudeUserMessage) string {
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

func printClaudeResultMessage(msg *ClaudeJSONOutput, resultBuilder *strings.Builder) {
	if msg.Result != "" {
		fmt.Println()
		fmt.Println("✅ Final Result")
		fmt.Println("==========================================")
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
func PrintAgentDiagnostics(result interface{}) {
	var numTurns int
	var inputTokens, outputTokens int
	var durationAPIMS int
	hasInfo := false

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

// ParseCodexJSONLRealtime parses Codex JSONL format in real-time.
func ParseCodexJSONLRealtime(reader io.Reader) (content []byte, result *CodexJSONOutput, err error) {
	var lastResult *CodexJSONOutput
	var lastAgentMessage string
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var baseMsg map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			log.Debugf("codex-json: non-JSON lines, error: %s", err)
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			continue
		}

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

	return []byte(lastAgentMessage), lastResult, nil
}

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

func hasThinkTags(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	return strings.Contains(lower, "<think>") && strings.Contains(lower, "</think>")
}

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

// ParseOpenCodeJSONLRealtime parses OpenCode JSONL format in real-time.
func ParseOpenCodeJSONLRealtime(reader io.Reader) (content []byte, result *OpenCodeJSONOutput, err error) {
	var resultBuilder strings.Builder
	var lastResult *OpenCodeJSONOutput
	var inStep bool
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			log.Debugf("opencode-json: non-JSON lines, error: %s", err)
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			continue
		}

		switch baseMsg.Type {
		case "step_start":
			var stepMsg OpenCodeStepStart
			if err := json.Unmarshal([]byte(line), &stepMsg); err == nil {
				if lastResult == nil {
					lastResult = &OpenCodeJSONOutput{}
				}
				lastResult.SessionID = stepMsg.SessionID
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
				if stepMsg.Part.Tokens != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &OpenCodeUsage{}
					}
					if stepMsg.Part.Tokens.Total > 0 {
						lastResult.Usage.InputTokens = stepMsg.Part.Tokens.Input
						lastResult.Usage.OutputTokens = stepMsg.Part.Tokens.Output
					}
				}
				elapsed := time.Since(startTime)
				lastResult.DurationAPIMS = int(elapsed.Milliseconds())
				inStep = false
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

func printOpenCodeText(msg *OpenCodeText, resultBuilder *strings.Builder) {
	if msg.Part.Text != "" {
		displayText := truncateText(msg.Part.Text, maxDisplayBytes, maxDisplayLines)
		fmt.Print("🤖 ")
		fmt.Println(indentSubsequentLines(displayText))
		flushStdout()
		resultBuilder.WriteString(msg.Part.Text)
	}
}

const maxInputValueLen = 100

func printOpenCodeToolUse(msg *OpenCodeToolUse, resultBuilder *strings.Builder) {
	if msg.Part.State == nil {
		return
	}

	toolType := msg.Part.Tool
	if toolType == "" {
		toolType = "unknown"
	}

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
	if len(inputParts) > 1 {
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

	if msg.Part.State.Output != "" {
		fmt.Printf("💬 ... %d bytes ...\n", len(msg.Part.State.Output))
		resultBuilder.WriteString(msg.Part.State.Output)
	}
	flushStdout()
}

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

// ParseGeminiJSONLRealtime parses Gemini-CLI JSONL output in real-time.
func ParseGeminiJSONLRealtime(reader io.Reader) (content []byte, result *GeminiJSONOutput, err error) {
	var lastResult *GeminiJSONOutput
	var lastAssistantText string
	startTime := time.Now()

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			log.Debugf("gemini-json: non-JSON line: %s", line)
			continue
		}

		switch baseMsg.Type {
		case "system":
			var sysMsg GeminiSystemMessage
			if err := json.Unmarshal([]byte(line), &sysMsg); err == nil {
				if sysMsg.Subtype == "init" {
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
				if lastResult == nil {
					lastResult = &GeminiJSONOutput{
						SessionID: asstMsg.SessionID,
					}
				}
				lastResult.NumTurns++
				log.Debugf("gemini-json: turn %d", lastResult.NumTurns)

				var assistantText strings.Builder
				printGeminiAssistantMessage(&asstMsg, &assistantText)
				lastAssistantText = assistantText.String()

				if asstMsg.Message.Usage != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &GeminiUsage{}
					}
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

	if lastResult != nil {
		elapsed := time.Since(startTime)
		lastResult.DurationAPIMS = int(elapsed.Milliseconds())
		lastResult.Result = lastAssistantText
	}

	return []byte(lastAssistantText), lastResult, nil
}
