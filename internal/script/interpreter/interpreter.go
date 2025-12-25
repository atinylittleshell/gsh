package interpreter

import (
	"io"
	"os"

	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
)

// Interpreter represents the gsh script interpreter
type Interpreter struct {
	env              *Environment
	mcpManager       *mcp.Manager
	providerRegistry *ProviderRegistry
	callStack        []StackFrame // Track call stack for error reporting
	logger           *zap.Logger  // Optional zap logger for log.* functions
	stdin            io.Reader    // Reader for input() function, defaults to os.Stdin
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

	interp := &Interpreter{
		env:              NewEnvironment(),
		mcpManager:       mcp.NewManager(),
		providerRegistry: registry,
		logger:           logger,
		stdin:            os.Stdin,
	}
	interp.registerBuiltins()
	return interp
}

// NewWithEnvironment creates a new interpreter with a custom environment
func NewWithEnvironment(env *Environment) *Interpreter {
	return NewWithEnvironmentAndLogger(env, nil)
}

// NewWithEnvironmentAndLogger creates a new interpreter with a custom environment and optional logger
func NewWithEnvironmentAndLogger(env *Environment, logger *zap.Logger) *Interpreter {
	registry := NewProviderRegistry()
	registry.Register(NewOpenAIProvider())

	interp := &Interpreter{
		env:              env,
		mcpManager:       mcp.NewManager(),
		providerRegistry: registry,
		logger:           logger,
		stdin:            os.Stdin,
	}
	interp.registerBuiltins()
	return interp
}

// SetStdin sets the stdin reader for the input() function
// This is useful for testing or for providing custom input sources
func (i *Interpreter) SetStdin(r io.Reader) {
	i.stdin = r
}

// Close cleans up resources used by the interpreter
func (i *Interpreter) Close() error {
	if i.mcpManager != nil {
		return i.mcpManager.Close()
	}
	return nil
}

// SetVariable defines or updates a variable in the interpreter's environment
func (i *Interpreter) SetVariable(name string, value Value) error {
	return i.env.Define(name, value)
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
