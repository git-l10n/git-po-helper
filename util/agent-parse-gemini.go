package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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

// GetNumTurns implements AgentStreamResult for GeminiJSONOutput.
func (r *GeminiJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
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
