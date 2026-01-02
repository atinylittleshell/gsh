package input

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistorySearchState_BasicOperations(t *testing.T) {
	state := NewHistorySearchState()

	// Initially not active
	assert.False(t, state.IsActive())
	assert.Equal(t, "", state.Query())
	assert.Equal(t, "", state.CurrentMatch())

	// Start search
	state.Start("original input", 5)
	assert.True(t, state.IsActive())
	assert.Equal(t, "", state.Query())
	assert.Equal(t, "original input", state.OriginalInput())

	// Add characters to query
	state.AddChar('g')
	assert.Equal(t, "g", state.Query())
	state.AddChar('i')
	state.AddChar('t')
	assert.Equal(t, "git", state.Query())

	// Delete character
	assert.True(t, state.DeleteChar())
	assert.Equal(t, "gi", state.Query())

	// Delete all characters
	assert.True(t, state.DeleteChar())
	assert.True(t, state.DeleteChar())
	assert.Equal(t, "", state.Query())

	// Delete on empty query returns false
	assert.False(t, state.DeleteChar())
}

func TestHistorySearchState_Matches(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("", 0)

	// Set matches
	matches := []string{"git status", "git commit", "git push"}
	state.SetMatches(matches)

	assert.Equal(t, 3, state.MatchCount())
	assert.Equal(t, 0, state.MatchIndex())
	assert.Equal(t, "git status", state.CurrentMatch())

	// Navigate to next match
	assert.True(t, state.NextMatch())
	assert.Equal(t, 1, state.MatchIndex())
	assert.Equal(t, "git commit", state.CurrentMatch())

	assert.True(t, state.NextMatch())
	assert.Equal(t, 2, state.MatchIndex())
	assert.Equal(t, "git push", state.CurrentMatch())

	// Can't go beyond last match
	assert.False(t, state.NextMatch())
	assert.Equal(t, 2, state.MatchIndex())

	// Navigate back
	assert.True(t, state.PrevMatch())
	assert.Equal(t, 1, state.MatchIndex())
	assert.Equal(t, "git commit", state.CurrentMatch())

	assert.True(t, state.PrevMatch())
	assert.Equal(t, 0, state.MatchIndex())

	// Can't go before first match
	assert.False(t, state.PrevMatch())
	assert.Equal(t, 0, state.MatchIndex())
}

func TestHistorySearchState_EmptyMatches(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("", 0)

	// No matches
	state.SetMatches(nil)
	assert.Equal(t, 0, state.MatchCount())
	assert.Equal(t, "", state.CurrentMatch())

	// Navigation does nothing with empty matches
	assert.False(t, state.NextMatch())
	assert.False(t, state.PrevMatch())
}

func TestHistorySearchState_Cancel(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("original text", 7)
	state.AddChar('g')
	state.SetMatches([]string{"git status"})

	// Cancel should restore original input
	originalInput, originalPos := state.Cancel()
	assert.Equal(t, "original text", originalInput)
	assert.Equal(t, 7, originalPos)
	assert.False(t, state.IsActive())
	assert.Equal(t, "", state.Query())
}

func TestHistorySearchState_Accept(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("original text", 0)
	state.SetMatches([]string{"git status", "git commit"})
	state.NextMatch() // Select "git commit"

	// Accept should return the selected match
	result := state.Accept()
	assert.Equal(t, "git commit", result)
	assert.False(t, state.IsActive())
}

func TestHistorySearchState_AcceptWithNoMatches(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("original text", 0)
	// No matches set

	// Accept with no matches should return original input
	result := state.Accept()
	assert.Equal(t, "original text", result)
	assert.False(t, state.IsActive())
}

func TestHistorySearchState_Reset(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("original", 3)
	state.AddChar('x')
	state.SetMatches([]string{"match"})

	state.Reset()
	assert.False(t, state.IsActive())
	assert.Equal(t, "", state.Query())
	assert.Equal(t, 0, state.MatchCount())
	assert.Equal(t, "", state.OriginalInput())
}

func TestHistorySearchState_QueryChangeResetsMatchIndex(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("", 0)
	state.SetMatches([]string{"a", "b", "c"})
	state.NextMatch()
	state.NextMatch()
	assert.Equal(t, 2, state.MatchIndex())

	// Adding char resets match index
	state.AddChar('x')
	assert.Equal(t, 0, state.MatchIndex())

	// Set to different index
	state.SetMatches([]string{"x1", "x2"})
	state.NextMatch()
	assert.Equal(t, 1, state.MatchIndex())

	// Deleting char resets match index
	state.DeleteChar()
	assert.Equal(t, 0, state.MatchIndex())
}

func TestHistorySearchState_SetQueryResetsMatchIndex(t *testing.T) {
	state := NewHistorySearchState()
	state.Start("", 0)
	state.SetMatches([]string{"a", "b"})
	state.NextMatch()
	assert.Equal(t, 1, state.MatchIndex())

	state.SetQuery("new")
	assert.Equal(t, "new", state.Query())
	assert.Equal(t, 0, state.MatchIndex())
}
