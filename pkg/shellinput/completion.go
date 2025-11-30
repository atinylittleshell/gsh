package shellinput

// CompletionCandidate represents a single completion suggestion
type CompletionCandidate struct {
	Value       string // The actual value to insert
	Display     string // What to show in the list (if different from Value)
	Description string // The description to show in the right column
	Suffix      string // Optional suffix to show as greyed-out inline suggestion (e.g., "/" for directories)
}

// CompletionProvider is the interface that provides completion suggestions
type CompletionProvider interface {
	// GetCompletions returns a list of completion suggestions for the current input
	// line and cursor position
	GetCompletions(line string, pos int) []CompletionCandidate

	// GetHelpInfo returns help information for special commands like #! and #/
	// Returns empty string if no help is available
	GetHelpInfo(line string, pos int) string
}

// completionState tracks the state of completion suggestions
type completionState struct {
	active       bool
	suggestions  []CompletionCandidate
	selected     int
	prefix       string // the part of the word being completed
	startPos     int    // where in the input the completion should be inserted
	endPos       int    // where in the input the completion should end
	showInfoBox  bool   // whether to show the completion info box
	originalText string // the original text before completion started
	helpInfo     string // help information to display for special commands
	showHelpBox  bool   // whether to show the help info box
}

func (cs *completionState) reset() {
	cs.active = false
	cs.suggestions = nil
	cs.selected = -1
	cs.prefix = ""
	cs.startPos = 0
	cs.endPos = 0
	cs.showInfoBox = false
	cs.originalText = ""
	cs.helpInfo = ""
	cs.showHelpBox = false
}

func (cs *completionState) nextSuggestion() string {
	if !cs.active || len(cs.suggestions) == 0 {
		return ""
	}
	cs.selected = (cs.selected + 1) % len(cs.suggestions)
	return cs.suggestions[cs.selected].Value
}

func (cs *completionState) prevSuggestion() string {
	if !cs.active || len(cs.suggestions) == 0 {
		return ""
	}
	cs.selected--
	if cs.selected < 0 {
		cs.selected = len(cs.suggestions) - 1
	}
	return cs.suggestions[cs.selected].Value
}

func (cs *completionState) currentSuggestion() string {
	if !cs.active || cs.selected < 0 || cs.selected >= len(cs.suggestions) {
		return ""
	}
	return cs.suggestions[cs.selected].Value
}

// hasMultipleCompletions returns true if there are multiple completion options
func (cs *completionState) hasMultipleCompletions() bool {
	return len(cs.suggestions) > 1
}

// shouldShowInfoBox returns true if the info box should be displayed
func (cs *completionState) shouldShowInfoBox() bool {
	return cs.active && cs.showInfoBox && cs.hasMultipleCompletions()
}

// shouldShowHelpBox returns true if the help box should be displayed
func (cs *completionState) shouldShowHelpBox() bool {
	return cs.showHelpBox && cs.helpInfo != ""
}

// activateInfoBox enables the info box display and stores original text
func (cs *completionState) activateInfoBox(originalText string) {
	cs.showInfoBox = true
	cs.originalText = originalText
}

// cancelCompletion restores the original text and resets state
func (cs *completionState) cancelCompletion() string {
	originalText := cs.originalText
	cs.reset()
	return originalText
}

// setHelpInfo sets the help information to display
func (cs *completionState) setHelpInfo(helpInfo string) {
	cs.helpInfo = helpInfo
	cs.showHelpBox = helpInfo != ""
}
