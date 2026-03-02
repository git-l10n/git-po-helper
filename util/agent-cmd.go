// Package util provides utility functions for agent command building.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// PlaceholderVars holds key-value pairs for placeholder replacement.
// Keys correspond to placeholder names in template (e.g. {{.prompt}}, {{.source}}).
type PlaceholderVars map[string]string

// ExecutePromptTemplate executes a Go text template with the given data.
// The template uses {{.key}} syntax (e.g. {{.source}}, {{.dest}}).
// Data is built from vars; the "prompt" key is excluded to avoid circular reference.
// Returns the executed template string or an error if template parsing/execution fails.
func ExecutePromptTemplate(tmpl string, vars PlaceholderVars) (string, error) {
	data := make(map[string]interface{})
	for k, v := range vars {
		if k != "prompt" {
			data[k] = v
		}
	}
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}
	return buf.String(), nil
}

// ReplacePlaceholders replaces placeholders in a template string with actual values.
// Uses Go text/template syntax: {{.key}}, e.g. {{.prompt}}, {{.source}}, {{.commit}}.
//
// Example:
//
//	ReplacePlaceholders("cmd -p {{.prompt}} -s {{.source}}", PlaceholderVars{
//	    "prompt": "update",
//	    "source": "po/zh_CN.po",
//	})
func ReplacePlaceholders(tmpl string, kv PlaceholderVars) (string, error) {
	data := make(map[string]interface{})
	for k, v := range kv {
		data[k] = v
	}
	t, err := template.New("cmd").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse command template: %w", err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute command template: %w", err)
	}
	return buf.String(), nil
}

// normalizeOutputFormat normalizes output format by converting underscores to hyphens
// and unifying stream-json/stream_json to json.
// This allows both "stream_json" and "stream-json" to be treated as "json".
func normalizeOutputFormat(format string) string {
	normalized := strings.ReplaceAll(format, "_", "-")
	// Unify stream-json to json (claude uses stream-json internally, but we simplify it to json)
	if normalized == "stream-json" {
		return "json"
	}
	return normalized
}

// SelectAgent selects an agent from the configuration based on the provided agent name.
// If agentName is empty, it auto-selects an agent (only works if exactly one agent is configured).
// Returns the selected agent and an error if selection fails.
// Validates that agent.Kind is one of the known types (claude, gemini, codex, opencode, echo).
func SelectAgent(cfg *config.AgentConfig, agentName string) (config.Agent, error) {
	var agent config.Agent

	if agentName != "" {
		// Use specified agent
		log.Debugf("using specified agent: %s", agentName)
		a, ok := cfg.Agents[agentName]
		if !ok {
			agentList := make([]string, 0, len(cfg.Agents))
			for k := range cfg.Agents {
				agentList = append(agentList, k)
			}
			log.Errorf("agent '%s' not found in configuration. Available agents: %v", agentName, agentList)
			return config.Agent{}, fmt.Errorf("agent '%s' not found in configuration\nAvailable agents: %s\nHint: Check git-po-helper.yaml for configured agents", agentName, strings.Join(agentList, ", "))
		}
		agent = a
	} else {
		// Auto-select agent
		log.Debugf("auto-selecting agent from configuration")
		if len(cfg.Agents) == 0 {
			log.Error("no agents configured")
			return config.Agent{}, fmt.Errorf("no agents configured\nHint: Add at least one agent to git-po-helper.yaml in the 'agents' section")
		}
		if len(cfg.Agents) > 1 {
			agentList := make([]string, 0, len(cfg.Agents))
			for k := range cfg.Agents {
				agentList = append(agentList, k)
			}
			log.Errorf("multiple agents configured (%s), --agent flag required", strings.Join(agentList, ", "))
			return config.Agent{}, fmt.Errorf("multiple agents configured (%s), please specify --agent\nHint: Use --agent flag to select one of the available agents", strings.Join(agentList, ", "))
		}
		for k, v := range cfg.Agents {
			agent, agentName = v, k
			break
		}
	}

	// Set agent.Kind initial value when empty: try agentName then command name
	if agent.Kind == "" {
		// Try agentName (config key) converted to lowercase
		if lower := strings.ToLower(agentName); config.KnownAgentKinds[lower] {
			agent.Kind = lower
		} else {
			// Try first command argument (command name): use basename for paths
			if len(agent.Cmd) > 0 {
				base := strings.ToLower(filepath.Base(agent.Cmd[0]))
				if config.KnownAgentKinds[base] {
					agent.Kind = base
				}
			}
		}
		if agent.Kind == "" {
			return config.Agent{}, fmt.Errorf(
				"agent '%s' has unknown kind (cmd=%v)\n"+
					"Hint: Add 'kind' field (claude, gemini, codex, opencode, echo, qwen) to agent in git-po-helper.yaml",
				agentName, agent.Cmd)
		}
	}

	// Validate agent.Kind is a known type
	if !config.KnownAgentKinds[agent.Kind] {
		return config.Agent{}, fmt.Errorf(
			"agent '%s' has unknown kind '%s' (must be one of: claude, gemini, codex, opencode, echo, qwen)\n"+
				"Hint: Set 'kind' to a valid value in git-po-helper.yaml", agentName, agent.Kind)
	}

	return agent, nil
}

// BuildAgentCommand builds an agent command by replacing placeholders in the agent's command template.
// Uses Go text/template syntax (e.g. {{.prompt}}, {{.source}}, {{.commit}}).
// For claude/codex/opencode/gemini commands, it adds stream-json parameters based on agent.Output.
// Uses agent.Kind for type-safe detection (Kind must be validated by SelectAgent).
func BuildAgentCommand(agent config.Agent, vars PlaceholderVars) ([]string, error) {
	cmd := make([]string, len(agent.Cmd))
	for i, arg := range agent.Cmd {
		resolved, err := ReplacePlaceholders(arg, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve command arg %q: %w", arg, err)
		}
		cmd[i] = resolved
	}

	// Use agent.Kind for type detection (validated by SelectAgent)
	kind := agent.Kind
	isClaude := kind == config.AgentKindClaude
	isCodex := kind == config.AgentKindCodex
	isOpencode := kind == config.AgentKindOpencode
	isGemini := kind == config.AgentKindGemini || kind == config.AgentKindQwen

	// For claude command, add --output-format parameter if output format is specified
	if isClaude {
		// Check if --output-format parameter already exists in the command
		hasOutputFormat := false
		for i, arg := range cmd {
			if arg == "--output-format" || arg == "-o" {
				hasOutputFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --output-format if it doesn't already exist
		if !hasOutputFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --output-format parameter for json format (claude uses stream-json internally)
			if outputFormat == "json" {
				cmd = append(cmd, "--verbose", "--output-format", "stream-json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For codex command, add --json parameter if output format is json
	if isCodex {
		// Check if --json parameter already exists in the command
		hasJSON := false
		for _, arg := range cmd {
			if arg == "--json" {
				hasJSON = true
				break
			}
		}

		// Only add --json if it doesn't already exist
		if !hasJSON {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --json parameter for json format (codex uses JSONL format)
			if outputFormat == "json" {
				cmd = append(cmd, "--json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For opencode command, add --format json parameter if output format is json
	if isOpencode {
		// Check if --format parameter already exists in the command
		hasFormat := false
		for i, arg := range cmd {
			if arg == "--format" {
				hasFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --format if it doesn't already exist
		if !hasFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --format json parameter for json format (opencode uses JSONL format)
			if outputFormat == "json" {
				cmd = append(cmd, "--format", "json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	// For gemini/qwen command, add --output-format stream-json parameter if output format is json
	// (Applicable to Claude Code and Gemini-CLI)
	if isGemini {
		// Check if --output-format or -o parameter already exists in the command
		hasOutputFormat := false
		for i, arg := range cmd {
			if arg == "--output-format" || arg == "-o" {
				hasOutputFormat = true
				// Skip the next argument (the format value)
				if i+1 < len(cmd) {
					_ = cmd[i+1]
				}
				break
			}
		}

		// Only add --output-format if it doesn't already exist
		if !hasOutputFormat {
			outputFormat := normalizeOutputFormat(agent.Output)
			if outputFormat == "" {
				outputFormat = "default"
			}

			// Add --output-format stream-json parameter for json format (gemini uses stream-json)
			if outputFormat == "json" {
				cmd = append(cmd, "--output-format", "stream-json")
			}
			// For "default" format, no additional parameter is needed
		}
	}

	return cmd, nil
}

// GetPotFilePath returns the full path to the POT file in the repository.
func GetPotFilePath() string {
	workDir := repository.WorkDirOrCwd()
	return filepath.Join(workDir, PoDir, GitPot)
}

// GetRawPrompt returns the prompt for the specified action from configuration, or an error if not configured.
// Supported actions: "update-pot", "update-po", "translate", "review", "local-orchestration-translation", "fix-po"
// If --prompt flag is provided via viper, it overrides the configuration value.
func GetRawPrompt(cfg *config.AgentConfig, action string) (string, error) {
	// Check if --prompt flag is provided via viper (from command line)
	// Check both agent-run--prompt and agent-test--prompt
	overridePrompt := viper.GetString("agent-run--prompt")
	if overridePrompt == "" {
		overridePrompt = viper.GetString("agent-test--prompt")
	}

	// If override prompt is provided, use it directly
	if overridePrompt != "" {
		log.Debugf("using override prompt from --prompt flag for action %s: %s", action, overridePrompt)
		return overridePrompt, nil
	}

	var prompt string
	var promptName string

	switch action {
	case "update-pot":
		prompt = cfg.Prompt.UpdatePot
		promptName = "prompt.update_pot"
	case "update-po":
		prompt = cfg.Prompt.UpdatePo
		promptName = "prompt.update_po"
	case "translate":
		prompt = cfg.Prompt.Translate
		promptName = "prompt.translate"
	case "review":
		prompt = cfg.Prompt.Review
		promptName = "prompt.review"
	case "local-orchestration-translation":
		prompt = cfg.Prompt.LocalOrchestrationTranslation
		promptName = "prompt.local_orchestration_translation"
	case "fix-po":
		prompt = cfg.Prompt.FixPo
		promptName = "prompt.fix_po"
	default:
		return "", fmt.Errorf("unknown action: %s\nHint: Supported actions are: update-pot, update-po, translate, review, local-orchestration-translation, fix-po", action)
	}

	if prompt == "" {
		log.Errorf("%s is not configured", promptName)
		return "", fmt.Errorf("%s is not configured\nHint: Add '%s' to git-po-helper.yaml", promptName, promptName)
	}
	log.Debugf("using %s prompt: %s", action, prompt)
	return prompt, nil
}
