package interpreter

import (
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Interpreter represents the gsh script interpreter
type Interpreter struct {
	env *Environment
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

// Variables returns all top-level variables as a map
func (r *EvalResult) Variables() map[string]Value {
	if r.Env == nil {
		return make(map[string]Value)
	}
	// Return a copy to prevent external modification
	vars := make(map[string]Value)
	for k, v := range r.Env.store {
		vars[k] = v
	}
	return vars
}

// New creates a new interpreter instance
func New() *Interpreter {
	return &Interpreter{
		env: NewEnvironment(),
	}
}

// NewWithEnvironment creates a new interpreter with a custom environment
func NewWithEnvironment(env *Environment) *Interpreter {
	return &Interpreter{
		env: env,
	}
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
