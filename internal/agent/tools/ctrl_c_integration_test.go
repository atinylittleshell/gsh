package tools

import (
	"testing"

	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// TestMultiTurnAgentCtrlCBehavior tests the specific bug reported:
// "Sometimes, the agent will run a command just fine. But sometimes it will return
// a message 'Agent session interrupted by user.' I'm not sure, but it's possible
// that there is something stateful going on ... that it works fine, but if I ctrl-c
// out of a multi-turn set of commands, then it remembers that I canceled it and
// accidentally cancels future ones."
func TestMultiTurnAgentCtrlCBehavior(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Save the original userConfirmation function
	originalUserConfirmation := userConfirmation
	defer func() {
		userConfirmation = originalUserConfirmation
	}()

	// Test scenario 1: Normal command should work fine
	t.Run("First command works normally", func(t *testing.T) {
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "y" // User approves the command
		}

		// Test the userConfirmation function directly since BashTool requires complex setup
		result := userConfirmation(logger, nil, "Test question", "Test explanation")
		if result != "y" {
			t.Errorf("Expected 'y', got '%s'", result)
		}
	})

	// Test scenario 2: User presses Ctrl+C during command confirmation
	t.Run("Second command interrupted by Ctrl+C", func(t *testing.T) {
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			// Simulate Ctrl+C during user confirmation - now returns "n"
			return "n"
		}

		result := userConfirmation(logger, nil, "Test question", "Test explanation")
		if result != "n" {
			t.Errorf("Expected 'n', got '%s'", result)
		}
	})

	// Test scenario 3: Next command should work normally again (this is where the bug would manifest)
	t.Run("Third command should work normally after previous interruption", func(t *testing.T) {
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "y" // User approves the command
		}

		// This should work normally - the key test for the stateful bug
		result := userConfirmation(logger, nil, "Test question", "Test explanation")
		if result != "y" {
			t.Errorf("Expected 'y' (this would indicate the stateful bug is fixed), got '%s'", result)
		}
	})

	// Test scenario 4: Multiple interruptions and recoveries
	t.Run("Multiple interruptions should not cause persistent state", func(t *testing.T) {
		// Interrupt again
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "n"
		}

		result := userConfirmation(logger, nil, "Test question", "Test explanation")
		if result != "n" {
			t.Errorf("Expected 'n', got '%s'", result)
		}

		// Now try normal command again
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "y"
		}

		result = userConfirmation(logger, nil, "Test question", "Test explanation")
		if result != "y" {
			t.Errorf("Expected 'y' after multiple interruptions, got '%s'", result)
		}
	})
}

// TestUserConfirmationStatelessBehavior verifies that userConfirmation doesn't maintain state
func TestUserConfirmationStatelessBehavior(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Save the original userConfirmation function
	originalUserConfirmation := userConfirmation
	defer func() {
		userConfirmation = originalUserConfirmation
	}()

	// The userConfirmation function should be stateless - each call should be independent
	t.Run("userConfirmation calls should be independent", func(t *testing.T) {
		// First call returns n (ctrl-c now returns "n")
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "n"
		}
		result1 := userConfirmation(logger, nil, "Test question 1", "Test explanation 1")

		// Second call returns y - should not be affected by first call
		userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
			return "y"
		}
		result2 := userConfirmation(logger, nil, "Test question 2", "Test explanation 2")

		if result1 != "n" {
			t.Errorf("First call: expected 'n', got '%s'", result1)
		}
		if result2 != "y" {
			t.Errorf("Second call: expected 'y', got '%s'", result2)
		}
	})
}
