package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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

// GetNumTurns implements AgentStreamResult for OpenCodeJSONOutput.
func (r *OpenCodeJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
}

const maxInputValueLen = 100

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
