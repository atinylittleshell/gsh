package input

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Action represents a keyboard action that can be triggered by key bindings.
type Action int

const (
	// ActionNone represents no action (used when a key doesn't match any binding).
	ActionNone Action = iota

	// Navigation actions
	ActionCharacterForward  // Move cursor one character forward (Ctrl+F, Right)
	ActionCharacterBackward // Move cursor one character backward (Ctrl+B, Left)
	ActionWordForward       // Move cursor one word forward (Alt+F, Alt+Right)
	ActionWordBackward      // Move cursor one word backward (Alt+B, Alt+Left)
	ActionLineStart         // Move cursor to start of line (Ctrl+A, Home)
	ActionLineEnd           // Move cursor to end of line (Ctrl+E, End)

	// Deletion actions
	ActionDeleteCharacterBackward // Delete character before cursor (Backspace, Ctrl+H)
	ActionDeleteCharacterForward  // Delete character at cursor (Delete, Ctrl+D)
	ActionDeleteWordBackward      // Delete word before cursor (Ctrl+W, Alt+Backspace)
	ActionDeleteWordForward       // Delete word after cursor (Alt+D, Alt+Delete)
	ActionDeleteBeforeCursor      // Delete all text before cursor (Ctrl+U)
	ActionDeleteAfterCursor       // Delete all text after cursor (Ctrl+K)

	// Vertical navigation (context-dependent: history or completion)
	ActionCursorUp   // Move up (Up, Ctrl+P) - history previous or completion previous
	ActionCursorDown // Move down (Down, Ctrl+N) - history next or completion next

	// Completion actions
	ActionComplete         // Trigger tab completion (Tab)
	ActionCompleteBackward // Cycle backwards through completions (Shift+Tab)

	// Special actions
	ActionSubmit      // Submit the current input (Enter)
	ActionCancel      // Cancel current operation (Escape)
	ActionInterrupt   // Send interrupt signal (Ctrl+C)
	ActionEOF         // End of file / exit (Ctrl+D when input is empty)
	ActionClearScreen // Clear the screen (Ctrl+L)
	ActionPaste       // Paste from clipboard (Ctrl+V)

	// Prediction actions
	ActionAcceptPrediction // Accept the current prediction (Right arrow at end of line)
)

// String returns the string representation of an Action.
func (a Action) String() string {
	switch a {
	case ActionNone:
		return "None"
	case ActionCharacterForward:
		return "CharacterForward"
	case ActionCharacterBackward:
		return "CharacterBackward"
	case ActionWordForward:
		return "WordForward"
	case ActionWordBackward:
		return "WordBackward"
	case ActionLineStart:
		return "LineStart"
	case ActionLineEnd:
		return "LineEnd"
	case ActionDeleteCharacterBackward:
		return "DeleteCharacterBackward"
	case ActionDeleteCharacterForward:
		return "DeleteCharacterForward"
	case ActionDeleteWordBackward:
		return "DeleteWordBackward"
	case ActionDeleteWordForward:
		return "DeleteWordForward"
	case ActionDeleteBeforeCursor:
		return "DeleteBeforeCursor"
	case ActionDeleteAfterCursor:
		return "DeleteAfterCursor"
	case ActionCursorUp:
		return "CursorUp"
	case ActionCursorDown:
		return "CursorDown"
	case ActionComplete:
		return "Complete"
	case ActionCompleteBackward:
		return "CompleteBackward"
	case ActionSubmit:
		return "Submit"
	case ActionCancel:
		return "Cancel"
	case ActionInterrupt:
		return "Interrupt"
	case ActionEOF:
		return "EOF"
	case ActionClearScreen:
		return "ClearScreen"
	case ActionPaste:
		return "Paste"
	case ActionAcceptPrediction:
		return "AcceptPrediction"
	default:
		return "Unknown"
	}
}

// KeyBinding represents a single key binding that maps a key to an action.
type KeyBinding struct {
	// Keys is the list of key sequences that trigger this binding.
	// Each string should be a valid tea.KeyMsg string representation.
	Keys []string
	// Action is the action to perform when this binding is triggered.
	Action Action
}

// KeyMap holds all key bindings for the input component.
// It provides a configurable mapping from key presses to actions.
// Lookup is O(1) using an internal hash map.
type KeyMap struct {
	bindings []KeyBinding
	lookup   map[string]Action
}

// NewKeyMap creates a new KeyMap with the given bindings.
func NewKeyMap(bindings []KeyBinding) *KeyMap {
	km := &KeyMap{
		bindings: bindings,
		lookup:   make(map[string]Action),
	}
	km.rebuildLookup()
	return km
}

// rebuildLookup rebuilds the internal lookup map from the bindings.
// This must be called after any modification to bindings.
func (km *KeyMap) rebuildLookup() {
	km.lookup = make(map[string]Action)
	for _, b := range km.bindings {
		for _, key := range b.Keys {
			km.lookup[key] = b.Action
		}
	}
}

// DefaultKeyMap returns a KeyMap with default Emacs-style key bindings.
func DefaultKeyMap() *KeyMap {
	return NewKeyMap([]KeyBinding{
		// Navigation
		{Keys: []string{"right", "ctrl+f"}, Action: ActionCharacterForward},
		{Keys: []string{"left", "ctrl+b"}, Action: ActionCharacterBackward},
		{Keys: []string{"alt+right", "ctrl+right", "alt+f"}, Action: ActionWordForward},
		{Keys: []string{"alt+left", "ctrl+left", "alt+b"}, Action: ActionWordBackward},
		{Keys: []string{"home", "ctrl+a"}, Action: ActionLineStart},
		{Keys: []string{"end", "ctrl+e"}, Action: ActionLineEnd},

		// Deletion
		{Keys: []string{"backspace", "ctrl+h"}, Action: ActionDeleteCharacterBackward},
		{Keys: []string{"delete", "ctrl+d"}, Action: ActionDeleteCharacterForward},
		{Keys: []string{"ctrl+w", "alt+backspace"}, Action: ActionDeleteWordBackward},
		{Keys: []string{"alt+d", "alt+delete"}, Action: ActionDeleteWordForward},
		{Keys: []string{"ctrl+u"}, Action: ActionDeleteBeforeCursor},
		{Keys: []string{"ctrl+k"}, Action: ActionDeleteAfterCursor},

		// Vertical navigation (context-dependent)
		{Keys: []string{"up", "ctrl+p"}, Action: ActionCursorUp},
		{Keys: []string{"down", "ctrl+n"}, Action: ActionCursorDown},

		// Completion
		{Keys: []string{"tab"}, Action: ActionComplete},
		{Keys: []string{"shift+tab"}, Action: ActionCompleteBackward},

		// Special keys
		{Keys: []string{"enter"}, Action: ActionSubmit},
		{Keys: []string{"esc"}, Action: ActionCancel},
		{Keys: []string{"ctrl+c"}, Action: ActionInterrupt},
		{Keys: []string{"ctrl+l"}, Action: ActionClearScreen},
		{Keys: []string{"ctrl+v"}, Action: ActionPaste},
	})
}

// Lookup finds the action for the given key message.
// Returns ActionNone if no binding matches.
// This is an O(1) operation using the internal lookup map.
func (km *KeyMap) Lookup(msg tea.KeyMsg) Action {
	if action, ok := km.lookup[msg.String()]; ok {
		return action
	}
	return ActionNone
}

// SetBinding adds or updates a key binding.
// If a binding for the same action already exists, it will be replaced.
func (km *KeyMap) SetBinding(binding KeyBinding) {
	// Look for existing binding with the same action
	for i, b := range km.bindings {
		if b.Action == binding.Action {
			km.bindings[i] = binding
			km.rebuildLookup()
			return
		}
	}
	// Add new binding
	km.bindings = append(km.bindings, binding)
	km.rebuildLookup()
}

// RemoveBinding removes all bindings for the given action.
func (km *KeyMap) RemoveBinding(action Action) {
	newBindings := make([]KeyBinding, 0, len(km.bindings))
	for _, b := range km.bindings {
		if b.Action != action {
			newBindings = append(newBindings, b)
		}
	}
	km.bindings = newBindings
	km.rebuildLookup()
}

// GetBinding returns the binding for the given action, or nil if not found.
func (km *KeyMap) GetBinding(action Action) *KeyBinding {
	for i := range km.bindings {
		if km.bindings[i].Action == action {
			return &km.bindings[i]
		}
	}
	return nil
}

// AddKeys adds additional keys to an existing action binding.
// If the action doesn't exist, a new binding is created.
func (km *KeyMap) AddKeys(action Action, keys ...string) {
	for i := range km.bindings {
		if km.bindings[i].Action == action {
			km.bindings[i].Keys = append(km.bindings[i].Keys, keys...)
			km.rebuildLookup()
			return
		}
	}
	// Action not found, create new binding
	km.bindings = append(km.bindings, KeyBinding{
		Keys:   keys,
		Action: action,
	})
	km.rebuildLookup()
}

// RemoveKeys removes specific keys from an action binding.
func (km *KeyMap) RemoveKeys(action Action, keys ...string) {
	for i := range km.bindings {
		if km.bindings[i].Action == action {
			newKeys := make([]string, 0, len(km.bindings[i].Keys))
			for _, existingKey := range km.bindings[i].Keys {
				shouldRemove := false
				for _, keyToRemove := range keys {
					if existingKey == keyToRemove {
						shouldRemove = true
						break
					}
				}
				if !shouldRemove {
					newKeys = append(newKeys, existingKey)
				}
			}
			km.bindings[i].Keys = newKeys
			km.rebuildLookup()
			return
		}
	}
}

// Clone creates a deep copy of the KeyMap.
func (km *KeyMap) Clone() *KeyMap {
	newBindings := make([]KeyBinding, len(km.bindings))
	for i, b := range km.bindings {
		newKeys := make([]string, len(b.Keys))
		copy(newKeys, b.Keys)
		newBindings[i] = KeyBinding{
			Keys:   newKeys,
			Action: b.Action,
		}
	}
	clone := &KeyMap{
		bindings: newBindings,
		lookup:   make(map[string]Action),
	}
	clone.rebuildLookup()
	return clone
}

// Bindings returns a copy of all bindings in the keymap.
func (km *KeyMap) Bindings() []KeyBinding {
	result := make([]KeyBinding, len(km.bindings))
	for i, b := range km.bindings {
		keys := make([]string, len(b.Keys))
		copy(keys, b.Keys)
		result[i] = KeyBinding{
			Keys:   keys,
			Action: b.Action,
		}
	}
	return result
}
