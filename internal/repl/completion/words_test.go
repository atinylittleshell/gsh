package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPreservingQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "multiple spaces",
			input:    "hello   world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "double quotes",
			input:    "echo \"hello world\"",
			expected: []string{"echo", "\"hello world\""},
		},
		{
			name:     "single quotes",
			input:    "echo 'hello world'",
			expected: []string{"echo", "'hello world'"},
		},
		{
			name:     "mixed quotes",
			input:    "echo \"hello 'there'\" 'world \"test\"'",
			expected: []string{"echo", "\"hello 'there'\"", "'world \"test\"'"},
		},
		{
			name:     "quotes with spaces around",
			input:    "ls  \"Program Files\"  'My Documents' ",
			expected: []string{"ls", "\"Program Files\"", "'My Documents'"},
		},
		{
			name:     "path with spaces",
			input:    "cat /path/to/\"My Documents\"/file.txt",
			expected: []string{"cat", "/path/to/\"My Documents\"/file.txt"},
		},
		{
			name:     "unclosed quotes",
			input:    "echo \"hello world",
			expected: []string{"echo", "\"hello world"},
		},
		{
			name:     "escaped quotes inside quotes",
			input:    "echo \"hello \\\"world\\\"\"",
			expected: []string{"echo", "\"hello \\\"world\\\"\""},
		},
		{
			name:     "quotes in middle of word",
			input:    "path/\"Program Files\"/file",
			expected: []string{"path/\"Program Files\"/file"},
		},
		{
			name:     "multiple quoted sections",
			input:    "cp \"My Files\"/*.txt 'Backup Folder'/",
			expected: []string{"cp", "\"My Files\"/*.txt", "'Backup Folder'/"},
		},
		{
			name:     "quotes at start of input",
			input:    "\"Documents and Settings\"/file.txt",
			expected: []string{"\"Documents and Settings\"/file.txt"},
		},
		{
			name:     "empty quotes",
			input:    "echo \"\" ''",
			expected: []string{"echo", "\"\"", "''"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitPreservingQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPreservingQuotesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "only spaces",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "only quotes",
			input:    "\"\"''",
			expected: []string{"\"\"''"},
		},
		{
			name:     "alternating spaces and quotes",
			input:    "\" \" ' ' \" \"",
			expected: []string{"\" \"", "' '", "\" \""},
		},
		{
			name:     "quotes with special characters",
			input:    "echo \"$HOME\" '$PATH'",
			expected: []string{"echo", "\"$HOME\"", "'$PATH'"},
		},
		{
			name:     "quotes with newlines",
			input:    "echo \"hello\nworld\"",
			expected: []string{"echo", "\"hello\nworld\""},
		},
		{
			name:     "quotes with tabs",
			input:    "echo \"hello\tworld\"",
			expected: []string{"echo", "\"hello\tworld\""},
		},
		{
			name:     "multiple adjacent quoted strings",
			input:    "echo\"hello\"'world'",
			expected: []string{"echo\"hello\"'world'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitPreservingQuotes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
