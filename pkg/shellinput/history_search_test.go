package shellinput

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestRichHistorySearch(t *testing.T) {
	model := New()
	model.Focus()

	// Setup rich history
	now := time.Now()
	history := []HistoryItem{
		{Command: "git push", Timestamp: now, Directory: "/home/user/project"},
		{Command: "git commit", Timestamp: now.Add(-1 * time.Hour), Directory: "/home/user/project"},
		{Command: "go test", Timestamp: now.Add(-2 * time.Hour), Directory: "/home/user/project"},
		{Command: "grep -r", Timestamp: now.Add(-3 * time.Hour), Directory: "/tmp"},
	}
	model.SetRichHistory(history)
	model.SetCurrentDirectory("/home/user/project")

	// Enter reverse search mode
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)
	assert.True(t, updatedModel.inReverseSearch, "Should be in reverse search mode")

	// Initial state should show all items (since query is empty)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 4)

	// Type 'g'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "g", updatedModel.reverseSearchQuery)
	// Matches: git push, git commit, go test, grep -r (all contain 'g')
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 4)

	// Type 'o' -> "go"
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "go", updatedModel.reverseSearchQuery)
	// Matches: "git commit" (g...o...) and "go test"
	// So we expect 2 matches.
	// Filter logic: "git push" (no o), "git commit" (g..o), "go test" (go), "grep -r" (g..r)
	// "grep -r" has 'r' not 'o'.
	// So indices 1 and 2.
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 2)
	// Assuming order is preserved from source or ranked?
	// fuzzy ranks matches. "go test" is likely better match for "go" than "git commit".
	// But let's just check presence.
	assert.Contains(t, updatedModel.historySearchState.filteredIndices, 1)
	assert.Contains(t, updatedModel.historySearchState.filteredIndices, 2)

	// Backspace to 'g'
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 4)

	// Navigation
	// Type "git" -> Matches: git push (0), git commit (1)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("it")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 2)
	assert.Equal(t, 0, updatedModel.historySearchState.selected) // First item selected

	// Down -> Select next (git commit)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, 1, updatedModel.historySearchState.selected)

	// Accept
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.inReverseSearch)
	assert.Equal(t, "git commit", updatedModel.Value())
}

func TestHistoryFiltering(t *testing.T) {
	model := New()
	model.Focus()

	// Setup rich history
	now := time.Now()
	history := []HistoryItem{
		{Command: "cmd1", Timestamp: now, Directory: "/dir1"},
		{Command: "cmd2", Timestamp: now, Directory: "/dir2"},
		{Command: "cmd3", Timestamp: now, Directory: "/dir1"},
	}
	model.SetRichHistory(history)
	model.SetCurrentDirectory("/dir1")

	// Enter reverse search
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)

	// Default: Filter All
	assert.Equal(t, HistoryFilterAll, updatedModel.historySearchState.filterMode)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 3)

	// Toggle Filter (Ctrl+F) -> Directory
	// Ctrl+F is handled in Update by checking string "ctrl+f"
	// We need to simulate the key press manually or through Update if we mapped it?
	// In my code I mapped msg.String() == "ctrl+f"
	// BubbleTea Ctrl+F is usually KeyCtrlF but I used string check.
	// Let's create a Msg with string "ctrl+f" or KeyCtrlF
	msg = tea.KeyMsg{Type: tea.KeyCtrlF}
	updatedModel, _ = updatedModel.Update(msg)

	assert.Equal(t, HistoryFilterDirectory, updatedModel.historySearchState.filterMode)
	// Should match cmd1 and cmd3 (dir1)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 2)
	assert.Equal(t, 0, updatedModel.historySearchState.filteredIndices[0]) // cmd1
	assert.Equal(t, 2, updatedModel.historySearchState.filteredIndices[1]) // cmd3

	// Toggle Filter -> All (cycling back, skipping Session for now)
	msg = tea.KeyMsg{Type: tea.KeyCtrlF}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, HistoryFilterAll, updatedModel.historySearchState.filterMode)
	assert.Len(t, updatedModel.historySearchState.filteredIndices, 3)
}

func TestRichHistorySearchCancel(t *testing.T) {
	model := New()
	model.Focus()
	model.SetRichHistory([]HistoryItem{{Command: "ls -la"}})

	// Enter reverse search
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updatedModel, _ := model.Update(msg)
	assert.True(t, updatedModel.inReverseSearch)

	// Cancel with Escape
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.inReverseSearch)
}
