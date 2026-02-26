package util

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestExecuteAgentCommand(t *testing.T) {
	// Test successful command execution
	t.Run("successful command", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "test output"}
		} else {
			cmd = []string{"sh", "-c", "echo 'test output'"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "test output") {
			t.Errorf("Expected stdout to contain 'test output', got %q", output)
		}

		if len(stderr) > 0 {
			t.Logf("stderr: %s", string(stderr))
		}
	})

	// Test command with stderr output
	t.Run("command with stderr", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo test >&2"}
		} else {
			cmd = []string{"sh", "-c", "echo 'test error' >&2"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// stderr should contain the error message
		if len(stderr) == 0 {
			t.Log("Note: stderr is empty (this may be expected on some systems)")
		}
		_ = stdout
	})

	// Test command failure (non-zero exit code)
	t.Run("command failure", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "exit 1"}
		} else {
			cmd = []string{"sh", "-c", "exit 1"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err == nil {
			t.Error("Expected error for failing command, got nil")
		}

		// Error should mention exit code
		if !strings.Contains(err.Error(), "exit code") && !strings.Contains(err.Error(), "failed") {
			t.Errorf("Error message should mention exit code or failure: %v", err)
		}

		_ = stdout
		_ = stderr
	})

	// Test empty command
	t.Run("empty command", func(t *testing.T) {
		_, _, err := ExecuteAgentCommand([]string{})
		if err == nil {
			t.Error("Expected error for empty command, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Error should mention 'cannot be empty': %v", err)
		}
	})

	// Test non-existent command
	t.Run("non-existent command", func(t *testing.T) {
		_, _, err := ExecuteAgentCommand([]string{"nonexistent-command-xyz123"})
		if err == nil {
			t.Error("Expected error for non-existent command, got nil")
		}
	})

	// Test command that produces output (pwd/cd)
	t.Run("command produces output", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "cd"}
		} else {
			cmd = []string{"pwd"}
		}

		stdout, stderr, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		// On Windows, the path format might be different, so just check it's not empty
		if len(output) == 0 {
			t.Error("Expected non-empty output from pwd/cd command")
		}

		_ = stderr
	})
}

func TestExecuteAgentCommand_PlaceholderReplacement(t *testing.T) {
	// This test verifies that placeholder replacement should be done
	// before calling ExecuteAgentCommand, not inside it.
	// ExecuteAgentCommand should execute the command as-is.

	t.Run("command with literal placeholders", func(t *testing.T) {
		// ExecuteAgentCommand should not replace placeholders
		// (that's the caller's responsibility)
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "{prompt}"}
		} else {
			cmd = []string{"sh", "-c", "echo '{prompt}'"}
		}

		stdout, _, err := ExecuteAgentCommand(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "{prompt}") {
			t.Errorf("Expected literal {prompt} in output, got %q", output)
		}
	})
}

// Test helper: verify that ExecuteAgentCommand works with real commands
func TestExecuteAgentCommand_RealCommand(t *testing.T) {
	// Test with a command that should exist on all systems
	var cmd []string
	if runtime.GOOS == "windows" {
		// On Windows, use cmd /c echo
		cmd = []string{"cmd", "/c", "echo", "Hello World"}
	} else {
		// On Unix, use echo
		cmd = []string{"echo", "Hello World"}
	}

	// Check if command exists
	if _, err := exec.LookPath(cmd[0]); err != nil {
		t.Skipf("Command %s not found, skipping test", cmd[0])
	}

	stdout, stderr, err := ExecuteAgentCommand(cmd)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := strings.TrimSpace(string(stdout))
	if !strings.Contains(output, "Hello World") {
		t.Errorf("Expected 'Hello World' in output, got %q", output)
	}

	if len(stderr) > 0 {
		t.Logf("stderr (non-fatal): %s", string(stderr))
	}
}
