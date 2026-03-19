// Package util provides business logic for agent-run parse-log command.
package util

import (
	"bytes"
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
)

// CmdAgentRunParseLog parses an agent JSONL log file and displays formatted output.
// Auto-detects format via detectAgentOutputFormat (same heuristics as streaming agent output)
// and parses with parseStreamByKind. Unrecognized JSONL defaults to Gemini/Qwen parser
// to match previous behavior.
// Each line in the file should be a JSON object. Supports system, assistant (with text,
// thinking, tool_use content types), user (tool_result), and result messages.
func CmdAgentRunParseLog(logFile string) error {
	raw, err := os.ReadFile(logFile)
	if err != nil {
		return fmt.Errorf("failed to read log file %s: %w", logFile, err)
	}

	kind := detectAgentOutputFormat(raw)
	if kind == "" {
		// Plain text or unknown JSON shape: use Gemini parser as fallback (legacy behavior).
		kind = config.AgentKindGemini
	}

	_, _, err = parseStreamByKind(kind, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("failed to parse log file: %w", err)
	}
	return nil
}
