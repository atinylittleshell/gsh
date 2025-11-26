package interpreter

import (
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Interpreter represents the gsh script interpreter
type Interpreter struct {
	env        *Environment
	mcpManager *mcp.Manager
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
	interp := &Interpreter{
		env:        NewEnvironment(),
		mcpManager: mcp.NewManager(),
	}
	interp.registerBuiltins()
	return interp
}

// NewWithEnvironment creates a new interpreter with a custom environment
func NewWithEnvironment(env *Environment) *Interpreter {
	interp := &Interpreter{
		env:        env,
		mcpManager: mcp.NewManager(),
	}
	interp.registerBuiltins()
	return interp
}

// Close cleans up resources used by the interpreter
func (i *Interpreter) Close() error {
	if i.mcpManager != nil {
		return i.mcpManager.Close()
	}
	return nil
}

// Eval evaluates a program and returns the result
func (i *Interpreter) Eval(program *parser.Program) (*EvalResult, error) {
	var finalResult Value = &NullValue{}

	for _, stmt := range program.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			return nil, err
		}
		finalResult = val
	}

	return &EvalResult{
		FinalResult: finalResult,
		Env:         i.env,
	}, nil
}
