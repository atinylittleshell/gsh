// Package input provides input handling for the gsh REPL.
package input

// HistorySearchState manages the state for reverse history search (Ctrl+R).
type HistorySearchState struct {
	// active indicates whether history search mode is active
	active bool

	// query is the current search query
	query string

	// matches contains the matching history entries (most recent first)
	matches []string

	// matchIndex is the current position in matches (0 = most recent match)
	matchIndex int

	// originalInput is the input text before search started
	originalInput string

	// originalCursorPos is the cursor position before search started
	originalCursorPos int
}

// NewHistorySearchState creates a new history search state.
func NewHistorySearchState() *HistorySearchState {
	return &HistorySearchState{}
}

// IsActive returns true if history search mode is active.
func (s *HistorySearchState) IsActive() bool {
	return s.active
}

// Query returns the current search query.
func (s *HistorySearchState) Query() string {
	return s.query
}

// CurrentMatch returns the currently selected match, or empty string if no matches.
func (s *HistorySearchState) CurrentMatch() string {
	if len(s.matches) == 0 || s.matchIndex < 0 || s.matchIndex >= len(s.matches) {
		return ""
	}
	return s.matches[s.matchIndex]
}

// MatchIndex returns the current match index.
func (s *HistorySearchState) MatchIndex() int {
	return s.matchIndex
}

// MatchCount returns the total number of matches.
func (s *HistorySearchState) MatchCount() int {
	return len(s.matches)
}

// OriginalInput returns the input text from before the search started.
func (s *HistorySearchState) OriginalInput() string {
	return s.originalInput
}

// Start begins a history search session, saving the current input state.
func (s *HistorySearchState) Start(currentInput string, cursorPos int) {
	s.active = true
	s.query = ""
	s.matches = nil
	s.matchIndex = 0
	s.originalInput = currentInput
	s.originalCursorPos = cursorPos
}

// SetQuery updates the search query.
func (s *HistorySearchState) SetQuery(query string) {
	s.query = query
	s.matchIndex = 0 // Reset to first match when query changes
}

// SetMatches updates the list of matching history entries.
func (s *HistorySearchState) SetMatches(matches []string) {
	s.matches = matches
	// Clamp matchIndex to valid range
	if s.matchIndex >= len(s.matches) {
		s.matchIndex = len(s.matches) - 1
	}
	if s.matchIndex < 0 {
		s.matchIndex = 0
	}
}

// NextMatch moves to the next (older) match.
// Returns true if the index changed.
func (s *HistorySearchState) NextMatch() bool {
	if len(s.matches) == 0 {
		return false
	}
	if s.matchIndex < len(s.matches)-1 {
		s.matchIndex++
		return true
	}
	return false
}

// PrevMatch moves to the previous (newer) match.
// Returns true if the index changed.
func (s *HistorySearchState) PrevMatch() bool {
	if len(s.matches) == 0 {
		return false
	}
	if s.matchIndex > 0 {
		s.matchIndex--
		return true
	}
	return false
}

// Cancel exits history search and returns the original input to restore.
func (s *HistorySearchState) Cancel() (originalInput string, originalCursorPos int) {
	originalInput = s.originalInput
	originalCursorPos = s.originalCursorPos
	s.Reset()
	return
}

// Accept exits history search, keeping the current match selected.
// Returns the selected command (or original input if no match).
func (s *HistorySearchState) Accept() string {
	result := s.CurrentMatch()
	if result == "" {
		result = s.originalInput
	}
	s.Reset()
	return result
}

// Reset clears the history search state.
func (s *HistorySearchState) Reset() {
	s.active = false
	s.query = ""
	s.matches = nil
	s.matchIndex = 0
	s.originalInput = ""
	s.originalCursorPos = 0
}

// AddChar adds a character to the search query.
func (s *HistorySearchState) AddChar(r rune) {
	s.query += string(r)
	s.matchIndex = 0 // Reset to first match when query changes
}

// DeleteChar removes the last character from the search query.
// Returns true if a character was deleted.
func (s *HistorySearchState) DeleteChar() bool {
	if len(s.query) > 0 {
		runes := []rune(s.query)
		s.query = string(runes[:len(runes)-1])
		s.matchIndex = 0 // Reset to first match when query changes
		return true
	}
	return false
}
