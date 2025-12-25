// Package agent provides the exec tool for executing shell commands.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/creack/pty"
)

// DefaultExecTimeout is the default timeout for command execution.
const DefaultExecTimeout = 60 * time.Second

// ExecResult contains the result of executing a shell command.
type ExecResult struct {
	Output   string // Combined stdout/stderr (PTY combines them)
	ExitCode int
}

// ExecuteCommand runs a shell command with PTY support.
// This allows live output display while also capturing the output.
// The liveOutput writer receives output in real-time as the command runs.
// If liveOutput is nil, output is only captured (not displayed).
// Note: stdin is not supported - commands that require interactive input will not work.
func ExecuteCommand(ctx context.Context, command string, liveOutput io.Writer) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// Set environment variables to disable interactive pagers and prompts
	// but preserve color output (PTY provides terminal capabilities)
	cmd.Env = append(os.Environ(),
		"PAGER=cat",             // Disable general pager
		"GIT_PAGER=cat",         // Disable git pager
		"GIT_TERMINAL_PROMPT=0", // Disable git credential prompts
	)

	// Create PTY for the command (for proper terminal output handling)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Capture output while also writing to provided writer
	var outputBuf bytes.Buffer
	var mu sync.Mutex // Protect concurrent access to outputBuf

	// Use MultiWriter to write to both the live output and capture buffer
	var writer io.Writer
	if liveOutput != nil {
		writer = io.MultiWriter(&safeWriter{w: liveOutput, mu: &mu}, &safeWriter{w: &outputBuf, mu: &mu})
	} else {
		writer = &outputBuf
	}

	// Copy PTY output in a goroutine
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		_, _ = io.Copy(writer, ptmx) // Error is expected when PTY closes
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for copy to finish (PTY will close after command exits)
	<-copyDone

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			// Context was cancelled
			return nil, ctx.Err()
		} else {
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	mu.Lock()
	output := outputBuf.String()
	mu.Unlock()

	return &ExecResult{
		Output:   output,
		ExitCode: exitCode,
	}, nil
}

// safeWriter wraps an io.Writer with mutex protection for thread safety.
type safeWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (sw *safeWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// ExecToolDefinition returns the tool definition for the exec tool.
func ExecToolDefinition() interpreter.ChatTool {
	return interpreter.ChatTool{
		Name:        "exec",
		Description: "Execute a bash command and return the output.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds for the command execution. Defaults to 60 seconds if not specified.",
				},
			},
			"required": []string{"command"},
		},
	}
}

// ExecuteExecTool handles execution of the exec tool.
func ExecuteExecTool(ctx context.Context, args map[string]interface{}, liveOutput io.Writer) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("exec tool requires 'command' argument as string")
	}

	// Parse timeout (optional, defaults to DefaultExecTimeout)
	timeout := DefaultExecTimeout
	if timeoutVal, ok := args["timeout"]; ok {
		switch v := timeoutVal.(type) {
		case float64:
			timeout = time.Duration(v) * time.Second
		case int:
			timeout = time.Duration(v) * time.Second
		case int64:
			timeout = time.Duration(v) * time.Second
		}
	}

	// Create a timeout context
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with live output
	result, err := ExecuteCommand(execCtx, command, liveOutput)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	// Truncate very long outputs to avoid overwhelming the model
	output := result.Output
	const maxOutputLen = 50000 // ~50KB limit
	truncated := false
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen]
		truncated = true
	}

	// Return result as JSON for the agent
	if truncated {
		return fmt.Sprintf(`{"output": %q, "exitCode": %d, "truncated": true}`, output, result.ExitCode), nil
	}
	return fmt.Sprintf(`{"output": %q, "exitCode": %d}`, output, result.ExitCode), nil
}
