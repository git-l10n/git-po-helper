// Package util provides utility functions for agent command building.
package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/git-l10n/git-po-helper/config"
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

// echoAgent implements config.Agent for echo (test agent, text output only).
type echoAgent struct {
	cmd []string
}

// BuildCommand returns the command with placeholders replaced (no format params).
func (a *echoAgent) BuildCommand(vars map[string]string) ([]string, error) {
	return replacePlaceholdersInCmd(a.cmd, vars)
}

// GetOutputFormat returns text (echo produces plain text).
func (a *echoAgent) GetOutputFormat() string {
	return config.OutputText
}

// SelectAgent selects an agent from the configuration based on the provided agent name.
// If agentName is empty, it auto-selects an agent (only works if exactly one agent is configured).
// Returns the selected agent entry and an error if selection fails.
// Validates that agent.Kind is one of the known types (claude, gemini, codex, opencode, echo).
func SelectAgent(cfg *config.AgentConfig, agentName string) (config.AgentEntry, error) {
	var entry config.AgentEntry

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
			return config.AgentEntry{}, fmt.Errorf("agent '%s' not found in configuration\nAvailable agents: %s\nHint: Check git-po-helper.yaml for configured agents", agentName, strings.Join(agentList, ", "))
		}
		entry = a
	} else {
		// Auto-select agent
		log.Debugf("auto-selecting agent from configuration")
		if len(cfg.Agents) == 0 {
			log.Error("no agents configured")
			return config.AgentEntry{}, fmt.Errorf("no agents configured\nHint: Add at least one agent to git-po-helper.yaml in the 'agents' section")
		}
		if len(cfg.Agents) > 1 {
			agentList := make([]string, 0, len(cfg.Agents))
			for k := range cfg.Agents {
				agentList = append(agentList, k)
			}
			log.Errorf("multiple agents configured (%s), --agent flag required", strings.Join(agentList, ", "))
			return config.AgentEntry{}, fmt.Errorf("multiple agents configured (%s), please specify --agent\nHint: Use --agent flag to select one of the available agents", strings.Join(agentList, ", "))
		}
		for k, v := range cfg.Agents {
			entry, agentName = v, k
			break
		}
	}

	// Set entry.Kind initial value when empty: try agentName then command name
	if entry.Kind == "" {
		// Try agentName (config key) converted to lowercase
		if lower := strings.ToLower(agentName); config.KnownAgentKinds[lower] {
			entry.Kind = lower
		} else {
			// Try first command argument (command name): use basename for paths
			if len(entry.Cmd) > 0 {
				base := strings.ToLower(filepath.Base(entry.Cmd[0]))
				if config.KnownAgentKinds[base] {
					entry.Kind = base
				}
			}
		}
		if entry.Kind == "" {
			return config.AgentEntry{}, fmt.Errorf(
				"agent '%s' has unknown kind (cmd=%v)\n"+
					"Hint: Add 'kind' field (claude, gemini, codex, opencode, echo, qwen, qoder) to agent in git-po-helper.yaml",
				agentName, entry.Cmd)
		}
	}

	// Validate entry.Kind is a known type
	if !config.KnownAgentKinds[entry.Kind] {
		return config.AgentEntry{}, fmt.Errorf(
			"agent '%s' has unknown kind '%s' (must be one of: claude, gemini, codex, opencode, echo, qwen, qoder)\n"+
				"Hint: Set 'kind' to a valid value in git-po-helper.yaml", agentName, entry.Kind)
	}

	return entry, nil
}

// BuildAgentCommand builds an agent command using the Agent interface.
// Returns the full command (with format params) and the output format.
func BuildAgentCommand(entry config.AgentEntry, vars PlaceholderVars) ([]string, string, error) {
	agent, err := NewAgentFromConfig(entry)
	if err != nil {
		return nil, "", err
	}
	cmd, err := agent.BuildCommand(map[string]string(vars))
	if err != nil {
		return nil, "", err
	}
	return cmd, agent.GetOutputFormat(), nil
}

// GetPotFilePath returns the full path to the POT file in the repository.
func GetPotFilePath() string {
	return filepath.Join(PoDir, GitPot)
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
	case "local-orchestration-review":
		prompt = cfg.Prompt.LocalOrchestrationReview
		promptName = "prompt.local_orchestration_review"
	case "local-orchestration-translation":
		prompt = cfg.Prompt.LocalOrchestrationTranslation
		promptName = "prompt.local_orchestration_translation"
	case "fix-po":
		prompt = cfg.Prompt.FixPo
		promptName = "prompt.fix_po"
	default:
		return "", fmt.Errorf("unknown action: %s\nHint: Supported actions are: update-pot, update-po, translate, local-orchestration-review, local-orchestration-translation, fix-po", action)
	}

	if prompt == "" {
		log.Errorf("%s is not configured", promptName)
		return "", fmt.Errorf("%s is not configured\nHint: Add '%s' to git-po-helper.yaml", promptName, promptName)
	}
	log.Debugf("using %s prompt: %s", action, prompt)
	return prompt, nil
}
