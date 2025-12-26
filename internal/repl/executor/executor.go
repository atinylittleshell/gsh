// Package executor provides command execution abstractions for the gsh REPL.
// It supports both bash command execution and gsh script execution,
// managing environment variables and working directory state.
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// threadSafeBuffer provides a thread-safe wrapper around bytes.Buffer
type threadSafeBuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write implements io.Writer interface
func (b *threadSafeBuffer) Write(p []byte) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

// String returns the contents of the buffer as a string
func (b *threadSafeBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

// ExecMiddleware is a function that wraps an ExecHandlerFunc to provide
// additional functionality (e.g., command interception, logging).
type ExecMiddleware = func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc

// REPLExecutor handles command execution for the REPL using mvdan/sh for
// bash execution and the gsh script interpreter for gsh script execution.
type REPLExecutor struct {
	runner      *interp.Runner
	interpreter *interpreter.Interpreter
	logger      *zap.Logger
	varsMutex   sync.RWMutex // Protects concurrent access to runner.Vars
}

// NewREPLExecutor creates a new REPLExecutor.
// The logger is optional (can be nil).
// The execHandlers are optional middleware for intercepting command execution
// (e.g., for analytics, history, completion).
func NewREPLExecutor(logger *zap.Logger, execHandlers ...ExecMiddleware) (*REPLExecutor, error) {
	env := expand.ListEnviron(os.Environ()...)

	runner, err := interp.New(
		interp.Interactive(true),
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandlers(execHandlers...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create bash runner: %w", err)
	}

	return &REPLExecutor{
		runner:      runner,
		interpreter: interpreter.NewWithLogger(logger),
		logger:      logger,
	}, nil
}

// ExecuteBash runs a bash command with output going to stdout/stderr.
// Returns the exit code and any execution error.
func (e *REPLExecutor) ExecuteBash(ctx context.Context, command string) (int, error) {
	prog, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return 1, fmt.Errorf("failed to parse bash command: %w", err)
	}

	err = e.runner.Run(ctx, prog)
	if err != nil {
		var exitStatus interp.ExitStatus
		if errors.As(err, &exitStatus) {
			return int(exitStatus), nil
		}
		return 1, err
	}

	return 0, nil
}

// ExecuteBashInSubshell runs a bash command in a subshell, capturing output.
// Returns stdout, stderr, exit code, and any execution error.
func (e *REPLExecutor) ExecuteBashInSubshell(ctx context.Context, command string) (string, string, int, error) {
	subShell := e.runner.Subshell()

	outBuf := &threadSafeBuffer{}
	errBuf := &threadSafeBuffer{}
	interp.StdIO(nil, io.Writer(outBuf), io.Writer(errBuf))(subShell) //nolint:errcheck

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", 1, fmt.Errorf("failed to parse bash command: %w", err)
	}

	if prog == nil {
		return "", "", 0, nil
	}

	err = subShell.Run(ctx, prog)

	// Extract exit code
	exitCode := 0
	if err != nil {
		var exitStatus interp.ExitStatus
		if errors.As(err, &exitStatus) {
			exitCode = int(exitStatus)
			// Non-zero exit code is not an execution error, just return the code
			return outBuf.String(), errBuf.String(), exitCode, nil
		}
		// Real execution error (parse error, etc.)
		return outBuf.String(), errBuf.String(), 1, err
	}

	return outBuf.String(), errBuf.String(), exitCode, nil
}

// ExecuteGsh runs a gsh script.
// Returns any execution error.
func (e *REPLExecutor) ExecuteGsh(ctx context.Context, script string) error {
	// Parse the script
	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	// Check for parsing errors
	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse errors: %s", strings.Join(p.Errors(), "; "))
	}

	// Execute the script
	_, err := e.interpreter.Eval(program)
	if err != nil {
		return fmt.Errorf("execution error: %w", err)
	}

	return nil
}

// GetEnv gets an environment variable value.
// This reads from the runner's Vars map, which is populated during command execution.
func (e *REPLExecutor) GetEnv(name string) string {
	e.varsMutex.RLock()
	defer e.varsMutex.RUnlock()
	if e.runner.Vars == nil {
		return ""
	}
	return e.runner.Vars[name].String()
}

// SetEnv sets an environment variable directly in the runner's Vars map.
// For variables that need to be available in subshells, use ExecuteBash with export.
func (e *REPLExecutor) SetEnv(name, value string) {
	e.varsMutex.Lock()
	defer e.varsMutex.Unlock()
	if e.runner.Vars == nil {
		e.runner.Vars = make(map[string]expand.Variable)
	}
	e.runner.Vars[name] = expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}
}

// GetPwd returns the current working directory.
func (e *REPLExecutor) GetPwd() string {
	return e.runner.Dir
}

// Close cleans up any resources held by the executor.
func (e *REPLExecutor) Close() error {
	if e.interpreter != nil {
		return e.interpreter.Close()
	}
	return nil
}

// Runner returns the underlying mvdan/sh runner.
// This is useful for advanced use cases that need direct access.
func (e *REPLExecutor) Runner() *interp.Runner {
	return e.runner
}

// Interpreter returns the underlying gsh interpreter.
// This is useful for advanced use cases that need direct access.
func (e *REPLExecutor) Interpreter() *interpreter.Interpreter {
	return e.interpreter
}

// RunBashScriptFromReader runs a bash script from an io.Reader.
func (e *REPLExecutor) RunBashScriptFromReader(ctx context.Context, reader io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(reader, name)
	if err != nil {
		return err
	}
	return e.runner.Run(ctx, prog)
}

// SyncEnvToOS syncs all exported environment variables from the bash runner
// to the OS environment. This is useful after loading bash config files like
// ~/.gshrc so that variables set there are available to the gsh interpreter
// via env.VAR_NAME.
func (e *REPLExecutor) SyncEnvToOS() {
	e.varsMutex.RLock()
	defer e.varsMutex.RUnlock()

	if e.runner.Vars == nil {
		return
	}

	for name, variable := range e.runner.Vars {
		// Only sync exported variables
		if variable.Exported {
			os.Setenv(name, variable.String())
		}
	}
}
