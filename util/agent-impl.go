// Package util provides Agent interface implementations and factory.
package util

import (
	"fmt"
	"strings"

	"github.com/git-l10n/git-po-helper/config"
)

// replacePlaceholdersInCmd replaces {{.key}} in each cmd arg.
func replacePlaceholdersInCmd(cmd []string, vars map[string]string) ([]string, error) {
	out := make([]string, len(cmd))
	for i, arg := range cmd {
		resolved, err := ReplacePlaceholders(arg, PlaceholderVars(vars))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve command arg %q: %w", arg, err)
		}
		out[i] = resolved
	}
	return out, nil
}

// hasOutputFormatInCmd returns true if cmd contains any of the given flag names.
// Used by codex (--json boolean flag).
func hasOutputFormatInCmd(cmd []string, flagNames ...string) bool {
	flags := make(map[string]bool)
	for _, n := range flagNames {
		flags[n] = true
	}
	for _, arg := range cmd {
		if flags[arg] {
			return true
		}
	}
	return false
}

// parseFormatFromCmd parses output format from cmd. Looks for flag, takes next arg.
// Returns (format, true) if flag found and value valid, (defaultFormat, false) otherwise.
func parseFormatFromCmd(cmd []string, defaultFormat string, flagNames ...string) (format string, found bool) {
	flags := make(map[string]bool)
	for _, n := range flagNames {
		flags[n] = true
	}
	for i, arg := range cmd {
		if flags[arg] && i+1 < len(cmd) {
			f := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(cmd[i+1]), "_", "-"))
			var parsed string
			switch {
			case f == "default":
				parsed = config.OutputText
			case config.ValidOutputFormats[f]:
				parsed = f
			default:
				parsed = defaultFormat
			}
			return parsed, true
		}
	}
	return defaultFormat, false
}

// NewAgentFromConfig creates an Agent implementation from config.
// Returns the Agent interface for the given kind, or an error if kind is unknown.
func NewAgentFromConfig(cfg config.AgentEntry) (config.Agent, error) {
	switch cfg.Kind {
	case config.AgentKindClaude:
		return &claudeAgent{cmd: cfg.Cmd}, nil
	case config.AgentKindCodex:
		return &codexAgent{cmd: cfg.Cmd}, nil
	case config.AgentKindGemini, config.AgentKindQwen:
		return &geminiAgent{cmd: cfg.Cmd}, nil
	case config.AgentKindOpencode:
		return &opencodeAgent{cmd: cfg.Cmd}, nil
	case config.AgentKindQoder:
		return &qoderAgent{cmd: cfg.Cmd}, nil
	case config.AgentKindEcho:
		return &echoAgent{cmd: cfg.Cmd}, nil
	default:
		return nil, fmt.Errorf("unknown agent kind: %s", cfg.Kind)
	}
}
