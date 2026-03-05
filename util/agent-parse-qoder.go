package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
)

// qoderAgent implements config.Agent for Qoder CLI.
type qoderAgent struct {
	cmd          []string
	outputFormat *struct {
		format   string
		explicit bool
	}
}

func (a *qoderAgent) getExplicitOutputFormat(flagNames ...string) (format string, found bool) {
	defaultFormat := config.OutputStreamJSON
	if a.outputFormat != nil {
		return a.outputFormat.format, a.outputFormat.explicit
	}
	format, found = parseFormatFromCmd(a.cmd, defaultFormat, flagNames...)
	a.outputFormat = &struct {
		format   string
		explicit bool
	}{format, found}
	return format, found
}

// BuildCommand returns the full command with format params added if missing.
func (a *qoderAgent) BuildCommand(vars map[string]string) ([]string, error) {
	cmd, err := replacePlaceholdersInCmd(a.cmd, vars)
	if err != nil {
		return nil, err
	}
	if format, explicit := a.getExplicitOutputFormat("--output-format", "-f"); !explicit {
		cmd = append(cmd, "--output-format", format)
	}
	return cmd, nil
}

// GetOutputFormat parses format from cmd or returns stream-json.
func (a *qoderAgent) GetOutputFormat() string {
	format, _ := a.getExplicitOutputFormat("--output-format", "-f")
	return format
}

// QoderUsage represents usage information in Qoder JSON output.
type QoderUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
}

// QoderSystemMessage represents a system initialization message in Qoder JSONL format.
type QoderSystemMessage struct {
	Type           string   `json:"type"`
	Subtype        string   `json:"subtype"`
	Provider       string   `json:"provider"`
	SessionID      string   `json:"session_id"`
	WorkingDir     string   `json:"working_dir"`
	Model          string   `json:"model"`
	Tools          []string `json:"tools"`
	PermissionMode string   `json:"permission_mode,omitempty"`
}

// QoderContent represents content blocks in Qoder messages.
type QoderContent struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     string `json:"input,omitempty"` // JSON string for function type
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// QoderMessage represents the message object in Qoder output.
type QoderMessage struct {
	ID      string            `json:"id"`
	Role    string            `json:"role"`
	Content []json.RawMessage `json:"content"`
	Usage   *QoderUsage       `json:"usage,omitempty"`
}

// QoderAssistantMessage represents an assistant message in Qoder JSONL format.
type QoderAssistantMessage struct {
	Type      string       `json:"type"`
	Subtype   string       `json:"subtype"`
	SessionID string       `json:"session_id"`
	Message   QoderMessage `json:"message"`
}

// QoderUserMessage represents a user message (tool result) in Qoder JSONL format.
type QoderUserMessage struct {
	Type      string       `json:"type"`
	Subtype   string       `json:"subtype"`
	SessionID string       `json:"session_id"`
	Message   QoderMessage `json:"message"`
}

// QoderResultMessage represents a result message in Qoder JSONL format.
type QoderResultMessage struct {
	Type      string       `json:"type"`
	Subtype   string       `json:"subtype"`
	SessionID string       `json:"session_id"`
	Done      bool         `json:"done"`
	Message   QoderMessage `json:"message"`
}

// QoderJSONOutput represents the unified parsed information from Qoder JSONL output.
type QoderJSONOutput struct {
	NumTurns      int         `json:"num_turns"`
	Usage         *QoderUsage `json:"usage,omitempty"`
	DurationAPIMS int         `json:"duration_api_ms"`
	Result        string      `json:"result"`
	SessionID     string      `json:"session_id"`
}

// GetNumTurns implements AgentStreamResult for QoderJSONOutput.
func (r *QoderJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

// parseQoderContentBlock parses a content block from Qoder assistant message.
func parseQoderContentBlock(raw json.RawMessage) (contentType, displayText, resultText string, ok bool) {
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &typeOnly); err != nil {
		return "", "", "", false
	}
	switch typeOnly.Type {
	case "text":
		var c struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", "", false
		}
		return "text", truncateText(c.Text, maxDisplayBytes, maxDisplayLines), c.Text, true
	case "function":
		var c struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input string `json:"input"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			return "", "", "", false
		}
		display := c.Name
		if c.Input != "" {
			display += ": " + truncateCommandDisplay(c.Input)
		}
		return "tool_use", display, "", true
	case "finish":
		return "finish", "", "", true
	default:
		return typeOnly.Type, fmt.Sprintf("... %d bytes ...", len(raw)), "", true
	}
}

func printQoderAssistantMessage(msg *QoderAssistantMessage, resultBuilder *strings.Builder) {
	if len(msg.Message.Content) == 0 {
		return
	}
	for _, raw := range msg.Message.Content {
		contentType, displayText, resultText, ok := parseQoderContentBlock(raw)
		if !ok {
			log.Debugf("qoder-json: assistant message: content type: %s", contentType)
			continue
		}
		if contentType == "finish" {
			continue
		}
		if displayText == "" {
			continue
		}
		var icon string
		switch contentType {
		case "text":
			icon = "🤖 "
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

func printQoderUserMessage(rawLine []byte, msg *QoderUserMessage) {
	size := len(rawLine)
	displayText := fmt.Sprintf("tool_result: ... %d bytes ...", size)
	fmt.Print("💬 ")
	fmt.Println(indentSubsequentLines(displayText))
	flushStdout()
}

// ParseQoderJSONLRealtime parses Qoder JSONL output in real-time.
func ParseQoderJSONLRealtime(reader io.Reader) (content []byte, result *QoderJSONOutput, err error) {
	var lastResult *QoderJSONOutput
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
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		}
		if err := json.Unmarshal([]byte(line), &baseMsg); err != nil {
			fmt.Print("❓ ")
			fmt.Println(indentSubsequentLines(line))
			log.Debugf("qoder-json: non-JSON line: %s", line)
			continue
		}

		switch baseMsg.Type {
		case "system":
			var sysMsg QoderSystemMessage
			if err := json.Unmarshal([]byte(line), &sysMsg); err == nil {
				if sysMsg.Subtype == "init" {
					fmt.Println()
					fmt.Println("🚀 Qoder Session Initialized")
					fmt.Println("==========================================")
					if sysMsg.Provider != "" {
						fmt.Printf("**Provider:** %s\n", sysMsg.Provider)
					}
					if sysMsg.Model != "" {
						fmt.Printf("**Model:** %s\n", sysMsg.Model)
					}
					if sysMsg.SessionID != "" {
						fmt.Printf("**Session ID:** %s\n", sysMsg.SessionID)
					}
					if sysMsg.WorkingDir != "" {
						fmt.Printf("**Working Directory:** %s\n", sysMsg.WorkingDir)
					}
					if len(sysMsg.Tools) > 0 {
						fmt.Printf("**Tools:** %s\n", strings.Join(sysMsg.Tools, ", "))
					}
					fmt.Println("==========================================")
					fmt.Println()
					flushStdout()
				}
				if lastResult == nil {
					lastResult = &QoderJSONOutput{SessionID: sysMsg.SessionID}
				}
			} else {
				log.Debugf("qoder-json: failed to parse system message: %v", err)
			}
		case "assistant":
			var asstMsg QoderAssistantMessage
			if err := json.Unmarshal([]byte(line), &asstMsg); err == nil {
				if lastResult == nil {
					lastResult = &QoderJSONOutput{SessionID: asstMsg.SessionID}
				}
				lastResult.NumTurns++
				log.Debugf("qoder-json: turn %d", lastResult.NumTurns)

				var assistantText strings.Builder
				printQoderAssistantMessage(&asstMsg, &assistantText)
				lastAssistantText = assistantText.String()

				if asstMsg.Message.Usage != nil {
					if lastResult.Usage == nil {
						lastResult.Usage = &QoderUsage{}
					}
					lastResult.Usage.InputTokens += asstMsg.Message.Usage.InputTokens
					lastResult.Usage.OutputTokens += asstMsg.Message.Usage.OutputTokens
				}
			} else {
				log.Debugf("qoder-json: failed to parse assistant message: %v", err)
			}
		case "user":
			var userMsg QoderUserMessage
			if err := json.Unmarshal([]byte(line), &userMsg); err == nil {
				printQoderUserMessage([]byte(line), &userMsg)
			} else {
				log.Debugf("qoder-json: failed to parse user message: %v", err)
			}
		case "result":
			if baseMsg.Subtype == "success" {
				var resultMsg QoderResultMessage
				if err := json.Unmarshal([]byte(line), &resultMsg); err == nil {
					for _, raw := range resultMsg.Message.Content {
						var c struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}
						if err := json.Unmarshal(raw, &c); err == nil && c.Type == "text" && c.Text != "" {
							fmt.Println()
							fmt.Println("✅ Final Result")
							fmt.Println("==========================================")
							lines := strings.Split(c.Text, "\n")
							for len(lines) > 0 && lines[len(lines)-1] == "" {
								lines = lines[:len(lines)-1]
							}
							for _, ln := range lines {
								fmt.Println(ln)
							}
							fmt.Println("==========================================")
							lastAssistantText = c.Text
							if lastResult != nil {
								lastResult.Result = c.Text
							}
							flushStdout()
						}
					}
				}
			}
		default:
			log.Debugf("qoder-json: unknown message type: %s", baseMsg.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to parse qoder JSONL: %w", err)
	}

	if lastResult != nil {
		elapsed := time.Since(startTime)
		lastResult.DurationAPIMS = int(elapsed.Milliseconds())
		lastResult.Result = lastAssistantText
	}

	return []byte(lastAssistantText), lastResult, nil
}
