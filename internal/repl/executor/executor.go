// Package executor provides command execution abstractions for the gsh REPL.
// It supports both bash command execution and gsh script execution,
// managing environment variables and working directory state.
package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
	shinterp "mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ExecMiddleware is a function that wraps an ExecHandlerFunc to provide
// additional functionality (e.g., command interception, logging).
type ExecMiddleware = func(next shinterp.ExecHandlerFunc) shinterp.ExecHandlerFunc

// REPLExecutor handles command execution for the REPL using the interpreter's
// shared sh runner for bash execution and gsh script execution.
type REPLExecutor struct {
	interpreter *interpreter.Interpreter
	logger      *zap.Logger
}

// NewREPLExecutor creates a new REPLExecutor.
// The interpreter is required and provides the shared sh runner for bash execution.
// The logger is optional (can be nil).
// The execHandlers are optional middleware for intercepting command execution
// (e.g., for analytics, history, completion).
func NewREPLExecutor(interp *interpreter.Interpreter, logger *zap.Logger, execHandlers ...ExecMiddleware) (*REPLExecutor, error) {
	if interp == nil {
		return nil, fmt.Errorf("interpreter is required")
	}

	// Configure the interpreter's runner for interactive use with exec handlers
	runner := interp.Runner()
	shinterp.Interactive(true)(runner)                     //nolint:errcheck
	shinterp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner) //nolint:errcheck
	shinterp.ExecHandlers(execHandlers...)(runner)         //nolint:errcheck

	return &REPLExecutor{
		interpreter: interp,
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

	runner := e.interpreter.Runner()
	mu := e.interpreter.RunnerMutex()

	mu.Lock()
	err = runner.Run(ctx, prog)
	mu.Unlock()

	if err != nil {
		var exitStatus shinterp.ExitStatus
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
	runner := e.interpreter.Runner()
	mu := e.interpreter.RunnerMutex()

	mu.RLock()
	defer mu.RUnlock()

	return bash.RunBashCommandInSubShellWithExitCode(ctx, runner, command)
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

// GetEnv gets an environment variable value from the interpreter's runner.
func (e *REPLExecutor) GetEnv(name string) string {
	return e.interpreter.GetEnv(name)
}

// SetEnv sets an environment variable in the interpreter's runner.
func (e *REPLExecutor) SetEnv(name, value string) {
	e.interpreter.SetEnv(name, value)
}

// GetPwd returns the current working directory from the interpreter's runner.
func (e *REPLExecutor) GetPwd() string {
	return e.interpreter.GetWorkingDir()
}

// Close cleans up any resources held by the executor.
func (e *REPLExecutor) Close() error {
	if e.interpreter != nil {
		return e.interpreter.Close()
	}
	return nil
}

// Runner returns the underlying mvdan/sh runner from the interpreter.
// This is useful for advanced use cases that need direct access.
func (e *REPLExecutor) Runner() *shinterp.Runner {
	return e.interpreter.Runner()
}

// AliasExists returns true if the given name is currently defined as a shell alias
// in the underlying mvdan/sh runner.
//
// Note: mvdan/sh keeps aliases in an unexported field, so we use reflection.
func (e *REPLExecutor) AliasExists(name string) bool {
	runner := e.interpreter.Runner()
	if runner == nil {
		return false
	}

	runnerValue := reflect.ValueOf(runner).Elem()
	aliasField := runnerValue.FieldByName("alias")
	if !aliasField.IsValid() || aliasField.IsNil() {
		return false
	}

	// aliasField is a map[string]interp.alias; we only care about keys.
	key := reflect.ValueOf(name)
	return aliasField.MapIndex(key).IsValid()
}

// FunctionExists returns true if the given name is currently defined as a shell function
// in the underlying mvdan/sh runner.
//
// Note: mvdan/sh's Funcs field is exported, but we still use reflection for consistency.
func (e *REPLExecutor) FunctionExists(name string) bool {
	runner := e.interpreter.Runner()
	if runner == nil {
		return false
	}

	runnerValue := reflect.ValueOf(runner).Elem()
	funcsField := runnerValue.FieldByName("Funcs")
	if !funcsField.IsValid() || funcsField.IsNil() {
		return false
	}

	// funcsField is a map[string]*syntax.Stmt; we only care about keys.
	key := reflect.ValueOf(name)
	return funcsField.MapIndex(key).IsValid()
}

// AliasOrFunctionExists returns true if the given name is defined as either
// a shell alias or a shell function. This is useful for syntax highlighting
// to recognize user-defined commands from config files like .gshenv and .gsh_profile.
func (e *REPLExecutor) AliasOrFunctionExists(name string) bool {
	return e.AliasExists(name) || e.FunctionExists(name)
}

// Interpreter returns the underlying gsh interpreter.
// This is useful for advanced use cases that need direct access.
func (e *REPLExecutor) Interpreter() *interpreter.Interpreter {
	return e.interpreter
}

// RunBashScriptFromReader runs a bash script from an io.Reader.
func (e *REPLExecutor) RunBashScriptFromReader(ctx context.Context, reader io.Reader, name string) error {
	runner := e.interpreter.Runner()
	mu := e.interpreter.RunnerMutex()

	mu.Lock()
	defer mu.Unlock()

	return bash.RunBashScriptFromReader(ctx, runner, reader, name)
}
