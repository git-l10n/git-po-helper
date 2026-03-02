package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/repository"
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

func TestLoadConfigFromFile_ValidFile(t *testing.T) {
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
  update_po: "update {{.source}} according to po/README.md"
agents:
  claude:
    cmd: ["claude", "-p", "{{.prompt}}"]
  gemini:
    cmd: ["gemini", "--prompt", "{{.prompt}}"]
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

func TestLoadConfigFromFile_InvalidYAML(t *testing.T) {
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
    cmd: ["claude", "-p", "{{.prompt}}"]
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

func TestMergeConfigs(t *testing.T) {
	baseConfig := &AgentConfig{
		DefaultLangCode: "en_US",
		Prompt: PromptConfig{
			UpdatePot: "base update pot",
			UpdatePo:  "base update po",
		},
		Agents: map[string]Agent{
			"claude": {
				Cmd:  []string{"claude", "-p", "{{.prompt}}"},
				Kind: AgentKindClaude,
			},
			"gemini": {
				Cmd:  []string{"gemini", "--prompt", "{{.prompt}}"},
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
				Cmd:  []string{"claude", "--new-flag", "{{.prompt}}"},
				Kind: AgentKindClaude,
			},
		},
	}

	merged := mergeConfigs(baseConfig, repoConfig, true)

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

func TestApplyDefaults_ConfigAgentsOverrideDefaultTest(t *testing.T) {
	// When config has agents, mergeConfigs(..., false) keeps config's agents (does not replace with default)
	config := &AgentConfig{
		Agents: map[string]Agent{
			"claude": {
				Cmd:  []string{"claude", "-p", "{{.prompt}}"},
				Kind: AgentKindClaude,
			},
		},
	}
	defaultCfg := getDefaultConfig()
	merged := mergeConfigs(config, defaultCfg, false)

	if len(merged.Agents) != 1 {
		t.Fatalf("expected 1 agent from config, got %d", len(merged.Agents))
	}
	if _, ok := merged.Agents["test"]; ok {
		t.Fatal("expected default 'test' agent to NOT be added when config has agents")
	}
	if _, ok := merged.Agents["claude"]; !ok {
		t.Fatal("expected 'claude' agent from config")
	}

	// When config has no agents, mergeConfigs(..., false) does not touch Agents (stays empty);
	// LoadAgentConfig copies default's Agents after all merges when merged.Agents is empty.
	config2 := &AgentConfig{
		Agents: make(map[string]Agent),
	}
	merged2 := mergeConfigs(config2, getDefaultConfig(), false)

	if len(merged2.Agents) != 0 {
		t.Fatalf("mergeConfigs(..., false) must not merge Agents; got %d agents", len(merged2.Agents))
	}
}

func TestMergeConfigs_NilBase(t *testing.T) {
	defaultCfg := getDefaultConfig()

	// mergeAgents true: result has overlay's values and overlay's agents
	merged := mergeConfigs(nil, defaultCfg, true)
	if merged.DefaultLangCode != defaultCfg.DefaultLangCode {
		t.Fatalf("expected DefaultLangCode from overlay, got %q", merged.DefaultLangCode)
	}
	if merged.Prompt.UpdatePot != defaultCfg.Prompt.UpdatePot {
		t.Fatalf("expected UpdatePot from overlay, got %q", merged.Prompt.UpdatePot)
	}
	if len(merged.Agents) != len(defaultCfg.Agents) {
		t.Fatalf("expected %d agents from overlay, got %d", len(defaultCfg.Agents), len(merged.Agents))
	}

	// mergeAgents false: result has overlay's non-Agent fields, Agents not merged (stay empty)
	merged2 := mergeConfigs(nil, defaultCfg, false)
	if merged2.DefaultLangCode != defaultCfg.DefaultLangCode {
		t.Fatalf("expected DefaultLangCode from overlay, got %q", merged2.DefaultLangCode)
	}
	if len(merged2.Agents) != 0 {
		t.Fatalf("mergeConfigs(nil, overlay, false) must not set Agents; got %d", len(merged2.Agents))
	}
}

func TestMergeConfigs_NilOverlay(t *testing.T) {
	base := &AgentConfig{
		DefaultLangCode: "fr_FR",
		Prompt:          PromptConfig{UpdatePot: "base pot"},
		Agents: map[string]Agent{
			"x": {Cmd: []string{"x"}, Kind: AgentKindEcho},
		},
	}

	merged := mergeConfigs(base, nil, true)
	if merged.DefaultLangCode != "fr_FR" {
		t.Fatalf("expected base DefaultLangCode fr_FR, got %q", merged.DefaultLangCode)
	}
	if merged.Prompt.UpdatePot != "base pot" {
		t.Fatalf("expected base UpdatePot, got %q", merged.Prompt.UpdatePot)
	}
	if len(merged.Agents) != 1 || merged.Agents["x"].Cmd[0] != "x" {
		t.Fatalf("expected base agent x, got %v", merged.Agents)
	}

	merged2 := mergeConfigs(base, nil, false)
	if merged2.DefaultLangCode != "fr_FR" || len(merged2.Agents) != 1 {
		t.Fatalf("expected base unchanged, got DefaultLangCode=%q agents=%d", merged2.DefaultLangCode, len(merged2.Agents))
	}
}

func TestMergeConfigs_DefaultMergeOverwritesNonAgents(t *testing.T) {
	// When mergeAgents is false, overlay still overwrites non-Agent fields when overlay has value; base's Agents are kept.
	base := &AgentConfig{
		DefaultLangCode: "fr_FR",
		Prompt:          PromptConfig{UpdatePot: "base pot", UpdatePo: "base po"},
		Agents: map[string]Agent{
			"custom": {Cmd: []string{"custom", "{{.prompt}}"}, Kind: AgentKindEcho},
		},
	}
	defaultCfg := getDefaultConfig()

	merged := mergeConfigs(base, defaultCfg, false)

	// Non-Agent fields come from overlay (default) when overlay has value
	if merged.DefaultLangCode != defaultCfg.DefaultLangCode {
		t.Fatalf("expected default DefaultLangCode, got %q", merged.DefaultLangCode)
	}
	if merged.Prompt.UpdatePot != defaultCfg.Prompt.UpdatePot {
		t.Fatalf("expected default UpdatePot, got %q", merged.Prompt.UpdatePot)
	}
	// Agents from base only (overlay Agents not merged when mergeAgents false)
	if len(merged.Agents) != 1 {
		t.Fatalf("expected 1 agent from base, got %d", len(merged.Agents))
	}
	if _, ok := merged.Agents["custom"]; !ok {
		t.Fatal("expected base agent 'custom' preserved")
	}
}

func TestMergeConfigs_AgentTestFields(t *testing.T) {
	runs := 3
	potBefore := 10
	base := &AgentConfig{
		AgentTest: AgentTestConfig{Runs: func() *int { x := 1; return &x }()},
	}
	overlay := &AgentConfig{
		AgentTest: AgentTestConfig{
			Runs:                   &runs,
			PotEntriesBeforeUpdate: &potBefore,
		},
	}

	merged := mergeConfigs(base, overlay, true)
	if merged.AgentTest.Runs == nil || *merged.AgentTest.Runs != 3 {
		t.Fatalf("expected Runs 3, got %v", merged.AgentTest.Runs)
	}
	if merged.AgentTest.PotEntriesBeforeUpdate == nil || *merged.AgentTest.PotEntriesBeforeUpdate != 10 {
		t.Fatalf("expected PotEntriesBeforeUpdate 10, got %v", merged.AgentTest.PotEntriesBeforeUpdate)
	}

	// mergeAgents false: overlay still overwrites *int when overlay has value
	merged2 := mergeConfigs(base, overlay, false)
	if merged2.AgentTest.Runs == nil || *merged2.AgentTest.Runs != 3 {
		t.Fatalf("expected Runs 3 with mergeAgents false, got %v", merged2.AgentTest.Runs)
	}
	if merged2.AgentTest.PotEntriesBeforeUpdate == nil || *merged2.AgentTest.PotEntriesBeforeUpdate != 10 {
		t.Fatalf("expected PotEntriesBeforeUpdate 10, got %v", merged2.AgentTest.PotEntriesBeforeUpdate)
	}
}

func TestLoadAgentConfig_CustomPath_ValidConfig(t *testing.T) {
	repository.OpenRepository("")
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "custom.yaml")
	yaml := `default_lang_code: "de_DE"
prompt:
  update_pot: "custom update pot"
agents:
  my-agent:
    cmd: ["my-agent", "{{.prompt}}"]
    kind: echo
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadAgentConfig returned nil config")
	}
	// Custom config overrides: default_lang_code and prompt from custom file (merge order: default then user/repo/custom)
	if cfg.DefaultLangCode != "de_DE" {
		t.Fatalf("expected default_lang_code de_DE, got %q", cfg.DefaultLangCode)
	}
	if cfg.Prompt.UpdatePot != "custom update pot" {
		t.Fatalf("expected custom UpdatePot, got %q", cfg.Prompt.UpdatePot)
	}
	// Custom defines my-agent; it is merged with user/repo agents by key (custom has highest priority)
	if _, ok := cfg.Agents["my-agent"]; !ok {
		t.Fatal("expected my-agent from custom config")
	}
	if cfg.Agents["my-agent"].Cmd[0] != "my-agent" {
		t.Fatalf("expected my-agent command, got %v", cfg.Agents["my-agent"].Cmd)
	}
}

func TestLoadAgentConfig_CustomPath_PartialConfigGetsDefaultAgents(t *testing.T) {
	repository.OpenRepository("")
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Partial config: only default_lang_code, no agents. After merge, Agents is empty so LoadAgentConfig copies default's Agents.
	configPath := filepath.Join(tmpDir, "partial.yaml")
	yaml := `default_lang_code: "it_IT"
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadAgentConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAgentConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadAgentConfig returned nil config")
	}
	if cfg.DefaultLangCode != "it_IT" {
		t.Fatalf("expected default_lang_code it_IT from custom file, got %q", cfg.DefaultLangCode)
	}
	// Partial config has no agents; LoadAgentConfig copies default's Agents when merged.Agents is empty.
	// (If user/repo configs added agents, we'd have those; we only require at least one agent here.)
	if len(cfg.Agents) < 1 {
		t.Fatalf("expected at least one agent (default copy or from repo), got %d", len(cfg.Agents))
	}
}

func TestLoadAgentConfig_CustomPath_MissingFile(t *testing.T) {
	repository.OpenRepository("")
	tmpDir, err := os.MkdirTemp("", "git-po-helper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "nonexistent.yaml")
	_, err = LoadAgentConfig(configPath)
	if err == nil {
		t.Fatal("LoadAgentConfig expected error for missing custom file")
	}
}
