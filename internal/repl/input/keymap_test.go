package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestActionString(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{ActionNone, "None"},
		{ActionCharacterForward, "CharacterForward"},
		{ActionCharacterBackward, "CharacterBackward"},
		{ActionWordForward, "WordForward"},
		{ActionWordBackward, "WordBackward"},
		{ActionLineStart, "LineStart"},
		{ActionLineEnd, "LineEnd"},
		{ActionDeleteCharacterBackward, "DeleteCharacterBackward"},
		{ActionDeleteCharacterForward, "DeleteCharacterForward"},
		{ActionDeleteWordBackward, "DeleteWordBackward"},
		{ActionDeleteWordForward, "DeleteWordForward"},
		{ActionDeleteBeforeCursor, "DeleteBeforeCursor"},
		{ActionDeleteAfterCursor, "DeleteAfterCursor"},
		{ActionCursorUp, "CursorUp"},
		{ActionCursorDown, "CursorDown"},
		{ActionComplete, "Complete"},
		{ActionCompleteBackward, "CompleteBackward"},
		{ActionSubmit, "Submit"},
		{ActionCancel, "Cancel"},
		{ActionInterrupt, "Interrupt"},
		{ActionEOF, "EOF"},
		{ActionClearScreen, "ClearScreen"},
		{ActionPaste, "Paste"},
		{ActionAcceptPrediction, "AcceptPrediction"},
		{Action(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("Action.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewKeyMap(t *testing.T) {
	bindings := []KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
		{Keys: []string{"ctrl+e"}, Action: ActionLineEnd},
	}

	km := NewKeyMap(bindings)

	if km == nil {
		t.Fatal("NewKeyMap returned nil")
	}

	if len(km.bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(km.bindings))
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	if km == nil {
		t.Fatal("DefaultKeyMap returned nil")
	}

	// Verify some expected default bindings
	expectedBindings := map[Action][]string{
		ActionCharacterForward:        {"right", "ctrl+f"},
		ActionCharacterBackward:       {"left", "ctrl+b"},
		ActionWordForward:             {"alt+right", "ctrl+right", "alt+f"},
		ActionWordBackward:            {"alt+left", "ctrl+left", "alt+b"},
		ActionLineStart:               {"home", "ctrl+a"},
		ActionLineEnd:                 {"end", "ctrl+e"},
		ActionDeleteCharacterBackward: {"backspace", "ctrl+h"},
		ActionDeleteCharacterForward:  {"delete", "ctrl+d"},
		ActionDeleteWordBackward:      {"ctrl+w", "alt+backspace"},
		ActionDeleteWordForward:       {"alt+d", "alt+delete"},
		ActionDeleteBeforeCursor:      {"ctrl+u"},
		ActionDeleteAfterCursor:       {"ctrl+k"},
		ActionCursorUp:                {"up", "ctrl+p"},
		ActionCursorDown:              {"down", "ctrl+n"},
		ActionComplete:                {"tab"},
		ActionCompleteBackward:        {"shift+tab"},
		ActionSubmit:                  {"enter"},
		ActionCancel:                  {"esc"},
		ActionInterrupt:               {"ctrl+c"},
		ActionClearScreen:             {"ctrl+l"},
		ActionPaste:                   {"ctrl+v"},
	}

	for action, expectedKeys := range expectedBindings {
		binding := km.GetBinding(action)
		if binding == nil {
			t.Errorf("Missing binding for action %s", action)
			continue
		}

		if len(binding.Keys) != len(expectedKeys) {
			t.Errorf("Action %s: expected %d keys, got %d", action, len(expectedKeys), len(binding.Keys))
			continue
		}

		for i, key := range expectedKeys {
			if binding.Keys[i] != key {
				t.Errorf("Action %s: key[%d] = %q, want %q", action, i, binding.Keys[i], key)
			}
		}
	}
}

// createKeyMsg creates a tea.KeyMsg for testing
func createKeyMsg(keyStr string) tea.KeyMsg {
	// Map common key strings to their tea.KeyMsg equivalents
	keyMap := map[string]tea.Key{
		"ctrl+a":        {Type: tea.KeyCtrlA},
		"ctrl+b":        {Type: tea.KeyCtrlB},
		"ctrl+c":        {Type: tea.KeyCtrlC},
		"ctrl+d":        {Type: tea.KeyCtrlD},
		"ctrl+e":        {Type: tea.KeyCtrlE},
		"ctrl+f":        {Type: tea.KeyCtrlF},
		"ctrl+h":        {Type: tea.KeyCtrlH},
		"ctrl+k":        {Type: tea.KeyCtrlK},
		"ctrl+l":        {Type: tea.KeyCtrlL},
		"ctrl+n":        {Type: tea.KeyCtrlN},
		"ctrl+p":        {Type: tea.KeyCtrlP},
		"ctrl+u":        {Type: tea.KeyCtrlU},
		"ctrl+v":        {Type: tea.KeyCtrlV},
		"ctrl+w":        {Type: tea.KeyCtrlW},
		"enter":         {Type: tea.KeyEnter},
		"tab":           {Type: tea.KeyTab},
		"shift+tab":     {Type: tea.KeyShiftTab},
		"esc":           {Type: tea.KeyEscape},
		"backspace":     {Type: tea.KeyBackspace},
		"delete":        {Type: tea.KeyDelete},
		"up":            {Type: tea.KeyUp},
		"down":          {Type: tea.KeyDown},
		"left":          {Type: tea.KeyLeft},
		"right":         {Type: tea.KeyRight},
		"home":          {Type: tea.KeyHome},
		"end":           {Type: tea.KeyEnd},
		"ctrl+left":     {Type: tea.KeyCtrlLeft},
		"ctrl+right":    {Type: tea.KeyCtrlRight},
		"alt+left":      {Type: tea.KeyLeft, Alt: true},
		"alt+right":     {Type: tea.KeyRight, Alt: true},
		"alt+backspace": {Type: tea.KeyBackspace, Alt: true},
		"alt+delete":    {Type: tea.KeyDelete, Alt: true},
		"alt+b":         {Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true},
		"alt+d":         {Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true},
		"alt+f":         {Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true},
	}

	if key, ok := keyMap[keyStr]; ok {
		return tea.KeyMsg(key)
	}

	// For single character keys
	if len(keyStr) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
	}

	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
}

func TestKeyMapLookup(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		keyStr   string
		expected Action
	}{
		// Navigation
		{"ctrl+f", ActionCharacterForward},
		{"right", ActionCharacterForward},
		{"ctrl+b", ActionCharacterBackward},
		{"left", ActionCharacterBackward},
		{"ctrl+a", ActionLineStart},
		{"home", ActionLineStart},
		{"ctrl+e", ActionLineEnd},
		{"end", ActionLineEnd},

		// Deletion
		{"backspace", ActionDeleteCharacterBackward},
		{"ctrl+h", ActionDeleteCharacterBackward},
		{"delete", ActionDeleteCharacterForward},
		{"ctrl+d", ActionDeleteCharacterForward},
		{"ctrl+w", ActionDeleteWordBackward},
		{"ctrl+u", ActionDeleteBeforeCursor},
		{"ctrl+k", ActionDeleteAfterCursor},

		// History
		{"up", ActionCursorUp},
		{"ctrl+p", ActionCursorUp},
		{"down", ActionCursorDown},
		{"ctrl+n", ActionCursorDown},

		// Completion
		{"tab", ActionComplete},
		{"shift+tab", ActionCompleteBackward},

		// Special
		{"enter", ActionSubmit},
		{"esc", ActionCancel},
		{"ctrl+c", ActionInterrupt},
		{"ctrl+l", ActionClearScreen},
		{"ctrl+v", ActionPaste},

		// Unknown key
		{"f12", ActionNone},
	}

	for _, tt := range tests {
		t.Run(tt.keyStr, func(t *testing.T) {
			msg := createKeyMsg(tt.keyStr)
			got := km.Lookup(msg)
			if got != tt.expected {
				t.Errorf("Lookup(%q) = %s, want %s", tt.keyStr, got, tt.expected)
			}
		})
	}
}

func TestKeyMapSetBinding(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
	})

	// Update existing binding
	km.SetBinding(KeyBinding{Keys: []string{"ctrl+a", "home"}, Action: ActionLineStart})

	binding := km.GetBinding(ActionLineStart)
	if binding == nil {
		t.Fatal("Expected to find ActionLineStart binding")
	}
	if len(binding.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(binding.Keys))
	}

	// Add new binding
	km.SetBinding(KeyBinding{Keys: []string{"ctrl+e"}, Action: ActionLineEnd})

	binding = km.GetBinding(ActionLineEnd)
	if binding == nil {
		t.Fatal("Expected to find ActionLineEnd binding")
	}
	if len(km.bindings) != 2 {
		t.Errorf("Expected 2 total bindings, got %d", len(km.bindings))
	}
}

func TestKeyMapRemoveBinding(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
		{Keys: []string{"ctrl+e"}, Action: ActionLineEnd},
		{Keys: []string{"ctrl+k"}, Action: ActionDeleteAfterCursor},
	})

	km.RemoveBinding(ActionLineEnd)

	if len(km.bindings) != 2 {
		t.Errorf("Expected 2 bindings after removal, got %d", len(km.bindings))
	}

	if km.GetBinding(ActionLineEnd) != nil {
		t.Error("ActionLineEnd binding should have been removed")
	}

	// Verify other bindings are intact
	if km.GetBinding(ActionLineStart) == nil {
		t.Error("ActionLineStart binding should still exist")
	}
	if km.GetBinding(ActionDeleteAfterCursor) == nil {
		t.Error("ActionDeleteAfterCursor binding should still exist")
	}
}

func TestKeyMapGetBinding(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
	})

	// Existing binding
	binding := km.GetBinding(ActionLineStart)
	if binding == nil {
		t.Fatal("Expected to find ActionLineStart binding")
	}
	if binding.Action != ActionLineStart {
		t.Errorf("Expected ActionLineStart, got %s", binding.Action)
	}

	// Non-existing binding
	binding = km.GetBinding(ActionLineEnd)
	if binding != nil {
		t.Error("Expected nil for non-existing binding")
	}
}

func TestKeyMapAddKeys(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
	})

	// Add to existing binding
	km.AddKeys(ActionLineStart, "home", "ctrl+left")

	binding := km.GetBinding(ActionLineStart)
	if binding == nil {
		t.Fatal("Expected to find ActionLineStart binding")
	}
	if len(binding.Keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(binding.Keys))
	}

	// Add to non-existing binding (creates new)
	km.AddKeys(ActionLineEnd, "ctrl+e", "end")

	binding = km.GetBinding(ActionLineEnd)
	if binding == nil {
		t.Fatal("Expected to find ActionLineEnd binding")
	}
	if len(binding.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(binding.Keys))
	}
}

func TestKeyMapRemoveKeys(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a", "home", "ctrl+left"}, Action: ActionLineStart},
	})

	km.RemoveKeys(ActionLineStart, "home", "ctrl+left")

	binding := km.GetBinding(ActionLineStart)
	if binding == nil {
		t.Fatal("Expected to find ActionLineStart binding")
	}
	if len(binding.Keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(binding.Keys))
	}
	if binding.Keys[0] != "ctrl+a" {
		t.Errorf("Expected 'ctrl+a', got %q", binding.Keys[0])
	}

	// Remove from non-existing action (no-op)
	km.RemoveKeys(ActionLineEnd, "ctrl+e")
	// Should not panic or cause issues
}

func TestKeyMapClone(t *testing.T) {
	original := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a", "home"}, Action: ActionLineStart},
		{Keys: []string{"ctrl+e"}, Action: ActionLineEnd},
	})

	clone := original.Clone()

	// Verify clone has same content
	if len(clone.bindings) != len(original.bindings) {
		t.Errorf("Clone has %d bindings, original has %d", len(clone.bindings), len(original.bindings))
	}

	// Modify original and verify clone is unaffected
	original.SetBinding(KeyBinding{Keys: []string{"ctrl+a"}, Action: ActionLineStart})
	original.AddKeys(ActionLineEnd, "end")

	cloneBinding := clone.GetBinding(ActionLineStart)
	if cloneBinding == nil {
		t.Fatal("Clone should have ActionLineStart binding")
	}
	if len(cloneBinding.Keys) != 2 {
		t.Errorf("Clone ActionLineStart should still have 2 keys, got %d", len(cloneBinding.Keys))
	}

	cloneBinding = clone.GetBinding(ActionLineEnd)
	if cloneBinding == nil {
		t.Fatal("Clone should have ActionLineEnd binding")
	}
	if len(cloneBinding.Keys) != 1 {
		t.Errorf("Clone ActionLineEnd should still have 1 key, got %d", len(cloneBinding.Keys))
	}
}

func TestKeyMapBindings(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
		{Keys: []string{"ctrl+e"}, Action: ActionLineEnd},
	})

	bindings := km.Bindings()

	if len(bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(bindings))
	}

	// Verify returned slice is a copy
	bindings[0].Keys = []string{"modified"}

	originalBinding := km.GetBinding(ActionLineStart)
	if originalBinding.Keys[0] == "modified" {
		t.Error("Modifying returned bindings should not affect original")
	}
}

func TestKeyMapEmptyKeyMap(t *testing.T) {
	km := NewKeyMap([]KeyBinding{})

	// Lookup should return ActionNone
	msg := createKeyMsg("ctrl+a")
	if got := km.Lookup(msg); got != ActionNone {
		t.Errorf("Empty keymap Lookup should return ActionNone, got %s", got)
	}

	// GetBinding should return nil
	if km.GetBinding(ActionLineStart) != nil {
		t.Error("Empty keymap GetBinding should return nil")
	}

	// Bindings should return empty slice
	if len(km.Bindings()) != 0 {
		t.Error("Empty keymap Bindings should return empty slice")
	}
}

func TestKeyMapMultipleKeysForSameAction(t *testing.T) {
	km := DefaultKeyMap()

	// Test that both 'right' and 'ctrl+f' map to the same action
	rightMsg := createKeyMsg("right")
	ctrlFMsg := createKeyMsg("ctrl+f")

	rightAction := km.Lookup(rightMsg)
	ctrlFAction := km.Lookup(ctrlFMsg)

	if rightAction != ctrlFAction {
		t.Errorf("Expected same action for 'right' and 'ctrl+f', got %s and %s", rightAction, ctrlFAction)
	}
	if rightAction != ActionCharacterForward {
		t.Errorf("Expected ActionCharacterForward, got %s", rightAction)
	}
}

func TestKeyMapWordNavigationBindings(t *testing.T) {
	km := DefaultKeyMap()

	// Test word navigation has all expected keys
	wordForward := km.GetBinding(ActionWordForward)
	if wordForward == nil {
		t.Fatal("Expected ActionWordForward binding")
	}

	expectedKeys := map[string]bool{
		"alt+right":  false,
		"ctrl+right": false,
		"alt+f":      false,
	}

	for _, key := range wordForward.Keys {
		if _, ok := expectedKeys[key]; ok {
			expectedKeys[key] = true
		}
	}

	for key, found := range expectedKeys {
		if !found {
			t.Errorf("Expected key %q in ActionWordForward binding", key)
		}
	}
}

func TestKeyMapDeletionBindings(t *testing.T) {
	km := DefaultKeyMap()

	// Verify all deletion actions have bindings
	deletionActions := []Action{
		ActionDeleteCharacterBackward,
		ActionDeleteCharacterForward,
		ActionDeleteWordBackward,
		ActionDeleteWordForward,
		ActionDeleteBeforeCursor,
		ActionDeleteAfterCursor,
	}

	for _, action := range deletionActions {
		binding := km.GetBinding(action)
		if binding == nil {
			t.Errorf("Missing binding for deletion action %s", action)
		}
		if len(binding.Keys) == 0 {
			t.Errorf("Deletion action %s has no keys", action)
		}
	}
}

func TestKeyMapSpecialKeysBindings(t *testing.T) {
	km := DefaultKeyMap()

	// Verify special keys have exactly one key bound
	specialActions := map[Action]string{
		ActionSubmit:      "enter",
		ActionCancel:      "esc",
		ActionInterrupt:   "ctrl+c",
		ActionClearScreen: "ctrl+l",
		ActionPaste:       "ctrl+v",
	}

	for action, expectedKey := range specialActions {
		binding := km.GetBinding(action)
		if binding == nil {
			t.Errorf("Missing binding for special action %s", action)
			continue
		}
		if len(binding.Keys) != 1 {
			t.Errorf("Special action %s should have exactly 1 key, got %d", action, len(binding.Keys))
			continue
		}
		if binding.Keys[0] != expectedKey {
			t.Errorf("Special action %s key = %q, want %q", action, binding.Keys[0], expectedKey)
		}
	}
}

func TestKeyMapLookupAfterModifications(t *testing.T) {
	km := NewKeyMap([]KeyBinding{
		{Keys: []string{"ctrl+a"}, Action: ActionLineStart},
	})

	// Verify initial lookup works
	msg := createKeyMsg("ctrl+a")
	if got := km.Lookup(msg); got != ActionLineStart {
		t.Errorf("Initial lookup failed: got %s, want ActionLineStart", got)
	}

	// Add keys and verify lookup is updated
	km.AddKeys(ActionLineStart, "home")
	homeMsg := createKeyMsg("home")
	if got := km.Lookup(homeMsg); got != ActionLineStart {
		t.Errorf("After AddKeys: lookup('home') = %s, want ActionLineStart", got)
	}

	// SetBinding and verify lookup is updated
	km.SetBinding(KeyBinding{Keys: []string{"ctrl+x"}, Action: ActionLineEnd})
	ctrlXMsg := createKeyMsg("ctrl+x")
	if got := km.Lookup(ctrlXMsg); got != ActionLineEnd {
		t.Errorf("After SetBinding: lookup('ctrl+x') = %s, want ActionLineEnd", got)
	}

	// RemoveKeys and verify lookup is updated
	km.RemoveKeys(ActionLineStart, "ctrl+a")
	if got := km.Lookup(msg); got != ActionNone {
		t.Errorf("After RemoveKeys: lookup('ctrl+a') = %s, want ActionNone", got)
	}
	// But 'home' should still work
	if got := km.Lookup(homeMsg); got != ActionLineStart {
		t.Errorf("After RemoveKeys: lookup('home') = %s, want ActionLineStart", got)
	}

	// RemoveBinding and verify lookup is updated
	km.RemoveBinding(ActionLineEnd)
	if got := km.Lookup(ctrlXMsg); got != ActionNone {
		t.Errorf("After RemoveBinding: lookup('ctrl+x') = %s, want ActionNone", got)
	}
}

func TestKeyMapCloneLookupIndependence(t *testing.T) {
	original := DefaultKeyMap()
	clone := original.Clone()

	// Modify original
	original.SetBinding(KeyBinding{Keys: []string{"f1"}, Action: ActionLineStart})

	// Verify clone lookup is not affected
	f1Msg := createKeyMsg("f1")
	if got := clone.Lookup(f1Msg); got != ActionNone {
		t.Errorf("Clone should not have f1 binding after modifying original, got %s", got)
	}

	// Verify original has the new binding
	if got := original.Lookup(f1Msg); got != ActionLineStart {
		t.Errorf("Original should have f1 binding, got %s", got)
	}
}
