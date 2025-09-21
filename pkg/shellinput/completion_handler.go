package shellinput

import (
	"strings"
	"unicode"
)

// getWordBoundary returns the start and end position of the word at the cursor
func (m *Model) getWordBoundary() (start, end int) {
	value := m.Value()
	if len(value) == 0 {
		return 0, 0
	}

	// Get cursor position
	pos := m.Position()

	// Find start of word
	start = pos
	for start > 0 && !unicode.IsSpace(rune(value[start-1])) {
		start--
	}

	// Find end of word
	end = pos
	for end < len(value) && !unicode.IsSpace(rune(value[end])) {
		end++
	}

	return start, end
}

// handleCompletion handles the TAB key press for completion
func (m *Model) handleCompletion() {
	if m.CompletionProvider == nil {
		return
	}

	if !m.completion.active {
		// Start a new completion
		start, end := m.getWordBoundary()
		suggestions := m.CompletionProvider.GetCompletions(m.Value(), m.Position())
		if len(suggestions) == 0 {
			m.resetCompletion() // Ensure completion state is reset
			return
		}

		// Check for context-sensitive completions (#/ and #! prefixes)
		value := m.Value()
		if len(value) >= 2 {
			if value[:2] == "#/" || value[:2] == "#!" {
				// For context-sensitive completions, use the beginning of the prefix
				start = 0
			}
		}

		// Check if this is a multi-word completion by examining the suggestions
		// If suggestions contain spaces, it might be a full phrase completion
		isMultiWord := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion, " ") {
				isMultiWord = true
				break
			}
		}

		// For multi-word completions, we need to find the start of the command
		if isMultiWord {
			// Find the start of the current command (go back to beginning of line or last space)
			line := m.Value()
			pos := m.Position()
			commandStart := pos
			for commandStart > 0 && !unicode.IsSpace(rune(line[commandStart-1])) {
				commandStart--
			}
			// If there's a space before this word, go back to find the start of the command
			if commandStart > 0 {
				// Find the start of the previous word
				prevWordStart := commandStart - 1
				for prevWordStart > 0 && unicode.IsSpace(rune(line[prevWordStart-1])) {
					prevWordStart--
				}
				for prevWordStart > 0 && !unicode.IsSpace(rune(line[prevWordStart-1])) {
					prevWordStart--
				}
				// If the suggestion starts with the same prefix as the current command, use the full command
				if len(suggestions) > 0 && strings.HasPrefix(suggestions[0], line[prevWordStart:commandStart]) {
					start = prevWordStart
				}
			}
		}

		m.completion.active = true
		m.completion.suggestions = suggestions
		m.completion.selected = -1
		m.completion.prefix = m.Value()[start:m.Position()]
		m.completion.startPos = start // Use the actual start position from word boundary
		m.completion.endPos = end     // Store the end position as well

		// Activate info box if there are multiple completions
		if len(suggestions) > 1 {
			m.completion.activateInfoBox(m.Value())
		}
	}

	// Get next suggestion (this works for both initial and subsequent TAB presses)
	suggestion := m.completion.nextSuggestion()
	if suggestion == "" {
		return
	}

	// For subsequent completions, we need to recalculate the boundaries
	// based on the current value, not the original value when completion started
	if m.completion.selected > 0 {
		start, end := m.getWordBoundary()

		// Check for context-sensitive completions (#/ and #! prefixes)
		value := m.Value()
		if len(value) >= 2 {
			if value[:2] == "#/" || value[:2] == "#!" {
				// For context-sensitive completions, use the beginning of the prefix
				start = 0
			}
		}

		// Check if this is a multi-word completion by examining the suggestions
		isMultiWord := false
		for _, suggestion := range m.completion.suggestions {
			if strings.Contains(suggestion, " ") {
				isMultiWord = true
				break
			}
		}

		// For multi-word completions, we need to find the start of the command
		if isMultiWord {
			// Find the start of the current command (go back to beginning of line or last space)
			line := m.Value()
			pos := m.Position()
			commandStart := pos
			for commandStart > 0 && !unicode.IsSpace(rune(line[commandStart-1])) {
				commandStart--
			}
			// If there's a space before this word, go back to find the start of the command
			if commandStart > 0 {
				// Find the start of the previous word
				prevWordStart := commandStart - 1
				for prevWordStart > 0 && unicode.IsSpace(rune(line[prevWordStart-1])) {
					prevWordStart--
				}
				for prevWordStart > 0 && !unicode.IsSpace(rune(line[prevWordStart-1])) {
					prevWordStart--
				}
				// If the suggestion starts with the same prefix as the current command, use the full command
				if len(m.completion.suggestions) > 0 && strings.HasPrefix(m.completion.suggestions[m.completion.selected], line[prevWordStart:commandStart]) {
					start = prevWordStart
				}
			}
		}

		m.completion.startPos = start
		m.completion.endPos = end
		m.completion.prefix = m.Value()[start:m.Position()]
	}

	// Apply the suggestion
	m.applySuggestion(suggestion)

	// Update help info for the selected completion
	m.updateHelpInfo()
}

// handleBackwardCompletion handles the Shift+TAB key press for completion
func (m *Model) handleBackwardCompletion() {
	if m.CompletionProvider == nil || !m.completion.active {
		return
	}

	suggestion := m.completion.prevSuggestion()
	if suggestion == "" {
		return
	}

	m.applySuggestion(suggestion)

	// Update help info for the selected completion
	m.updateHelpInfo()
}

// applySuggestion replaces the current word with the suggestion
func (m *Model) applySuggestion(suggestion string) {
	value := m.Value()
	if m.completion.startPos > len(value) {
		return
	}

	// Use the stored end position from completion state
	// This ensures consistency with the start position
	end := m.completion.endPos
	if end > len(value) {
		end = len(value)
	}

	// Replace the current word with the suggestion
	newValue := value[:m.completion.startPos] + suggestion
	if end < len(value) {
		newValue += value[end:]
	}
	m.SetValue(newValue)

	// Move cursor to end of inserted suggestion
	m.SetCursor(m.completion.startPos + len(suggestion))

	// Update the end position to reflect the new completion
	m.completion.endPos = m.completion.startPos + len(suggestion)
}

// resetCompletion resets the completion state
func (m *Model) resetCompletion() {
	m.completion.reset()
}

// updateHelpInfo updates the help information based on current input
func (m *Model) updateHelpInfo() {
	if m.CompletionProvider == nil {
		return
	}

	var helpInfo string

	// If completion is active and a suggestion is selected, show help for the selected suggestion
	if m.completion.active && m.completion.selected >= 0 && m.completion.selected < len(m.completion.suggestions) {
		selectedSuggestion := m.completion.suggestions[m.completion.selected]
		// For help purposes, we want to get help for the selected suggestion
		// We'll use the length of the suggestion as the position to ensure we get the full command
		helpInfo = m.CompletionProvider.GetHelpInfo(selectedSuggestion, len(selectedSuggestion))
	} else {
		// Normal case: use current input
		helpInfo = m.CompletionProvider.GetHelpInfo(m.Value(), m.Position())
	}

	m.completion.setHelpInfo(helpInfo)
}

// cancelCompletion cancels the current completion and restores original text
func (m *Model) cancelCompletion() {
	if m.completion.active && m.completion.originalText != "" {
		originalText := m.completion.cancelCompletion()
		m.SetValue(originalText)
		m.SetCursor(len(originalText))
	} else {
		m.resetCompletion()
	}
}
