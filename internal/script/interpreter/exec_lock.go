package interpreter

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// goroutineCallStacks provides per-goroutine call stack storage.
// This allows concurrent script execution without call stack corruption.
//
// Each goroutine gets its own isolated call stack for error reporting,
// similar to how each thread in other languages has its own call stack.
type goroutineCallStacks struct {
	mu     sync.RWMutex
	stacks map[int64][]StackFrame
}

// newGoroutineCallStacks creates a new per-goroutine call stack storage.
func newGoroutineCallStacks() *goroutineCallStacks {
	return &goroutineCallStacks{
		stacks: make(map[int64][]StackFrame),
	}
}

// push adds a frame to the current goroutine's call stack.
func (g *goroutineCallStacks) push(functionName, location string) {
	gid := getGoroutineID()
	g.mu.Lock()
	defer g.mu.Unlock()

	g.stacks[gid] = append(g.stacks[gid], StackFrame{
		FunctionName: functionName,
		Location:     location,
	})
}

// pop removes the top frame from the current goroutine's call stack.
func (g *goroutineCallStacks) pop() {
	gid := getGoroutineID()
	g.mu.Lock()
	defer g.mu.Unlock()

	stack := g.stacks[gid]
	if len(stack) > 0 {
		g.stacks[gid] = stack[:len(stack)-1]
	}
	// Clean up empty stacks to prevent memory leaks
	if len(g.stacks[gid]) == 0 {
		delete(g.stacks, gid)
	}
}

// get returns a copy of the current goroutine's call stack.
func (g *goroutineCallStacks) get() []StackFrame {
	gid := getGoroutineID()
	g.mu.RLock()
	defer g.mu.RUnlock()

	stack := g.stacks[gid]
	if stack == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	result := make([]StackFrame, len(stack))
	copy(result, stack)
	return result
}

// getGoroutineID returns the current goroutine's ID.
// This is used for per-goroutine call stack isolation.
//
// Note: This uses runtime.Stack() which is somewhat expensive, but:
// 1. It's only called when pushing/popping stack frames (during function calls)
// 2. The alternative (passing context through all functions) would be invasive
// 3. This is a well-known pattern in Go for goroutine identification
func getGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Stack trace starts with "goroutine <id> [..."
	// Parse the ID from the stack trace
	field := bytes.Fields(buf[:n])[1]
	id, _ := strconv.ParseInt(string(field), 10, 64)
	return id
}
