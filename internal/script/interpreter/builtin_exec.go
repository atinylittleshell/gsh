package interpreter

import (
	"context"
	"fmt"
	"time"

	"github.com/atinylittleshell/gsh/internal/bash"
)

// builtinExec implements the exec() function for executing shell commands
// exec(command: string, options?: {timeout?: number}): {stdout: string, stderr: string, exitCode: number}
func (i *Interpreter) builtinExec(args []Value) (Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("exec() takes 1 or 2 arguments (command: string, options?: object), got %d", len(args))
	}

	// First argument: command (string)
	cmdValue, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("exec() first argument must be a string, got %s", args[0].Type())
	}
	command := cmdValue.Value

	// Second argument (optional): options object
	timeout := 60 * time.Second // Default timeout
	if len(args) == 2 {
		optsValue, ok := args[1].(*ObjectValue)
		if !ok {
			return nil, fmt.Errorf("exec() second argument must be an object, got %s", args[1].Type())
		}

		// Parse timeout option if provided
		timeoutVal := optsValue.GetPropertyValue("timeout")
		if timeoutVal.Type() != ValueTypeNull {
			if timeoutNum, ok := timeoutVal.(*NumberValue); ok {
				timeout = time.Duration(timeoutNum.Value) * time.Millisecond
			} else {
				return nil, fmt.Errorf("exec() options.timeout must be a number (milliseconds), got %s", timeoutVal.Type())
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute the command in a subshell
	stdout, stderr, exitCode, err := i.executeBashInSubshell(ctx, command)

	// Check for context timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("exec() command timed out after %v", timeout)
	}

	// If there's an execution error (not just non-zero exit code), return it
	if err != nil {
		return nil, fmt.Errorf("exec() failed: %w", err)
	}

	// Return result as an object with stdout, stderr, and exitCode
	result := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"stdout":   {Value: &StringValue{Value: stdout}},
			"stderr":   {Value: &StringValue{Value: stderr}},
			"exitCode": {Value: &NumberValue{Value: float64(exitCode)}},
		},
	}

	return result, nil
}

// executeBashInSubshell executes a bash command in a subshell and returns stdout, stderr, and exit code
// It uses a subshell clone of the interpreter's runner to inherit env vars and working directory
func (i *Interpreter) executeBashInSubshell(ctx context.Context, command string) (string, string, int, error) {
	i.runnerMu.RLock()
	runner := i.runner
	i.runnerMu.RUnlock()

	return bash.RunBashCommandInSubShellWithExitCode(ctx, runner, command)
}
