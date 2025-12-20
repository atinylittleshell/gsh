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

func TestGetRuneWidthZeroWidthCharacters(t *testing.T) {
	// Zero-width characters should return width 0
	tests := []struct {
		name     string
		char     rune
		expected int
	}{
		{
			name:     "variation selector 16 (FE0F)",
			char:     '\uFE0F',
			expected: 0,
		},
		{
			name:     "variation selector 15 (FE0E)",
			char:     '\uFE0E',
			expected: 0,
		},
		{
			name:     "zero width space (200B)",
			char:     '\u200B',
			expected: 0,
		},
		{
			name:     "zero width non-joiner (200C)",
			char:     '\u200C',
			expected: 0,
		},
		{
			name:     "zero width joiner (200D)",
			char:     '\u200D',
			expected: 0,
		},
		{
			name:     "byte order mark / zero width no-break space (FEFF)",
			char:     '\uFEFF',
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRuneWidth(tt.char)
			if result != tt.expected {
				t.Errorf("GetRuneWidth(%U) = %d, want %d", tt.char, result, tt.expected)
			}
		})
	}
}

func TestStringWidthWithVariationSelector(t *testing.T) {
	// The keyboard emoji with variation selector should be calculated correctly
	// ⌨️ = U+2328 (KEYBOARD) + U+FE0F (VARIATION SELECTOR-16)
	// The variation selector should have width 0
	keyboard := "⌨️"

	width := stringWidthWithAnsi(keyboard)

	// Width should be 1 or 2 (depending on terminal), not 2 or 3
	// In test environment (non-terminal), keyboard base char has width 1 per runewidth
	// Variation selector has width 0
	// So total should be 1
	if width != 1 {
		t.Errorf("stringWidthWithAnsi(%q) = %d, want 1 (in test environment)", keyboard, width)
	}
}

func TestCoachTipWithEmojiAlignment(t *testing.T) {
	// Simulate the coach tip scenario that was causing misalignment
	// Line 1: "⌨️ Alias for exit" (keyboard emoji with variation selector)
	// Line 2: "You use 'exit' often."
	// Both lines should calculate width correctly for right-alignment

	line1 := "⌨️ Alias for exit"
	line2 := "You use 'exit' often."

	width1 := stringWidthWithAnsi(line1)
	width2 := stringWidthWithAnsi(line2)

	// In test environment:
	// Line 1: ⌨(1) + ️(0) + space(1) + "Alias for exit"(14) = 16
	// Line 2: "You use 'exit' often." = 21
	expectedWidth1 := 16
	expectedWidth2 := 21

	if width1 != expectedWidth1 {
		t.Errorf("stringWidthWithAnsi(%q) = %d, want %d", line1, width1, expectedWidth1)
	}
	if width2 != expectedWidth2 {
		t.Errorf("stringWidthWithAnsi(%q) = %d, want %d", line2, width2, expectedWidth2)
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
