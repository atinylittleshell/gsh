package input

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestNewHighlighter(t *testing.T) {
	h := NewHighlighter()
	if h == nil {
		t.Fatal("NewHighlighter returned nil")
	}
	if h.parser == nil {
		t.Error("parser should not be nil")
	}
	if h.cmdCache == nil {
		t.Error("cmdCache should not be nil")
	}
	if h.styles == nil {
		t.Error("styles should not be nil")
	}
}

func TestHighlightEmpty(t *testing.T) {
	h := NewHighlighter()
	result := h.Highlight("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestHighlightAgentMode(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "simple agent message",
			input:    "# hello",
			contains: []string{"#", "hello"},
		},
		{
			name:     "agent command",
			input:    "# /agents",
			contains: []string{"#", "/agents"},
		},
		{
			name:     "agent clear command",
			input:    "# /clear",
			contains: []string{"#", "/clear"},
		},
		{
			name:     "agent with leading space",
			input:    "  # test",
			contains: []string{"#", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// All parts should still be present (ignoring ANSI codes)
			for _, part := range tt.contains {
				if !strings.Contains(result, part) {
					t.Errorf("expected result to contain %q, got %q", part, result)
				}
			}
			// Text should be preserved after stripping ANSI
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHighlightCommandExists(t *testing.T) {
	h := NewHighlighter()

	// Test with a command that definitely exists
	result := h.Highlight("ls")
	// Text should be preserved
	stripped := ansi.Strip(result)
	if stripped != "ls" {
		t.Errorf("text not preserved: expected %q, got %q", "ls", stripped)
	}
}

func TestHighlightCommandNotExists(t *testing.T) {
	h := NewHighlighter()

	// Test with a command that definitely doesn't exist
	input := "thiscommanddoesnotexist12345"
	result := h.Highlight(input)
	// Text should be preserved
	stripped := ansi.Strip(result)
	if stripped != input {
		t.Errorf("text not preserved: expected %q, got %q", input, stripped)
	}
}

func TestHighlightStrings(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "double quoted string",
			input: `echo "hello world"`,
		},
		{
			name:  "single quoted string",
			input: `echo 'hello world'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Text should be preserved
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHighlightVariables(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple variable",
			input: "echo $HOME",
		},
		{
			name:  "braced variable",
			input: "echo ${HOME}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Text should be preserved
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHighlightFlags(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "short flag",
			input: "ls -la",
		},
		{
			name:  "long flag",
			input: "ls --all",
		},
		{
			name:  "multiple flags",
			input: "grep -r --include=*.go pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Text should be preserved
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHighlightOperators(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "pipe",
			input: "ls | grep foo",
		},
		{
			name:  "and operator",
			input: "cmd1 && cmd2",
		},
		{
			name:  "or operator",
			input: "cmd1 || cmd2",
		},
		{
			name:  "redirect output",
			input: "echo hello > file.txt",
		},
		{
			name:  "redirect input",
			input: "cat < file.txt",
		},
		{
			name:  "append redirect",
			input: "echo hello >> file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Text should be preserved
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHighlightBasicFallback(t *testing.T) {
	h := NewHighlighter()

	// Test with incomplete/invalid syntax that triggers basic fallback
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unclosed double quote",
			input: `echo "hello`,
		},
		{
			name:  "unclosed single quote",
			input: `echo 'hello`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Should not panic and should return something
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestCommandExistsCache(t *testing.T) {
	h := NewHighlighter()

	// First call - should check PATH
	exists1 := h.commandExists("ls")

	// Second call - should use cache
	exists2 := h.commandExists("ls")

	if exists1 != exists2 {
		t.Error("cache should return same result")
	}

	// Clear cache
	h.ClearCache()

	// Should still work after clearing
	exists3 := h.commandExists("ls")
	if exists1 != exists3 {
		t.Error("should return same result after cache clear")
	}
}

func TestHighlightPreservesText(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple command",
			input: "ls -la",
		},
		{
			name:  "command with args",
			input: "echo hello world",
		},
		{
			name:  "agent mode",
			input: "# hello",
		},
		{
			name:  "pipe",
			input: "ls | grep foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			// Strip ANSI codes and verify text is preserved
			stripped := ansi.Strip(result)
			if stripped != tt.input {
				t.Errorf("text not preserved: expected %q, got %q", tt.input, stripped)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isOperator", func(t *testing.T) {
		if !isOperator('|') {
			t.Error("| should be an operator")
		}
		if !isOperator('&') {
			t.Error("& should be an operator")
		}
		if !isOperator(';') {
			t.Error("; should be an operator")
		}
		if !isOperator('>') {
			t.Error("> should be an operator")
		}
		if !isOperator('<') {
			t.Error("< should be an operator")
		}
		if isOperator('a') {
			t.Error("a should not be an operator")
		}
	})

	t.Run("isAlphaNum", func(t *testing.T) {
		if !isAlphaNum('a') {
			t.Error("a should be alphanumeric")
		}
		if !isAlphaNum('Z') {
			t.Error("Z should be alphanumeric")
		}
		if !isAlphaNum('5') {
			t.Error("5 should be alphanumeric")
		}
		if isAlphaNum('-') {
			t.Error("- should not be alphanumeric")
		}
	})
}
