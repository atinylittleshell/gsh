package gline

import (
	"fmt"
	"testing"
	"time"
)

// TestSpecificInfiniteLoopScenario attempts to reproduce the exact infinite loop scenario
// mentioned in the bot's comment about findMatchingParen consistently returning -1
func TestSpecificInfiniteLoopScenario(t *testing.T) {
	// Let's create a scenario that might cause the infinite loop
	// The bot mentioned "if findMatchingParen consistently returns -1 due to malformed input"

	// First, let's test the hasIncompleteConstructs function directly
	testCases := []struct {
		input    string
		expected bool
		desc     string
	}{
		// These cases should trigger the command substitution removal logic
		{"echo $(foo) (bar", true, "Command substitution + unmatched parentheses"},
		{"echo $(foo $(bar) (baz", true, "Nested command substitution + unmatched"},
		{"$(echo test) $(foo (bar", true, "Mixed complete and incomplete"},
		{"echo $(foo (bar $(baz", true, "Multiple levels of nesting"},
		{"$(echo (test (foo (bar", true, "Deep nesting that might cause issues"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Case_%d_%s", i+1, tc.desc), func(t *testing.T) {
			fmt.Printf("Testing hasIncompleteConstructs with: %q\n", tc.input)

			// Test the function directly with a timeout
			done := make(chan struct{})
			var result bool

			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("  PANIC in hasIncompleteConstructs: %v\n", r)
					}
					close(done)
				}()

				result = hasIncompleteConstructs(tc.input)
				fmt.Printf("  hasIncompleteConstructs(%q) = %v\n", tc.input, result)
			}()

			select {
			case <-done:
				if result != tc.expected {
					t.Errorf("hasIncompleteConstructs(%q) = %v, want %v", tc.input, result, tc.expected)
				}
			case <-time.After(1 * time.Second):
				t.Errorf("TIMEOUT in hasIncompleteConstructs for input: %q", tc.input)
				fmt.Printf("  ⚠️  TIMEOUT in hasIncompleteConstructs!\n")
			}
		})
	}
}

// TestCommandSubstitutionRemovalEdgeCases tests edge cases that might cause infinite loops
func TestCommandSubstitutionRemovalEdgeCases(t *testing.T) {
	// These are edge cases that might cause the infinite loop
	edgeCases := []struct {
		input string
		desc  string
	}{
		{"$(", "Minimal command substitution start"},
		{"$(foo", "Incomplete command substitution"},
		{"$(foo (bar", "Command substitution with unmatched parentheses"},
		{"$(foo (bar (baz", "Multiple levels of unmatched parentheses"},
		{"$(foo) (bar", "Complete command substitution + unmatched parentheses"},
		{"$(foo) $(bar (baz", "Mixed complete and incomplete"},
		{"echo $(foo (bar $(baz", "Complex nesting with command substitution"},
		{"$(echo $(foo (bar $(baz", "Very deep nesting"},
		{"$(echo test) $(foo (bar $(baz", "Multiple command substitutions with incomplete"},
		{"$(echo (test (foo (bar (baz", "Extremely deep nesting"},
	}

	for i, tc := range edgeCases {
		t.Run(fmt.Sprintf("EdgeCase_%d_%s", i+1, tc.desc), func(t *testing.T) {
			fmt.Printf("Testing edge case %d: %q\n", i+1, tc.input)

			state := NewMultilineState()

			done := make(chan struct{})
			var complete bool
			var prompt string

			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("  PANIC in AddLine: %v\n", r)
					}
					close(done)
				}()

				start := time.Now()
				complete, prompt = state.AddLine(tc.input)
				elapsed := time.Since(start)

				fmt.Printf("  Result: complete=%v, prompt=%q, time=%v\n", complete, prompt, elapsed)
			}()

			select {
			case <-done:
				fmt.Printf("  ✓ Completed normally\n")
			case <-time.After(2 * time.Second):
				t.Errorf("TIMEOUT after 2s - Potential infinite loop for input: %q", tc.input)
				fmt.Printf("  ⚠️  TIMEOUT - Potential infinite loop detected!\n")
			}
		})
	}
}

// TestMalformedInputScenarios tests specifically malformed inputs that might cause infinite loops
func TestMalformedInputScenarios(t *testing.T) {
	// These inputs are specifically designed to potentially cause infinite loops
	malformedCases := []struct {
		input string
		desc  string
	}{
		// Cases where findMatchingParen might return -1 repeatedly
		{"$(echo (test", "findMatchingParen should return -1"},
		{"$(echo (test (foo", "Multiple unmatched opening"},
		{"$(echo (test (foo (bar", "Three levels of unmatched"},
		{"$(echo (test (foo (bar (baz", "Four levels of unmatched"},

		// Cases with mixed parentheses
		{"$(echo (test) (foo", "Mixed matched and unmatched"},
		{"$(echo (test (foo) (bar", "Complex mixed pattern"},

		// Cases that might cause string manipulation issues
		{"$(echo test (foo", "Simple case that might loop"},
		{"echo $(foo (bar $(baz", "Alternating pattern"},
		{"$(echo $(foo (bar $(baz $(qux", "Very complex alternating"},
	}

	for i, tc := range malformedCases {
		t.Run(fmt.Sprintf("Malformed_%d_%s", i+1, tc.desc), func(t *testing.T) {
			fmt.Printf("Testing malformed case %d: %q\n", i+1, tc.input)

			// Test both the direct function and the full multiline state
			done := make(chan struct{})

			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("  PANIC: %v\n", r)
					}
					close(done)
				}()

				// First test hasIncompleteConstructs directly
				start := time.Now()
				result1 := hasIncompleteConstructs(tc.input)
				elapsed1 := time.Since(start)

				fmt.Printf("  hasIncompleteConstructs(%q) = %v, time=%v\n", tc.input, result1, elapsed1)

				// Then test with full multiline state
				state := NewMultilineState()
				start = time.Now()
				complete, prompt := state.AddLine(tc.input)
				elapsed2 := time.Since(start)

				fmt.Printf("  AddLine(%q) = (complete=%v, prompt=%q), time=%v\n", tc.input, complete, prompt, elapsed2)
			}()

			select {
			case <-done:
				fmt.Printf("  ✓ Both functions completed normally\n")
			case <-time.After(3 * time.Second):
				t.Errorf("TIMEOUT after 3s - Potential infinite loop for input: %q", tc.input)
				fmt.Printf("  ⚠️  TIMEOUT - Potential infinite loop detected!\n")
			}
		})
	}
}
