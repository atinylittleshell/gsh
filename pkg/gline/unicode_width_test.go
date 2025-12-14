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
