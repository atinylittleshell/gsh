package shellinput

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionBoxView_SingleColumn(t *testing.T) {
	m := New()
	m.suggestions = [][]rune{[]rune("A"), []rune("B"), []rune("C")}
	m.ShowSuggestions = true

	// Manually set up completion state
	m.completion.active = true
	m.completion.suggestions = []string{"A", "B", "C"}
	m.completion.showInfoBox = true
	m.completion.selected = 0

	// Height 3, enough for all items
	view := m.CompletionBoxView(3, true)

	// Should show 3 lines
	lines := strings.Split(strings.TrimSpace(view), "\n")
	assert.Equal(t, 3, len(lines))
	assert.Contains(t, lines[0], "A")
	assert.Contains(t, lines[1], "B")
	assert.Contains(t, lines[2], "C")
}

func TestCompletionBoxView_TwoColumns(t *testing.T) {
	m := New()
	m.ShowSuggestions = true

	// 6 items
	items := []string{"1", "2", "3", "4", "5", "6"}
	m.completion.active = true
	m.completion.suggestions = items
	m.completion.showInfoBox = true
	m.completion.selected = 0

	// Height 3. Capacity = 3 * 2 = 6. All fits.
	view := m.CompletionBoxView(3, true)

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

func TestCompletionBoxView_DisableColumns(t *testing.T) {
	m := New()
	m.ShowSuggestions = true

	// 6 items
	items := []string{"1", "2", "3", "4", "5", "6"}
	m.completion.active = true
	m.completion.suggestions = items
	m.completion.showInfoBox = true
	m.completion.selected = 0

	// Height 3. useColumns = false.
	// Should show 3 items (scrolling), single column.
	view := m.CompletionBoxView(3, false)

	lines := strings.Split(strings.TrimSpace(view), "\n")
	assert.Equal(t, 3, len(lines))

	// Line 1: "1"
	assert.Contains(t, lines[0], "1")
	assert.NotContains(t, lines[0], "4") // 4 should not be visible or on same line

	// Line 2: "2"
	assert.Contains(t, lines[1], "2")

	// Line 3: "3"
	assert.Contains(t, lines[2], "3")
}

func TestCompletionBoxView_Paging(t *testing.T) {
	m := New()
	m.ShowSuggestions = true

	// 12 items
	items := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	m.completion.active = true
	m.completion.suggestions = items
	m.completion.showInfoBox = true

	// Height 3. Columns 2. Capacity 6.
	// Page 0: 1-6
	// Page 1: 7-12

	// Select item 7 (index 6). Should trigger Page 1.
	m.completion.selected = 6

	view := m.CompletionBoxView(3, true)
	lines := strings.Split(strings.TrimSpace(view), "\n")

	assert.Equal(t, 3, len(lines))

	// Line 1: "7" and "10"
	assert.Contains(t, lines[0], "> 7")
	assert.Contains(t, lines[0], "10")

	// Ensure "1" is NOT present
	assert.NotContains(t, view, " 1 ")
}
