package shellinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

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
