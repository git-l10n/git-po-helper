// Package util provides utility functions for agent execution.
package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// flushStdout flushes stdout to ensure agent output (🤖 etc.) is visible immediately.
// Without this, stdout may be buffered when not a TTY, causing output to appear only with -v
// (which produces more stderr activity that can trigger flushing in some environments).
func flushStdout() {
	_ = os.Stdout.Sync()
}

// ExecuteAgentCommand executes an agent command and captures both stdout and stderr.
// The command is executed in the specified working directory.
//
// Parameters:
//   - cmd: Command and arguments as a slice (e.g., []string{"claude", "-p", "{{.prompt}}"})
//   - workDir: Working directory for command execution (empty string uses current working directory).
//     To use repository root, pass repository.WorkDir() explicitly.
//
// Returns:
//   - stdout: Standard output from the command
//   - stderr: Standard error from the command
//   - error: Error if command execution fails (includes non-zero exit codes)
//
// The function:
//   - Replaces placeholders in command arguments using ReplacePlaceholders
//   - Executes the command in the specified working directory
//   - Captures both stdout and stderr separately
//   - Returns an error if the command exits with a non-zero status code
func ExecuteAgentCommand(cmd []string) ([]byte, []byte, error) {
	if len(cmd) == 0 {
		return nil, nil, fmt.Errorf("command cannot be empty")
	}

	cwd, _ := os.Getwd()

	// Replace placeholders in command arguments
	// Note: Placeholders should be replaced before calling this function,
	// but we'll handle it here for safety
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	log.Debugf("executing agent command: %s (workDir: %s)", strings.Join(cmd, " "), cwd)

	// Capture stdout and stderr separately
	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	// Execute the command
	err := execCmd.Run()
	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()

	// Check for execution errors
	if err != nil {
		// If command exited with non-zero status, include stderr in error message
		if exitError, ok := err.(*exec.ExitError); ok {
			return stdout, stderr, fmt.Errorf("agent command failed with exit code %d: %w\nstderr: %s",
				exitError.ExitCode(), err, string(stderr))
		}
		return stdout, stderr, fmt.Errorf("failed to execute agent command: %w\nstderr: %s", err, string(stderr))
	}

	log.Debugf("agent command completed successfully (stdout: %d bytes, stderr: %d bytes)",
		len(stdout), len(stderr))

	return stdout, stderr, nil
}

// ExecuteAgentCommandStream executes an agent command and returns a reader for real-time stdout streaming.
// The command is executed in the specified working directory.
// This function is used for json format (stream-json internally) to process output in real-time.
//
// Parameters:
//   - cmd: Command and arguments as a slice
//   - workDir: Working directory for command execution
//
// Returns:
//   - stdoutReader: io.ReadCloser for reading stdout in real-time
//   - stderr: Standard error from the command (captured after execution)
//   - cmdProcess: *exec.Cmd for waiting on command completion
//   - error: Error if command setup fails
func ExecuteAgentCommandStream(cmd []string) (stdoutReader io.ReadCloser, stderrBuf *bytes.Buffer, cmdProcess *exec.Cmd, err error) {
	if len(cmd) == 0 {
		return nil, nil, nil, fmt.Errorf("command cannot be empty")
	}

	// Create command
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	log.Debugf("executing agent command (streaming): %s", strings.Join(cmd, " "))

	// Get stdout pipe for real-time reading
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr separately
	var stderrBuffer bytes.Buffer
	execCmd.Stderr = &stderrBuffer

	// Start command execution
	if err := execCmd.Start(); err != nil {
		stdoutPipe.Close()
		return nil, nil, nil, fmt.Errorf("failed to start agent command: %w", err)
	}

	return stdoutPipe, &stderrBuffer, execCmd, nil
}
