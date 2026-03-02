package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
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

// GetNumTurns implements AgentStreamResult for ClaudeJSONOutput.
func (r *ClaudeJSONOutput) GetNumTurns() int {
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

// parseClaudeContentBlock is used by both Claude and Gemini assistant message display.
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
