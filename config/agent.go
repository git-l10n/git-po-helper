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

//go:embed prompts/local-orchestration-translation.md
var promptLocalOrchestrationTranslation string

//go:embed prompts/fix-po.txt
var promptFixPo string

// AgentConfig holds the complete agent configuration.
type AgentConfig struct {
	DefaultLangCode string           `yaml:"default_lang_code"`
	Prompt          PromptConfig     `yaml:"prompt"`
	AgentTest       AgentTestConfig  `yaml:"agent-test"`
	Agents          map[string]Agent `yaml:"agents"`
}

// PromptConfig holds prompt templates for different operations.
type PromptConfig struct {
	UpdatePot                     string `yaml:"update_pot"`
	UpdatePo                      string `yaml:"update_po"`
	Translate                     string `yaml:"translate"`
	Review                        string `yaml:"review"`
	LocalOrchestrationTranslation string `yaml:"local_orchestration_translation"`
	FixPo                         string `yaml:"fix_po"`
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

// GitPoHelperConfigFileName is the name of the agent config file (user home and repository root).
const GitPoHelperConfigFileName = ".git-po-helper.yaml"

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
			UpdatePot:                     loadEmbeddedPrompt(promptUpdatePot),
			UpdatePo:                      loadEmbeddedPrompt(promptUpdatePo),
			Translate:                     loadEmbeddedPrompt(promptTranslate),
			Review:                        loadEmbeddedPrompt(promptReview),
			LocalOrchestrationTranslation: loadEmbeddedPrompt(promptLocalOrchestrationTranslation),
			FixPo:                         loadEmbeddedPrompt(promptFixPo),
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

// loadUserHomeConfig loads ~/AgentConfigFileName if it exists.
// Returns (config, nil) on success, (nil, nil) when the file is missing, (nil, err) on read/parse error.
func loadUserHomeConfig() (*AgentConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	userConfigPath := filepath.Join(homeDir, GitPoHelperConfigFileName)
	if _, err := os.Stat(userConfigPath); err != nil {
		return nil, nil
	}
	config, err := loadConfigFromFile(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config from %s: %w", userConfigPath, err)
	}
	log.Debugf("loaded user config from %s", userConfigPath)
	return config, nil
}

// loadRepoConfig loads <repo-root>/AgentConfigFileName if it exists.
// Returns (config, nil) on success, (nil, nil) when the file is missing, (nil, err) on read/parse error.
func loadRepoConfig() (*AgentConfig, error) {
	workDir := repository.WorkDir()
	repoConfigPath := filepath.Join(workDir, GitPoHelperConfigFileName)
	if _, err := os.Stat(repoConfigPath); err != nil {
		return nil, nil
	}
	config, err := loadConfigFromFile(repoConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load repo config from %s: %w", repoConfigPath, err)
	}
	log.Debugf("loaded repo config from %s", repoConfigPath)
	return config, nil
}

// LoadAgentConfig loads agent configuration. Merge order: default first (mergeAgents false), then user home, repo root, custom file (mergeAgents true). After all merges, if Agents is still empty, default's Agents are copied.
// Returns the configuration and an error. If no config files are found and no custom path was given, returns a default config with a warning (not an error).
func LoadAgentConfig(customConfigPath string) (*AgentConfig, error) {
	var merged AgentConfig
	configsLoaded := false

	// 1. Base: merge default config first (mergeAgents false — fill unset fields only, do not touch Agents)
	defaultCfg := getDefaultConfig()
	merged = *mergeConfigs(&merged, defaultCfg, false)

	// 2. Overlay: user home config (mergeAgents true)
	if userConfig, err := loadUserHomeConfig(); err != nil {
		return nil, err
	} else if userConfig != nil {
		merged = *mergeConfigs(&merged, userConfig, true)
		configsLoaded = true
	}

	// 3. Overlay: repository root config (mergeAgents true)
	if repoConfig, err := loadRepoConfig(); err != nil {
		return nil, err
	} else if repoConfig != nil {
		merged = *mergeConfigs(&merged, repoConfig, true)
		configsLoaded = true
	}

	// 4. Overlay: custom config file (mergeAgents true)
	if customConfigPath != "" {
		customConfig, err := loadConfigFromFile(customConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", customConfigPath, err)
		}
		log.Debugf("loaded custom config from %s", customConfigPath)
		merged = *mergeConfigs(&merged, customConfig, true)
		configsLoaded = true
	}

	if !configsLoaded {
		workDir := repository.WorkDir()
		repoConfigPath := filepath.Join(workDir, GitPoHelperConfigFileName)
		log.Warnf("no configuration files found (checked ~/%s and %s), using defaults",
			GitPoHelperConfigFileName, repoConfigPath)
		return getDefaultConfig(), nil
	}

	// 5. If Agents is still empty after all merges, use default's Agents
	if len(merged.Agents) == 0 {
		merged.Agents = make(map[string]Agent)
		for k, v := range defaultCfg.Agents {
			merged.Agents[k] = v
		}
	}
	return &merged, nil
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

// mergeConfigs merges baseConfig and overlay. mergeAgents controls Agents behavior:
// - mergeAgents true: overlay overrides base; Agents are merged by key (overlay adds or overrides).
// - mergeAgents false: overlay fills only unset fields in base; Agents are not modified (no merge, no copy).
func mergeConfigs(baseConfig, overlay *AgentConfig, mergeAgents bool) *AgentConfig {
	result := &AgentConfig{
		Agents: make(map[string]Agent),
	}

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

	if overlay != nil {
		if overlay.DefaultLangCode != "" {
			result.DefaultLangCode = overlay.DefaultLangCode
		}
		if overlay.Prompt.UpdatePot != "" {
			result.Prompt.UpdatePot = overlay.Prompt.UpdatePot
		}
		if overlay.Prompt.UpdatePo != "" {
			result.Prompt.UpdatePo = overlay.Prompt.UpdatePo
		}
		if overlay.Prompt.Translate != "" {
			result.Prompt.Translate = overlay.Prompt.Translate
		}
		if overlay.Prompt.Review != "" {
			result.Prompt.Review = overlay.Prompt.Review
		}
		if overlay.Prompt.LocalOrchestrationTranslation != "" {
			result.Prompt.LocalOrchestrationTranslation = overlay.Prompt.LocalOrchestrationTranslation
		}
		if overlay.Prompt.FixPo != "" {
			result.Prompt.FixPo = overlay.Prompt.FixPo
		}
		if overlay.AgentTest.Runs != nil {
			result.AgentTest.Runs = overlay.AgentTest.Runs
		}
		if overlay.AgentTest.PotEntriesBeforeUpdate != nil {
			result.AgentTest.PotEntriesBeforeUpdate = overlay.AgentTest.PotEntriesBeforeUpdate
		}
		if overlay.AgentTest.PotEntriesAfterUpdate != nil {
			result.AgentTest.PotEntriesAfterUpdate = overlay.AgentTest.PotEntriesAfterUpdate
		}
		if overlay.AgentTest.PoEntriesBeforeUpdate != nil {
			result.AgentTest.PoEntriesBeforeUpdate = overlay.AgentTest.PoEntriesBeforeUpdate
		}
		if overlay.AgentTest.PoEntriesAfterUpdate != nil {
			result.AgentTest.PoEntriesAfterUpdate = overlay.AgentTest.PoEntriesAfterUpdate
		}
		if overlay.AgentTest.PoNewEntriesAfterUpdate != nil {
			result.AgentTest.PoNewEntriesAfterUpdate = overlay.AgentTest.PoNewEntriesAfterUpdate
		}
		if overlay.AgentTest.PoFuzzyEntriesAfterUpdate != nil {
			result.AgentTest.PoFuzzyEntriesAfterUpdate = overlay.AgentTest.PoFuzzyEntriesAfterUpdate
		}
		if mergeAgents && overlay.Agents != nil {
			for k, v := range overlay.Agents {
				result.Agents[k] = v
			}
		}
	}

	return result
}
