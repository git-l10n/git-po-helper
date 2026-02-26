// Package config provides configuration structures and loading for agent commands.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/git-l10n/git-po-helper/repository"
	log "github.com/sirupsen/logrus"
)

//go:embed prompts/update_pot.txt
var promptUpdatePot string

//go:embed prompts/update_po.txt
var promptUpdatePo string

//go:embed prompts/translate.txt
var promptTranslate string

//go:embed prompts/review.txt
var promptReview string

// AgentConfig holds the complete agent configuration.
type AgentConfig struct {
	DefaultLangCode string           `yaml:"default_lang_code"`
	Prompt          PromptConfig     `yaml:"prompt"`
	AgentTest       AgentTestConfig  `yaml:"agent-test"`
	Agents          map[string]Agent `yaml:"agents"`
}

// PromptConfig holds prompt templates for different operations.
type PromptConfig struct {
	UpdatePot string `yaml:"update_pot"`
	UpdatePo  string `yaml:"update_po"`
	Translate string `yaml:"translate"`
	Review    string `yaml:"review"`
}

// AgentTestConfig holds configuration for agent-test command.
type AgentTestConfig struct {
	Runs                      *int `yaml:"runs"`
	PotEntriesBeforeUpdate    *int `yaml:"pot_entries_before_update"`
	PotEntriesAfterUpdate     *int `yaml:"pot_entries_after_update"`
	PoEntriesBeforeUpdate     *int `yaml:"po_entries_before_update"`
	PoEntriesAfterUpdate      *int `yaml:"po_entries_after_update"`
	PoNewEntriesAfterUpdate   *int `yaml:"po_new_entries_after_update"`
	PoFuzzyEntriesAfterUpdate *int `yaml:"po_fuzzy_entries_after_update"`
}

// KnownAgentKinds defines the valid agent kinds. Kind must be one of these for type-safe detection.
const (
	AgentKindClaude   = "claude"
	AgentKindGemini   = "gemini"
	AgentKindCodex    = "codex"
	AgentKindOpencode = "opencode"
	AgentKindEcho     = "echo" // Test agent, no stream-json
	AgentKindQwen     = "qwen" // Alias for gemini-compatible CLI
)

// KnownAgentKinds is the set of valid agent kinds for validation.
var KnownAgentKinds = map[string]bool{
	AgentKindClaude:   true,
	AgentKindGemini:   true,
	AgentKindCodex:    true,
	AgentKindOpencode: true,
	AgentKindEcho:     true,
	AgentKindQwen:     true,
}

// Agent holds configuration for a single agent.
type Agent struct {
	Cmd    []string `yaml:"cmd"`
	Kind   string   `yaml:"kind"`   // Agent kind: "claude", "gemini", "codex", or "opencode"
	Output string   `yaml:"output"` // Output format: "default", "json", or "stream_json"
}

// getSystemLocale gets the system locale from environment variables.
// It checks LC_ALL, LC_MESSAGES, LANG in order of priority.
// Returns a locale string like "en_US" or "zh_CN", or "en_US" as fallback.
func getSystemLocale() string {
	// Check locale environment variables in order of priority
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LC_MESSAGES")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}

	// Parse locale string (format: language_territory.encoding or language_territory@variant)
	// Examples: "en_US.UTF-8", "zh_CN.UTF-8", "C", "POSIX"
	if locale != "" {
		// Remove encoding suffix (.UTF-8, .utf8, etc.)
		parts := strings.Split(locale, ".")
		locale = parts[0]

		// Remove variant suffix (@variant)
		parts = strings.Split(locale, "@")
		locale = parts[0]

		// Handle special cases: "C" and "POSIX" default to "en_US"
		if locale == "C" || locale == "POSIX" {
			locale = "en_US"
		}

		// Validate format (should be like "en_US" or "zh_CN")
		if strings.Contains(locale, "_") {
			return locale
		}

		// If only language code (e.g., "en"), try to get full locale
		// For now, we'll use it as-is or default to en_US
		if len(locale) >= 2 {
			// Try to construct a valid locale (e.g., "en" -> "en_US")
			// This is a simple heuristic
			return locale + "_US"
		}
	}

	// Default fallback
	return "en_US"
}

// loadEmbeddedPrompt loads a prompt from an embedded string and trims whitespace.
func loadEmbeddedPrompt(prompt string) string {
	return strings.TrimSpace(prompt)
}

// getDefaultConfig returns a default AgentConfig with sensible defaults.
// Prompt templates are loaded from embedded files in config/prompts/.
func getDefaultConfig() *AgentConfig {
	defaultRuns := 1
	systemLocale := getSystemLocale()

	return &AgentConfig{
		DefaultLangCode: systemLocale,
		Prompt: PromptConfig{
			UpdatePot: loadEmbeddedPrompt(promptUpdatePot),
			UpdatePo:  loadEmbeddedPrompt(promptUpdatePo),
			Translate: loadEmbeddedPrompt(promptTranslate),
			Review:    loadEmbeddedPrompt(promptReview),
		},
		AgentTest: AgentTestConfig{
			Runs: &defaultRuns,
		},
		Agents: map[string]Agent{
			"claude": {
				Cmd:    []string{"claude", "--dangerously-skip-permissions", "-p", "{{.prompt}}"},
				Kind:   AgentKindClaude,
				Output: "json",
			},
			"codex": {
				Cmd:    []string{"codex", "exec", "--yolo", "{{.prompt}}"},
				Kind:   AgentKindCodex,
				Output: "json",
			},
			"opencode": {
				Cmd:    []string{"opencode", "run", "--thinking", "{{.prompt}}"},
				Kind:   AgentKindOpencode,
				Output: "json",
			},
			"gemini": {
				Cmd:    []string{"gemini", "--yolo", "{{.prompt}}"},
				Kind:   AgentKindGemini,
				Output: "json",
			},
			"echo": {
				Cmd:  []string{"echo", "{{.prompt}}"},
				Kind: AgentKindEcho,
			},
		},
	}
}

// LoadAgentConfig loads agent configuration from multiple locations with priority:
// 1. User home directory: ~/.git-po-helper.yaml (lower priority)
// 2. Repository root: <repo-root>/git-po-helper.yaml (higher priority, overrides user config)
// Returns the configuration and an error. If both config files are missing, it returns
// a default empty config with a warning (not an error).
func LoadAgentConfig() (*AgentConfig, error) {
	var baseConfig, repoConfig AgentConfig
	configsLoaded := false

	// Load user home directory config first (lower priority)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(homeDir, ".git-po-helper.yaml")
		if _, err := os.Stat(userConfigPath); err == nil {
			config, err := loadConfigFromFile(userConfigPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load user config from %s: %w", userConfigPath, err)
			}
			baseConfig = *config
			configsLoaded = true
			log.Debugf("loaded user config from %s", userConfigPath)
		}
	}

	// Load repository root config (higher priority, overrides user config)
	workDir := repository.WorkDir()
	repoConfigPath := filepath.Join(workDir, "git-po-helper.yaml")
	if _, err := os.Stat(repoConfigPath); err == nil {
		config, err := loadConfigFromFile(repoConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load repo config from %s: %w", repoConfigPath, err)
		}
		repoConfig = *config
		configsLoaded = true
		log.Debugf("loaded repo config from %s", repoConfigPath)
	}

	// If no config files were found, return default config
	if !configsLoaded {
		userConfigPath := ""
		if homeDir != "" {
			userConfigPath = filepath.Join(homeDir, ".git-po-helper.yaml")
		} else {
			userConfigPath = "~/.git-po-helper.yaml"
		}
		log.Warnf("no configuration files found (checked %s and %s), using defaults",
			userConfigPath, repoConfigPath)
		return getDefaultConfig(), nil
	}

	// Merge configurations: repo config overrides user config
	mergedConfig := mergeConfigs(&baseConfig, &repoConfig)

	// Initialize Agents map if nil
	if mergedConfig.Agents == nil {
		mergedConfig.Agents = make(map[string]Agent)
	}

	// Apply defaults for missing values
	applyDefaults(mergedConfig)

	// Validate configuration
	if err := mergedConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return mergedConfig, nil
}

// applyDefaults applies default values to a configuration for missing fields.
func applyDefaults(cfg *AgentConfig) {
	defaultConfig := getDefaultConfig()

	// Apply default_lang_code if not set
	if cfg.DefaultLangCode == "" {
		cfg.DefaultLangCode = defaultConfig.DefaultLangCode
	}

	// Apply prompt defaults if not set
	if cfg.Prompt.UpdatePot == "" {
		cfg.Prompt.UpdatePot = defaultConfig.Prompt.UpdatePot
	}
	if cfg.Prompt.UpdatePo == "" {
		cfg.Prompt.UpdatePo = defaultConfig.Prompt.UpdatePo
	}
	if cfg.Prompt.Translate == "" {
		cfg.Prompt.Translate = defaultConfig.Prompt.Translate
	}
	if cfg.Prompt.Review == "" {
		cfg.Prompt.Review = defaultConfig.Prompt.Review
	}

	// Apply agent-test defaults if not set
	if cfg.AgentTest.Runs == nil {
		cfg.AgentTest.Runs = defaultConfig.AgentTest.Runs
	}

	// Apply default agent only if no agents configured
	// If config file has agents, use them and don't add default test agent
	if len(cfg.Agents) == 0 {
		cfg.Agents = map[string]Agent{
			"test": {
				Cmd:  []string{"echo", "{{.prompt}}"},
				Kind: AgentKindEcho,
			},
		}
	}

	// Apply default Kind for agents that don't have it (backward compatibility)
	for key, agent := range cfg.Agents {
		if agent.Kind == "" {
			if defaultAgent, ok := defaultConfig.Agents[key]; ok {
				agent.Kind = defaultAgent.Kind
				cfg.Agents[key] = agent
			}
		}
	}
}

// loadConfigFromFile loads and parses a YAML config file without validation.
// This is used internally to load configs that will be merged.
func loadConfigFromFile(configPath string) (*AgentConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	return &config, nil
}

// mergeConfigs merges two AgentConfig structs, with repoConfig taking priority over baseConfig.
func mergeConfigs(baseConfig, repoConfig *AgentConfig) *AgentConfig {
	result := &AgentConfig{
		Agents: make(map[string]Agent),
	}

	// Start with base config
	if baseConfig != nil {
		result.DefaultLangCode = baseConfig.DefaultLangCode
		result.Prompt = baseConfig.Prompt
		result.AgentTest = baseConfig.AgentTest
		if baseConfig.Agents != nil {
			for k, v := range baseConfig.Agents {
				result.Agents[k] = v
			}
		}
	}

	// Override with repo config (higher priority)
	if repoConfig != nil {
		if repoConfig.DefaultLangCode != "" {
			result.DefaultLangCode = repoConfig.DefaultLangCode
		}
		// Merge Prompt config
		if repoConfig.Prompt.UpdatePot != "" {
			result.Prompt.UpdatePot = repoConfig.Prompt.UpdatePot
		}
		if repoConfig.Prompt.UpdatePo != "" {
			result.Prompt.UpdatePo = repoConfig.Prompt.UpdatePo
		}
		if repoConfig.Prompt.Translate != "" {
			result.Prompt.Translate = repoConfig.Prompt.Translate
		}
		if repoConfig.Prompt.Review != "" {
			result.Prompt.Review = repoConfig.Prompt.Review
		}
		// Merge AgentTest config (pointer fields need special handling)
		if repoConfig.AgentTest.Runs != nil {
			result.AgentTest.Runs = repoConfig.AgentTest.Runs
		}
		if repoConfig.AgentTest.PotEntriesBeforeUpdate != nil {
			result.AgentTest.PotEntriesBeforeUpdate = repoConfig.AgentTest.PotEntriesBeforeUpdate
		}
		if repoConfig.AgentTest.PotEntriesAfterUpdate != nil {
			result.AgentTest.PotEntriesAfterUpdate = repoConfig.AgentTest.PotEntriesAfterUpdate
		}
		if repoConfig.AgentTest.PoEntriesBeforeUpdate != nil {
			result.AgentTest.PoEntriesBeforeUpdate = repoConfig.AgentTest.PoEntriesBeforeUpdate
		}
		if repoConfig.AgentTest.PoEntriesAfterUpdate != nil {
			result.AgentTest.PoEntriesAfterUpdate = repoConfig.AgentTest.PoEntriesAfterUpdate
		}
		if repoConfig.AgentTest.PoNewEntriesAfterUpdate != nil {
			result.AgentTest.PoNewEntriesAfterUpdate = repoConfig.AgentTest.PoNewEntriesAfterUpdate
		}
		if repoConfig.AgentTest.PoFuzzyEntriesAfterUpdate != nil {
			result.AgentTest.PoFuzzyEntriesAfterUpdate = repoConfig.AgentTest.PoFuzzyEntriesAfterUpdate
		}
		// Merge Agents (repo config agents override base config agents)
		if repoConfig.Agents != nil {
			for k, v := range repoConfig.Agents {
				result.Agents[k] = v
			}
		}
	}

	return result
}

// Validate validates the agent configuration and returns an error if invalid.
func (c *AgentConfig) Validate() error {
	// Check if at least one agent is configured
	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	// Validate each agent
	for name, agent := range c.Agents {
		if len(agent.Cmd) == 0 {
			return fmt.Errorf("agent '%s' has empty command", name)
		}
		if agent.Kind == "" {
			return fmt.Errorf("agent '%s' has empty kind (must be one of: claude, gemini, codex, opencode, echo, qwen)", name)
		}
		if !KnownAgentKinds[agent.Kind] {
			return fmt.Errorf("agent '%s' has unknown kind '%s' (must be one of: claude, gemini, codex, opencode, echo, qwen)", name, agent.Kind)
		}
	}

	// Validate that update_pot prompt is set (required for update-pot command)
	if c.Prompt.UpdatePot == "" {
		return fmt.Errorf("prompt.update_pot is required")
	}

	return nil
}
