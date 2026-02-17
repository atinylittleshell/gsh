package interpreter

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	FunctionName string // Name of the function/tool being executed
	Location     string // Source location (line:column or description)
}

// RuntimeError represents an error that occurred during script execution
// It includes a stack trace for debugging
type RuntimeError struct {
	Message    string
	StackTrace []StackFrame
}

// Error implements the error interface
func (e *RuntimeError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)

	if len(e.StackTrace) > 0 {
		sb.WriteString("\n\nStack trace:")
		for i := len(e.StackTrace) - 1; i >= 0; i-- {
			frame := e.StackTrace[i]
			fmt.Fprintf(&sb, "\n  at %s (%s)", frame.FunctionName, frame.Location)
		}
	}

	return sb.String()
}

// NewRuntimeError creates a new runtime error with a message
func NewRuntimeError(format string, args ...interface{}) *RuntimeError {
	return &RuntimeError{
		Message:    fmt.Sprintf(format, args...),
		StackTrace: []StackFrame{},
	}
}

// WrapError wraps a standard error as a RuntimeError
func WrapError(err error, location string) *RuntimeError {
	// If it's already a RuntimeError, just add to its stack
	if rte, ok := err.(*RuntimeError); ok {
		return rte
	}

	// Otherwise, create a new RuntimeError
	return &RuntimeError{
		Message:    err.Error(),
		StackTrace: []StackFrame{},
	}
}

// AddStackFrame adds a frame to the stack trace
func (e *RuntimeError) AddStackFrame(functionName, location string) {
	e.StackTrace = append(e.StackTrace, StackFrame{
		FunctionName: functionName,
		Location:     location,
	})
}

// ThrownError represents an error explicitly thrown by a throw statement.
// It is a regular Go error (not a ControlFlowError), so it will be caught
// by existing try/catch logic.
type ThrownError struct {
	Value Value       // the thrown gsh value
	Token lexer.Token // source location of the throw statement
}

// Error implements the error interface
func (e *ThrownError) Error() string {
	return e.Value.String()
}
