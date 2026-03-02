// Package util provides business logic for agent-run misc commands (show-config, parse-log).
package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/flag"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// CmdAgentRunShowConfig displays the current agent configuration in YAML format.
func CmdAgentRunShowConfig() error {
	// Load configuration
	log.Debugf("loading agent configuration")
	cfg, err := config.LoadAgentConfig(flag.AgentConfigFile())
	if err != nil {
		log.Errorf("failed to load agent configuration: %v", err)
		return fmt.Errorf("failed to load agent configuration: %w", err)
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
// Auto-detects format: Claude (claude_code_version) vs Qwen/Gemini (qwen_code_version or Gemini-style).
// Each line in the file should be a JSON object. Supports system, assistant (with text,
// thinking, tool_use content types), user (tool_result), and result messages.
func CmdAgentRunParseLog(logFile string) error {
	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	firstLine, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read log file: %w", err)
	}
	parseReader := io.MultiReader(strings.NewReader(firstLine), reader)

	if strings.Contains(firstLine, "claude_code_version") {
		_, _, err = ParseClaudeStreamJSONRealtime(parseReader)
	} else if strings.Contains(firstLine, `"type":"step_start"`) || strings.Contains(firstLine, `"type": "step_start"`) {
		// OpenCode format
		_, _, err = ParseOpenCodeJSONLRealtime(parseReader)
	} else if strings.Contains(firstLine, "thread.started") {
		// Codex format
		_, _, err = ParseCodexJSONLRealtime(parseReader)
	} else {
		// Qwen/Gemini format (qwen_code_version or Gemini-style system init)
		_, _, err = ParseGeminiJSONLRealtime(parseReader)
	}
	if err != nil {
		return fmt.Errorf("failed to parse log file: %w", err)
	}
	return nil
}
