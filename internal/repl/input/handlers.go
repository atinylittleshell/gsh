package input

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// handleSubmit handles the Enter key.
func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	m.result = Result{
		Type:  ResultSubmit,
		Value: m.buffer.Text(),
	}
	return m, tea.Quit
}

// handleInterrupt handles Ctrl+C.
func (m Model) handleInterrupt() (tea.Model, tea.Cmd) {
	m.result = Result{
		Type:  ResultInterrupt,
		Value: "",
	}
	return m, tea.Quit
}

// handleEOF handles Ctrl+D on empty input.
func (m Model) handleEOF() (tea.Model, tea.Cmd) {
	m.result = Result{
		Type:  ResultEOF,
		Value: "",
	}
	return m, tea.Quit
}

// handleCancel handles the Escape key.
func (m Model) handleCancel() (tea.Model, tea.Cmd) {
	if m.completion.IsActive() {
		m.completion.Reset()
	}
	return m, nil
}

// handleCharacterForward handles moving cursor forward or accepting prediction.
func (m Model) handleCharacterForward() (tea.Model, tea.Cmd) {
	if m.buffer.Pos() < m.buffer.Len() {
		// Normal case: move cursor forward
		m.buffer.SetPos(m.buffer.Pos() + 1)
	} else if m.currentPrediction != "" && strings.HasPrefix(m.currentPrediction, m.buffer.Text()) {
		// At end of input with valid prediction: accept prediction
		return m.handleAcceptPrediction()
	}
	return m, nil
}

// handleAcceptPrediction accepts the current prediction.
func (m Model) handleAcceptPrediction() (tea.Model, tea.Cmd) {
	if m.currentPrediction == "" {
		return m, nil
	}

	text := m.buffer.Text()
	if !strings.HasPrefix(m.currentPrediction, text) {
		return m, nil
	}

	// Accept the prediction
	m.buffer.SetText(m.currentPrediction)
	m.currentPrediction = ""

	return m, nil
}

// handleDeleteCharacterBackward handles Backspace.
func (m Model) handleDeleteCharacterBackward() (tea.Model, tea.Cmd) {
	if m.buffer.Len() == 0 {
		// Clear prediction when deleting from empty
		m.currentPrediction = ""
		return m, nil
	}

	oldText := m.buffer.Text()
	m.buffer.DeleteCharBackward()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleDeleteCharacterForward handles Delete.
func (m Model) handleDeleteCharacterForward() (tea.Model, tea.Cmd) {
	oldText := m.buffer.Text()
	m.buffer.DeleteCharForward()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleDeleteWordBackward handles Ctrl+W / Alt+Backspace.
func (m Model) handleDeleteWordBackward() (tea.Model, tea.Cmd) {
	oldText := m.buffer.Text()
	m.buffer.DeleteWordBackward()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleDeleteWordForward handles Alt+D.
func (m Model) handleDeleteWordForward() (tea.Model, tea.Cmd) {
	oldText := m.buffer.Text()
	m.buffer.DeleteWordForward()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleDeleteBeforeCursor handles Ctrl+U.
func (m Model) handleDeleteBeforeCursor() (tea.Model, tea.Cmd) {
	oldText := m.buffer.Text()
	m.buffer.DeleteBeforeCursor()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleDeleteAfterCursor handles Ctrl+K.
func (m Model) handleDeleteAfterCursor() (tea.Model, tea.Cmd) {
	oldText := m.buffer.Text()
	m.buffer.DeleteAfterCursor()
	newText := m.buffer.Text()

	if oldText != newText {
		return m.onTextChanged()
	}
	return m, nil
}

// handleInsertRunes inserts characters at the cursor position.
func (m Model) handleInsertRunes(runes []rune) (tea.Model, tea.Cmd) {
	// Sanitize input: replace tabs and newlines with spaces
	sanitized := sanitizeRunes(runes)
	m.buffer.InsertRunes(sanitized)
	m.historyIndex = 0
	m.hasNavigatedHistory = false

	return m.onTextChanged()
}

// handlePaste handles pasted text.
func (m Model) handlePaste(text string) (tea.Model, tea.Cmd) {
	// Sanitize pasted text
	sanitized := sanitizeRunes([]rune(text))
	m.buffer.InsertRunes(sanitized)
	m.historyIndex = 0
	m.hasNavigatedHistory = false

	return m.onTextChanged()
}

// handleHistoryPrevious navigates to the previous history entry (older).
func (m Model) handleHistoryPrevious() (tea.Model, tea.Cmd) {
	if len(m.historyValues) == 0 {
		return m, nil
	}

	// Save current input if this is the first navigation
	if !m.hasNavigatedHistory {
		m.savedCurrentInput = m.buffer.Text()
		m.hasNavigatedHistory = true
	}

	// Move to older history entry
	if m.historyIndex < len(m.historyValues) {
		m.historyIndex++
		m.buffer.SetText(m.historyValues[m.historyIndex-1])
	}

	return m, nil
}

// handleHistoryNext navigates to the next history entry (newer).
func (m Model) handleHistoryNext() (tea.Model, tea.Cmd) {
	if m.historyIndex <= 0 {
		return m, nil
	}

	m.historyIndex--
	if m.historyIndex == 0 {
		// Return to current input
		m.buffer.SetText(m.savedCurrentInput)
	} else {
		m.buffer.SetText(m.historyValues[m.historyIndex-1])
	}

	return m, nil
}

// handleComplete handles Tab completion.
func (m Model) handleComplete() (tea.Model, tea.Cmd) {
	if m.completionProvider == nil {
		return m, nil
	}

	if m.completion.IsActive() {
		// Already in completion mode, cycle to next suggestion
		suggestion := m.completion.NextSuggestion()
		if suggestion != "" {
			m.applyCompletion(suggestion)
		}
		return m, nil
	}

	// Start new completion
	text := m.buffer.Text()
	pos := m.buffer.Pos()

	suggestions := m.completionProvider.GetCompletions(text, pos)
	if len(suggestions) == 0 {
		return m, nil
	}

	// Find word boundaries for the completion
	start, end := GetWordBoundary(text, pos)

	// Get the prefix being completed
	prefix := ""
	if start < len(text) {
		if end > len(text) {
			end = len(text)
		}
		prefix = text[start:end]
	}

	// Activate completion
	m.completion.Activate(suggestions, prefix, start, end)
	m.completion.SetOriginalText(text)

	// If only one suggestion, apply it immediately
	if len(suggestions) == 1 {
		m.applyCompletion(suggestions[0])
		m.completion.Reset()
	} else {
		// Select first suggestion
		suggestion := m.completion.NextSuggestion()
		if suggestion != "" {
			m.applyCompletion(suggestion)
		}
	}

	return m, nil
}

// handleCompleteBackward handles Shift+Tab.
func (m Model) handleCompleteBackward() (tea.Model, tea.Cmd) {
	if !m.completion.IsActive() {
		return m, nil
	}

	suggestion := m.completion.PrevSuggestion()
	if suggestion != "" {
		m.applyCompletion(suggestion)
	}

	return m, nil
}

// handleCompletionAction handles actions when completion is active.
func (m Model) handleCompletionAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionComplete:
		return m.handleComplete()
	case ActionCompleteBackward:
		return m.handleCompleteBackward()
	case ActionCancel:
		// Restore original text
		originalText := m.completion.Cancel()
		if originalText != "" {
			m.buffer.SetText(originalText)
		}
		return m, nil
	}
	return m, nil
}

// applyCompletion applies a completion suggestion to the buffer.
func (m *Model) applyCompletion(suggestion string) {
	start := m.completion.StartPos()
	end := m.completion.EndPos()
	text := m.buffer.Text()

	result := ApplySuggestion(text, suggestion, start, end)
	m.buffer.SetText(result.NewText)
	m.buffer.SetPos(result.NewCursorPos)

	// Update completion boundaries for next cycle
	newStart, newEnd := GetWordBoundary(result.NewText, result.NewCursorPos)
	m.completion.UpdateBoundaries(suggestion, newStart, newEnd)
}

// sanitizeRunes cleans up input runes by replacing tabs and newlines with spaces.
func sanitizeRunes(runes []rune) []rune {
	result := make([]rune, len(runes))
	for i, r := range runes {
		switch r {
		case '\t', '\n', '\r':
			result[i] = ' '
		default:
			result[i] = r
		}
	}
	return result
}
