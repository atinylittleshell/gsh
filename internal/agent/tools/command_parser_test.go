package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "ls -la",
			expected: []string{"ls -la"},
		},
		{
			name:     "semicolon separated commands",
			input:    "ls; pwd; echo hello",
			expected: []string{"ls", "pwd", "echo hello"},
		},
		{
			name:     "AND operator commands",
			input:    "ls && pwd && echo success",
			expected: []string{"ls", "pwd", "echo success"},
		},
		{
			name:     "OR operator commands",
			input:    "test -f file || echo 'file not found'",
			expected: []string{"test -f file", "echo 'file not found'"},
		},
		{
			name:     "pipe commands",
			input:    "ls -la | grep txt | sort",
			expected: []string{"ls -la", "grep txt", "sort"},
		},
		{
			name:     "mixed operators",
			input:    "ls && pwd; echo done || echo failed",
			expected: []string{"ls", "pwd", "echo done", "echo failed"},
		},
		{
			name:     "subshell commands",
			input:    "(cd /tmp && ls)",
			expected: []string{"cd /tmp", "ls"},
		},
		{
			name:     "nested subshells",
			input:    "(cd /tmp && (ls -la && pwd))",
			expected: []string{"cd /tmp", "ls -la", "pwd"},
		},
		{
			name:     "command substitution",
			input:    "echo $(date)",
			expected: []string{"echo $(date)", "date"},
		},
		{
			name:     "complex command substitution",
			input:    "echo $(ls | head -1) && pwd",
			expected: []string{"echo $(ls)", "pwd", "ls", "head -1"},
		},
		{
			name:     "malicious command injection",
			input:    "ls; rm -rf /",
			expected: []string{"ls", "rm -rf /"},
		},
		{
			name:     "complex malicious injection",
			input:    "ls && (echo safe && rm -rf /tmp/*)",
			expected: []string{"ls", "echo safe", "rm -rf /tmp/*"},
		},
		{
			name:     "quoted arguments",
			input:    "echo 'hello world' && ls \"my file.txt\"",
			expected: []string{"echo 'hello world'", "ls \"my file.txt\""},
		},
		{
			name:     "empty command",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only",
			input:    "   \t\n  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCommandsErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid syntax - unmatched parenthesis",
			input: "ls (",
		},
		{
			name:  "invalid syntax - unmatched quote",
			input: "echo 'unclosed quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.input)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestValidateCompoundCommand(t *testing.T) {
	patterns := []string{
		"^ls.*",
		"^pwd.*",
		"^echo.*",
		"^git status.*",
	}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "single approved command",
			command:  "ls -la",
			expected: true,
		},
		{
			name:     "multiple approved commands",
			command:  "ls && pwd && echo done",
			expected: true,
		},
		{
			name:     "single unapproved command",
			command:  "rm -rf /",
			expected: false,
		},
		{
			name:     "mixed approved and unapproved",
			command:  "ls && rm -rf /",
			expected: false,
		},
		{
			name:     "malicious injection attempt",
			command:  "ls; rm -rf /",
			expected: false,
		},
		{
			name:     "approved pipe commands",
			command:  "ls | grep txt",
			expected: false, // grep is not in approved patterns
		},
		{
			name:     "approved subshell commands",
			command:  "(ls && pwd)",
			expected: true,
		},
		{
			name:     "unapproved subshell commands",
			command:  "(ls && rm file)",
			expected: false,
		},
		{
			name:     "command substitution with approved commands",
			command:  "echo $(pwd)",
			expected: true,
		},
		{
			name:     "command substitution with unapproved commands",
			command:  "echo $(rm file)",
			expected: false,
		},
		{
			name:     "empty command",
			command:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateCompoundCommand(tt.command, patterns)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCompoundCommandErrors(t *testing.T) {
	patterns := []string{"^ls.*"}

	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "invalid syntax",
			command: "ls (",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateCompoundCommand(tt.command, patterns)
			assert.Error(t, err)
			assert.False(t, result)
		})
	}
}

func TestGenerateCompoundCommandRegex(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "single command",
			command:  "ls -la",
			expected: []string{"^ls.*"},
		},
		{
			name:     "multiple commands",
			command:  "ls && pwd && echo hello",
			expected: []string{"^ls.*", "^pwd.*", "^echo hello.*"}, // "hello" is correctly identified as a subcommand-like argument
		},
		{
			name:     "git commands with subcommands",
			command:  "git status && git commit -m 'test'",
			expected: []string{"^git status.*", "^git commit.*"},
		},
		{
			name:     "pipe commands",
			command:  "ls | grep txt | sort",
			expected: []string{"^ls.*", "^grep txt.*", "^sort.*"}, // "txt" is correctly identified as a subcommand-like argument
		},
		{
			name:     "subshell commands",
			command:  "(cd /tmp && ls)",
			expected: []string{"^cd.*", "^ls.*"},
		},
		{
			name:     "command substitution",
			command:  "echo $(date)",
			expected: []string{"^echo.*", "^date.*"},
		},
		{
			name:     "duplicate commands",
			command:  "ls && ls -la && ls",
			expected: []string{"^ls.*"}, // Should deduplicate
		},
		{
			name:     "empty command",
			command:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCompoundCommandRegex(tt.command)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateCompoundCommandRegexErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "invalid syntax",
			command: "ls (",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCompoundCommandRegex(tt.command)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestSecurityScenarios(t *testing.T) {
	// Test various security attack scenarios
	patterns := []string{"^ls.*", "^echo.*"}

	securityTests := []struct {
		name     string
		command  string
		expected bool
		reason   string
	}{
		{
			name:     "basic injection with semicolon",
			command:  "ls; rm -rf /",
			expected: false,
			reason:   "rm command should not be approved",
		},
		{
			name:     "injection with AND operator",
			command:  "ls && rm -rf /",
			expected: false,
			reason:   "rm command should not be approved",
		},
		{
			name:     "injection with OR operator",
			command:  "ls || rm -rf /",
			expected: false,
			reason:   "rm command should not be approved",
		},
		{
			name:     "injection in subshell",
			command:  "(ls && rm -rf /)",
			expected: false,
			reason:   "rm command in subshell should not be approved",
		},
		{
			name:     "injection in command substitution",
			command:  "echo $(ls && rm -rf /)",
			expected: false,
			reason:   "rm command in substitution should not be approved",
		},
		{
			name:     "nested injection",
			command:  "ls && (echo safe && rm -rf /)",
			expected: false,
			reason:   "nested rm command should not be approved",
		},
		{
			name:     "pipe injection",
			command:  "ls | rm -rf /",
			expected: false,
			reason:   "rm command in pipe should not be approved",
		},
		{
			name:     "complex nested injection",
			command:  "ls && (echo $(pwd && rm file) && ls)",
			expected: false,
			reason:   "rm command in nested substitution should not be approved",
		},
		{
			name:     "legitimate complex command",
			command:  "ls && echo $(ls -la) && (echo done && ls)",
			expected: true,
			reason:   "all commands should be approved",
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateCompoundCommand(tt.command, patterns)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

func TestExtractFromStatement(t *testing.T) {
	// Test edge cases for extractFromStatement function
	// This function is tested indirectly via ExtractCommands
	// Testing with nil would be an invalid use case in real scenarios

	// We can test via ExtractCommands which exercises this path
	commands, err := ExtractCommands("echo hello")
	assert.NoError(t, err)
	assert.Equal(t, []string{"echo hello"}, commands)
}

func TestExtractFromCommand(t *testing.T) {
	// Test coverage for different command types via ExtractCommands
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "background command with &",
			input:    "sleep 5 &",
			expected: []string{"sleep 5"},
		},
		{
			name:     "redirected command",
			input:    "echo hello > output.txt",
			expected: []string{"echo hello"},
		},
		{
			name:     "command with input redirection",
			input:    "sort < input.txt",
			expected: []string{"sort"},
		},
		{
			name:     "command with error redirection",
			input:    "ls nonexistent 2>/dev/null",
			expected: []string{"ls nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCallCommand(t *testing.T) {
	// Test complex call command extraction via ExtractCommands
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "command with variables",
			input:    "echo $USER $HOME",
			expected: []string{"echo $USER $HOME"},
		},
		{
			name:     "command with glob patterns",
			input:    "ls *.txt *.log",
			expected: []string{"ls *.txt *.log"},
		},
		{
			name:     "command with escape sequences",
			input:    "echo \"hello\\nworld\"",
			expected: []string{"echo \"hello\\nworld\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractWordString(t *testing.T) {
	// Test word string extraction with complex quoting
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single quoted with spaces",
			input:    "echo 'hello world with spaces'",
			expected: []string{"echo 'hello world with spaces'"},
		},
		{
			name:     "double quoted with variables",
			input:    "echo \"Hello $USER, today is $(date)\"",
			expected: []string{"echo \"Hello , today is \"", "date"},
		},
		{
			name:     "mixed quoting",
			input:    "echo 'single' \"double\" unquoted",
			expected: []string{"echo 'single' \"double\" unquoted"},
		},
		{
			name:     "nested command substitution",
			input:    "echo $(echo $(date))",
			expected: []string{"echo $(echo $(date))", "echo $(date)", "date"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"ls", "pwd", "echo"},
			expected: []string{"ls", "pwd", "echo"},
		},
		{
			name:     "with duplicates",
			input:    []string{"ls", "pwd", "ls", "echo", "pwd"},
			expected: []string{"ls", "pwd", "echo"},
		},
		{
			name:     "with empty strings",
			input:    []string{"ls", "", "pwd", "   ", "echo"},
			expected: []string{"ls", "pwd", "echo"},
		},
		{
			name:     "with whitespace",
			input:    []string{" ls ", "pwd", " ls", "echo "},
			expected: []string{"ls", "pwd", "echo"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string(nil),
		},
		{
			name:     "nil input",
			input:    nil,
			expected: []string(nil),
		},
		{
			name:     "all empty strings",
			input:    []string{"", "   ", "\t\n"},
			expected: []string(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateCommands(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCompoundCommandWithInvalidRegex(t *testing.T) {
	// Test with invalid regex patterns
	patterns := []string{
		"^ls.*",
		"[invalid regex", // Invalid regex
		"^pwd.*",
	}

	// Should still work with valid patterns, skipping invalid ones
	result, err := ValidateCompoundCommand("ls -la", patterns)
	assert.NoError(t, err)
	assert.True(t, result) // Should match the valid "^ls.*" pattern

	// Test with completely invalid pattern
	result, err = ValidateCompoundCommand("pwd", patterns)
	assert.NoError(t, err)
	assert.True(t, result) // Should match the valid "^pwd.*" pattern
}

func TestValidateCompoundCommandEdgeCases(t *testing.T) {
	patterns := []string{"^ls.*"}

	tests := []struct {
		name     string
		command  string
		patterns []string
		expected bool
	}{
		{
			name:     "empty patterns list",
			command:  "ls -la",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "nil patterns list",
			command:  "ls -la",
			patterns: nil,
			expected: false,
		},
		{
			name:     "whitespace only command",
			command:  "   \t\n  ",
			patterns: patterns,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateCompoundCommand(tt.command, tt.patterns)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateCompoundCommandRegexEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
		hasError bool
	}{
		{
			name:     "whitespace only command",
			command:  "   \t\n  ",
			expected: []string{},
			hasError: false,
		},
		{
			name:     "command with only separators",
			command:  "&&;;||",
			expected: []string{}, // No actual commands to extract
			hasError: true,       // Invalid syntax
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCompoundCommandRegex(tt.command)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestComplexNestingScenarios(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "deeply nested subshells",
			command:  "(ls && (pwd && (echo hello)))",
			expected: []string{"ls", "pwd", "echo hello"},
		},
		{
			name:     "multiple command substitutions",
			command:  "echo $(date) $(whoami) $(pwd)",
			expected: []string{"echo $(date) $(whoami) $(pwd)", "date", "whoami", "pwd"},
		},
		{
			name:     "mixed operators with subshells",
			command:  "(ls && pwd) || (echo failed && exit 1)",
			expected: []string{"ls", "pwd", "echo failed", "exit 1"},
		},
		{
			name:     "command substitution in subshell",
			command:  "(echo $(ls) && pwd)",
			expected: []string{"echo $(ls)", "pwd", "ls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.command)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractCommandsFromCmdSubst(t *testing.T) {
	// Test command substitution extraction
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "simple command substitution",
			command:  "echo $(date)",
			expected: []string{"echo $(date)", "date"},
		},
		{
			name:     "command substitution with pipes",
			command:  "echo $(ls | head -1)",
			expected: []string{"echo $(ls)", "ls", "head -1"},
		},
		{
			name:     "backtick command substitution",
			command:  "echo `date`",
			expected: []string{"echo $(date)", "date"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractCommands(tt.command)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
