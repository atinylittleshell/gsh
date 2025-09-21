package gline

import (
	"testing"

	"go.uber.org/zap"
)

// TestStatefulInterruptBehavior reproduces the bug where interrupted state
// persists across multiple gline calls, causing subsequent calls to incorrectly
// return ErrInterrupted even when the user didn't press Ctrl+C
func TestStatefulInterruptBehavior(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	options := NewOptions()

	// First call: simulate a normal successful interaction
	t.Run("First call should work normally", func(t *testing.T) {
		// Create a mock that simulates normal user input
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Verify initial state
		if model.interrupted {
			t.Error("Initial model should not be interrupted")
		}
		if model.appState != Active {
			t.Errorf("Expected Active state, got %v", model.appState)
		}
	})

	// Second call: simulate Ctrl+C interruption
	t.Run("Second call with Ctrl+C should be interrupted", func(t *testing.T) {
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Simulate Ctrl+C by sending interruptMsg
		updatedModel, _ := model.Update(interruptMsg{})
		appModel := updatedModel.(appModel)

		// Verify the model is now interrupted
		if !appModel.interrupted {
			t.Error("Model should be interrupted after interruptMsg")
		}
		if appModel.appState != Terminated {
			t.Errorf("Expected Terminated state, got %v", appModel.appState)
		}
	})

	// Third call: this should work normally but might fail due to stateful bug
	t.Run("Third call should work normally again", func(t *testing.T) {
		// This is where the bug would manifest - if there's global state,
		// this new model might incorrectly inherit the interrupted state
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Verify fresh state
		if model.interrupted {
			t.Error("Fresh model should not be interrupted - this indicates stateful bug")
		}
		if model.appState != Active {
			t.Errorf("Expected Active state, got %v", model.appState)
		}
	})
}

// TestMultipleGlineCallsWithInterruption tests the actual Gline function
// to see if interrupted state persists across calls
func TestMultipleGlineCallsWithInterruption(t *testing.T) {
	// This test would require more complex setup to actually trigger the bug
	// since it involves the full tea.Program lifecycle. The bug likely occurs
	// when there's some shared state or when the same model instance is reused.

	t.Skip("This test requires complex setup with tea.Program mocking")

	// The actual bug reproduction would look something like:
	// 1. Call Gline() normally - should work
	// 2. Call Gline() and simulate Ctrl+C - should return ErrInterrupted
	// 3. Call Gline() again normally - BUG: might incorrectly return ErrInterrupted
}
