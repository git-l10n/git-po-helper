package util

import (
	"strings"
	"testing"

	"github.com/git-l10n/git-po-helper/config"
	"github.com/spf13/viper"
)

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		template string
		kv       PlaceholderVars
		expected string
		wantErr  bool
	}{
		{
			name:     "all placeholders",
			template: "cmd -p {{.prompt}} -s {{.source}} -c {{.commit}}",
			kv:       PlaceholderVars{"prompt": "update pot", "source": "po/zh_CN.po", "commit": "HEAD"},
			expected: "cmd -p update pot -s po/zh_CN.po -c HEAD",
		},
		{
			name:     "only prompt placeholder",
			template: "cmd -p {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "update pot"},
			expected: "cmd -p update pot",
		},
		{
			name:     "multiple occurrences",
			template: "{{.prompt}} {{.prompt}} {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "test"},
			expected: "test test test",
		},
		{
			name:     "empty values",
			template: "cmd -p {{.prompt}} -s {{.source}} -c {{.commit}}",
			kv:       PlaceholderVars{"prompt": "", "source": "", "commit": ""},
			expected: "cmd -p  -s  -c ",
		},
		{
			name:     "no placeholders",
			template: "cmd -p test",
			kv:       PlaceholderVars{"prompt": "update pot", "source": "po/zh_CN.po", "commit": "HEAD"},
			expected: "cmd -p test",
		},
		{
			name:     "special characters in values",
			template: "cmd -p {{.prompt}}",
			kv:       PlaceholderVars{"prompt": "update 'pot' file"},
			expected: "cmd -p update 'pot' file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReplacePlaceholders(tt.template, tt.kv)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplacePlaceholders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExecutePromptTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     PlaceholderVars
		expected string
		wantErr  bool
	}{
		{
			name:     "source and dest",
			template: `Review "{{.source}}" and fix in "{{.dest}}"`,
			vars:     PlaceholderVars{"prompt": "ignored", "source": "po/zh_CN.po", "dest": "po/zh_CN.po"},
			expected: `Review "po/zh_CN.po" and fix in "po/zh_CN.po"`,
		},
		{
			name:     "no template vars",
			template: "Update file po/git.pot",
			vars:     PlaceholderVars{"prompt": "x"},
			expected: "Update file po/git.pot",
		},
		{
			name:     "literal braces",
			template: `Placeholders like {{` + "`{name}`" + `}} preserved`,
			vars:     PlaceholderVars{},
			expected: `Placeholders like {name} preserved`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExecutePromptTemplate(tt.template, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecutePromptTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetPrompt(t *testing.T) {
	tests := []struct {
		name             string
		action           string
		cfg              *config.AgentConfig
		agentRunPrompt   string
		agentTestPrompt  string
		expected         string
		expectedContains string // if set and expected is empty, check result contains this
		expectError      bool
		errorContains    string
	}{
		{
			name:   "use config prompt when no override",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config update pot prompt",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "config update pot prompt",
			expectError:     false,
		},
		{
			name:   "override with agent-run--prompt",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config update pot prompt",
				},
			},
			agentRunPrompt:  "override prompt from agent-run",
			agentTestPrompt: "",
			expected:        "override prompt from agent-run",
			expectError:     false,
		},
		{
			name:   "override with agent-test--prompt when agent-run--prompt is empty",
			action: "update-po",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePo: "config update po prompt",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "override prompt from agent-test",
			expected:        "override prompt from agent-test",
			expectError:     false,
		},
		{
			name:   "agent-run--prompt takes priority over agent-test--prompt",
			action: "translate",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					Translate: "config translate prompt",
				},
			},
			agentRunPrompt:  "override from agent-run",
			agentTestPrompt: "override from agent-test",
			expected:        "override from agent-run",
			expectError:     false,
		},
		{
			name:   "error when config prompt is empty and no override",
			action: "review",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					Review: "",
				},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "",
			expectError:     true,
			errorContains:   "prompt.review is not configured",
		},
		{
			name:   "error for unknown action",
			action: "unknown-action",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{},
			},
			agentRunPrompt:  "",
			agentTestPrompt: "",
			expected:        "",
			expectError:     true,
			errorContains:   "unknown action",
		},
		{
			name:   "override works for all actions",
			action: "update-pot",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePot: "config prompt",
					UpdatePo:  "config prompt",
					Translate: "config prompt",
					Review:    "config prompt",
				},
			},
			agentRunPrompt:  "override prompt",
			agentTestPrompt: "",
			expected:        "override prompt",
			expectError:     false,
		},
		{
			name:   "override works even when config is empty",
			action: "update-po",
			cfg: &config.AgentConfig{
				Prompt: config.PromptConfig{
					UpdatePo: "",
				},
			},
			agentRunPrompt:  "override prompt",
			agentTestPrompt: "",
			expected:        "override prompt",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original viper values
			originalAgentRunPrompt := viper.GetString("agent-run--prompt")
			originalAgentTestPrompt := viper.GetString("agent-test--prompt")

			// Set viper values for test
			if tt.agentRunPrompt != "" {
				viper.Set("agent-run--prompt", tt.agentRunPrompt)
			} else {
				viper.Set("agent-run--prompt", "")
			}
			if tt.agentTestPrompt != "" {
				viper.Set("agent-test--prompt", tt.agentTestPrompt)
			} else {
				viper.Set("agent-test--prompt", "")
			}

			// Run test
			result, err := GetRawPrompt(tt.cfg, tt.action)

			// Restore original viper values
			if originalAgentRunPrompt != "" {
				viper.Set("agent-run--prompt", originalAgentRunPrompt)
			} else {
				viper.Set("agent-run--prompt", "")
			}
			if originalAgentTestPrompt != "" {
				viper.Set("agent-test--prompt", originalAgentTestPrompt)
			} else {
				viper.Set("agent-test--prompt", "")
			}

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedContains != "" {
				if !strings.Contains(result, tt.expectedContains) {
					t.Errorf("Expected result to contain %q, got %q", tt.expectedContains, result)
				}
			} else if result != tt.expected {
				t.Errorf("Expected prompt %q, got %q", tt.expected, result)
			}
		})
	}
}
