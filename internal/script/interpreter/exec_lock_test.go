package interpreter

import (
	"sync"
	"testing"
)

func TestGoroutineCallStacks_BasicPushPop(t *testing.T) {
	stacks := newGoroutineCallStacks()

	stacks.push("func1", "file.gsh:1")
	stacks.push("func2", "file.gsh:2")

	stack := stacks.get()
	if len(stack) != 2 {
		t.Errorf("expected 2 frames, got %d", len(stack))
	}
	if stack[0].FunctionName != "func1" {
		t.Errorf("expected func1, got %s", stack[0].FunctionName)
	}
	if stack[1].FunctionName != "func2" {
		t.Errorf("expected func2, got %s", stack[1].FunctionName)
	}

	stacks.pop()
	stack = stacks.get()
	if len(stack) != 1 {
		t.Errorf("expected 1 frame after pop, got %d", len(stack))
	}

	stacks.pop()
	stack = stacks.get()
	if len(stack) != 0 {
		t.Errorf("expected 0 frames after second pop, got %d", len(stack))
	}
}

func TestGoroutineCallStacks_IsolatedPerGoroutine(t *testing.T) {
	stacks := newGoroutineCallStacks()
	var wg sync.WaitGroup

	// Main goroutine pushes its frames
	stacks.push("main_func", "main.gsh:1")

	// Start another goroutine that pushes different frames
	wg.Add(1)
	var otherStack []StackFrame
	go func() {
		defer wg.Done()
		stacks.push("other_func1", "other.gsh:1")
		stacks.push("other_func2", "other.gsh:2")
		otherStack = stacks.get()
	}()
	wg.Wait()

	// Main goroutine's stack should be unaffected
	mainStack := stacks.get()
	if len(mainStack) != 1 {
		t.Errorf("expected main stack to have 1 frame, got %d", len(mainStack))
	}
	if mainStack[0].FunctionName != "main_func" {
		t.Errorf("expected main_func, got %s", mainStack[0].FunctionName)
	}

	// Other goroutine should have had its own stack
	if len(otherStack) != 2 {
		t.Errorf("expected other stack to have 2 frames, got %d", len(otherStack))
	}
	if otherStack[0].FunctionName != "other_func1" {
		t.Errorf("expected other_func1, got %s", otherStack[0].FunctionName)
	}
}

func TestGoroutineCallStacks_ConcurrentAccess(t *testing.T) {
	stacks := newGoroutineCallStacks()
	var wg sync.WaitGroup

	// Launch many goroutines that all push/pop concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each goroutine does its own push/pop sequence
			stacks.push("func1", "file.gsh:1")
			stacks.push("func2", "file.gsh:2")
			stacks.push("func3", "file.gsh:3")

			stack := stacks.get()
			if len(stack) != 3 {
				t.Errorf("goroutine %d: expected 3 frames, got %d", id, len(stack))
			}

			stacks.pop()
			stacks.pop()
			stacks.pop()

			stack = stacks.get()
			if len(stack) != 0 {
				t.Errorf("goroutine %d: expected 0 frames after pops, got %d", id, len(stack))
			}
		}(i)
	}

	wg.Wait()
}

func TestGoroutineCallStacks_GetReturnsCopy(t *testing.T) {
	stacks := newGoroutineCallStacks()

	stacks.push("func1", "file.gsh:1")

	// Get a copy
	stack1 := stacks.get()

	// Push more
	stacks.push("func2", "file.gsh:2")

	// Original copy should be unchanged
	if len(stack1) != 1 {
		t.Errorf("expected original copy to have 1 frame, got %d", len(stack1))
	}

	// New get should have 2
	stack2 := stacks.get()
	if len(stack2) != 2 {
		t.Errorf("expected new get to have 2 frames, got %d", len(stack2))
	}
}

func TestGoroutineCallStacks_CleanupEmptyStacks(t *testing.T) {
	stacks := newGoroutineCallStacks()

	stacks.push("func1", "file.gsh:1")
	stacks.pop()

	// After popping all frames, the stack should be cleaned up
	// We can verify this by checking internal state
	stacks.mu.RLock()
	gid := getGoroutineID()
	_, exists := stacks.stacks[gid]
	stacks.mu.RUnlock()

	if exists {
		t.Error("expected stack to be cleaned up after all pops")
	}
}

func TestGetGoroutineID(t *testing.T) {
	// Get ID from main goroutine
	mainID := getGoroutineID()
	if mainID <= 0 {
		t.Errorf("expected positive goroutine ID, got %d", mainID)
	}

	// Get ID from another goroutine
	var otherID int64
	done := make(chan bool)
	go func() {
		otherID = getGoroutineID()
		done <- true
	}()
	<-done

	// IDs should be different
	if mainID == otherID {
		t.Error("expected different goroutine IDs for different goroutines")
	}

	// Same goroutine should return same ID
	sameID := getGoroutineID()
	if mainID != sameID {
		t.Errorf("expected same ID %d, got %d", mainID, sameID)
	}
}
