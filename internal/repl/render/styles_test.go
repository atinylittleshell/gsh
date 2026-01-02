package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStyledSymbol(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		success bool
		wantLen int // We just verify the output is non-empty since styled output varies
	}{
		{
			name:    "exec symbol",
			symbol:  SymbolExec,
			success: true,
			wantLen: 1,
		},
		{
			name:    "tool pending symbol",
			symbol:  SymbolToolPending,
			success: true,
			wantLen: 1,
		},
		{
			name:    "tool complete success",
			symbol:  SymbolToolComplete,
			success: true,
			wantLen: 1,
		},
		{
			name:    "tool complete failure",
			symbol:  SymbolToolComplete,
			success: false,
			wantLen: 1,
		},
		{
			name:    "success symbol",
			symbol:  SymbolSuccess,
			success: true,
			wantLen: 1,
		},
		{
			name:    "error symbol",
			symbol:  SymbolError,
			success: false,
			wantLen: 1,
		},
		{
			name:    "system message symbol",
			symbol:  SymbolSystemMessage,
			success: true,
			wantLen: 1,
		},
		{
			name:    "unknown symbol passthrough",
			symbol:  "?",
			success: true,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StyledSymbol(tt.symbol, tt.success)
			// Result should contain the original symbol (possibly with ANSI codes)
			assert.NotEmpty(t, result)
		})
	}
}

func TestSymbolConstants(t *testing.T) {
	// Verify symbols are defined correctly
	assert.Equal(t, "▶", SymbolExec)
	assert.Equal(t, "○", SymbolToolPending)
	assert.Equal(t, "●", SymbolToolComplete)
	assert.Equal(t, "✓", SymbolSuccess)
	assert.Equal(t, "✗", SymbolError)
	assert.Equal(t, "→", SymbolSystemMessage)
}
