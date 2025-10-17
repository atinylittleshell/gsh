package shellinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockContextCompletionProvider implements CompletionProvider for testing context-sensitive completions
type mockContextCompletionProvider struct{}

// mockShortLongCompletionProvider tests completions of different lengths
type mockShortLongCompletionProvider struct{}

func (m *mockShortLongCompletionProvider) GetCompletions(line string, pos int) []string {
	if line == "@!" {
		return []string{"@!short", "@!longer_completion"}
	}
	return []string{}
}

func (m *mockShortLongCompletionProvider) GetHelpInfo(line string, pos int) string {
	return ""
}

func (m *mockContextCompletionProvider) GetCompletions(line string, pos int) []string {
	// Handle exact matches first
	switch line {
	case "@/":
		return []string{"@/macro1", "@/macro2", "@/macro3"}
	case "@!":
		return []string{"@!gsh_analytics", "@!gsh_evaluate", "@!history", "@!complete"}
	case "@/m":
		return []string{"@/macro1", "@/macro2"}
	case "@!g":
		return []string{"@!gsh_analytics", "@!gsh_evaluate"}
	case "@/macro1":
		return []string{"@/macro1", "@/macro2", "@/macro3"} // Return all macros for cycling
	}
	return []string{}
}

func (m *mockContextCompletionProvider) GetHelpInfo(line string, pos int) string {
	return ""
}

func TestContextSensitiveCompletions(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = &mockContextCompletionProvider{}

	// Test @/ macro completion
	model.SetValue("@/")
	model.SetCursor(2) // cursor at end of "@/"

	// First TAB should complete to "@/macro1"
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "@/macro1", updatedModel.Value(), "First TAB should complete '@/ to '@/macro1'")
	assert.True(t, updatedModel.completion.active, "Completion should be active")

	// Second TAB should complete to "@/macro2"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@/macro2", updatedModel.Value(), "Second TAB should complete to '@/macro2'")

	// Third TAB should cycle to "@/macro3" (cycling through all options)
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@/macro3", updatedModel.Value(), "Third TAB should cycle to '@/macro3'")

	// Test Shift+TAB cycles backwards
	msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@/macro2", updatedModel.Value(), "Shift+TAB should cycle backwards to '@/macro2'")

	// Test @! builtin command completion
	model.SetValue("@!")
	model.SetCursor(2) // cursor at end of "@!"

	// First TAB should complete to "@!gsh_analytics"
	msg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "@!gsh_analytics", updatedModel.Value(), "First TAB should complete '@! to '@!gsh_analytics'")

	// Second TAB should complete to "@!gsh_evaluate"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@!gsh_evaluate", updatedModel.Value(), "Second TAB should complete to '@!gsh_evaluate'")

	// Test completion reset on other key press
	model.SetValue("@/")
	model.SetCursor(2)
	updatedModel, _ = model.Update(msg) // Activate completion
	assert.True(t, updatedModel.completion.active, "Completion should be active")

	// Press space to reset completion
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}
	updatedModel, _ = updatedModel.Update(msg)
	assert.False(t, updatedModel.completion.active, "Completion should be reset on other key press")
}

func TestContextSensitivePartialCompletions(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = &mockContextCompletionProvider{}

	// Test partial @/ completion
	model.SetValue("@/m")
	model.SetCursor(3) // cursor at end of "@/m"

	// TAB should complete to "@/macro1" (filtering based on 'm')
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "@/macro1", updatedModel.Value(), "TAB should filter and complete '@/m to '@/macro1'")

	// Test partial @! completion
	model.SetValue("@!g")
	model.SetCursor(3) // cursor at end of "@!g"

	// TAB should complete to "@!gsh_analytics" (filtering based on 'g')
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "@!gsh_analytics", updatedModel.Value(), "TAB should filter and complete '@!g to '@!gsh_analytics'")
}

func TestContextSensitiveCompletionEdgeCases(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = &mockContextCompletionProvider{}

	// Test no completion available
	model.SetValue("@/xyz")
	model.SetCursor(5)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(msg)
	// When no completions are available, the completion system should not activate
	// So the value should remain unchanged and completion should not be active
	assert.Equal(t, "@/xyz", updatedModel.Value(), "Value should not change when no completion available")
	assert.False(t, updatedModel.completion.active, "Completion should not be active when no suggestions available")

	// Test empty @/ completion
	model.SetValue("@/")
	model.SetCursor(2)

	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "@/macro1", updatedModel.Value(), "Should complete empty '@/ to first macro")
	assert.True(t, updatedModel.completion.active, "Completion should be active")

	// Test single match
	model.SetValue("@/macro1")
	model.SetCursor(8) // cursor at end of "@/macro1"

	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "@/macro1", updatedModel.Value(), "Should stay the same when only one match")
	assert.True(t, updatedModel.completion.active, "Completion should still be active for cycling")
}

func TestTextRetentionFix(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = &mockContextCompletionProvider{}

	// Test cycling through completions of different lengths
	// This specifically tests the fix for the text retention issue
	model.SetValue("@!")
	model.SetCursor(2) // cursor at end of "@!"

	// First TAB should complete to "@!gsh_analytics" (longer completion)
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "@!gsh_analytics", updatedModel.Value(), "First TAB should complete '@! to '@!gsh_analytics'")
	assert.True(t, updatedModel.completion.active, "Completion should be active")

	// Second TAB should complete to "@!gsh_evaluate" (same length)
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@!gsh_evaluate", updatedModel.Value(), "Second TAB should complete to '@!gsh_evaluate'")
	assert.True(t, updatedModel.completion.active, "Completion should still be active")

	// Third TAB should cycle to "@!history" - this tests that no text retention occurs
	// Before the fix, this would result in "@!historyuate" (retaining characters from the previous completion)
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@!history", updatedModel.Value(), "Third TAB should cycle to '@!history' without text retention")
	assert.True(t, updatedModel.completion.active, "Completion should still be active")

	// Test with a shorter completion following a longer one
	// Create a mock provider that has completions of different lengths
	shortLongProvider := &mockShortLongCompletionProvider{}
	model.CompletionProvider = shortLongProvider

	model.SetValue("@!")
	model.SetCursor(2)

	// First completion: first in the list (short)
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "@!short", updatedModel.Value(), "Should complete to first completion")

	// Second completion: second in the list (longer) - this was the main bug scenario
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@!longer_completion", updatedModel.Value(), "Should complete to longer completion without text retention issues")

	// Third completion: cycle back to first (short) - this tests the fix
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "@!short", updatedModel.Value(), "Should complete back to shorter completion without retaining old text")
}
