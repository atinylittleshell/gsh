package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// ExecuteNativeExecTool executes a shell command with PTY support.
// This is the shared implementation used by both gsh.tools.exec and the REPL agent.
// The working_directory must be provided in args and must be an absolute path.
func ExecuteNativeExecTool(ctx context.Context, args map[string]interface{}, liveOutput io.Writer) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("exec tool requires 'command' argument as string")
	}

	// Get working_directory (required, must be absolute path)
	workingDir, ok := args["working_directory"].(string)
	if !ok || workingDir == "" {
		return "", fmt.Errorf("exec tool requires 'working_directory' argument as string")
	}
	if !filepath.IsAbs(workingDir) {
		return "", fmt.Errorf("exec tool requires 'working_directory' to be an absolute path, got: %s", workingDir)
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
	result, err := ExecuteCommandWithPTY(execCtx, command, liveOutput, workingDir)

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

// ExecResult contains the result of executing a shell command.
type ExecResult struct {
	Output   string // Combined stdout/stderr (PTY combines them)
	ExitCode int
}

// ExecuteCommandWithPTY runs a shell command with PTY support.
// This allows live output display while also capturing the output.
// The liveOutput writer receives output in real-time as the command runs.
// If liveOutput is nil, output is only captured (not displayed).
// The workingDir parameter specifies the directory to run the command in.
func ExecuteCommandWithPTY(ctx context.Context, command string, liveOutput io.Writer, workingDir string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// Set working directory
	cmd.Dir = workingDir

	// Set environment variables to disable interactive pagers and prompts
	cmd.Env = append(os.Environ(),
		"PAGER=cat",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
	)

	// Create PTY for the command
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Set PTY size
	cols, rows := 80, 24
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 && height > 0 {
		cols, rows = width, height
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})

	// Capture output while also writing to provided writer
	var outputBuf bytes.Buffer
	var mu sync.Mutex

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
		_, _ = io.Copy(writer, ptmx)
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for copy to finish
	<-copyDone

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
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

const execToolName = "exec"
const execToolDescription = "Execute a bash command and return the output."

func execToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"working_directory": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the directory where the command should be executed",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds for the command execution. Defaults to 60 seconds if not specified.",
			},
		},
		"required": []string{"command", "working_directory"},
	}
}

// ExecToolDefinition returns the ChatTool definition for the exec tool.
func ExecToolDefinition() ChatTool {
	return ChatTool{
		Name:        execToolName,
		Description: execToolDescription,
		Parameters:  execToolParameters(),
	}
}

// CreateExecNativeTool creates the exec native tool for use in gsh.tools.
// Output is streamed to stderr in real-time.
func CreateExecNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        execToolName,
		Description: execToolDescription,
		Parameters:  execToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeExecTool(context.Background(), args, os.Stderr)
		},
	}
}
