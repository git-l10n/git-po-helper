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
	const headLen, tailLen = 256, 128
	if len(s) <= headLen+tailLen {
		return s
	}
	return s[:headLen] + " [...] " + s[len(s)-tailLen:]
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

// GetAgentDiagnostics copies diagnostic fields from streamResult into result. No printing.
func GetAgentDiagnostics(result *AgentRunResult, streamResult AgentStreamResult) {
	if result == nil || streamResult == nil {
		return
	}
	if n := streamResult.GetNumTurns(); n > 0 {
		result.NumTurns = n
	}
	switch r := streamResult.(type) {
	case *GeminiJSONOutput:
		if r == nil {
			return
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				result.AgentInputTokens = r.Usage.InputTokens
			}
			if r.Usage.OutputTokens > 0 {
				result.AgentOutputTokens = r.Usage.OutputTokens
			}
		}
		if r.DurationAPIMS > 0 {
			result.AgentDurationAPIMS = r.DurationAPIMS
		}
	case *ClaudeJSONOutput:
		if r == nil {
			return
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				result.AgentInputTokens = r.Usage.InputTokens
			}
			if r.Usage.OutputTokens > 0 {
				result.AgentOutputTokens = r.Usage.OutputTokens
			}
		}
		if r.DurationAPIMS > 0 {
			result.AgentDurationAPIMS = r.DurationAPIMS
		}
	case *CodexJSONOutput:
		if r == nil {
			return
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				result.AgentInputTokens = r.Usage.InputTokens
			}
			if r.Usage.OutputTokens > 0 {
				result.AgentOutputTokens = r.Usage.OutputTokens
			}
		}
		if r.DurationAPIMS > 0 {
			result.AgentDurationAPIMS = r.DurationAPIMS
		}
	case *OpenCodeJSONOutput:
		if r == nil {
			return
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				result.AgentInputTokens = r.Usage.InputTokens
			}
			if r.Usage.OutputTokens > 0 {
				result.AgentOutputTokens = r.Usage.OutputTokens
			}
		}
		if r.DurationAPIMS > 0 {
			result.AgentDurationAPIMS = r.DurationAPIMS
		}
	case *QoderJSONOutput:
		if r == nil {
			return
		}
		if r.Usage != nil {
			if r.Usage.InputTokens > 0 {
				result.AgentInputTokens = r.Usage.InputTokens
			}
			if r.Usage.OutputTokens > 0 {
				result.AgentOutputTokens = r.Usage.OutputTokens
			}
		}
		if r.DurationAPIMS > 0 {
			result.AgentDurationAPIMS = r.DurationAPIMS
		}
	default:
		return
	}
}

// PrintAgentDiagnosticsFromResult prints diagnostics from AgentRunResult (fields set by GetAgentDiagnostics).
// Format aligns with PrintReviewReportResult (ReviewStatLabelWidth, two-space indent, label: value).
func PrintAgentDiagnosticsFromResult(result *AgentRunResult) {
	if result == nil {
		return
	}
	hasInfo := result.NumTurns > 0 || result.AgentInputTokens > 0 || result.AgentOutputTokens > 0 || result.AgentDurationAPIMS > 0
	if !hasInfo {
		return
	}
	w := ReviewStatLabelWidth
	fmt.Println()
	fmt.Println("📊 Agent Diagnostics")
	fmt.Println()
	if result.NumTurns > 0 {
		fmt.Printf("  %-*s %d\n", w, "Num turns:", result.NumTurns)
	}
	if result.AgentInputTokens > 0 {
		fmt.Printf("  %-*s %d\n", w, "Input tokens:", result.AgentInputTokens)
	}
	if result.AgentOutputTokens > 0 {
		fmt.Printf("  %-*s %d\n", w, "Output tokens:", result.AgentOutputTokens)
	}
	if result.AgentDurationAPIMS > 0 {
		durationSec := float64(result.AgentDurationAPIMS) / 1000.0
		fmt.Printf("  %-*s %.2f s\n", w, "API duration:", durationSec)
	}
	fmt.Println()
	flushStdout()
}
