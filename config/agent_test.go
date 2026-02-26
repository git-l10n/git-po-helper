package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFromFile_MissingFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "git-po-helper.yaml")

	// Test missing file - should return error (loadConfigFromFile doesn't handle missing files)
	config, err := loadConfigFromFile(configPath)
	if err == nil {
		t.Fatal("loadConfigFromFile should return error for missing file")
	}
	if config != nil {
		t.Fatal("loadConfigFromFile should return nil config for missing file")
	}
}

func TestLoadAgentConfig_ValidFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "git-po-helper.yaml")

	// Create a valid YAML config file
	validYAML := `default_lang_code: "zh_CN"
prompt:
  update_pot: "update po/git.pot according to po/README.md"
  update_po: "update {source} according to po/README.md"
agents:
  claude:
    cmd: ["claude", "-p", "{prompt}"]
  gemini:
    cmd: ["gemini", "--prompt", "{prompt}"]
`

	if err := os.WriteFile(configPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFromFile should succeed for valid file, got error: %v", err)
	}
	if config == nil {
		t.Fatal("loadConfigFromFile should return config, got nil")
	}
	if config.DefaultLangCode != "zh_CN" {
		t.Fatalf("expected DefaultLangCode 'zh_CN', got '%s'", config.DefaultLangCode)
	}
	if config.Prompt.UpdatePot != "update po/git.pot according to po/README.md" {
		t.Fatalf("expected UpdatePot prompt, got '%s'", config.Prompt.UpdatePot)
	}
	if len(config.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(config.Agents))
	}
	if config.Agents["claude"].Cmd[0] != "claude" {
		t.Fatalf("expected claude agent command, got %v", config.Agents["claude"].Cmd)
	}
}

func TestLoadAgentConfig_InvalidYAML(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "git-po-helper.yaml")

	// Create an invalid YAML file
	invalidYAML := `default_lang_code: "zh_CN"
prompt:
  update_pot: "update po/git.pot according to po/README.md"
agents:
  claude:
    cmd: ["claude", "-p", "{prompt}"]
    invalid: [unclosed bracket
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := loadConfigFromFile(configPath)
	if err == nil {
		t.Fatal("loadConfigFromFile should return error for invalid YAML")
	}
	if config != nil {
		t.Fatal("loadConfigFromFile should return nil config for invalid YAML")
	}
}

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "update pot",
				},
				Agents: map[string]Agent{
					"claude": {
						Cmd:  []string{"claude", "-p", "{prompt}"},
						Kind: AgentKindClaude,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no agents",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "update pot",
				},
				Agents: map[string]Agent{},
			},
			wantErr: true,
			errMsg:  "at least one agent must be configured",
		},
		{
			name: "empty agent command",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "update pot",
				},
				Agents: map[string]Agent{
					"claude": {
						Cmd:  []string{},
						Kind: AgentKindClaude,
					},
				},
			},
			wantErr: true,
			errMsg:  "agent 'claude' has empty command",
		},
		{
			name: "missing update_pot prompt",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "",
				},
				Agents: map[string]Agent{
					"claude": {
						Cmd:  []string{"claude", "-p", "{prompt}"},
						Kind: AgentKindClaude,
					},
				},
			},
			wantErr: true,
			errMsg:  "prompt.update_pot is required",
		},
		{
			name: "unknown agent kind",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "update pot",
				},
				Agents: map[string]Agent{
					"custom": {
						Cmd:  []string{"custom", "{prompt}"},
						Kind: "unknown",
					},
				},
			},
			wantErr: true,
			errMsg:  "agent 'custom' has unknown kind 'unknown' (must be one of: claude, gemini, codex, opencode, echo, qwen)",
		},
		{
			name: "empty agent kind",
			config: &AgentConfig{
				Prompt: PromptConfig{
					UpdatePot: "update pot",
				},
				Agents: map[string]Agent{
					"custom": {
						Cmd:  []string{"custom", "{prompt}"},
						Kind: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "agent 'custom' has empty kind (must be one of: claude, gemini, codex, opencode, echo, qwen)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate() expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Fatalf("Validate() expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	baseConfig := &AgentConfig{
		DefaultLangCode: "en_US",
		Prompt: PromptConfig{
			UpdatePot: "base update pot",
			UpdatePo:  "base update po",
		},
		Agents: map[string]Agent{
			"claude": {
				Cmd:  []string{"claude", "-p", "{prompt}"},
				Kind: AgentKindClaude,
			},
			"gemini": {
				Cmd:  []string{"gemini", "--prompt", "{prompt}"},
				Kind: AgentKindGemini,
			},
		},
	}

	repoConfig := &AgentConfig{
		DefaultLangCode: "zh_CN",
		Prompt: PromptConfig{
			UpdatePot: "repo update pot",
		},
		Agents: map[string]Agent{
			"claude": {
				Cmd:  []string{"claude", "--new-flag", "{prompt}"},
				Kind: AgentKindClaude,
			},
		},
	}

	merged := mergeConfigs(baseConfig, repoConfig)

	// Check that repo config overrides base config
	if merged.DefaultLangCode != "zh_CN" {
		t.Fatalf("expected DefaultLangCode 'zh_CN', got '%s'", merged.DefaultLangCode)
	}

	// Check that repo config overrides base prompt
	if merged.Prompt.UpdatePot != "repo update pot" {
		t.Fatalf("expected UpdatePot 'repo update pot', got '%s'", merged.Prompt.UpdatePot)
	}

	// Check that base prompt fields are preserved if not overridden
	if merged.Prompt.UpdatePo != "base update po" {
		t.Fatalf("expected UpdatePo 'base update po', got '%s'", merged.Prompt.UpdatePo)
	}

	// Check that repo agents override base agents
	if len(merged.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(merged.Agents))
	}
	if merged.Agents["claude"].Cmd[1] != "--new-flag" {
		t.Fatalf("expected claude agent to have --new-flag, got %v", merged.Agents["claude"].Cmd)
	}

	// Check that base agents are preserved if not overridden
	if merged.Agents["gemini"].Cmd[0] != "gemini" {
		t.Fatalf("expected gemini agent to be preserved, got %v", merged.Agents["gemini"].Cmd)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()

	// Check default_lang_code (should be system locale or en_US)
	if config.DefaultLangCode == "" {
		t.Fatal("DefaultLangCode should not be empty")
	}

	// Check prompt defaults (should not be empty and should contain key placeholders)
	if config.Prompt.UpdatePot == "" {
		t.Fatal("UpdatePot should not be empty")
	}
	if !strings.Contains(config.Prompt.UpdatePot, "po/git.pot") {
		t.Fatalf("UpdatePot should contain 'po/git.pot', got '%s'", config.Prompt.UpdatePot)
	}
	if config.Prompt.UpdatePo == "" {
		t.Fatal("UpdatePo should not be empty")
	}
	if !strings.Contains(config.Prompt.UpdatePo, "{{.source}}") {
		t.Fatalf("UpdatePo should contain '{{.source}}', got '%s'", config.Prompt.UpdatePo)
	}
	if config.Prompt.Translate == "" {
		t.Fatal("Translate should not be empty")
	}
	if !strings.Contains(config.Prompt.Translate, "{{.source}}") {
		t.Fatalf("Translate should contain '{{.source}}', got '%s'", config.Prompt.Translate)
	}
	if config.Prompt.Review == "" {
		t.Fatal("Review should not be empty")
	}
	if !strings.Contains(config.Prompt.Review, "{{.source}}") {
		t.Fatalf("Review should contain '{{.source}}', got '%s'", config.Prompt.Review)
	}
	if !strings.Contains(config.Prompt.Review, "JSON") {
		t.Fatalf("Review should contain 'JSON' (extended prompt), got '%s'", config.Prompt.Review)
	}

	// Check agent-test defaults
	if config.AgentTest.Runs == nil {
		t.Fatal("AgentTest.Runs should not be nil")
	}
	if *config.AgentTest.Runs != 1 {
		t.Fatalf("expected Runs default 1, got %d", *config.AgentTest.Runs)
	}

	// Check default agents
	if len(config.Agents) != 5 {
		t.Fatalf("expected 5 default agents, got %d", len(config.Agents))
	}
	testAgent, ok := config.Agents["echo"]
	if !ok {
		t.Fatal("expected 'echo' agent in default config")
	}
	if len(testAgent.Cmd) != 2 {
		t.Fatalf("expected echo agent command with 2 args, got %d", len(testAgent.Cmd))
	}
	if testAgent.Cmd[0] != "echo" {
		t.Fatalf("expected echo agent command 'echo', got '%s'", testAgent.Cmd[0])
	}
	if testAgent.Cmd[1] != "{{.prompt}}" {
		t.Fatalf("expected echo agent command '{{.prompt}}', got '%s'", testAgent.Cmd[1])
	}
}

func TestGetSystemLocale(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		expect string
	}{
		{
			name:   "LC_ALL set",
			env:    map[string]string{"LC_ALL": "zh_CN.UTF-8"},
			expect: "zh_CN",
		},
		{
			name:   "LANG set",
			env:    map[string]string{"LANG": "en_US.UTF-8"},
			expect: "en_US",
		},
		{
			name:   "C locale",
			env:    map[string]string{"LANG": "C"},
			expect: "en_US",
		},
		{
			name:   "POSIX locale",
			env:    map[string]string{"LANG": "POSIX"},
			expect: "en_US",
		},
		{
			name:   "language only",
			env:    map[string]string{"LANG": "en"},
			expect: "en_US",
		},
		{
			name:   "no locale env",
			env:    map[string]string{},
			expect: "en_US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			originalLCALL := os.Getenv("LC_ALL")
			originalLCMESSAGES := os.Getenv("LC_MESSAGES")
			originalLANG := os.Getenv("LANG")

			// Set test env
			os.Unsetenv("LC_ALL")
			os.Unsetenv("LC_MESSAGES")
			os.Unsetenv("LANG")
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			// Test
			result := getSystemLocale()
			if result != tt.expect {
				t.Fatalf("expected locale '%s', got '%s'", tt.expect, result)
			}

			// Restore original env
			if originalLCALL != "" {
				os.Setenv("LC_ALL", originalLCALL)
			} else {
				os.Unsetenv("LC_ALL")
			}
			if originalLCMESSAGES != "" {
				os.Setenv("LC_MESSAGES", originalLCMESSAGES)
			} else {
				os.Unsetenv("LC_MESSAGES")
			}
			if originalLANG != "" {
				os.Setenv("LANG", originalLANG)
			} else {
				os.Unsetenv("LANG")
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	// Test with empty config
	config := &AgentConfig{
		Agents: make(map[string]Agent),
	}

	applyDefaults(config)

	// Check that defaults are applied
	if config.DefaultLangCode == "" {
		t.Fatal("DefaultLangCode should be set after applyDefaults")
	}
	if config.Prompt.UpdatePot == "" {
		t.Fatal("Prompt.UpdatePot should be set after applyDefaults")
	}
	if config.AgentTest.Runs == nil {
		t.Fatal("AgentTest.Runs should be set after applyDefaults")
	}
	if *config.AgentTest.Runs != 1 {
		t.Fatalf("expected Runs default 1, got %d", *config.AgentTest.Runs)
	}
	if len(config.Agents) == 0 {
		t.Fatal("Agents should have default 'test' agent after applyDefaults")
	}
	if _, ok := config.Agents["test"]; !ok {
		t.Fatal("expected 'test' agent after applyDefaults")
	}

	// Test with partial config (should not override existing values)
	config2 := &AgentConfig{
		DefaultLangCode: "fr_FR",
		Prompt: PromptConfig{
			UpdatePot: "custom update pot",
		},
		Agents: map[string]Agent{
			"custom": {
				Cmd:  []string{"custom", "cmd"},
				Kind: AgentKindEcho,
			},
		},
	}

	applyDefaults(config2)

	// Check that existing values are preserved
	if config2.DefaultLangCode != "fr_FR" {
		t.Fatalf("expected DefaultLangCode 'fr_FR' to be preserved, got '%s'", config2.DefaultLangCode)
	}
	if config2.Prompt.UpdatePot != "custom update pot" {
		t.Fatalf("expected UpdatePot 'custom update pot' to be preserved, got '%s'", config2.Prompt.UpdatePot)
	}
	if len(config2.Agents) != 1 {
		t.Fatalf("expected 1 agent to be preserved, got %d", len(config2.Agents))
	}
	// Check that missing prompt fields are filled
	if config2.Prompt.UpdatePo == "" {
		t.Fatal("Prompt.UpdatePo should be set by applyDefaults")
	}
}

func TestLoadAgentConfig_ConfigOverridesDefaults(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a config file with custom values
	configPath := filepath.Join(tmpDir, "git-po-helper.yaml")
	customYAML := `default_lang_code: "fr_FR"
prompt:
  update_pot: "custom update pot prompt"
agent-test:
  runs: 10
agents:
  custom-agent:
    kind: echo
    cmd: ["custom", "agent", "{prompt}"]
`

	if err := os.WriteFile(configPath, []byte(customYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Mock repository.WorkDir() to return tmpDir
	// We need to test LoadAgentConfig, but it uses repository.WorkDir()
	// For now, let's test the merge and applyDefaults logic directly

	// Test that config values override defaults
	config, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFromFile should succeed, got error: %v", err)
	}

	// Apply defaults
	applyDefaults(config)

	// Check that config values are preserved (not overridden by defaults)
	if config.DefaultLangCode != "fr_FR" {
		t.Fatalf("expected DefaultLangCode 'fr_FR' from config, got '%s'", config.DefaultLangCode)
	}
	if config.Prompt.UpdatePot != "custom update pot prompt" {
		t.Fatalf("expected UpdatePot 'custom update pot prompt' from config, got '%s'", config.Prompt.UpdatePot)
	}
	if config.AgentTest.Runs == nil || *config.AgentTest.Runs != 10 {
		t.Fatalf("expected Runs 10 from config, got %v", config.AgentTest.Runs)
	}

	// Check that config agents are used (not default test agent)
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent from config, got %d", len(config.Agents))
	}
	if _, ok := config.Agents["test"]; ok {
		t.Fatal("expected default 'test' agent to be removed when config has agents")
	}
	if _, ok := config.Agents["custom-agent"]; !ok {
		t.Fatal("expected 'custom-agent' from config")
	}
	if config.Agents["custom-agent"].Cmd[0] != "custom" {
		t.Fatalf("expected custom-agent command, got %v", config.Agents["custom-agent"].Cmd)
	}

	// Check that missing prompt fields are filled with defaults
	if config.Prompt.UpdatePo == "" {
		t.Fatal("Prompt.UpdatePo should be filled with default")
	}
	if config.Prompt.UpdatePo != "Update file \"{{.source}}\" according to @po/AGENTS.md." {
		t.Fatalf("expected UpdatePo default, got '%s'", config.Prompt.UpdatePo)
	}
}

func TestApplyDefaults_ConfigAgentsOverrideDefaultTest(t *testing.T) {
	// Test that when config has agents, default test agent is not added
	config := &AgentConfig{
		Agents: map[string]Agent{
			"claude": {
				Cmd:  []string{"claude", "-p", "{prompt}"},
				Kind: AgentKindClaude,
			},
		},
	}

	applyDefaults(config)

	// Check that only config agents are present
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent from config, got %d", len(config.Agents))
	}
	if _, ok := config.Agents["test"]; ok {
		t.Fatal("expected default 'test' agent to NOT be added when config has agents")
	}
	if _, ok := config.Agents["claude"]; !ok {
		t.Fatal("expected 'claude' agent from config")
	}

	// Test that when config has empty agents, default test agent is added
	config2 := &AgentConfig{
		Agents: make(map[string]Agent),
	}

	applyDefaults(config2)

	// Check that default test agent is added
	if len(config2.Agents) != 1 {
		t.Fatalf("expected 1 default agent, got %d", len(config2.Agents))
	}
	if _, ok := config2.Agents["test"]; !ok {
		t.Fatal("expected default 'test' agent to be added when config has no agents")
	}
}

func TestLoadAgentConfig_PartialConfigWithDefaults(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a partial config file (only agents, missing prompts)
	configPath := filepath.Join(tmpDir, "git-po-helper.yaml")
	partialYAML := `agents:
  my-agent:
    cmd: ["my-agent", "{prompt}"]
`

	if err := os.WriteFile(configPath, []byte(partialYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := loadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFromFile should succeed, got error: %v", err)
	}

	// Apply defaults
	applyDefaults(config)

	// Check that config agents are preserved
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent from config, got %d", len(config.Agents))
	}
	if _, ok := config.Agents["test"]; ok {
		t.Fatal("expected default 'test' agent to NOT be added when config has agents")
	}
	if _, ok := config.Agents["my-agent"]; !ok {
		t.Fatal("expected 'my-agent' from config")
	}

	// Check that missing prompts are filled with defaults
	if config.Prompt.UpdatePot == "" {
		t.Fatal("Prompt.UpdatePot should be filled with default")
	}
	if config.Prompt.UpdatePot != "Update file \"po/git.pot\" according to @po/AGENTS.md." {
		t.Fatalf("expected UpdatePot default, got '%s'", config.Prompt.UpdatePot)
	}

	// Check that default_lang_code is set
	if config.DefaultLangCode == "" {
		t.Fatal("DefaultLangCode should be set by applyDefaults")
	}

	// Check that runs is set
	if config.AgentTest.Runs == nil {
		t.Fatal("AgentTest.Runs should be set by applyDefaults")
	}
	if *config.AgentTest.Runs != 1 {
		t.Fatalf("expected Runs default 1, got %d", *config.AgentTest.Runs)
	}
}
