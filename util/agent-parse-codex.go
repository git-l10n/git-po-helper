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

// GetNumTurns implements AgentStreamResult for CodexJSONOutput.
func (r *CodexJSONOutput) GetNumTurns() int {
	if r == nil {
		return 0
	}
	return r.NumTurns
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
