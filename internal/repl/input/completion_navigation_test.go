package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionNavigationWithArrowKeys(t *testing.T) {
	// Create a model with a completion provider that returns multiple items
	provider := &mockCompletionProvider{
		completions: []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"},
	}

	model := New(Config{
		Prompt:             "> ",
		CompletionProvider: provider,
	})

	// Type "cat " to set up for file completion
	model.buffer.SetText("cat ")
	model.buffer.SetPos(4)

	// Press Tab to trigger completion
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := model.Update(tabKey)
	model = newModel.(Model)

	// Verify completion is active and visible
	assert.True(t, model.completion.IsActive(), "Completion should be active after Tab")
	assert.True(t, model.completion.IsVisible(), "Completion should be visible with multiple suggestions")
	assert.Equal(t, 4, len(model.completion.Suggestions()), "Should have 4 suggestions")

	// Get the initial view
	initialView := model.View()
	t.Logf("Initial view after Tab:\n%s", initialView)
	assert.NotEmpty(t, initialView, "Initial view should not be empty")
	assert.Contains(t, initialView, "> ", "View should contain prompt")

	// Press Down arrow
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = model.Update(downKey)
	model = newModel.(Model)

	// Get the view after pressing down
	viewAfterDown := model.View()
	t.Logf("View after Down arrow:\n%s", viewAfterDown)

	// The view should still contain the prompt and input
	assert.NotEmpty(t, viewAfterDown, "View after Down should not be empty")
	assert.Contains(t, viewAfterDown, "> ", "View should still contain prompt after Down")
	assert.Contains(t, viewAfterDown, "cat ", "View should still contain input text after Down")

	// Check completion state after down arrow
	t.Logf("Completion active after Down: %v", model.completion.IsActive())
	t.Logf("Completion visible after Down: %v", model.completion.IsVisible())
}

func TestCompletionStateAfterArrowKeys(t *testing.T) {
	provider := &mockCompletionProvider{
		completions: []string{"file1.txt", "file2.txt", "file3.txt"},
	}

	model := New(Config{
		Prompt:             "> ",
		CompletionProvider: provider,
	})

	model.buffer.SetText("cat ")
	model.buffer.SetPos(4)

	// Press Tab to trigger completion
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := model.Update(tabKey)
	model = newModel.(Model)

	require.True(t, model.completion.IsActive())
	initialSelected := model.completion.Selected()
	t.Logf("Initial selected index: %d", initialSelected)

	// Check what action Down arrow maps to
	keymap := DefaultKeyMap()
	downAction := keymap.Lookup(tea.KeyMsg{Type: tea.KeyDown})
	t.Logf("Down arrow maps to action: %v", downAction)
	assert.Equal(t, ActionCursorDown, downAction)

	upAction := keymap.Lookup(tea.KeyMsg{Type: tea.KeyUp})
	t.Logf("Up arrow maps to action: %v", upAction)
	assert.Equal(t, ActionCursorUp, upAction)

	// Press Down arrow - this should either navigate completion OR trigger history
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = model.Update(downKey)
	model = newModel.(Model)

	t.Logf("After Down - Completion active: %v, Selected: %d", model.completion.IsActive(), model.completion.Selected())

	// The problem: Down arrow resets completion because it's not in the "allowed" list
	// in handleKeyMsg lines 286-288
}

func TestKeyMapActionsForVerticalNavigation(t *testing.T) {
	keymap := DefaultKeyMap()

	// Check what actions arrow keys map to
	tests := []struct {
		name     string
		key      tea.KeyMsg
		expected Action
	}{
		{"Down arrow", tea.KeyMsg{Type: tea.KeyDown}, ActionCursorDown},
		{"Up arrow", tea.KeyMsg{Type: tea.KeyUp}, ActionCursorUp},
		{"Tab", tea.KeyMsg{Type: tea.KeyTab}, ActionComplete},
		{"Shift+Tab", tea.KeyMsg{Type: tea.KeyShiftTab}, ActionCompleteBackward},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := keymap.Lookup(tt.key)
			t.Logf("%s maps to %v (expected %v)", tt.name, action, tt.expected)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestRendererViewContainsPromptAfterCompletionReset(t *testing.T) {
	renderer := NewRenderer(DefaultRenderConfig())
	renderer.SetWidth(80)

	buffer := NewBuffer()
	buffer.SetText("cat ")

	completion := NewCompletionState()
	completion.Activate([]string{"file1.txt", "file2.txt", "file3.txt"}, "", 4, 4)
	completion.NextSuggestion() // Select first item

	// Render with completion visible
	view1 := renderer.RenderFullView("> ", buffer, "", true, completion, nil, 0)
	t.Logf("View with completion:\n%s", view1)
	assert.Contains(t, view1, "> ", "View should contain prompt")
	assert.Contains(t, view1, "file1.txt", "View should contain completion suggestions")

	// Reset completion (simulating what happens when escape is pressed)
	completion.Reset()

	// Render without completion
	view2 := renderer.RenderFullView("> ", buffer, "", true, completion, nil, 0)
	t.Logf("View after completion reset:\n%s", view2)

	// The view should still contain the prompt and input
	assert.Contains(t, view2, "> ", "View should contain prompt")
	assert.Contains(t, view2, "cat ", "View should contain input text")
	// Completion suggestions should no longer be in the view
	assert.NotContains(t, view2, "file1.txt", "View should not contain completion suggestions after reset")
}

func TestCompletionNotResetByArrowKeys(t *testing.T) {
	// This test documents the CURRENT behavior and what SHOULD happen
	provider := &mockCompletionProvider{
		completions: []string{"file1.txt", "file2.txt", "file3.txt"},
	}

	model := New(Config{
		Prompt:             "> ",
		CompletionProvider: provider,
	})

	model.buffer.SetText("cat ")
	model.buffer.SetPos(4)

	// Press Tab to trigger completion
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := model.Update(tabKey)
	model = newModel.(Model)

	require.True(t, model.completion.IsActive(), "Completion should be active")
	selectedBefore := model.completion.Selected()

	// Press Down arrow
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = model.Update(downKey)
	model = newModel.(Model)

	// CURRENT BEHAVIOR: Completion gets reset because Down maps to ActionCursorDown
	// which is not in the "allowed during completion" list
	t.Logf("Completion active after Down: %v (expected: true for proper UX)", model.completion.IsActive())
	t.Logf("Selected before: %d, after: %d", selectedBefore, model.completion.Selected())

	// This documents the bug: completion should remain active and navigate
	// For now, we're just documenting behavior
	if !model.completion.IsActive() {
		t.Log("BUG: Down arrow should navigate completion, not dismiss it")
	}
}
