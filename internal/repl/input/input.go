// Package input provides a unified line input component for the gsh REPL.
// It merges functionality from pkg/gline and pkg/shellinput into a single
// cohesive Bubble Tea component that handles text input, cursor management,
// key bindings, tab completion, and LLM prediction integration.
package input

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// ResultType indicates the type of result from the input component.
type ResultType int

const (
	// ResultNone indicates no result yet (still editing).
	ResultNone ResultType = iota
	// ResultSubmit indicates the user submitted the input (Enter).
	ResultSubmit
	// ResultInterrupt indicates the user interrupted (Ctrl+C).
	ResultInterrupt
	// ResultEOF indicates end of input (Ctrl+D on empty line).
	ResultEOF
)

// Result contains the outcome of an input session.
type Result struct {
	// Type indicates what action caused the input to complete.
	Type ResultType
	// Value is the input text (empty for interrupt/EOF).
	Value string
}

// Model is the Bubble Tea model for the unified input component.
// It coordinates the buffer, keymap, completion, prediction, and rendering.
type Model struct {
	// Core state
	buffer  *Buffer
	keymap  *KeyMap
	focused bool

	// Prompt
	prompt string

	// History navigation
	historyValues       []string
	historyIndex        int // 0 = current input, 1+ = history entries
	savedCurrentInput   string
	hasNavigatedHistory bool

	// Completion
	completion         *CompletionState
	completionProvider CompletionProvider

	// Prediction
	prediction        *PredictionState
	currentPrediction string

	// Rendering
	renderer  *Renderer
	width     int
	minHeight int

	// Info panel content (help text, etc.)
	infoContent InfoPanelContent

	// Result state
	result Result

	// Logger
	logger *zap.Logger
}

// Config holds configuration for creating a new Model.
type Config struct {
	// Prompt is the prompt string to display.
	Prompt string

	// HistoryValues is the list of previous commands for history navigation.
	// Index 0 is the most recent.
	HistoryValues []string

	// CompletionProvider provides tab completion suggestions.
	CompletionProvider CompletionProvider

	// PredictionState manages command predictions.
	PredictionState *PredictionState

	// KeyMap provides key bindings. If nil, DefaultKeyMap is used.
	KeyMap *KeyMap

	// RenderConfig provides styling. If nil, DefaultRenderConfig is used.
	RenderConfig *RenderConfig

	// MinHeight is the minimum number of lines to render.
	MinHeight int

	// Width is the initial terminal width.
	Width int

	// Logger for debug output. If nil, a no-op logger is used.
	Logger *zap.Logger
}

// New creates a new input Model with the given configuration.
func New(cfg Config) Model {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	keymap := cfg.KeyMap
	if keymap == nil {
		keymap = DefaultKeyMap()
	}

	renderConfig := cfg.RenderConfig
	if renderConfig == nil {
		defaultConfig := DefaultRenderConfig()
		renderConfig = &defaultConfig
	}

	width := cfg.Width
	if width <= 0 {
		width = 80
	}

	renderer := NewRenderer(*renderConfig)
	renderer.SetWidth(width)

	return Model{
		buffer:             NewBuffer(),
		keymap:             keymap,
		focused:            true,
		prompt:             cfg.Prompt,
		historyValues:      cfg.HistoryValues,
		historyIndex:       0,
		completion:         NewCompletionState(),
		completionProvider: cfg.CompletionProvider,
		prediction:         cfg.PredictionState,
		renderer:           renderer,
		width:              width,
		minHeight:          cfg.MinHeight,
		result:             Result{Type: ResultNone},
		logger:             logger,
	}
}

// Init implements tea.Model. It triggers an initial prediction request.
func (m Model) Init() tea.Cmd {
	if m.prediction != nil {
		// Trigger initial prediction for empty input (null-state prediction)
		return m.requestPrediction("")
	}
	return nil
}

// Update implements tea.Model. It handles all input events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.renderer.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case predictionResultMsg:
		return m.handlePredictionResult(msg)

	case pasteMsg:
		return m.handlePaste(string(msg))
	}

	return m, nil
}

// View implements tea.Model. It renders the input component.
func (m Model) View() string {
	if m.result.Type != ResultNone {
		// Input is complete, render final state
		return m.renderFinalView()
	}

	return m.renderer.RenderFullView(
		m.prompt,
		m.buffer,
		m.currentPrediction,
		m.focused,
		m.completion,
		m.infoContent,
		m.minHeight,
	)
}

// Result returns the current result. Check Type != ResultNone to see if complete.
func (m Model) Result() Result {
	return m.result
}

// Value returns the current input text.
func (m Model) Value() string {
	return m.buffer.Text()
}

// SetValue sets the input text and moves cursor to end.
func (m *Model) SetValue(text string) {
	m.buffer.SetText(text)
	m.historyIndex = 0
	m.hasNavigatedHistory = false
}

// Focus sets the focus state on the model.
func (m *Model) Focus() {
	m.focused = true
}

// Blur removes focus from the model.
func (m *Model) Blur() {
	m.focused = false
}

// Focused returns whether the model is focused.
func (m Model) Focused() bool {
	return m.focused
}

// SetPrompt updates the prompt string.
func (m *Model) SetPrompt(prompt string) {
	m.prompt = prompt
}

// Prompt returns the current prompt string.
func (m Model) Prompt() string {
	return m.prompt
}

// SetHistoryValues updates the history values for navigation.
func (m *Model) SetHistoryValues(values []string) {
	m.historyValues = values
	m.historyIndex = 0
	m.hasNavigatedHistory = false
}

// Reset clears the input state for a new input session.
func (m *Model) Reset() {
	m.buffer.Clear()
	m.completion.Reset()
	if m.prediction != nil {
		m.prediction.Reset()
	}
	m.currentPrediction = ""
	m.historyIndex = 0
	m.savedCurrentInput = ""
	m.hasNavigatedHistory = false
	m.result = Result{Type: ResultNone}
	m.infoContent = nil
}

// Buffer returns the underlying buffer (for testing).
func (m Model) Buffer() *Buffer {
	return m.buffer
}

// Completion returns the completion state (for testing).
func (m Model) Completion() *CompletionState {
	return m.completion
}

// CurrentPrediction returns the current prediction text.
func (m Model) CurrentPrediction() string {
	return m.currentPrediction
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Look up the action for this key
	action := m.keymap.Lookup(msg)

	// When completion is active, handle navigation keys specially
	if m.completion.IsActive() {
		switch action {
		case ActionComplete, ActionCursorDown:
			// Tab or Down arrow: cycle forward through completions
			return m.handleComplete()
		case ActionCompleteBackward, ActionCursorUp:
			// Shift+Tab or Up arrow: cycle backward through completions
			return m.handleCompleteBackward()
		case ActionCancel:
			// Escape: cancel completion
			return m.handleCompletionAction(action)
		case ActionSubmit:
			// Enter: accept current completion and submit
			m.completion.Reset()
			return m.handleSubmit()
		}
		// For other keys, reset completion and continue with normal handling
		m.completion.Reset()
	}

	// Handle special actions first
	switch action {
	case ActionSubmit:
		return m.handleSubmit()

	case ActionInterrupt:
		return m.handleInterrupt()

	case ActionDeleteCharacterForward:
		// Ctrl+D on empty input triggers EOF
		if m.buffer.Len() == 0 {
			return m.handleEOF()
		}
		return m.handleDeleteCharacterForward()

	case ActionEOF:
		return m.handleEOF()

	case ActionClearScreen:
		return m, tea.ClearScreen

	case ActionPaste:
		return m, Paste

	case ActionComplete:
		return m.handleComplete()

	case ActionCompleteBackward:
		return m.handleCompleteBackward()

	case ActionCancel:
		return m.handleCancel()

	case ActionAcceptPrediction:
		return m.handleAcceptPrediction()

	// Navigation actions
	case ActionCharacterForward:
		return m.handleCharacterForward()

	case ActionCharacterBackward:
		m.buffer.SetPos(m.buffer.Pos() - 1)
		return m, nil

	case ActionWordForward:
		m.buffer.WordForward()
		return m, nil

	case ActionWordBackward:
		m.buffer.WordBackward()
		return m, nil

	case ActionLineStart:
		m.buffer.CursorStart()
		return m, nil

	case ActionLineEnd:
		m.buffer.CursorEnd()
		return m, nil

	// Deletion actions
	case ActionDeleteCharacterBackward:
		return m.handleDeleteCharacterBackward()

	case ActionDeleteWordBackward:
		return m.handleDeleteWordBackward()

	case ActionDeleteWordForward:
		return m.handleDeleteWordForward()

	case ActionDeleteBeforeCursor:
		return m.handleDeleteBeforeCursor()

	case ActionDeleteAfterCursor:
		return m.handleDeleteAfterCursor()

	// Vertical navigation (history when completion not active)
	case ActionCursorUp:
		return m.handleHistoryPrevious()

	case ActionCursorDown:
		return m.handleHistoryNext()

	default:
		// Insert regular characters
		if len(msg.Runes) > 0 {
			return m.handleInsertRunes(msg.Runes)
		}
	}

	return m, nil
}

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

// renderFinalView renders the view after input is complete.
func (m Model) renderFinalView() string {
	// For interrupt, we don't render anything (shell will handle it)
	if m.result.Type == ResultInterrupt {
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
