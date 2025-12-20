package gline

import (
	"testing"
)

func TestParseDSRResponse(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		expected int
	}{
		{
			name:     "simple response column 2",
			response: []byte("\x1b[1;2R"),
			expected: 2,
		},
		{
			name:     "simple response column 3",
			response: []byte("\x1b[1;3R"),
			expected: 3,
		},
		{
			name:     "double digit row and column",
			response: []byte("\x1b[10;15R"),
			expected: 15,
		},
		{
			name:     "column 1",
			response: []byte("\x1b[1;1R"),
			expected: 1,
		},
		{
			name:     "empty response",
			response: []byte{},
			expected: -1,
		},
		{
			name:     "invalid response - no ESC",
			response: []byte("[1;2R"),
			expected: -1,
		},
		{
			name:     "invalid response - no bracket",
			response: []byte("\x1b1;2R"),
			expected: -1,
		},
		{
			name:     "response with leading garbage",
			response: []byte("garbage\x1b[5;7R"),
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDSRResponse(tt.response)
			if result != tt.expected {
				t.Errorf("parseDSRResponse(%q) = %d, want %d", tt.response, result, tt.expected)
			}
		})
	}
}

func TestGetLightningBoltWidth(t *testing.T) {
	// Get the width - should be 1 or 2 depending on terminal
	// In test environment (non-terminal), should return default of 1
	width := GetLightningBoltWidth()

	// Width should be either 1 or 2 (the two valid options for this character)
	if width < 1 || width > 2 {
		t.Errorf("GetLightningBoltWidth() = %d, want 1 or 2", width)
	}

	// Call it again to verify sync.Once returns the same cached value
	width2 := GetLightningBoltWidth()
	if width != width2 {
		t.Errorf("GetLightningBoltWidth() returned different values: %d vs %d", width, width2)
	}
}

func TestProbeTerminalCharWidthNonTerminal(t *testing.T) {
	// When running in a non-terminal context (like tests), probeTerminalCharWidth
	// should return the default width of 1
	width := probeTerminalCharWidth('⚡')

	// In test environment (not a terminal), should return default of 1
	if width != 1 {
		t.Errorf("probeTerminalCharWidth('⚡') in non-terminal = %d, want 1", width)
	}
}

func TestWordwrapWithRuneWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "simple text fits",
			input:    "hello world",
			width:    20,
			expected: "hello world",
		},
		{
			name:     "text wraps at word boundary",
			input:    "hello world",
			width:    6,
			expected: "hello\nworld",
		},
		{
			name:     "multiple words wrap",
			input:    "one two three four",
			width:    8,
			expected: "one two\nthree\nfour",
		},
		{
			name:     "preserves existing newlines",
			input:    "line1\nline2",
			width:    20,
			expected: "line1\nline2",
		},
		{
			name:     "long word breaks",
			input:    "abcdefghij",
			width:    5,
			expected: "abcde\nfghij",
		},
		{
			name:     "width zero returns original",
			input:    "test",
			width:    0,
			expected: "test",
		},
		{
			name:     "negative width returns original",
			input:    "test",
			width:    -5,
			expected: "test",
		},
		{
			name:     "empty string",
			input:    "",
			width:    10,
			expected: "",
		},
		{
			name:     "text with emoji - fits in width",
			input:    "🔥 fire",
			width:    10,
			expected: "🔥 fire",
		},
		{
			name:     "text with emoji - needs wrapping",
			input:    "🔥 fire emoji here",
			width:    10,
			expected: "🔥 fire\nemoji here",
		},
		{
			name:     "coach tip style content",
			input:    "🔥 Day 5 streak (1.2x XP)",
			width:    20,
			// In test environment emoji width=1, so: 🔥(1)+' '(1)+Day(3)+' '(1)+5(1)+' '(1)+streak(6)+' '(1)+(1.2x(5)=20
			// XP) doesn't fit (20+1+3=24>20), so it wraps
			expected: "🔥 Day 5 streak (1.2x\nXP)",
		},
		{
			name:     "multiple emojis",
			input:    "📊 Today: 10 commands, +50 XP",
			width:    15,
			expected: "📊 Today: 10\ncommands, +50\nXP",
		},
		{
			name:     "preserves ANSI codes",
			input:    "\x1b[31mred\x1b[0m text",
			width:    5,
			expected: "\x1b[31mred\x1b[0m\ntext",
		},
		{
			name:     "ANSI codes don't count towards width",
			input:    "\x1b[31mred\x1b[0m blue",
			width:    8,
			expected: "\x1b[31mred\x1b[0m blue",
		},
		{
			name:     "tabs handled as word boundaries",
			input:    "one\ttwo\tthree",
			width:    10,
			expected: "one\ttwo\nthree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WordwrapWithRuneWidth(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("WordwrapWithRuneWidth(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}
