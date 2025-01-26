/*
This file is forked from the textinput component from
github.com/charmbracelet/bubbles

# MIT License

# Copyright (c) 2020-2023 Charmbracelet, Inc

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package shellinput

import (
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/runeutil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/rivo/uniseg"
)

// Internal messages for clipboard operations.
type (
	pasteMsg    string
	pasteErrMsg struct{ error }
)

// EchoMode sets the input behavior of the text input field.
type EchoMode int

const (
	// EchoNormal displays text as is. This is the default behavior.
	EchoNormal EchoMode = iota

	// EchoPassword displays the EchoCharacter mask instead of actual
	// characters. This is commonly used for password fields.
	EchoPassword

	// EchoNone displays nothing as characters are entered. This is commonly
	// seen for password fields on the command line.
	EchoNone
)

// ValidateFunc is a function that returns an error if the input is invalid.
type ValidateFunc func(string) error

// KeyMap is the key bindings for different actions within the textinput.
type KeyMap struct {
	CharacterForward        key.Binding
	CharacterBackward       key.Binding
	WordForward             key.Binding
	WordBackward            key.Binding
	DeleteWordBackward      key.Binding
	DeleteWordForward       key.Binding
	DeleteAfterCursor       key.Binding
	DeleteBeforeCursor      key.Binding
	DeleteCharacterBackward key.Binding
	DeleteCharacterForward  key.Binding
	LineStart               key.Binding
	LineEnd                 key.Binding
	Paste                   key.Binding
	NextValue               key.Binding
	PrevValue               key.Binding
	Complete                key.Binding
	PrevSuggestion          key.Binding
}

// DefaultKeyMap is the default set of key bindings for navigating and acting
// upon the textinput.
var DefaultKeyMap = KeyMap{
	CharacterForward:        key.NewBinding(key.WithKeys("right", "ctrl+f")),
	CharacterBackward:       key.NewBinding(key.WithKeys("left", "ctrl+b")),
	WordForward:             key.NewBinding(key.WithKeys("alt+right", "ctrl+right", "alt+f")),
	WordBackward:            key.NewBinding(key.WithKeys("alt+left", "ctrl+left", "alt+b")),
	DeleteWordBackward:      key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w")),
	DeleteWordForward:       key.NewBinding(key.WithKeys("alt+delete", "alt+d")),
	DeleteAfterCursor:       key.NewBinding(key.WithKeys("ctrl+k")),
	DeleteBeforeCursor:      key.NewBinding(key.WithKeys("ctrl+u")),
	DeleteCharacterBackward: key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
	Complete:                key.NewBinding(key.WithKeys("tab")),
	PrevSuggestion:          key.NewBinding(key.WithKeys("shift+tab")),
	DeleteCharacterForward:  key.NewBinding(key.WithKeys("delete", "ctrl+d")),
	LineStart:               key.NewBinding(key.WithKeys("home", "ctrl+a")),
	LineEnd:                 key.NewBinding(key.WithKeys("end", "ctrl+e")),
	Paste:                   key.NewBinding(key.WithKeys("ctrl+v")),
	NextValue:               key.NewBinding(key.WithKeys("down", "ctrl+n")),
	PrevValue:               key.NewBinding(key.WithKeys("up", "ctrl+p")),
}

// Model is the Bubble Tea model for this text input element.
type Model struct {
	Err error

	// General settings.
	Prompt        string
	EchoMode      EchoMode
	EchoCharacter rune
	Cursor        cursor.Model

	// Completion settings
	CompletionProvider CompletionProvider
	completion         completionState

	// Deprecated: use [cursor.BlinkSpeed] instead.
	BlinkSpeed time.Duration

	// Styles. These will be applied as inline styles.
	//
	// For an introduction to styling with Lip Gloss see:
	// https://github.com/charmbracelet/lipgloss
	PromptStyle     lipgloss.Style
	TextStyle       lipgloss.Style
	CompletionStyle lipgloss.Style

	// Deprecated: use Cursor.Style instead.
	CursorStyle lipgloss.Style

	// CharLimit is the maximum amount of characters this input element will
	// accept. If 0 or less, there's no limit.
	CharLimit int

	// Width marks the horizontal boundary for this component to render within.
	// Content that exceeds this width will be wrapped.
	// If 0 or less this setting is ignored.
	Width int

	// KeyMap encodes the keybindings recognized by the widget.
	KeyMap KeyMap

	// focus indicates whether user input focus should be on this input
	// component. When false, ignore keyboard input and hide the cursor.
	focus bool

	// Cursor position.
	pos int

	// Validate is a function that checks whether or not the text within the
	// input is valid. If it is not valid, the `Err` field will be set to the
	// error returned by the function. If the function is not defined, all
	// input is considered valid.
	Validate ValidateFunc

	// rune sanitizer for input.
	rsan runeutil.Sanitizer

	// Should the input suggest to complete
	ShowSuggestions bool

	// suggestions is a list of suggestions that may be used to complete the
	// input.
	suggestions            [][]rune
	matchedSuggestions     [][]rune
	currentSuggestionIndex int

	// values[0] is the current value. other indices represent history values
	// that can be navigated with the up and down arrow keys.
	values             [][]rune
	selectedValueIndex int
}

// New creates a new model with default settings.
func New() Model {
	return Model{
		Prompt:          "> ",
		EchoCharacter:   '*',
		CharLimit:       0,
		ShowSuggestions: false,
		CompletionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Cursor:          cursor.New(),
		KeyMap:          DefaultKeyMap,

		suggestions: [][]rune{},
		focus:       false,
		pos:         0,

		values:             [][]rune{{}},
		selectedValueIndex: 0,
	}
}

// SetValue sets the value of the text input.
func (m *Model) SetValue(s string) {
	// Clean up any special characters in the input provided by the
	// caller. This avoids bugs due to e.g. tab characters and whatnot.
	runes := m.san().Sanitize([]rune(s))
	err := m.validate(runes)
	m.setValueInternal(runes, err)
}

func (m *Model) setValueInternal(runes []rune, err error) {
	m.Err = err

	empty := len(m.values[m.selectedValueIndex]) == 0

	if m.CharLimit > 0 && len(runes) > m.CharLimit {
		m.values[0] = runes[:m.CharLimit]
	} else {
		m.values[0] = runes
	}
	m.selectedValueIndex = 0
	if (m.pos == 0 && empty) || m.pos > len(m.values[0]) {
		m.SetCursor(len(m.values[0]))
	}
}

// Value returns the value of the text input.
func (m Model) Value() string {
	return string(m.values[m.selectedValueIndex])
}

// Position returns the cursor position.
func (m Model) Position() int {
	return m.pos
}

// SetCursor moves the cursor to the given position. If the position is
// out of bounds the cursor will be moved to the start or end accordingly.
func (m *Model) SetCursor(pos int) {
	m.pos = clamp(pos, 0, len(m.values[m.selectedValueIndex]))
}

// CursorStart moves the cursor to the start of the input field.
func (m *Model) CursorStart() {
	m.SetCursor(0)
}

// CursorEnd moves the cursor to the end of the input field.
func (m *Model) CursorEnd() {
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

// Focused returns the focus state on the model.
func (m Model) Focused() bool {
	return m.focus
}

// Focus sets the focus state on the model. When the model is in focus it can
// receive keyboard input and the cursor will be shown.
func (m *Model) Focus() tea.Cmd {
	m.focus = true
	return m.Cursor.Focus()
}

// Blur removes the focus state on the model.  When the model is blurred it can
// not receive keyboard input and the cursor will be hidden.
func (m *Model) Blur() {
	m.focus = false
	m.Cursor.Blur()
}

// Reset sets the input to its default state with no input.
func (m *Model) Reset() {
	m.values = [][]rune{{}}
	m.selectedValueIndex = 0
	m.SetCursor(0)
}

// SetSuggestions sets the suggestions for the input.
func (m *Model) SetSuggestions(suggestions []string) {
	m.suggestions = make([][]rune, len(suggestions))
	for i, s := range suggestions {
		m.suggestions[i] = []rune(s)
	}

	m.updateSuggestions()
}

// SetHistoryValues sets the suggestions for the input.
func (m *Model) SetHistoryValues(historyValues []string) {
	m.values = append([][]rune{m.values[0]}, make([][]rune, len(historyValues))...)

	for i, s := range historyValues {
		m.values[i+1] = m.san().Sanitize([]rune(s))
	}

	// reset value index if the selected index is out of bounds
	if m.selectedValueIndex >= len(m.values) {
		m.selectedValueIndex = 0
	}
}

// rsan initializes or retrieves the rune sanitizer.
func (m *Model) san() runeutil.Sanitizer {
	if m.rsan == nil {
		// Textinput has all its input on a single line so collapse
		// newlines/tabs to single spaces.
		m.rsan = runeutil.NewSanitizer(
			runeutil.ReplaceTabs(" "), runeutil.ReplaceNewlines(" "))
	}
	return m.rsan
}

func (m *Model) insertRunesFromUserInput(v []rune) {
	// Clean up any special characters in the input provided by the
	// clipboard. This avoids bugs due to e.g. tab characters and
	// whatnot.
	paste := m.san().Sanitize(v)

	var availSpace int
	if m.CharLimit > 0 {
		availSpace = m.CharLimit - len(m.values[m.selectedValueIndex])

		// If the char limit's been reached, cancel.
		if availSpace <= 0 {
			return
		}

		// If there's not enough space to paste the whole thing cut the pasted
		// runes down so they'll fit.
		if availSpace < len(paste) {
			paste = paste[:availSpace]
		}
	}

	result := make([]rune, len(m.values[m.selectedValueIndex])+len(paste))

	copy(result, m.values[m.selectedValueIndex][:m.pos])
	copy(result[m.pos:], paste)
	copy(result[m.pos+len(paste):], m.values[m.selectedValueIndex][m.pos:])
	m.pos += len(paste)

	inputErr := m.validate(result)
	m.setValueInternal(result, inputErr)
}

// deleteBeforeCursor deletes all text before the cursor.
func (m *Model) deleteBeforeCursor() {
	newValue := cloneRunes(m.values[m.selectedValueIndex][m.pos:])
	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
	m.SetCursor(0)
}

// deleteAfterCursor deletes all text after the cursor. If input is masked
// delete everything after the cursor so as not to reveal word breaks in the
// masked input.
func (m *Model) deleteAfterCursor() {
	newValue := cloneRunes(m.values[m.selectedValueIndex][:m.pos])
	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
	m.SetCursor(len(m.values[0]))
}

// deleteWordBackward deletes the word left to the cursor.
func (m *Model) deleteWordBackward() {
	if m.pos == 0 || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.deleteBeforeCursor()
		return
	}

	// Linter note: it's critical that we acquire the initial cursor position
	// here prior to altering it via SetCursor() below. As such, moving this
	// call into the corresponding if clause does not apply here.
	oldPos := m.pos //nolint:ifshort

	m.SetCursor(m.pos - 1)
	for unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
		if m.pos <= 0 {
			break
		}
		// ignore series of whitespace before cursor
		m.SetCursor(m.pos - 1)
	}

	for m.pos > 0 {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
			m.SetCursor(m.pos - 1)
		} else {
			if m.pos > 0 {
				// keep the previous space
				m.SetCursor(m.pos + 1)
			}
			break
		}
	}

	var newValue []rune
	if oldPos > len(m.values[m.selectedValueIndex]) {
		newValue = cloneRunes(m.values[m.selectedValueIndex][:m.pos])
	} else {
		newValue = cloneConcatRunes(m.values[m.selectedValueIndex][:m.pos], m.values[m.selectedValueIndex][oldPos:])
	}
	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
}

// deleteWordForward deletes the word right to the cursor. If input is masked
// delete everything after the cursor so as not to reveal word breaks in the
// masked input.
func (m *Model) deleteWordForward() {
	if m.pos >= len(m.values[m.selectedValueIndex]) || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.deleteAfterCursor()
		return
	}

	oldPos := m.pos
	m.SetCursor(m.pos + 1)
	for unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
		// ignore series of whitespace after cursor
		m.SetCursor(m.pos + 1)

		if m.pos >= len(m.values[m.selectedValueIndex]) {
			break
		}
	}

	for m.pos < len(m.values[m.selectedValueIndex]) {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
			m.SetCursor(m.pos + 1)
		} else {
			break
		}
	}

	var newValue []rune
	if m.pos > len(m.values[m.selectedValueIndex]) {
		newValue = cloneRunes(m.values[m.selectedValueIndex][:oldPos])
	} else {
		newValue = cloneConcatRunes(m.values[m.selectedValueIndex][:oldPos], m.values[m.selectedValueIndex][m.pos:])
	}
	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
	m.SetCursor(oldPos)
}

// wordBackward moves the cursor one word to the left. If input is masked, move
// input to the start so as not to reveal word breaks in the masked input.
func (m *Model) wordBackward() {
	if m.pos == 0 || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.CursorStart()
		return
	}

	i := m.pos - 1
	for i >= 0 {
		if unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos - 1)
			i--
		} else {
			break
		}
	}

	for i >= 0 {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos - 1)
			i--
		} else {
			break
		}
	}
}

// wordForward moves the cursor one word to the right. If the input is masked,
// move input to the end so as not to reveal word breaks in the masked input.
func (m *Model) wordForward() {
	if m.pos >= len(m.values[m.selectedValueIndex]) || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.CursorEnd()
		return
	}

	i := m.pos
	for i < len(m.values[m.selectedValueIndex]) {
		if unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos + 1)
			i++
		} else {
			break
		}
	}

	for i < len(m.values[m.selectedValueIndex]) {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos + 1)
			i++
		} else {
			break
		}
	}
}

func (m Model) echoTransform(v string) string {
	switch m.EchoMode {
	case EchoPassword:
		return strings.Repeat(string(m.EchoCharacter), uniseg.StringWidth(v))
	case EchoNone:
		return ""
	case EchoNormal:
		return v
	default:
		return v
	}
}

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	// Let's remember where the position of the cursor currently is so that if
	// the cursor position changes, we can reset the blink.
	oldPos := m.pos

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Reset completion state for any key except TAB and Shift+TAB
		if !key.Matches(msg, m.KeyMap.Complete) && !key.Matches(msg, m.KeyMap.PrevSuggestion) {
			m.resetCompletion()
		}

		switch {
		case key.Matches(msg, m.KeyMap.Complete):
			m.handleCompletion()
			return m, nil
		case key.Matches(msg, m.KeyMap.PrevSuggestion) && m.completion.active:
			m.handleBackwardCompletion()
			return m, nil
		case key.Matches(msg, m.KeyMap.DeleteWordBackward):
			m.deleteWordBackward()
		case key.Matches(msg, m.KeyMap.DeleteCharacterBackward):
			m.Err = nil
			if len(m.values[m.selectedValueIndex]) > 0 {
				newValue := cloneConcatRunes(m.values[m.selectedValueIndex][:max(0, m.pos-1)], m.values[m.selectedValueIndex][m.pos:])
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
				if m.pos > 0 {
					m.SetCursor(m.pos - 1)
				}
			}
		case key.Matches(msg, m.KeyMap.WordBackward):
			m.wordBackward()
		case key.Matches(msg, m.KeyMap.CharacterBackward):
			if m.pos > 0 {
				m.SetCursor(m.pos - 1)
			}
		case key.Matches(msg, m.KeyMap.WordForward):
			m.wordForward()
		case key.Matches(msg, m.KeyMap.CharacterForward):
			if m.pos < len(m.values[m.selectedValueIndex]) {
				m.SetCursor(m.pos + 1)
			} else if m.canAcceptSuggestion() {
				newValue := cloneConcatRunes(
					m.values[m.selectedValueIndex],
					m.matchedSuggestions[m.currentSuggestionIndex][len(m.values[m.selectedValueIndex]):],
				)
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
				m.CursorEnd()
			}
		case key.Matches(msg, m.KeyMap.LineStart):
			m.CursorStart()
		case key.Matches(msg, m.KeyMap.DeleteCharacterForward):
			if len(m.values[m.selectedValueIndex]) > 0 && m.pos < len(m.values[m.selectedValueIndex]) {
				newValue := cloneConcatRunes(m.values[m.selectedValueIndex][:m.pos], m.values[m.selectedValueIndex][m.pos+1:])
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
			}
		case key.Matches(msg, m.KeyMap.LineEnd):
			m.CursorEnd()
		case key.Matches(msg, m.KeyMap.DeleteAfterCursor):
			m.deleteAfterCursor()
		case key.Matches(msg, m.KeyMap.DeleteBeforeCursor):
			m.deleteBeforeCursor()
		case key.Matches(msg, m.KeyMap.Paste):
			return m, Paste
		case key.Matches(msg, m.KeyMap.DeleteWordForward):
			m.deleteWordForward()
		case key.Matches(msg, m.KeyMap.NextValue):
			m.nextValue()
		case key.Matches(msg, m.KeyMap.PrevValue):
			m.previousValue()
		default:
			// Input one or more regular characters.
			m.insertRunesFromUserInput(msg.Runes)
		}

		// Check again if can be completed
		// because value might be something that does not match the completion prefix
		m.updateSuggestions()

	case pasteMsg:
		m.insertRunesFromUserInput([]rune(msg))

	case pasteErrMsg:
		m.Err = msg
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.Cursor, cmd = m.Cursor.Update(msg)
	cmds = append(cmds, cmd)

	if oldPos != m.pos && m.Cursor.Mode() == cursor.CursorBlink {
		m.Cursor.Blink = false
		cmds = append(cmds, m.Cursor.BlinkCmd())
	}

	return m, tea.Batch(cmds...)
}

// View renders the textinput in its current state.
func (m Model) View() string {
	styleText := m.TextStyle.Inline(true).Render

	value := m.values[m.selectedValueIndex]
	pos := max(0, m.pos)
	v := m.PromptStyle.Render(m.Prompt) + styleText(m.echoTransform(string(value[:pos])))

	if pos < len(value) { //nolint:nestif
		char := m.echoTransform(string(value[pos]))
		m.Cursor.SetChar(char)
		v += m.Cursor.View()                                   // cursor and text under it
		v += styleText(m.echoTransform(string(value[pos+1:]))) // text after cursor
		v += m.completionView(0)                               // suggested completion
	} else {
		if m.canAcceptSuggestion() {
			suggestion := m.matchedSuggestions[m.currentSuggestionIndex]
			if len(value) < len(suggestion) {
				m.Cursor.TextStyle = m.CompletionStyle
				m.Cursor.SetChar(m.echoTransform(string(suggestion[pos])))
				v += m.Cursor.View()
				v += m.completionView(1)
			} else {
				m.Cursor.SetChar(" ")
				v += m.Cursor.View()
			}
		} else {
			m.Cursor.SetChar(" ")
			v += m.Cursor.View()
		}
	}

	totalWidth := uniseg.StringWidth(v)

	// If a max width is set, we need to respect the horizontal boundary
	if m.Width > 0 {
		if totalWidth <= m.Width {
			// fill empty spaces with the background color
			padding := max(0, m.Width-totalWidth)
			if totalWidth+padding <= m.Width && pos < len(value) {
				padding++
			}
			v += styleText(strings.Repeat(" ", padding))
		} else {
			v = wrap.String(v, m.Width)
		}
	}

	return v
}

// Blink is a command used to initialize cursor blinking.
func Blink() tea.Msg {
	return cursor.Blink()
}

// Paste is a command for pasting from the clipboard into the text input.
func Paste() tea.Msg {
	str, err := clipboard.ReadAll()
	if err != nil {
		return pasteErrMsg{err}
	}
	return pasteMsg(str)
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Deprecated.

// Deprecated: use cursor.Mode.
type CursorMode int

const (
	// Deprecated: use cursor.CursorBlink.
	CursorBlink = CursorMode(cursor.CursorBlink)
	// Deprecated: use cursor.CursorStatic.
	CursorStatic = CursorMode(cursor.CursorStatic)
	// Deprecated: use cursor.CursorHide.
	CursorHide = CursorMode(cursor.CursorHide)
)

func (c CursorMode) String() string {
	return cursor.Mode(c).String()
}

// Deprecated: use cursor.Mode().
func (m Model) CursorMode() CursorMode {
	return CursorMode(m.Cursor.Mode())
}

// Deprecated: use cursor.SetMode().
func (m *Model) SetCursorMode(mode CursorMode) tea.Cmd {
	return m.Cursor.SetMode(cursor.Mode(mode))
}

func (m Model) completionView(offset int) string {
	var (
		value = m.values[m.selectedValueIndex]
		style = m.CompletionStyle.Inline(true).Render
	)

	if m.canAcceptSuggestion() {
		suggestion := m.matchedSuggestions[m.currentSuggestionIndex]
		if len(value) < len(suggestion) {
			return style(string(suggestion[len(value)+offset:]))
		}
	}
	return ""
}

func (m *Model) getSuggestions(sugs [][]rune) []string {
	suggestions := make([]string, len(sugs))
	for i, s := range sugs {
		suggestions[i] = string(s)
	}
	return suggestions
}

// AvailableSuggestions returns the list of available suggestions.
func (m *Model) AvailableSuggestions() []string {
	return m.getSuggestions(m.suggestions)
}

// MatchedSuggestions returns the list of matched suggestions.
func (m *Model) MatchedSuggestions() []string {
	return m.getSuggestions(m.matchedSuggestions)
}

// CurrentSuggestion returns the currently selected suggestion index.
func (m *Model) CurrentSuggestionIndex() int {
	return m.currentSuggestionIndex
}

// CurrentSuggestion returns the currently selected suggestion.
func (m *Model) CurrentSuggestion() string {
	if m.currentSuggestionIndex >= len(m.matchedSuggestions) {
		return ""
	}

	return string(m.matchedSuggestions[m.currentSuggestionIndex])
}

// canAcceptSuggestion returns whether there is an acceptable suggestion to
// autocomplete the current value.
func (m *Model) canAcceptSuggestion() bool {
	return len(m.matchedSuggestions) > 0
}

// updateSuggestions refreshes the list of matching suggestions.
func (m *Model) updateSuggestions() {
	if !m.ShowSuggestions {
		return
	}

	if len(m.suggestions) <= 0 {
		m.matchedSuggestions = [][]rune{}
		return
	}

	matches := [][]rune{}
	for _, s := range m.suggestions {
		suggestion := string(s)

		if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(string(m.values[m.selectedValueIndex]))) {
			matches = append(matches, []rune(suggestion))
		}
	}
	if !reflect.DeepEqual(matches, m.matchedSuggestions) {
		m.currentSuggestionIndex = 0
	}

	m.matchedSuggestions = matches
}

func (m *Model) nextValue() {
	if len(m.values) == 1 {
		return
	}

	m.selectedValueIndex--
	if m.selectedValueIndex < 0 {
		m.selectedValueIndex = 0
	}
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

func (m *Model) previousValue() {
	if len(m.values) == 1 {
		return
	}

	m.selectedValueIndex++
	if m.selectedValueIndex >= len(m.values) {
		m.selectedValueIndex = len(m.values) - 1
	}
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

func (m Model) validate(v []rune) error {
	if m.Validate != nil {
		return m.Validate(string(v))
	}
	return nil
}

func cloneRunes(r []rune) []rune {
	clone := make([]rune, len(r))
	copy(clone, r)
	return clone
}

func cloneConcatRunes(r1, r2 []rune) []rune {
	clone := make([]rune, len(r1)+len(r2))
	copy(clone, r1)
	copy(clone[len(r1):], r2)
	return clone
}
