package shellinput

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupCompletionModel(items []string) Model {
	m := New()
	m.ShowSuggestions = true
	m.completion.active = true
	// Convert strings to CompletionCandidate
	candidates := make([]CompletionCandidate, len(items))
	for i, item := range items {
		candidates[i] = CompletionCandidate{Value: item, Display: item}
	}
	m.completion.suggestions = candidates
	m.completion.showInfoBox = true
	m.completion.selected = 0
	// Also set suggestions field for consistency, though logic mostly uses completion.suggestions
	runes := make([][]rune, len(items))
	for i, item := range items {
		runes[i] = []rune(item)
	}
	m.suggestions = runes
	return m
}

func TestCompletionBoxView_SingleColumn(t *testing.T) {
	m := setupCompletionModel([]string{"A", "B", "C"})

	// Height 3. Items <= Height, so single column forced.
	// Width 100.
	view := m.CompletionBoxView(3, 100)

	// Should show 3 lines
	lines := strings.Split(strings.TrimSpace(view), "\n")
	assert.Equal(t, 3, len(lines))
	assert.Contains(t, lines[0], "A")
	assert.Contains(t, lines[1], "B")
	assert.Contains(t, lines[2], "C")
}

func TestCompletionBoxView_TwoColumns(t *testing.T) {
	// 6 items
	m := setupCompletionModel([]string{"1", "2", "3", "4", "5", "6"})

	// Height 3.
	// Item width: 1 char + 4 (padding) = 5.
	// Min item width set to 10 in code.
	// To get 2 columns: width >= 20.
	view := m.CompletionBoxView(3, 25)

	lines := strings.Split(strings.TrimSpace(view), "\n")
	assert.Equal(t, 3, len(lines), "Should have 3 lines of output")

	// Check content of line 1. Should have "1" and "4"
	// 1 is selected
	assert.Contains(t, lines[0], "> 1")
	assert.Contains(t, lines[0], "4")

	// Line 2: "2" and "5"
	assert.Contains(t, lines[1], "2")
	assert.Contains(t, lines[1], "5")

	// Line 3: "3" and "6"
	assert.Contains(t, lines[2], "3")
	assert.Contains(t, lines[2], "6")
}

func TestCompletionBoxView_NarrowWidth(t *testing.T) {
	// 6 items
	m := setupCompletionModel([]string{"1", "2", "3", "4", "5", "6"})

	// Height 3.
	// Width 10. Should result in 1 column (10/10 = 1).
	view := m.CompletionBoxView(3, 10)

	lines := strings.Split(strings.TrimSpace(view), "\n")
	assert.Equal(t, 3, len(lines))

	// Line 1: "1"
	assert.Contains(t, lines[0], "1")
	assert.NotContains(t, lines[0], "4") // 4 should not be visible

	// Line 2: "2"
	assert.Contains(t, lines[1], "2")

	// Line 3: "3"
	assert.Contains(t, lines[2], "3")
}

func TestCompletionBoxView_Paging(t *testing.T) {
	// 12 items
	m := setupCompletionModel([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"})

	// Height 3. Width 25 (2 columns). Capacity 6.
	// Page 0: 1-6
	// Page 1: 7-12

	// Select item 7 (index 6). Should trigger Page 1.
	m.completion.selected = 6

	view := m.CompletionBoxView(3, 25)
	lines := strings.Split(strings.TrimSpace(view), "\n")

	assert.Equal(t, 3, len(lines))

	// Line 1: "7" and "10"
	assert.Contains(t, lines[0], "> 7")
	assert.Contains(t, lines[0], "10")

	// Ensure "1" is NOT present
	assert.NotContains(t, view, " 1 ")
}
