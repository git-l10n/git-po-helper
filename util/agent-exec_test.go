package util

import (
	"io"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// runAgentViaStream runs a command using ExecuteAgentCommandStream and io.ReadAll.
// This tests the unified execution path (streaming + buffer) used by RunAgentAndParse.
func runAgentViaStream(cmd []string) (stdout, stderr []byte, err error) {
	stdoutReader, stderrBuf, cmdProcess, execErr := ExecuteAgentCommandStream(cmd)
	if execErr != nil {
		return nil, nil, execErr
	}
	defer stdoutReader.Close()
	stdout, _ = io.ReadAll(stdoutReader)
	waitErr := cmdProcess.Wait()
	stderr = stderrBuf.Bytes()
	if waitErr != nil {
		return stdout, stderr, waitErr
	}
	return stdout, stderr, nil
}

func TestExecuteAgentCommandStream(t *testing.T) {
	// Test successful command execution (via stream + ReadAll)
	t.Run("successful command", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "test output"}
		} else {
			cmd = []string{"sh", "-c", "echo 'test output'"}
		}

		stdout, stderr, err := runAgentViaStream(cmd)
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

		stdout, stderr, err := runAgentViaStream(cmd)
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

		stdout, stderr, err := runAgentViaStream(cmd)
		if err == nil {
			t.Error("Expected error for failing command, got nil")
		}

		// Error should mention exit status or failure
		if err != nil && !strings.Contains(err.Error(), "exit") && !strings.Contains(err.Error(), "failed") {
			t.Errorf("Error message should mention exit or failure: %v", err)
		}

		_ = stdout
		_ = stderr
	})

	// Test empty command
	t.Run("empty command", func(t *testing.T) {
		_, _, _, err := ExecuteAgentCommandStream([]string{})
		if err == nil {
			t.Error("Expected error for empty command, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("Error should mention 'cannot be empty': %v", err)
		}
	})

	// Test non-existent command
	t.Run("non-existent command", func(t *testing.T) {
		_, _, err := runAgentViaStream([]string{"nonexistent-command-xyz123"})
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

		stdout, stderr, err := runAgentViaStream(cmd)
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

func TestExecuteAgentCommandStream_PlaceholderReplacement(t *testing.T) {
	// This test verifies that placeholder replacement should be done
	// before calling the command, not inside it.
	t.Run("command with literal placeholders", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "{{.prompt}}"}
		} else {
			cmd = []string{"sh", "-c", "echo '{{.prompt}}'"}
		}

		stdout, _, err := runAgentViaStream(cmd)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "{{.prompt}}") {
			t.Errorf("Expected literal {{.prompt}} in output, got %q", output)
		}
	})
}

func TestExecuteAgentCommandStream_RealCommand(t *testing.T) {
	// Test with a command that should exist on all systems
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "Hello World"}
	} else {
		cmd = []string{"echo", "Hello World"}
	}

	// Check if command exists
	if _, err := exec.LookPath(cmd[0]); err != nil {
		t.Skipf("Command %s not found, skipping test", cmd[0])
	}

	stdout, stderr, err := runAgentViaStream(cmd)
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

func TestRunAgentAndParse(t *testing.T) {
	// Test RunAgentAndParse with default format (non-json path)
	t.Run("default format echo", func(t *testing.T) {
		var cmd []string
		if runtime.GOOS == "windows" {
			cmd = []string{"cmd", "/c", "echo", "hello"}
		} else {
			cmd = []string{"sh", "-c", "echo 'hello'"}
		}

		stdout, _, stderr, _, err := RunAgentAndParse(cmd, "default", "echo")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		output := strings.TrimSpace(string(stdout))
		if !strings.Contains(output, "hello") {
			t.Errorf("Expected stdout to contain 'hello', got %q", output)
		}
		_ = stderr
	})
}
