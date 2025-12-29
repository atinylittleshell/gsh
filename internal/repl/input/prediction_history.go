package input

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// onTextChanged is called after any text modification.
// It triggers prediction updates and other text-change handling.
func (m Model) onTextChanged() (tea.Model, tea.Cmd) {
	text := m.buffer.Text()

	// Check if prediction still applies
	if m.currentPrediction != "" && !strings.HasPrefix(m.currentPrediction, text) {
		m.currentPrediction = ""
	}

	// Request new prediction
	cmd := m.requestPrediction(text)

	return m, cmd
}

// requestPrediction initiates an async prediction request.
func (m Model) requestPrediction(input string) tea.Cmd {
	if m.prediction == nil {
		return nil
	}

	resultChan := m.prediction.OnInputChanged(input)
	if resultChan == nil {
		return nil
	}

	return func() tea.Msg {
		result := <-resultChan
		return predictionResultMsg(result)
	}
}

// handlePredictionResult processes a prediction result.
func (m Model) handlePredictionResult(msg predictionResultMsg) (tea.Model, tea.Cmd) {
	result := PredictionResult(msg)

	if result.Error != nil {
		m.logger.Debug("prediction error", zap.Error(result.Error))
		return m, nil
	}

	// Update prediction if state ID matches
	if m.prediction != nil && m.prediction.SetPrediction(result.StateID, result.Prediction) {
		m.currentPrediction = result.Prediction
	}

	return m, nil
}

// handleHistorySearchStart starts history search mode.
func (m Model) handleHistorySearchStart() (tea.Model, tea.Cmd) {
	// Start history search, saving current input
	m.historySearch.Start(m.buffer.Text(), m.buffer.Pos())
	// Clear predictions while in search mode
	m.currentPrediction = ""
	return m, nil
}

// handleHistorySearchKey handles key input while in history search mode.
func (m Model) handleHistorySearchKey(msg tea.KeyMsg, action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionSubmit:
		// Accept current match and exit search mode
		result := m.historySearch.Accept()
		m.buffer.SetText(result)
		return m, nil

	case ActionCancel, ActionInterrupt:
		// Cancel search, restore original input
		originalInput, originalPos := m.historySearch.Cancel()
		m.buffer.SetText(originalInput)
		m.buffer.SetPos(originalPos)
		return m, nil

	case ActionHistorySearchBackward:
		// Ctrl+R again: go to next (older) match
		m.historySearch.NextMatch()
		if match := m.historySearch.CurrentMatch(); match != "" {
			m.buffer.SetText(match)
		}
		return m, nil

	case ActionCursorUp:
		// Up arrow: go to next (older) match
		m.historySearch.NextMatch()
		if match := m.historySearch.CurrentMatch(); match != "" {
			m.buffer.SetText(match)
		}
		return m, nil

	case ActionCursorDown:
		// Down arrow: go to previous (newer) match
		m.historySearch.PrevMatch()
		if match := m.historySearch.CurrentMatch(); match != "" {
			m.buffer.SetText(match)
		}
		return m, nil

	case ActionDeleteCharacterBackward:
		// Backspace: delete character from search query
		if m.historySearch.DeleteChar() {
			m.updateHistorySearchMatches()
			if match := m.historySearch.CurrentMatch(); match != "" {
				m.buffer.SetText(match)
			} else if m.historySearch.Query() == "" {
				// If query is empty, show original input
				m.buffer.SetText(m.historySearch.OriginalInput())
			}
		}
		return m, nil

	case ActionCharacterForward, ActionCharacterBackward, ActionLineStart, ActionLineEnd:
		// Accept current match and exit search mode, keeping cursor movement
		result := m.historySearch.Accept()
		m.buffer.SetText(result)
		// Now handle the navigation action
		return m.handleKeyMsg(msg)

	default:
		// Regular character input: add to search query
		if len(msg.Runes) > 0 {
			for _, r := range msg.Runes {
				// Skip control characters
				if r >= 32 {
					m.historySearch.AddChar(r)
				}
			}
			m.updateHistorySearchMatches()
			if match := m.historySearch.CurrentMatch(); match != "" {
				m.buffer.SetText(match)
			} else {
				// No matches, keep showing original or empty
				m.buffer.SetText(m.historySearch.OriginalInput())
			}
			return m, nil
		}
	}

	return m, nil
}

// updateHistorySearchMatches updates the matches based on current query.
func (m *Model) updateHistorySearchMatches() {
	query := m.historySearch.Query()
	if query == "" {
		m.historySearch.SetMatches(nil)
		return
	}

	var matches []string

	// Use the search function if provided
	if m.historySearchFunc != nil {
		matches = m.historySearchFunc(query)
	} else {
		// Fall back to searching historyValues
		matches = m.searchHistoryValues(query)
	}

	m.historySearch.SetMatches(matches)
}

// searchHistoryValues searches the in-memory history values for matches.
func (m *Model) searchHistoryValues(query string) []string {
	var matches []string
	queryLower := strings.ToLower(query)
	for _, cmd := range m.historyValues {
		if strings.Contains(strings.ToLower(cmd), queryLower) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// renderFinalView renders the view after input is complete.
func (m Model) renderFinalView() string {
	// For interrupt and submit, we don't render anything
	// The REPL will handle printing the final line so it persists in terminal history
	if m.result.Type == ResultInterrupt || m.result.Type == ResultSubmit {
		return ""
	}

	// Render final input line without cursor/prediction
	return m.renderer.RenderInputLine(m.prompt, m.buffer, "", false)
}

// predictionResultMsg wraps a PredictionResult for the tea.Msg interface.
type predictionResultMsg PredictionResult

// pasteMsg is sent when paste content is available.
type pasteMsg string

// Paste returns a command that reads from the clipboard.
func Paste() tea.Msg {
	str, err := clipboard.ReadAll()
	if err != nil {
		return nil
	}
	return pasteMsg(str)
}
