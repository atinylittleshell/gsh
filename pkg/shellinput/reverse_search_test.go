package shellinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestReverseSearch(t *testing.T) {
	model := New()
	model.Focus()

	// Setup history
	history := []string{"git push", "git commit", "go test", "grep -r"}
	model.SetHistoryValues(history)

	// Enter reverse search mode
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)
	assert.True(t, updatedModel.inReverseSearch, "Should be in reverse search mode")
	assert.Equal(t, "", updatedModel.reverseSearchQuery, "Query should be empty initially")

	// Type 'g'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "g", updatedModel.reverseSearchQuery, "Query should update")
	// "git push" is the first match in history (index 1)
	assert.Len(t, updatedModel.reverseSearchMatches, 1, "Should have 1 match so far")
	assert.Equal(t, 1, updatedModel.reverseSearchMatches[0], "First match should be index 1 (git push)")

	// Type 'o' -> "go"
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "go", updatedModel.reverseSearchQuery, "Query should update")
	// "go test" is at index 3
	assert.Len(t, updatedModel.reverseSearchMatches, 1, "Should have 1 match")
	assert.Equal(t, 3, updatedModel.reverseSearchMatches[0], "Match should be index 3 (go test)")

	// Backspace to 'g'
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "g", updatedModel.reverseSearchQuery, "Query should update")
	// Should be back to "git push" (index 1) as we search from beginning for new query
	assert.Equal(t, 1, updatedModel.reverseSearchMatches[0], "Match should be index 1 (git push)")

	// Press Ctrl+R again to find next match
	msg = tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ = updatedModel.Update(msg)
	// Next match with 'g' is "git commit" (index 2)
	assert.Equal(t, 2, len(updatedModel.reverseSearchMatches), "Should have found 2 matches in sequence")
	assert.Equal(t, 2, updatedModel.reverseSearchMatches[1], "Second match should be index 2 (git commit)")

	// Press Ctrl+R again
	updatedModel, _ = updatedModel.Update(msg)
	// Next match with 'g' is "go test" (index 3)
	assert.Equal(t, 3, len(updatedModel.reverseSearchMatches), "Should have found 3 matches in sequence")
	assert.Equal(t, 3, updatedModel.reverseSearchMatches[2], "Third match should be index 3 (go test)")

	// Press Ctrl+R again
	updatedModel, _ = updatedModel.Update(msg)
	// Next match with 'g' is "grep -r" (index 4)
	assert.Equal(t, 4, len(updatedModel.reverseSearchMatches), "Should have found 4 matches in sequence")
	assert.Equal(t, 4, updatedModel.reverseSearchMatches[3], "Fourth match should be index 4 (grep -r)")

	// Accept match
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.inReverseSearch, "Should exit reverse search mode")
	assert.Equal(t, "grep -r", updatedModel.Value(), "Value should be the selected match")
}

func TestReverseSearchCancel(t *testing.T) {
	model := New()
	model.Focus()
	model.SetHistoryValues([]string{"ls -la"})

	// Enter reverse search
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)
	assert.True(t, updatedModel.inReverseSearch)

	// Type 'l'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "l", updatedModel.reverseSearchQuery)

	// Cancel with Ctrl+G
	msg = tea.KeyMsg{Type: tea.KeyCtrlG}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.inReverseSearch)
	assert.Equal(t, "", updatedModel.Value(), "Value should be empty as we started with empty input")
}

func TestReverseSearchNoMatch(t *testing.T) {
	model := New()
	model.Focus()
	model.SetHistoryValues([]string{"ls"})

	// Enter reverse search
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)

	// Type 'z' (no match)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Empty(t, updatedModel.reverseSearchMatches, "Should have no matches")
}

func TestReverseSearchUTF8(t *testing.T) {
	model := New()
	model.Focus()
	model.SetHistoryValues([]string{"echo 🚀", "echo 👋"})

	// Enter reverse search
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)

	// Type '👋'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("👋")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "👋", updatedModel.reverseSearchQuery)
	assert.Equal(t, 1, len(updatedModel.reverseSearchMatches), "Should match 'echo 👋'")

	// Backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "", updatedModel.reverseSearchQuery, "Should correctly delete multibyte character")
}

func TestReverseSearchNavigation(t *testing.T) {
	model := New()
	model.Focus()

	// History: [1]"echo A", [2]"echo B", [3]"echo C"
	history := []string{"echo A", "echo B", "echo C"}
	model.SetHistoryValues(history)

	// Enter search, type "echo"
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	model, _ = model.Update(msg)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("echo")}
	model, _ = model.Update(msg)

	// Should match [1]"echo A"
	assert.Equal(t, 1, model.reverseSearchMatches[model.reverseSearchMatchIndex])

	// Up Arrow -> Next match (older) -> [2]"echo B"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	model, _ = model.Update(msg)
	assert.Equal(t, 2, model.reverseSearchMatches[model.reverseSearchMatchIndex])

	// Up Arrow -> Next match (older) -> [3]"echo C"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	model, _ = model.Update(msg)
	assert.Equal(t, 3, model.reverseSearchMatches[model.reverseSearchMatchIndex])

	// Down Arrow -> Prev match (newer) -> [2]"echo B"
	msg = tea.KeyMsg{Type: tea.KeyDown}
	model, _ = model.Update(msg)
	assert.Equal(t, 2, model.reverseSearchMatches[model.reverseSearchMatchIndex])

	// Down Arrow -> Prev match (newer) -> [1]"echo A"
	msg = tea.KeyMsg{Type: tea.KeyDown}
	model, _ = model.Update(msg)
	assert.Equal(t, 1, model.reverseSearchMatches[model.reverseSearchMatchIndex])

	// Right Arrow -> Accept
	msg = tea.KeyMsg{Type: tea.KeyRight}
	model, _ = model.Update(msg)
	assert.False(t, model.inReverseSearch)
	assert.Equal(t, "echo A", model.Value())
}

func TestReverseSearchDuplicates(t *testing.T) {
	model := New()
	model.Focus()

	// History: [1]"echo A", [2]"echo B", [3]"echo A", [4]"echo C"
	history := []string{"echo A", "echo B", "echo A", "echo C"}
	model.SetHistoryValues(history)

	// Enter search, type "echo"
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	model, _ = model.Update(msg)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("echo")}
	model, _ = model.Update(msg)

	// Should match [1]"echo A"
	assert.Equal(t, 1, model.reverseSearchMatches[model.reverseSearchMatchIndex])
	assert.Equal(t, "echo A", string(model.values[model.reverseSearchMatches[0]]))

	// Up Arrow -> Next match (older) -> [2]"echo B"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	model, _ = model.Update(msg)
	assert.Equal(t, 2, model.reverseSearchMatches[model.reverseSearchMatchIndex])
	assert.Equal(t, "echo B", string(model.values[model.reverseSearchMatches[1]]))

	// Up Arrow -> Next match (older) -> Should skip [3]"echo A" (duplicate) and find [4]"echo C"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	model, _ = model.Update(msg)
	assert.Equal(t, 4, model.reverseSearchMatches[model.reverseSearchMatchIndex])
	assert.Equal(t, "echo C", string(model.values[model.reverseSearchMatches[2]]))

	// Verify we only have 3 unique matches
	assert.Len(t, model.reverseSearchMatches, 3)
}
