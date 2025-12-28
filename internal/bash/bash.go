package bash

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// RunBashScriptFromReader parses and runs a bash script from an io.Reader.
// The script is executed in the provided runner (not a subshell).
func RunBashScriptFromReader(ctx context.Context, runner *interp.Runner, reader io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(reader, name)
	if err != nil {
		return err
	}
	return runner.Run(ctx, prog)
}

// RunBashScriptFromFile parses and runs a bash script from a file.
// The script is executed in the provided runner (not a subshell).
func RunBashScriptFromFile(ctx context.Context, runner *interp.Runner, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return RunBashScriptFromReader(ctx, runner, f, filePath)
}

// RunBashCommandInSubShell runs a bash command in a subshell and captures stdout/stderr.
// Returns stdout, stderr, and any error. Non-zero exit codes are returned as errors.
// Deprecated: Use RunBashCommandInSubShellWithExitCode for better exit code handling.
func RunBashCommandInSubShell(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	stdout, stderr, exitCode, err := RunBashCommandInSubShellWithExitCode(ctx, runner, command)
	if err != nil {
		return stdout, stderr, err
	}
	if exitCode != 0 {
		return stdout, stderr, interp.ExitStatus(exitCode)
	}
	return stdout, stderr, nil
}

// RunBashCommandInSubShellWithExitCode runs a bash command in a subshell and captures stdout/stderr.
// Returns stdout, stderr, exit code, and any execution error.
// A non-zero exit code is NOT treated as an error - check the exit code separately.
func RunBashCommandInSubShellWithExitCode(ctx context.Context, runner *interp.Runner, command string) (string, string, int, error) {
	subShell := runner.Subshell()

	outBuf := &threadSafeBuffer{}
	errBuf := &threadSafeBuffer{}
	interp.StdIO(nil, outBuf, errBuf)(subShell) //nolint:errcheck

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", 1, fmt.Errorf("failed to parse bash command: %w", err)
	}

	if prog == nil {
		// Empty command
		return "", "", 0, nil
	}

	err = subShell.Run(ctx, prog)

	// Extract exit code
	exitCode := 0
	if err != nil {
		var exitStatus interp.ExitStatus
		if errors.As(err, &exitStatus) {
			exitCode = int(exitStatus)
			// Non-zero exit code is not an execution error
			return outBuf.String(), errBuf.String(), exitCode, nil
		}
		// Real execution error
		return outBuf.String(), errBuf.String(), 1, err
	}

	return outBuf.String(), errBuf.String(), exitCode, nil
}

// RunBashCommand runs a bash command in the main runner and captures stdout/stderr.
// WARNING: This temporarily redirects the runner's stdio, which is not thread-safe.
// Consider using RunBashCommandInSubShell for safer concurrent execution.
func RunBashCommand(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	outBuf := &threadSafeBuffer{}
	errBuf := &threadSafeBuffer{}
	interp.StdIO(nil, outBuf, errBuf)(runner)                  //nolint:errcheck
	defer interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner) //nolint:errcheck

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", err
	}

	err = runner.Run(ctx, prog)
	if err != nil {
		return "", "", err
	}

	return outBuf.String(), errBuf.String(), nil
}
