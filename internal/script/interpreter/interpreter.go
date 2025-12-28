package interpreter

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Interpreter represents the gsh script interpreter
type Interpreter struct {
	env              *Environment
	mcpManager       *mcp.Manager
	providerRegistry *ProviderRegistry
	callStack        []StackFrame   // Track call stack for error reporting
	logger           *zap.Logger    // Optional zap logger for log.* functions
	stdin            io.Reader      // Reader for input() function, defaults to os.Stdin
	runner           *interp.Runner // Shared sh runner for env vars, working dir, and exec()
	runnerMu         sync.RWMutex   // Protects runner access
}

// EvalResult represents the result of evaluating a program
type EvalResult struct {
	FinalResult Value        // The value of the last statement executed
	Env         *Environment // The environment after execution (contains all variables)
}

// Value returns the final result value for convenient access
func (r *EvalResult) Value() Value {
	return r.FinalResult
}

// Variables returns all top-level variables as a map (excluding built-ins)
func (r *EvalResult) Variables() map[string]Value {
	if r.Env == nil {
		return make(map[string]Value)
	}
	// Return a copy to prevent external modification, excluding built-ins
	vars := make(map[string]Value)
	for k, v := range r.Env.store {
		// Skip built-in functions and objects
		if isBuiltin(k) {
			continue
		}
		vars[k] = v
	}
	return vars
}

// New creates a new interpreter instance
func New() *Interpreter {
	return NewWithLogger(nil)
}

// NewWithLogger creates a new interpreter with an optional zap logger
// When logger is provided, log.* functions will write to the zap logger
// When logger is nil, log.* functions will write to stderr with prefixes
func NewWithLogger(logger *zap.Logger) *Interpreter {
	registry := NewProviderRegistry()
	registry.Register(NewOpenAIProvider())

	// Create sh runner with OS environment
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
	)
	if err != nil {
		// This should never fail with basic options
		panic(fmt.Sprintf("failed to create sh runner: %v", err))
	}

	i := &Interpreter{
		env:              NewEnvironment(),
		mcpManager:       mcp.NewManager(),
		providerRegistry: registry,
		logger:           logger,
		stdin:            os.Stdin,
		runner:           runner,
	}
	i.registerBuiltins()
	return i
}

// NewWithEnvironment creates a new interpreter with a custom environment
func NewWithEnvironment(env *Environment) *Interpreter {
	return NewWithEnvironmentAndLogger(env, nil)
}

// NewWithEnvironmentAndLogger creates a new interpreter with a custom environment and optional logger
func NewWithEnvironmentAndLogger(gshEnv *Environment, logger *zap.Logger) *Interpreter {
	registry := NewProviderRegistry()
	registry.Register(NewOpenAIProvider())

	// Create sh runner with OS environment
	shEnv := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(
		interp.Env(shEnv),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
	)
	if err != nil {
		// This should never fail with basic options
		panic(fmt.Sprintf("failed to create sh runner: %v", err))
	}

	i := &Interpreter{
		env:              gshEnv,
		mcpManager:       mcp.NewManager(),
		providerRegistry: registry,
		logger:           logger,
		stdin:            os.Stdin,
		runner:           runner,
	}
	i.registerBuiltins()
	return i
}

// SetStdin sets the stdin reader for the input() function
// This is useful for testing or for providing custom input sources
func (i *Interpreter) SetStdin(r io.Reader) {
	i.stdin = r
}

// Runner returns the underlying sh runner
// This is used by the REPL executor to share the same runner for bash commands
func (i *Interpreter) Runner() *interp.Runner {
	return i.runner
}

// RunnerMutex returns the mutex protecting the runner
// The REPL executor should hold this lock when accessing the runner
func (i *Interpreter) RunnerMutex() *sync.RWMutex {
	return &i.runnerMu
}

// GetWorkingDir returns the current working directory from the sh runner
func (i *Interpreter) GetWorkingDir() string {
	i.runnerMu.RLock()
	defer i.runnerMu.RUnlock()
	return i.runner.Dir
}

// GetEnv returns an environment variable
// It first checks runner.Vars (for variables set during the session),
// then falls back to os.Getenv (for inherited environment variables)
func (i *Interpreter) GetEnv(name string) string {
	i.runnerMu.RLock()
	defer i.runnerMu.RUnlock()
	// First check runner.Vars for variables set during the session
	if i.runner.Vars != nil {
		if v, ok := i.runner.Vars[name]; ok {
			return v.String()
		}
	}
	// Fall back to OS environment for inherited variables
	return os.Getenv(name)
}

// SetEnv sets an environment variable by running an export command through the runner
// This ensures the variable is properly inherited by subshells
func (i *Interpreter) SetEnv(name, value string) {
	i.runnerMu.Lock()
	defer i.runnerMu.Unlock()

	// Escape the value for shell (simple escaping - wrap in single quotes and escape single quotes)
	escapedValue := strings.ReplaceAll(value, "'", "'\"'\"'")
	cmd := fmt.Sprintf("export %s='%s'", name, escapedValue)

	prog, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		// Fallback to direct assignment if parsing fails
		if i.runner.Vars == nil {
			i.runner.Vars = make(map[string]expand.Variable)
		}
		i.runner.Vars[name] = expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}
		return
	}

	// Run the export command - ignore errors since this is best-effort
	_ = i.runner.Run(context.Background(), prog)
}

// UnsetEnv removes an environment variable by running an unset command through the runner
// This ensures the variable is properly removed from subshells
func (i *Interpreter) UnsetEnv(name string) {
	i.runnerMu.Lock()
	defer i.runnerMu.Unlock()

	cmd := fmt.Sprintf("unset %s", name)
	prog, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		// Fallback to direct deletion if parsing fails
		if i.runner.Vars != nil {
			delete(i.runner.Vars, name)
		}
		return
	}

	// Run the unset command - ignore errors since this is best-effort
	_ = i.runner.Run(context.Background(), prog)
}

// EvalString parses and evaluates a source string in the interpreter
// This is useful for evaluating multiple scripts into the same interpreter
func (i *Interpreter) EvalString(source string) (*EvalResult, error) {
	lex := lexer.New(source)
	p := parser.New(lex)
	program := p.ParseProgram()

	// Check for parser errors
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.Errors(), "; "))
	}

	return i.Eval(program)
}

// Close cleans up resources used by the interpreter
func (i *Interpreter) Close() error {
	if i.mcpManager != nil {
		return i.mcpManager.Close()
	}
	return nil
}

// SetVariable defines or updates a variable in the interpreter's environment
func (i *Interpreter) SetVariable(name string, value Value) {
	i.env.Set(name, value)
}

// GetVariables returns all top-level variables from the interpreter's environment (excluding built-ins)
func (i *Interpreter) GetVariables() map[string]Value {
	vars := make(map[string]Value)
	for k, v := range i.env.store {
		// Skip built-in functions and objects
		if isBuiltin(k) {
			continue
		}
		vars[k] = v
	}
	return vars
}

// Eval evaluates a program and returns the result
func (i *Interpreter) Eval(program *parser.Program) (*EvalResult, error) {
	var finalResult Value = &NullValue{}

	for _, stmt := range program.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			return nil, i.wrapError(err, stmt)
		}
		finalResult = val
	}

	return &EvalResult{
		FinalResult: finalResult,
		Env:         i.env,
	}, nil
}

// wrapError wraps an error with stack trace information
func (i *Interpreter) wrapError(err error, node parser.Node) error {
	// Don't wrap control flow errors
	if _, isControlFlow := err.(*ControlFlowError); isControlFlow {
		return err
	}

	// If it's already a RuntimeError, add current stack
	if rte, ok := err.(*RuntimeError); ok {
		// Add all frames from the call stack
		for _, frame := range i.callStack {
			rte.AddStackFrame(frame.FunctionName, frame.Location)
		}
		return rte
	}

	// Otherwise, create a new RuntimeError
	rte := &RuntimeError{
		Message:    err.Error(),
		StackTrace: make([]StackFrame, len(i.callStack)),
	}
	copy(rte.StackTrace, i.callStack)

	return rte
}

// pushStackFrame adds a frame to the call stack
func (i *Interpreter) pushStackFrame(functionName, location string) {
	i.callStack = append(i.callStack, StackFrame{
		FunctionName: functionName,
		Location:     location,
	})
}

// popStackFrame removes the top frame from the call stack
func (i *Interpreter) popStackFrame() {
	if len(i.callStack) > 0 {
		i.callStack = i.callStack[:len(i.callStack)-1]
	}
}
