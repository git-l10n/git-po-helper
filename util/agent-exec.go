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
