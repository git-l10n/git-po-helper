// Package util provides business logic for agent-run misc commands (show-config, parse-log).
package util

import (
	"bytes"
	"fmt"
	"os"

	"github.com/git-l10n/git-po-helper/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// CmdAgentShowConfig displays the current agent configuration in YAML format.
// Used by both agent-run and agent-test show-config commands.
func CmdAgentShowConfig() error {
	cfg, err := LoadAgentConfigForCmd()
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return err
	}

	// Marshal configuration to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		log.Errorf("failed to marshal configuration to YAML: %v", err)
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Display the configuration
	fmt.Println("# Agent Configuration")
	fmt.Println("# This is the merged configuration from:")
	fmt.Println("# - User home directory: ~/.git-po-helper.yaml (lower priority)")
	fmt.Println("# - Repository root: <repo-root>/git-po-helper.yaml (higher priority)")
	fmt.Println()
	os.Stdout.Write(yamlData)

	return nil
}

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
