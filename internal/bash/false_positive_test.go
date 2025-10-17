package bash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreprocessTypesetCommands_FalsePositives(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expected     string
		shouldChange bool
	}{
		// False positive in strings
		{
			name:         "string literal with typeset -f",
			input:        `echo "typeset -f"`,
			expected:     `echo "typeset -f"`,
			shouldChange: false,
		},
		{
			name:         "string literal with declare -f",
			input:        `echo "declare -f"`,
			expected:     `echo "declare -f"`,
			shouldChange: false,
		},
		{
			name:         "single quoted string",
			input:        `echo 'typeset -f'`,
			expected:     `echo 'typeset -f'`,
			shouldChange: false,
		},
		{
			name: "heredoc content",
			input: `cat <<EOF
typeset -f
declare -f
EOF`,
			expected: `cat <<EOF
typeset -f
declare -f
EOF`,
			shouldChange: false,
		},

		// False positive in comments
		{
			name:         "comment line",
			input:        `# typeset -f should not be changed`,
			expected:     `# typeset -f should not be changed`,
			shouldChange: false,
		},
		{
			name:         "inline comment",
			input:        `echo hello # typeset -f comment`,
			expected:     `echo hello # typeset -f comment`,
			shouldChange: false,
		},

		// False positive in function names/variables
		{
			name:         "function name containing pattern",
			input:        `my_typeset -f() { echo "function"; }`,
			expected:     `my_typeset -f() { echo "function"; }`,
			shouldChange: false,
		},
		{
			name:         "variable assignment",
			input:        `cmd="typeset -f"`,
			expected:     `cmd="typeset -f"`,
			shouldChange: false,
		},

		// False positive in URLs/paths
		{
			name:         "URL containing pattern",
			input:        `curl http://example.com/typeset-f`,
			expected:     `curl http://example.com/typeset-f`,
			shouldChange: false,
		},
		{
			name:         "file path",
			input:        `cat /path/to/typeset -f.txt`,
			expected:     `cat /path/to/typeset -f.txt`,
			shouldChange: false,
		},

		// Edge cases with extra spaces that might cause issues
		{
			name:         "multiple spaces in string",
			input:        `echo "typeset  -f"`,
			expected:     `echo "typeset  -f"`,
			shouldChange: false,
		},
		{
			name:         "tab separated in string",
			input:        `echo "typeset	-f"`,
			expected:     `echo "typeset	-f"`,
			shouldChange: false,
		},

		// Valid transformations that should work
		{
			name:         "actual command",
			input:        `typeset -f`,
			expected:     `gsh_typeset -f`,
			shouldChange: true,
		},
		{
			name:         "actual command with extra spaces",
			input:        `typeset  -f`,
			expected:     `gsh_typeset -f`,
			shouldChange: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			if tc.shouldChange {
				assert.NotEqual(t, tc.input, result, "input %q should have been changed", tc.input)
				assert.Equal(t, tc.expected, result, "input %q should be transformed to %q", tc.input, tc.expected)
			} else {
				assert.Equal(t, tc.expected, result, "input %q should remain unchanged", tc.input)
			}
		})
	}
}

func TestPreprocessTypesetCommands_CurrentBehavior(t *testing.T) {
	// This test demonstrates the current problematic behavior
	testCases := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "string_false_positive",
			input:       `echo "typeset -f"`,
			description: "Currently transforms string content incorrectly",
		},
		{
			name:        "comment_false_positive",
			input:       `# typeset -f comment`,
			description: "Currently transforms comment content incorrectly",
		},
		{
			name:        "variable_false_positive",
			input:       `cmd="typeset -f"`,
			description: "Currently transforms variable assignment incorrectly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			t.Logf("Input: %q", tc.input)
			t.Logf("Output: %q", result)
			t.Logf("Description: %s", tc.description)
			t.Logf("Problem: String replacement approach cannot distinguish between actual commands and contextual usage")
		})
	}
}
