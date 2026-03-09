package util

import (
	"testing"

	"github.com/git-l10n/git-po-helper/config"
)

func TestNewAgentFromConfig(t *testing.T) {
	tests := []struct {
		entry config.AgentEntry
		want  string
	}{
		{config.AgentEntry{Cmd: []string{"claude", "-p", "{{.prompt}}"}, Kind: config.AgentKindClaude}, config.OutputStreamJSON},
		{config.AgentEntry{Cmd: []string{"claude", "--output-format", "text", "-p", "{{.prompt}}"}, Kind: config.AgentKindClaude}, config.OutputText},
		{config.AgentEntry{Cmd: []string{"echo", "{{.prompt}}"}, Kind: config.AgentKindEcho}, config.OutputText},
		{config.AgentEntry{Cmd: []string{"codex", "exec", "{{.prompt}}"}, Kind: config.AgentKindCodex}, config.OutputStreamJSON},
		{config.AgentEntry{Cmd: []string{"qodercli", "-p", "{{.prompt}}"}, Kind: config.AgentKindQoder}, config.OutputStreamJSON},
	}
	for i, tt := range tests {
		agent, err := NewAgentFromConfig(tt.entry)
		if err != nil {
			t.Errorf("case %d: NewAgentFromConfig: %v", i, err)
			continue
		}
		got := agent.GetOutputFormat()
		if got != tt.want {
			t.Errorf("case %d: GetOutputFormat() = %q, want %q", i, got, tt.want)
		}
	}
}

func TestAgentBuildCommand(t *testing.T) {
	// Test agent.BuildCommand + agent.GetOutputFormat flow (design doc 3.7)
	entry := config.AgentEntry{
		Cmd:  []string{"claude", "-p", "{{.prompt}}"},
		Kind: config.AgentKindClaude,
	}
	agent, err := NewAgentFromConfig(entry)
	if err != nil {
		t.Fatalf("NewAgentFromConfig: %v", err)
	}
	vars := PlaceholderVars{"prompt": "hello"}
	cmd, err := agent.BuildCommand(map[string]string(vars))
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	outputFormat := agent.GetOutputFormat()
	if outputFormat != config.OutputStreamJSON {
		t.Errorf("GetOutputFormat() = %q, want %q", outputFormat, config.OutputStreamJSON)
	}
	if len(cmd) < 4 {
		t.Errorf("expected cmd to have --verbose --output-format stream-json appended, got %v", cmd)
	}
	found := false
	for _, arg := range cmd {
		if arg == "hello" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'hello' in cmd, got %v", cmd)
	}
}

func TestBuildAgentCommand(t *testing.T) {
	entry := config.AgentEntry{
		Cmd:  []string{"claude", "-p", "{{.prompt}}"},
		Kind: config.AgentKindClaude,
	}
	vars := PlaceholderVars{"prompt": "hello"}
	cmd, outputFormat, err := BuildAgentCommand(entry, vars)
	if err != nil {
		t.Fatalf("BuildAgentCommand: %v", err)
	}
	if outputFormat != config.OutputStreamJSON {
		t.Errorf("outputFormat = %q, want %q", outputFormat, config.OutputStreamJSON)
	}
	if len(cmd) < 4 {
		t.Errorf("expected cmd to have --verbose --output-format stream-json appended, got %v", cmd)
	}
	// Check placeholder was replaced
	found := false
	for _, arg := range cmd {
		if arg == "hello" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'hello' in cmd, got %v", cmd)
	}
}
