package shellinput

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCompletionProvider implements CompletionProvider for testing
type mockCompletionProvider struct {
}

func (m *mockCompletionProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	// Check for exact prefix matches first
	if strings.HasPrefix(line, "git ch") {
		return []CompletionCandidate{
			{Value: "git checkout"},
			{Value: "git cherry-pick"},
		}
	}
	if strings.HasPrefix(line, "gi") {
		return []CompletionCandidate{
			{Value: "git"},
			{Value: "gist"},
			{Value: "give"},
		}
	}
	// Return empty slice for no completions
	return []CompletionCandidate{}
}

func (m *mockCompletionProvider) GetHelpInfo(line string, pos int) string {
	// Return empty string for tests - no help info needed
	return ""
}

type trackingCompletionProvider struct {
	lastLine string
	lastPos  int
}

func (t *trackingCompletionProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	return nil
}

func (t *trackingCompletionProvider) GetHelpInfo(line string, pos int) string {
	t.lastLine = line
	t.lastPos = pos
	return line
}

func TestCompletion(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = &mockCompletionProvider{}

	// Test basic completion
	model.SetValue("gi")
	model.SetCursor(2) // cursor at end of "gi"

	// First TAB should extend to the shared prefix only
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "gi", updatedModel.Value(), "First TAB should leave input at common prefix when multiple matches exist")
	assert.Equal(t, 2, updatedModel.Position(), "Cursor should remain at end of current prefix")
	assert.True(t, updatedModel.completion.active, "Completion should be active")

	// Second TAB should complete to "git"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "git", updatedModel.Value(), "Second TAB should complete to 'git'")
	assert.Equal(t, 3, updatedModel.Position(), "Cursor should be at end of completion")

	// Third TAB should complete to "gist"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "gist", updatedModel.Value(), "Third TAB should complete to 'gist'")

	// Fourth TAB should complete to "give"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "give", updatedModel.Value(), "Fourth TAB should complete to 'give'")

	// Fifth TAB should cycle back to "git"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "git", updatedModel.Value(), "Fourth TAB should cycle back to 'git'")

	// Test Shift+TAB cycles backwards
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "give", updatedModel.Value(), "Shift+TAB should cycle backwards to 'give'")

	// Test completion reset on other key press
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.completion.active, "Completion should be reset on other key press")

	// Test multi-word completion
	updatedModel.SetValue("git ch")
	updatedModel.SetCursor(6) // cursor at end of "git ch"

	// TAB should complete to the shared prefix "git che"
	msg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "git che", updatedModel.Value(), "TAB should extend 'git ch' to shared prefix 'git che'")

	// Second TAB should complete to "git checkout"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "git checkout", updatedModel.Value(), "Second TAB should complete to 'git checkout'")

	// Test no completion available
	model = New() // Reset model state
	model.Focus()
	model.CompletionProvider = &mockCompletionProvider{}
	model.SetValue("xyz")
	model.SetCursor(3)
	msg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "xyz", updatedModel.Value(), "Value should not change when no completion available")
	assert.False(t, updatedModel.completion.active, "Completion should not be active when no suggestions available")
}

func TestUpdate(t *testing.T) {
	model := New()
	model.Focus()
	model.SetValue("hello world")

	// Test backspace
	model.SetCursor(11)
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "hello worl", updatedModel.Value(), "Backspace should delete the character before the cursor")

	// Test rune input
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "hello world", updatedModel.Value(), "Rune input should insert the character at the cursor position")

	// Test delete forward
	updatedModel.SetCursor(4)
	msg = tea.KeyMsg{Type: tea.KeyDelete}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "hell world", updatedModel.Value(), "Delete should remove the character after the cursor")

	// Test moving cursor forward
	msg = tea.KeyMsg{Type: tea.KeyRight}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, 5, updatedModel.Position(), "Cursor should move forward")

	// Test moving cursor backward
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, 4, updatedModel.Position(), "Cursor should move backward")

	// Test PrevValue, changing current value to "first"
	updatedModel.SetHistoryValues([]string{"first", "second", "third"})
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "first", updatedModel.Value(), "PrevValue should move to the previous value in history")
	assert.Equal(t, 5, updatedModel.Position(), "Cursor should move to the end")

	// PrevValue again, now "second"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "second", updatedModel.Value(), "PrevValue should move to the previous value in history")
	assert.Equal(t, 6, updatedModel.Position(), "Cursor should move to the end")

	// PrevValue again, now "third"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "third", updatedModel.Value(), "PrevValue should move to the previous value in history")
	assert.Equal(t, 5, updatedModel.Position(), "Cursor should move to the end")

	// PrevValue again, still "third"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "third", updatedModel.Value(), "PrevValue should move to the previous value in history")

	// NextValue, back to "second"
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "second", updatedModel.Value(), "NextValue should move to the next value in history")

	// NextValue again, back to "first"
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "first", updatedModel.Value(), "NextValue should move to the next value in history")

	// NextValue again, back to user input
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "hell world", updatedModel.Value(), "NextValue should now return the user input value")

	// PrevValue again, now "first"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "first", updatedModel.Value(), "PrevValue should move to the previous value in history")

	// Enter a key 'd' should append character to the end of the line
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "firstd", updatedModel.Value(), "Rune input should insert the character at the cursor position")

	// Test deleting word backward, which should only affect current input not history
	msg = tea.KeyMsg{Type: tea.KeyCtrlW}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "", updatedModel.Value(), "Ctrl+W should delete the word 'first' before the cursor")

	// PrevValue again, should still get "first"
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "first", updatedModel.Value(), "PrevValue should move to the previous value in history")

	// NextValue, back to ""
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "", updatedModel.Value(), "NextValue should move back to the user input value")

	// Test deleting word forward
	updatedModel.SetValue("hello world")
	updatedModel.SetCursor(6)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "hello ", updatedModel.Value(), "Alt+D should delete the word after the cursor")

	// Test moving to the start of the line
	msg = tea.KeyMsg{Type: tea.KeyCtrlA}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, 0, updatedModel.Position(), "Home key should move the cursor to the start of the line")

	// Test moving to the end of the line
	msg = tea.KeyMsg{Type: tea.KeyCtrlE}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, len(updatedModel.Value()), updatedModel.Position(), "End key should move the cursor to the end of the line")
}

func TestCtrlKTruncatesSuggestionsAndHelp(t *testing.T) {
	provider := &trackingCompletionProvider{}

	model := New()
	model.Focus()
	model.ShowSuggestions = true
	model.CompletionProvider = provider
	model.SetSuggestions([]string{"hello"})
	model.SetValue("hello")
	model.SetCursor(3) // after "hel"

	// Ensure suggestions are available before trimming
	model.updateSuggestions()
	assert.NotEmpty(t, model.MatchedSuggestions())

	provider.lastLine = ""
	msg := tea.KeyMsg{Type: tea.KeyCtrlK}
	updatedModel, _ := model.Update(msg)

	assert.Equal(t, "hel", updatedModel.Value(), "Ctrl+K should trim user-entered text")
	assert.Empty(t, updatedModel.MatchedSuggestions(), "Ctrl+K should clear active suggestions")
	assert.Equal(t, "hel", provider.lastLine, "Help should reflect the truncated command")
	assert.Equal(t, updatedModel.Position(), provider.lastPos, "Help should use the truncated cursor position")
}

func TestCtrlUAndCtrlWRespectSuggestionsAndYank(t *testing.T) {
	provider := &trackingCompletionProvider{}

	model := New()
	model.Focus()
	model.ShowSuggestions = true
	model.CompletionProvider = provider
	model.SetSuggestions([]string{"hello world"})
	model.SetValue("hello world")
	model.SetCursor(5) // after "hello"
	model.updateSuggestions()

	provider.lastLine = ""
	msg := tea.KeyMsg{Type: tea.KeyCtrlU}
	updatedModel, _ := model.Update(msg)

	assert.Equal(t, " world", updatedModel.Value(), "Ctrl+U should trim to the cursor and keep trailing text")
	assert.Empty(t, updatedModel.MatchedSuggestions(), "Ctrl+U should clear active suggestions")
	assert.Equal(t, " world", provider.lastLine, "Help should reflect the truncated command after Ctrl+U")
	assert.Equal(t, updatedModel.Position(), provider.lastPos, "Help should use the updated cursor position after Ctrl+U")

	// Now test Ctrl+W and Ctrl+Y using the same model
	updatedModel.SetValue("hello world")
	updatedModel.SetCursor(len(updatedModel.Value()))
	updatedModel.updateSuggestions()
	provider.lastLine = ""

	msg = tea.KeyMsg{Type: tea.KeyCtrlW}
	updatedModel, _ = updatedModel.Update(msg)

	assert.Equal(t, "hello ", updatedModel.Value(), "Ctrl+W should delete the word before the cursor")
	assert.Empty(t, updatedModel.MatchedSuggestions(), "Ctrl+W should clear active suggestions")
	assert.Equal(t, "hello ", provider.lastLine, "Help should reflect the truncated command after Ctrl+W")
	assert.Equal(t, updatedModel.Position(), provider.lastPos, "Help should use the updated cursor position after Ctrl+W")

	msg = tea.KeyMsg{Type: tea.KeyCtrlY}
	updatedModel, _ = updatedModel.Update(msg)

	assert.Equal(t, "hello world", updatedModel.Value(), "Ctrl+Y should paste the most recently killed text")
	assert.NotEmpty(t, updatedModel.MatchedSuggestions(), "Ctrl+Y should resume suggestions with the restored command")
	assert.Equal(t, "hello world", provider.lastLine, "Help should reflect the yanked command")
	assert.Equal(t, updatedModel.Position(), provider.lastPos, "Help should use the cursor position after yanking")
}

func TestKillRingAppendAndYankPop(t *testing.T) {
	model := New()
	model.Focus()

	model.SetValue("hello world mars")
	model.SetCursor(5) // after "hello"

	// First forward kill (Alt+D)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "hello mars", updatedModel.Value(), "Alt+D should delete the next word")
	require.Len(t, updatedModel.killRing, 1, "Kill ring should capture the deleted word")
	assert.Equal(t, " world", string(updatedModel.killRing[0]))

	// Second forward kill (Ctrl+K) should append to the previous kill ring entry
	msg = tea.KeyMsg{Type: tea.KeyCtrlK}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "hello", updatedModel.Value(), "Ctrl+K should delete to the end of the line")
	require.Len(t, updatedModel.killRing, 1, "Kills in the same direction should append to the latest entry")
	assert.Equal(t, " world mars", string(updatedModel.killRing[0]))

	// New kill should start a fresh entry
	updatedModel.SetValue("alpha beta gamma")
	updatedModel.SetCursor(len(updatedModel.Value()))

	msg = tea.KeyMsg{Type: tea.KeyCtrlW}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "alpha beta ", updatedModel.Value(), "Ctrl+W should delete the previous word")
	require.Len(t, updatedModel.killRing, 2, "A new kill direction should start a new entry")
	assert.Equal(t, "gamma", string(updatedModel.killRing[0]))

	// Yank and yank-pop to rotate through the ring
	msg = tea.KeyMsg{Type: tea.KeyCtrlY}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "alpha beta gamma", updatedModel.Value(), "Ctrl+Y should yank the latest kill")

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}, Alt: true}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "alpha beta  world mars", updatedModel.Value(), "Alt+Y should yank-pop to the previous kill")
}
