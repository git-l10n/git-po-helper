// Package util provides agent JSONL parsing and display utilities.
package util

import (
	"fmt"
	"strings"
)

const (
	// maxDisplayBytes is the maximum number of bytes to display for agent messages (4KB).
	maxDisplayBytes = 4096
	// maxDisplayLines is the maximum number of lines to display for agent messages.
	maxDisplayLines = 10
)

// AgentStreamResult is the common interface for agent stream parsing results.
// Implemented by *CodexJSONOutput, *OpenCodeJSONOutput, *GeminiJSONOutput, *ClaudeJSONOutput.
type AgentStreamResult interface {
	GetNumTurns() int
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
