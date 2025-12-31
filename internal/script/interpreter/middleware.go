package interpreter

import (
	"fmt"
	"sync"
)

// MiddlewareManager manages the chain of input middleware functions.
// Middleware runs in registration order (first registered = first to run).
type MiddlewareManager struct {
	mu          sync.RWMutex
	middlewares []*middlewareEntry
	nextID      int
}

// middlewareEntry holds a middleware function and its unique ID
type middlewareEntry struct {
	id     string
	tool   *ToolValue
	interp *Interpreter
}

// NewMiddlewareManager creates a new middleware manager
func NewMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{
		middlewares: make([]*middlewareEntry, 0),
		nextID:      1,
	}
}

// Use registers a middleware function. Returns a unique ID that can be used to remove it.
// Middleware runs in registration order (first registered = first to run).
func (mm *MiddlewareManager) Use(tool *ToolValue, interp *Interpreter) string {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	id := fmt.Sprintf("middleware_%d", mm.nextID)
	mm.nextID++

	entry := &middlewareEntry{
		id:     id,
		tool:   tool,
		interp: interp,
	}
	mm.middlewares = append(mm.middlewares, entry)

	return id
}

// Remove removes a middleware by its ID. Returns true if removed, false if not found.
func (mm *MiddlewareManager) Remove(id string) bool {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i, entry := range mm.middlewares {
		if entry.id == id {
			mm.middlewares = append(mm.middlewares[:i], mm.middlewares[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveByTool removes a middleware by its tool reference. Returns true if removed.
func (mm *MiddlewareManager) RemoveByTool(tool *ToolValue) bool {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for i, entry := range mm.middlewares {
		if entry.tool == tool {
			mm.middlewares = append(mm.middlewares[:i], mm.middlewares[i+1:]...)
			return true
		}
	}
	return false
}

// Len returns the number of registered middlewares
func (mm *MiddlewareManager) Len() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return len(mm.middlewares)
}

// MiddlewareResult represents the result of middleware chain execution
type MiddlewareResult struct {
	Handled bool   // If true, input was handled by middleware
	Input   string // Possibly modified input (for fall-through to shell)
}

// ExecuteChain executes the middleware chain with the given input.
// Returns MiddlewareResult indicating whether input was handled or should fall through to shell.
func (mm *MiddlewareManager) ExecuteChain(input string, interp *Interpreter) (*MiddlewareResult, error) {
	mm.mu.RLock()
	middlewares := make([]*middlewareEntry, len(mm.middlewares))
	copy(middlewares, mm.middlewares)
	mm.mu.RUnlock()

	// Create initial context
	ctx := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"input": {Value: &StringValue{Value: input}},
		},
	}

	// Execute the chain recursively
	return mm.executeMiddleware(middlewares, 0, ctx, interp)
}

// executeMiddleware executes a single middleware and handles the chain continuation
func (mm *MiddlewareManager) executeMiddleware(middlewares []*middlewareEntry, index int, ctx *ObjectValue, interp *Interpreter) (*MiddlewareResult, error) {
	// If we've exhausted all middleware, fall through to shell
	if index >= len(middlewares) {
		// Get the (possibly modified) input from context
		inputVal := ctx.GetPropertyValue("input")
		inputStr := ""
		if sv, ok := inputVal.(*StringValue); ok {
			inputStr = sv.Value
		}
		return &MiddlewareResult{
			Handled: false,
			Input:   inputStr,
		}, nil
	}

	entry := middlewares[index]

	// Create the next() function that continues the chain
	nextFn := &BuiltinValue{
		Name: "next",
		Fn: func(args []Value) (Value, error) {
			// Get ctx from args (middleware may have modified it)
			var nextCtx *ObjectValue
			if len(args) > 0 {
				if obj, ok := args[0].(*ObjectValue); ok {
					nextCtx = obj
				}
			}
			if nextCtx == nil {
				nextCtx = ctx // Use original if not passed
			}

			// Execute next middleware in chain
			result, err := mm.executeMiddleware(middlewares, index+1, nextCtx, interp)
			if err != nil {
				return nil, err
			}

			// Return result as object so middleware can inspect/modify it
			return &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"handled": {Value: &BoolValue{Value: result.Handled}},
					"input":   {Value: &StringValue{Value: result.Input}},
				},
			}, nil
		},
	}

	// Call the middleware with (ctx, next)
	result, err := interp.CallTool(entry.tool, []Value{ctx, nextFn})
	if err != nil {
		return nil, fmt.Errorf("middleware error: %w", err)
	}

	// Check if middleware returned { handled: true }
	if obj, ok := result.(*ObjectValue); ok {
		handledVal := obj.GetPropertyValue("handled")
		if bv, ok := handledVal.(*BoolValue); ok && bv.Value {
			return &MiddlewareResult{Handled: true}, nil
		}

		// Check if result has input (possibly modified)
		if inputVal := obj.GetPropertyValue("input"); inputVal.Type() == ValueTypeString {
			if sv, ok := inputVal.(*StringValue); ok {
				return &MiddlewareResult{
					Handled: false,
					Input:   sv.Value,
				}, nil
			}
		}
	}

	// If result is null or doesn't have handled: true, treat as fall-through
	inputVal := ctx.GetPropertyValue("input")
	inputStr := ""
	if sv, ok := inputVal.(*StringValue); ok {
		inputStr = sv.Value
	}
	return &MiddlewareResult{
		Handled: false,
		Input:   inputStr,
	}, nil
}

// Clear removes all registered middlewares
func (mm *MiddlewareManager) Clear() {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.middlewares = make([]*middlewareEntry, 0)
}
