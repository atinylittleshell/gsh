package input

import (
	"unicode"
)

// InfoPanelContent is the interface for content that can be displayed in the
// info panel overlay. Different content types (completions, help text, errors, etc.)
// implement this interface to provide their rendering and interaction logic.
type InfoPanelContent interface {
	// Render returns the styled string to display in the info panel.
	// The width parameter indicates the available width for rendering.
	Render(width int) string

	// IsInteractive returns true if this content responds to navigation actions.
	// Interactive content can handle actions like cycling through options.
	IsInteractive() bool

	// HandleAction processes a keyboard action and returns the updated content
	// and whether the action was handled. If handled is true, the action should
	// not be processed further by the parent component.
	HandleAction(action Action) (content InfoPanelContent, handled bool)

	// IsVisible returns true if this content should be displayed.
	IsVisible() bool
}

// CompletionProvider is the interface that provides completion suggestions.
// Implementations should return completion candidates based on the current
// input line and cursor position.
type CompletionProvider interface {
	// GetCompletions returns a list of completion suggestions for the current input
	// line and cursor position. Returns an empty slice if no completions are available.
	GetCompletions(line string, pos int) []string

	// GetHelpInfo returns help information for the current input.
	// This can be used to show documentation or hints for commands.
	// Returns empty string if no help is available.
	GetHelpInfo(line string, pos int) string
}

// CompletionState tracks the state of tab completion and implements InfoPanelContent.
// It manages the list of suggestions, current selection, and the text
// boundaries where the completion should be applied.
type CompletionState struct {
	// active indicates whether completion mode is currently active
	active bool

	// suggestions is the list of completion candidates
	suggestions []string

	// selected is the index of the currently selected suggestion (-1 if none)
	selected int

	// prefix is the text being completed (the partial word before cursor)
	prefix string

	// startPos is the position in the input where the completion starts
	startPos int

	// endPos is the position in the input where the completion ends
	endPos int

	// originalText stores the input text before completion started
	// (used for cancellation)
	originalText string
}

// Ensure CompletionState implements InfoPanelContent.
var _ InfoPanelContent = (*CompletionState)(nil)

// NewCompletionState creates a new CompletionState in its initial (inactive) state.
func NewCompletionState() *CompletionState {
	return &CompletionState{
		selected: -1,
	}
}

// Reset clears all completion state and returns to inactive mode.
func (cs *CompletionState) Reset() {
	cs.active = false
	cs.suggestions = nil
	cs.selected = -1
	cs.prefix = ""
	cs.startPos = 0
	cs.endPos = 0
	cs.originalText = ""
}

// Render implements InfoPanelContent. It returns a string representation of the
// completion suggestions for display in the info panel.
func (cs *CompletionState) Render(width int) string {
	if !cs.IsVisible() {
		return ""
	}

	// Build a simple list view of suggestions
	// The actual styling will be handled by the render package
	var result string
	for i, suggestion := range cs.suggestions {
		if i > 0 {
			result += "\n"
		}
		if i == cs.selected {
			result += "> " + suggestion
		} else {
			result += "  " + suggestion
		}
	}

	return result
}

// IsInteractive implements InfoPanelContent. Returns true because completion
// supports cycling through suggestions with Tab/Shift+Tab.
func (cs *CompletionState) IsInteractive() bool {
	return true
}

// HandleAction implements InfoPanelContent. It processes completion-related
// actions like cycling through suggestions.
func (cs *CompletionState) HandleAction(action Action) (InfoPanelContent, bool) {
	if !cs.active {
		return cs, false
	}

	switch action {
	case ActionComplete:
		cs.NextSuggestion()
		return cs, true

	case ActionCompleteBackward:
		cs.PrevSuggestion()
		return cs, true

	case ActionCancel:
		cs.Reset()
		return cs, true

	default:
		return cs, false
	}
}

// IsVisible implements InfoPanelContent. Returns true if there are multiple
// suggestions to display.
func (cs *CompletionState) IsVisible() bool {
	return cs.active && len(cs.suggestions) > 1
}

// IsActive returns true if completion mode is currently active.
func (cs *CompletionState) IsActive() bool {
	return cs.active
}

// Suggestions returns the current list of completion suggestions.
func (cs *CompletionState) Suggestions() []string {
	return cs.suggestions
}

// Selected returns the index of the currently selected suggestion.
// Returns -1 if no suggestion is selected.
func (cs *CompletionState) Selected() int {
	return cs.selected
}

// Prefix returns the text prefix being completed.
func (cs *CompletionState) Prefix() string {
	return cs.prefix
}

// StartPos returns the start position of the completion in the input.
func (cs *CompletionState) StartPos() int {
	return cs.startPos
}

// EndPos returns the end position of the completion in the input.
func (cs *CompletionState) EndPos() int {
	return cs.endPos
}

// OriginalText returns the original input text before completion started.
func (cs *CompletionState) OriginalText() string {
	return cs.originalText
}

// HasMultipleCompletions returns true if there are multiple completion options.
func (cs *CompletionState) HasMultipleCompletions() bool {
	return len(cs.suggestions) > 1
}

// CurrentSuggestion returns the currently selected suggestion.
// Returns empty string if no suggestion is selected or completion is inactive.
func (cs *CompletionState) CurrentSuggestion() string {
	if !cs.active || cs.selected < 0 || cs.selected >= len(cs.suggestions) {
		return ""
	}
	return cs.suggestions[cs.selected]
}

// NextSuggestion advances to the next suggestion and returns it.
// Wraps around to the first suggestion after the last one.
// Returns empty string if completion is inactive or there are no suggestions.
func (cs *CompletionState) NextSuggestion() string {
	if !cs.active || len(cs.suggestions) == 0 {
		return ""
	}
	cs.selected = (cs.selected + 1) % len(cs.suggestions)
	return cs.suggestions[cs.selected]
}

// PrevSuggestion moves to the previous suggestion and returns it.
// Wraps around to the last suggestion before the first one.
// Returns empty string if completion is inactive or there are no suggestions.
func (cs *CompletionState) PrevSuggestion() string {
	if !cs.active || len(cs.suggestions) == 0 {
		return ""
	}
	cs.selected--
	if cs.selected < 0 {
		cs.selected = len(cs.suggestions) - 1
	}
	return cs.suggestions[cs.selected]
}

// Activate starts a new completion session with the given suggestions.
// It sets up the completion boundaries based on the provided positions.
func (cs *CompletionState) Activate(suggestions []string, prefix string, startPos, endPos int) {
	cs.active = true
	cs.suggestions = suggestions
	cs.selected = -1
	cs.prefix = prefix
	cs.startPos = startPos
	cs.endPos = endPos
}

// SetOriginalText stores the original input text for potential cancellation.
func (cs *CompletionState) SetOriginalText(text string) {
	cs.originalText = text
}

// UpdateBoundaries updates the start and end positions for the completion.
// This is used when cycling through completions to update the boundaries
// based on the current input state.
func (cs *CompletionState) UpdateBoundaries(prefix string, startPos, endPos int) {
	cs.prefix = prefix
	cs.startPos = startPos
	cs.endPos = endPos
}

// Cancel cancels the current completion and returns the original text.
// After calling this, the completion state is reset.
func (cs *CompletionState) Cancel() string {
	originalText := cs.originalText
	cs.Reset()
	return originalText
}

// HelpContent displays contextual help or documentation text.
// It implements InfoPanelContent for passive, non-interactive display.
type HelpContent struct {
	text string
}

// Ensure HelpContent implements InfoPanelContent.
var _ InfoPanelContent = (*HelpContent)(nil)

// NewHelpContent creates a new HelpContent with the given text.
func NewHelpContent(text string) *HelpContent {
	return &HelpContent{text: text}
}

// Render implements InfoPanelContent. Returns the help text.
func (h *HelpContent) Render(width int) string {
	if !h.IsVisible() {
		return ""
	}
	return h.text
}

// IsInteractive implements InfoPanelContent. Returns false because help
// content is read-only and doesn't respond to navigation.
func (h *HelpContent) IsInteractive() bool {
	return false
}

// HandleAction implements InfoPanelContent. Help content doesn't handle
// any actions, so it always returns false.
func (h *HelpContent) HandleAction(action Action) (InfoPanelContent, bool) {
	return h, false
}

// IsVisible implements InfoPanelContent. Returns true if there is help text.
func (h *HelpContent) IsVisible() bool {
	return h.text != ""
}

// Text returns the help text content.
func (h *HelpContent) Text() string {
	return h.text
}

// SetText updates the help text content.
func (h *HelpContent) SetText(text string) {
	h.text = text
}

// Clear removes the help text content.
func (h *HelpContent) Clear() {
	h.text = ""
}

// GetWordBoundary finds the start and end positions of the word at the given
// cursor position in the input text. A word is defined as a sequence of
// non-whitespace characters.
func GetWordBoundary(text string, cursorPos int) (start, end int) {
	if len(text) == 0 {
		return 0, 0
	}

	runes := []rune(text)
	runeLen := len(runes)

	// Clamp cursor position to valid range
	if cursorPos < 0 {
		cursorPos = 0
	}
	if cursorPos > runeLen {
		cursorPos = runeLen
	}

	// Find start of word (go backwards from cursor)
	start = cursorPos
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	// Find end of word (go forwards from cursor)
	end = cursorPos
	for end < runeLen && !unicode.IsSpace(runes[end]) {
		end++
	}

	return start, end
}

// CompletionResult represents the result of applying a completion suggestion.
type CompletionResult struct {
	// NewText is the resulting text after applying the completion
	NewText string
	// NewCursorPos is the new cursor position after applying the completion
	NewCursorPos int
}

// ApplySuggestion applies a completion suggestion to the given text.
// It replaces the text between startPos and endPos with the suggestion.
// Returns the new text and the new cursor position.
func ApplySuggestion(text string, suggestion string, startPos, endPos int) CompletionResult {
	if startPos > len(text) {
		startPos = len(text)
	}
	if endPos > len(text) {
		endPos = len(text)
	}
	if startPos > endPos {
		startPos = endPos
	}

	// Build the new text: before + suggestion + after
	newText := text[:startPos] + suggestion
	if endPos < len(text) {
		newText += text[endPos:]
	}

	// Cursor goes to the end of the inserted suggestion
	newCursorPos := startPos + len(suggestion)

	return CompletionResult{
		NewText:      newText,
		NewCursorPos: newCursorPos,
	}
}
