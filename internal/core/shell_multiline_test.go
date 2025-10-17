package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/pkg/gline"
)

// TestShellMultilineCommands tests that the shell properly handles multiline commands
func TestShellMultilineCommands(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{
			name:     "backslash continuation",
			inputs:   []string{"echo hello \\", "world"},
			expected: "hello world",
		},
		{
			name:     "incomplete quotes",
			inputs:   []string{`echo "hello`, `world"`},
			expected: "hello\nworld",
		},
		{
			name:     "function definition",
			inputs:   []string{"myfunc() {", "echo test", "}"},
			expected: "",
		},
		{
			name:     "here document",
			inputs:   []string{"cat <<EOF", "line1", "line2", "EOF"},
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock multiline state to simulate user input
			state := gline.NewMultilineState()

			// Simulate adding lines one by one
			var complete bool
			var prompt string

			for i, input := range tt.inputs {
				complete, prompt = state.AddLine(input)

				// For all but the last input, we expect incomplete
				if i < len(tt.inputs)-1 {
					if complete {
						t.Errorf("Expected incomplete command at line %d, but got complete", i)
					}
					if prompt != ">" {
						t.Errorf("Expected '>' prompt, got '%s'", prompt)
					}
				}
			}

			// Final input should be complete
			if !complete {
				t.Errorf("Expected complete command, but got incomplete")
			}

			// Verify we get the complete command
			completeCommand := state.GetCompleteCommand()
			if completeCommand == "" {
				t.Errorf("Expected non-empty complete command")
			}

			// Execute the command and validate the expected output
			if tt.expected != "" {
				output, err := executeBashCommand(completeCommand)
				if err != nil {
					t.Errorf("Failed to execute command: %v", err)
				} else if output != tt.expected {
					t.Errorf("Expected output '%s', got '%s'", tt.expected, output)
				}
			}

			// Basic validation that the command contains our inputs
			for _, input := range tt.inputs {
				if !strings.Contains(completeCommand, input) {
					t.Errorf("Expected complete command to contain '%s', got: %s", input, completeCommand)
				}
			}
		})
	}
}

// executeBashCommand executes a bash command and returns its output
func executeBashCommand(command string) (string, error) {
	// Create a bash command that will execute the multiline command
	cmd := exec.Command("bash", "-c", command)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		// Return stderr if there's an error, as it might contain useful information
		if stderr.Len() > 0 {
			return "", fmt.Errorf("command failed: %v, stderr: %s", err, stderr.String())
		}
		return "", err
	}

	// Return the output, trimming any trailing whitespace
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// TestShellMultilineCancellation tests that Ctrl+C properly cancels multiline input
func TestShellMultilineCancellation(t *testing.T) {
	state := gline.NewMultilineState()

	// Add a line
	complete, prompt := state.AddLine("echo hello \\")
	if complete {
		t.Errorf("Expected incomplete command")
	}
	if prompt != ">" {
		t.Errorf("Expected '>' prompt, got '%s'", prompt)
	}

	// Reset (simulating Ctrl+C)
	state.Reset()

	// Should be empty and inactive
	if state.IsActive() {
		t.Errorf("Expected state to be inactive after reset")
	}

	completeCommand := state.GetCompleteCommand()
	if completeCommand != "" {
		t.Errorf("Expected empty command after reset, got: %s", completeCommand)
	}
}

// TestShellMultilineEdgeCases tests edge cases for multiline input
func TestShellMultilineEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected bool // whether the final command should be complete
	}{
		{
			name:     "empty lines in backslash continuation",
			inputs:   []string{"echo hello \\", "", "world"},
			expected: true,
		},
		{
			name:     "mixed quotes",
			inputs:   []string{`echo "start`, `'finish"`},
			expected: true,
		},
		{
			name:     "command substitution",
			inputs:   []string{"echo $(echo", "hello)"},
			expected: true,
		},
		{
			name:     "backticks",
			inputs:   []string{"echo `echo", "hello`"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := gline.NewMultilineState()

			var complete bool
			for _, input := range tt.inputs {
				complete, _ = state.AddLine(input)
			}

			if complete != tt.expected {
				t.Errorf("Expected complete=%v, got %v", tt.expected, complete)
			}
		})
	}
}
