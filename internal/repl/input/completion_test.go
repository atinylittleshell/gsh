package input

import (
	"strings"
	"testing"
)

// TestNewCompletionState tests the creation of a new CompletionState.
func TestNewCompletionState(t *testing.T) {
	cs := NewCompletionState()

	if cs.IsActive() {
		t.Error("new CompletionState should not be active")
	}
	if cs.Selected() != -1 {
		t.Errorf("new CompletionState should have selected = -1, got %d", cs.Selected())
	}
	if len(cs.Suggestions()) != 0 {
		t.Error("new CompletionState should have no suggestions")
	}
	if cs.Prefix() != "" {
		t.Error("new CompletionState should have empty prefix")
	}
	if cs.StartPos() != 0 {
		t.Errorf("new CompletionState should have startPos = 0, got %d", cs.StartPos())
	}
	if cs.EndPos() != 0 {
		t.Errorf("new CompletionState should have endPos = 0, got %d", cs.EndPos())
	}
	if cs.IsVisible() {
		t.Error("new CompletionState should not be visible")
	}
}

// TestCompletionStateReset tests the Reset method.
func TestCompletionStateReset(t *testing.T) {
	cs := NewCompletionState()

	// Activate with some state
	cs.Activate([]string{"foo", "bar", "baz"}, "f", 0, 1)
	cs.SetOriginalText("original")
	cs.NextSuggestion() // Select first suggestion

	// Verify state is set
	if !cs.IsActive() {
		t.Error("CompletionState should be active after Activate")
	}

	// Reset
	cs.Reset()

	// Verify all state is cleared
	if cs.IsActive() {
		t.Error("CompletionState should not be active after Reset")
	}
	if cs.Selected() != -1 {
		t.Errorf("selected should be -1 after Reset, got %d", cs.Selected())
	}
	if len(cs.Suggestions()) != 0 {
		t.Error("suggestions should be empty after Reset")
	}
	if cs.Prefix() != "" {
		t.Error("prefix should be empty after Reset")
	}
	if cs.StartPos() != 0 {
		t.Error("startPos should be 0 after Reset")
	}
	if cs.EndPos() != 0 {
		t.Error("endPos should be 0 after Reset")
	}
	if cs.OriginalText() != "" {
		t.Error("originalText should be empty after Reset")
	}
	if cs.IsVisible() {
		t.Error("should not be visible after Reset")
	}
}

// TestCompletionStateActivate tests the Activate method.
func TestCompletionStateActivate(t *testing.T) {
	cs := NewCompletionState()
	suggestions := []string{"apple", "apricot", "avocado"}

	cs.Activate(suggestions, "a", 5, 6)

	if !cs.IsActive() {
		t.Error("CompletionState should be active after Activate")
	}
	if len(cs.Suggestions()) != 3 {
		t.Errorf("expected 3 suggestions, got %d", len(cs.Suggestions()))
	}
	if cs.Selected() != -1 {
		t.Errorf("selected should be -1 initially, got %d", cs.Selected())
	}
	if cs.Prefix() != "a" {
		t.Errorf("prefix should be 'a', got '%s'", cs.Prefix())
	}
	if cs.StartPos() != 5 {
		t.Errorf("startPos should be 5, got %d", cs.StartPos())
	}
	if cs.EndPos() != 6 {
		t.Errorf("endPos should be 6, got %d", cs.EndPos())
	}
}

// TestCompletionStateNextSuggestion tests cycling forward through suggestions.
func TestCompletionStateNextSuggestion(t *testing.T) {
	cs := NewCompletionState()
	suggestions := []string{"first", "second", "third"}
	cs.Activate(suggestions, "", 0, 0)

	// First call should select index 0
	s := cs.NextSuggestion()
	if s != "first" {
		t.Errorf("expected 'first', got '%s'", s)
	}
	if cs.Selected() != 0 {
		t.Errorf("expected selected = 0, got %d", cs.Selected())
	}

	// Second call should select index 1
	s = cs.NextSuggestion()
	if s != "second" {
		t.Errorf("expected 'second', got '%s'", s)
	}
	if cs.Selected() != 1 {
		t.Errorf("expected selected = 1, got %d", cs.Selected())
	}

	// Third call should select index 2
	s = cs.NextSuggestion()
	if s != "third" {
		t.Errorf("expected 'third', got '%s'", s)
	}
	if cs.Selected() != 2 {
		t.Errorf("expected selected = 2, got %d", cs.Selected())
	}

	// Fourth call should wrap around to index 0
	s = cs.NextSuggestion()
	if s != "first" {
		t.Errorf("expected 'first' (wrap around), got '%s'", s)
	}
	if cs.Selected() != 0 {
		t.Errorf("expected selected = 0 (wrap around), got %d", cs.Selected())
	}
}

// TestCompletionStatePrevSuggestion tests cycling backward through suggestions.
func TestCompletionStatePrevSuggestion(t *testing.T) {
	cs := NewCompletionState()
	suggestions := []string{"first", "second", "third"}
	cs.Activate(suggestions, "", 0, 0)

	// First call should wrap to last (index 2)
	s := cs.PrevSuggestion()
	if s != "third" {
		t.Errorf("expected 'third', got '%s'", s)
	}
	if cs.Selected() != 2 {
		t.Errorf("expected selected = 2, got %d", cs.Selected())
	}

	// Second call should select index 1
	s = cs.PrevSuggestion()
	if s != "second" {
		t.Errorf("expected 'second', got '%s'", s)
	}
	if cs.Selected() != 1 {
		t.Errorf("expected selected = 1, got %d", cs.Selected())
	}

	// Third call should select index 0
	s = cs.PrevSuggestion()
	if s != "first" {
		t.Errorf("expected 'first', got '%s'", s)
	}
	if cs.Selected() != 0 {
		t.Errorf("expected selected = 0, got %d", cs.Selected())
	}

	// Fourth call should wrap to last (index 2)
	s = cs.PrevSuggestion()
	if s != "third" {
		t.Errorf("expected 'third' (wrap around), got '%s'", s)
	}
	if cs.Selected() != 2 {
		t.Errorf("expected selected = 2 (wrap around), got %d", cs.Selected())
	}
}

// TestCompletionStateNextPrevInactive tests Next/Prev on inactive state.
func TestCompletionStateNextPrevInactive(t *testing.T) {
	cs := NewCompletionState()

	// Should return empty string when inactive
	if s := cs.NextSuggestion(); s != "" {
		t.Errorf("NextSuggestion on inactive state should return empty, got '%s'", s)
	}
	if s := cs.PrevSuggestion(); s != "" {
		t.Errorf("PrevSuggestion on inactive state should return empty, got '%s'", s)
	}
}

// TestCompletionStateNextPrevEmpty tests Next/Prev with empty suggestions.
func TestCompletionStateNextPrevEmpty(t *testing.T) {
	cs := NewCompletionState()
	cs.Activate([]string{}, "", 0, 0)

	if s := cs.NextSuggestion(); s != "" {
		t.Errorf("NextSuggestion with empty suggestions should return empty, got '%s'", s)
	}
	if s := cs.PrevSuggestion(); s != "" {
		t.Errorf("PrevSuggestion with empty suggestions should return empty, got '%s'", s)
	}
}

// TestCompletionStateCurrentSuggestion tests the CurrentSuggestion method.
func TestCompletionStateCurrentSuggestion(t *testing.T) {
	cs := NewCompletionState()

	// Inactive state
	if s := cs.CurrentSuggestion(); s != "" {
		t.Errorf("CurrentSuggestion on inactive state should return empty, got '%s'", s)
	}

	// Active but no selection yet
	cs.Activate([]string{"foo", "bar"}, "", 0, 0)
	if s := cs.CurrentSuggestion(); s != "" {
		t.Errorf("CurrentSuggestion with no selection should return empty, got '%s'", s)
	}

	// After selecting
	cs.NextSuggestion()
	if s := cs.CurrentSuggestion(); s != "foo" {
		t.Errorf("CurrentSuggestion should be 'foo', got '%s'", s)
	}

	cs.NextSuggestion()
	if s := cs.CurrentSuggestion(); s != "bar" {
		t.Errorf("CurrentSuggestion should be 'bar', got '%s'", s)
	}
}

// TestCompletionStateHasMultipleCompletions tests the HasMultipleCompletions method.
func TestCompletionStateHasMultipleCompletions(t *testing.T) {
	cs := NewCompletionState()

	// No suggestions
	if cs.HasMultipleCompletions() {
		t.Error("HasMultipleCompletions should be false with no suggestions")
	}

	// One suggestion
	cs.Activate([]string{"only"}, "", 0, 0)
	if cs.HasMultipleCompletions() {
		t.Error("HasMultipleCompletions should be false with one suggestion")
	}

	// Multiple suggestions
	cs.Reset()
	cs.Activate([]string{"one", "two"}, "", 0, 0)
	if !cs.HasMultipleCompletions() {
		t.Error("HasMultipleCompletions should be true with multiple suggestions")
	}
}

// TestCompletionStateIsVisible tests the IsVisible method.
func TestCompletionStateIsVisible(t *testing.T) {
	cs := NewCompletionState()

	// Inactive state
	if cs.IsVisible() {
		t.Error("IsVisible should be false when inactive")
	}

	// Active with single suggestion
	cs.Activate([]string{"only"}, "", 0, 0)
	if cs.IsVisible() {
		t.Error("IsVisible should be false with single suggestion")
	}

	// Active with multiple suggestions
	cs.Reset()
	cs.Activate([]string{"foo", "bar"}, "", 0, 0)
	if !cs.IsVisible() {
		t.Error("IsVisible should be true with multiple suggestions")
	}
}

// TestCompletionStateUpdateBoundaries tests the UpdateBoundaries method.
func TestCompletionStateUpdateBoundaries(t *testing.T) {
	cs := NewCompletionState()
	cs.Activate([]string{"foo"}, "f", 0, 1)

	cs.UpdateBoundaries("fo", 0, 2)

	if cs.Prefix() != "fo" {
		t.Errorf("prefix should be 'fo', got '%s'", cs.Prefix())
	}
	if cs.StartPos() != 0 {
		t.Errorf("startPos should be 0, got %d", cs.StartPos())
	}
	if cs.EndPos() != 2 {
		t.Errorf("endPos should be 2, got %d", cs.EndPos())
	}
}

// TestCompletionStateCancel tests the Cancel method.
func TestCompletionStateCancel(t *testing.T) {
	cs := NewCompletionState()
	cs.Activate([]string{"foo", "bar"}, "f", 0, 1)
	cs.SetOriginalText("original text")
	cs.NextSuggestion()

	original := cs.Cancel()

	if original != "original text" {
		t.Errorf("Cancel should return original text, got '%s'", original)
	}
	if cs.IsActive() {
		t.Error("CompletionState should be inactive after Cancel")
	}
}

// TestCompletionStateSingleSuggestion tests behavior with a single suggestion.
func TestCompletionStateSingleSuggestion(t *testing.T) {
	cs := NewCompletionState()
	cs.Activate([]string{"onlyone"}, "o", 0, 1)

	// Should cycle through the single suggestion
	s := cs.NextSuggestion()
	if s != "onlyone" {
		t.Errorf("expected 'onlyone', got '%s'", s)
	}

	s = cs.NextSuggestion()
	if s != "onlyone" {
		t.Errorf("expected 'onlyone' again, got '%s'", s)
	}

	s = cs.PrevSuggestion()
	if s != "onlyone" {
		t.Errorf("expected 'onlyone' with prev, got '%s'", s)
	}
}

// TestCompletionStateSetOriginalText tests SetOriginalText.
func TestCompletionStateSetOriginalText(t *testing.T) {
	cs := NewCompletionState()
	cs.Activate([]string{"foo", "bar"}, "", 0, 0)

	cs.SetOriginalText("my original text")

	if cs.OriginalText() != "my original text" {
		t.Errorf("OriginalText should be 'my original text', got '%s'", cs.OriginalText())
	}
}

// TestGetWordBoundary tests the GetWordBoundary function.
func TestGetWordBoundary(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		cursorPos int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "empty string",
			text:      "",
			cursorPos: 0,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "single word at start",
			text:      "hello",
			cursorPos: 0,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "single word in middle",
			text:      "hello",
			cursorPos: 2,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "single word at end",
			text:      "hello",
			cursorPos: 5,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "multiple words - first word",
			text:      "hello world",
			cursorPos: 3,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "multiple words - at space",
			text:      "hello world",
			cursorPos: 5,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "multiple words - second word start",
			text:      "hello world",
			cursorPos: 6,
			wantStart: 6,
			wantEnd:   11,
		},
		{
			name:      "multiple words - second word middle",
			text:      "hello world",
			cursorPos: 8,
			wantStart: 6,
			wantEnd:   11,
		},
		{
			name:      "cursor beyond text length",
			text:      "hello",
			cursorPos: 10,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "negative cursor position",
			text:      "hello",
			cursorPos: -1,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "unicode characters",
			text:      "héllo wörld",
			cursorPos: 3,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "multiple spaces",
			text:      "hello   world",
			cursorPos: 7,
			wantStart: 7,
			wantEnd:   7,
		},
		{
			name:      "leading space",
			text:      " hello",
			cursorPos: 0,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "trailing space",
			text:      "hello ",
			cursorPos: 6,
			wantStart: 6,
			wantEnd:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := GetWordBoundary(tt.text, tt.cursorPos)
			if gotStart != tt.wantStart {
				t.Errorf("GetWordBoundary() start = %d, want %d", gotStart, tt.wantStart)
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("GetWordBoundary() end = %d, want %d", gotEnd, tt.wantEnd)
			}
		})
	}
}

// TestApplySuggestion tests the ApplySuggestion function.
func TestApplySuggestion(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		suggestion    string
		startPos      int
		endPos        int
		wantText      string
		wantCursorPos int
	}{
		{
			name:          "replace word at start",
			text:          "hel world",
			suggestion:    "hello",
			startPos:      0,
			endPos:        3,
			wantText:      "hello world",
			wantCursorPos: 5,
		},
		{
			name:          "replace word in middle",
			text:          "hello wor today",
			suggestion:    "world",
			startPos:      6,
			endPos:        9,
			wantText:      "hello world today",
			wantCursorPos: 11,
		},
		{
			name:          "replace word at end",
			text:          "hello wor",
			suggestion:    "world",
			startPos:      6,
			endPos:        9,
			wantText:      "hello world",
			wantCursorPos: 11,
		},
		{
			name:          "insert at empty position",
			text:          "hello  world",
			suggestion:    "big",
			startPos:      6,
			endPos:        6,
			wantText:      "hello big world",
			wantCursorPos: 9,
		},
		{
			name:          "empty text",
			text:          "",
			suggestion:    "hello",
			startPos:      0,
			endPos:        0,
			wantText:      "hello",
			wantCursorPos: 5,
		},
		{
			name:          "replace entire text",
			text:          "old",
			suggestion:    "new",
			startPos:      0,
			endPos:        3,
			wantText:      "new",
			wantCursorPos: 3,
		},
		{
			name:          "startPos beyond text length",
			text:          "hello",
			suggestion:    "world",
			startPos:      10,
			endPos:        10,
			wantText:      "helloworld",
			wantCursorPos: 10,
		},
		{
			name:          "endPos beyond text length",
			text:          "hello",
			suggestion:    "world",
			startPos:      3,
			endPos:        10,
			wantText:      "helworld",
			wantCursorPos: 8,
		},
		{
			name:          "startPos greater than endPos",
			text:          "hello",
			suggestion:    "world",
			startPos:      4,
			endPos:        2,
			wantText:      "heworldllo",
			wantCursorPos: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplySuggestion(tt.text, tt.suggestion, tt.startPos, tt.endPos)
			if result.NewText != tt.wantText {
				t.Errorf("ApplySuggestion() NewText = %q, want %q", result.NewText, tt.wantText)
			}
			if result.NewCursorPos != tt.wantCursorPos {
				t.Errorf("ApplySuggestion() NewCursorPos = %d, want %d", result.NewCursorPos, tt.wantCursorPos)
			}
		})
	}
}

// MockCompletionProvider implements CompletionProvider for testing.
type MockCompletionProvider struct {
	completions []string
	helpInfo    string
}

func (m *MockCompletionProvider) GetCompletions(line string, pos int) []string {
	return m.completions
}

func (m *MockCompletionProvider) GetHelpInfo(line string, pos int) string {
	return m.helpInfo
}

// TestCompletionProviderInterface tests that MockCompletionProvider implements CompletionProvider.
func TestCompletionProviderInterface(t *testing.T) {
	var _ CompletionProvider = (*MockCompletionProvider)(nil)

	provider := &MockCompletionProvider{
		completions: []string{"foo", "bar", "baz"},
		helpInfo:    "Test help info",
	}

	completions := provider.GetCompletions("test", 4)
	if len(completions) != 3 {
		t.Errorf("expected 3 completions, got %d", len(completions))
	}

	helpInfo := provider.GetHelpInfo("test", 4)
	if helpInfo != "Test help info" {
		t.Errorf("expected 'Test help info', got '%s'", helpInfo)
	}
}

// TestInfoPanelContentInterface tests that CompletionState and HelpContent implement InfoPanelContent.
func TestInfoPanelContentInterface(t *testing.T) {
	var _ InfoPanelContent = (*CompletionState)(nil)
	var _ InfoPanelContent = (*HelpContent)(nil)
}

// TestCompletionStateIsInteractive tests the IsInteractive method.
func TestCompletionStateIsInteractive(t *testing.T) {
	cs := NewCompletionState()

	if !cs.IsInteractive() {
		t.Error("CompletionState should always be interactive")
	}
}

// TestCompletionStateHandleAction tests the HandleAction method.
func TestCompletionStateHandleAction(t *testing.T) {
	t.Run("inactive state ignores actions", func(t *testing.T) {
		cs := NewCompletionState()

		_, handled := cs.HandleAction(ActionComplete)
		if handled {
			t.Error("HandleAction should not handle actions when inactive")
		}
	})

	t.Run("ActionComplete cycles forward", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo", "bar", "baz"}, "", 0, 0)

		_, handled := cs.HandleAction(ActionComplete)
		if !handled {
			t.Error("HandleAction should handle ActionComplete")
		}
		if cs.Selected() != 0 {
			t.Errorf("expected selected = 0, got %d", cs.Selected())
		}

		cs.HandleAction(ActionComplete)
		if cs.Selected() != 1 {
			t.Errorf("expected selected = 1, got %d", cs.Selected())
		}
	})

	t.Run("ActionCompleteBackward cycles backward", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo", "bar", "baz"}, "", 0, 0)

		_, handled := cs.HandleAction(ActionCompleteBackward)
		if !handled {
			t.Error("HandleAction should handle ActionCompleteBackward")
		}
		if cs.Selected() != 2 {
			t.Errorf("expected selected = 2, got %d", cs.Selected())
		}
	})

	t.Run("ActionCancel resets state", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo", "bar"}, "", 0, 0)
		cs.NextSuggestion()

		_, handled := cs.HandleAction(ActionCancel)
		if !handled {
			t.Error("HandleAction should handle ActionCancel")
		}
		if cs.IsActive() {
			t.Error("CompletionState should be inactive after ActionCancel")
		}
	})

	t.Run("unhandled actions return false", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo"}, "", 0, 0)

		_, handled := cs.HandleAction(ActionSubmit)
		if handled {
			t.Error("HandleAction should not handle ActionSubmit")
		}

		_, handled = cs.HandleAction(ActionCharacterForward)
		if handled {
			t.Error("HandleAction should not handle ActionCharacterForward")
		}
	})
}

// TestCompletionStateRender tests the Render method.
func TestCompletionStateRender(t *testing.T) {
	t.Run("inactive state returns empty", func(t *testing.T) {
		cs := NewCompletionState()

		result := cs.Render(80)
		if result != "" {
			t.Errorf("Render should return empty for inactive state, got %q", result)
		}
	})

	t.Run("single suggestion returns empty", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"only"}, "", 0, 0)

		result := cs.Render(80)
		if result != "" {
			t.Errorf("Render should return empty for single suggestion, got %q", result)
		}
	})

	t.Run("multiple suggestions renders list", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo", "bar", "baz"}, "", 0, 0)

		result := cs.Render(80)
		if !strings.Contains(result, "foo") {
			t.Error("Render should contain 'foo'")
		}
		if !strings.Contains(result, "bar") {
			t.Error("Render should contain 'bar'")
		}
		if !strings.Contains(result, "baz") {
			t.Error("Render should contain 'baz'")
		}
	})

	t.Run("selected suggestion is marked", func(t *testing.T) {
		cs := NewCompletionState()
		cs.Activate([]string{"foo", "bar"}, "", 0, 0)
		cs.NextSuggestion() // Select first

		result := cs.Render(80)
		if !strings.Contains(result, "> foo") {
			t.Errorf("Render should mark selected suggestion with '>', got %q", result)
		}
		if !strings.Contains(result, "  bar") {
			t.Errorf("Render should indent unselected suggestion, got %q", result)
		}
	})
}

// TestHelpContent tests the HelpContent type.
func TestHelpContent(t *testing.T) {
	t.Run("NewHelpContent creates with text", func(t *testing.T) {
		h := NewHelpContent("test help")
		if h.Text() != "test help" {
			t.Errorf("expected 'test help', got '%s'", h.Text())
		}
	})

	t.Run("empty help is not visible", func(t *testing.T) {
		h := NewHelpContent("")
		if h.IsVisible() {
			t.Error("empty HelpContent should not be visible")
		}
	})

	t.Run("non-empty help is visible", func(t *testing.T) {
		h := NewHelpContent("some help")
		if !h.IsVisible() {
			t.Error("non-empty HelpContent should be visible")
		}
	})

	t.Run("Render returns text when visible", func(t *testing.T) {
		h := NewHelpContent("help text")
		result := h.Render(80)
		if result != "help text" {
			t.Errorf("expected 'help text', got '%s'", result)
		}
	})

	t.Run("Render returns empty when not visible", func(t *testing.T) {
		h := NewHelpContent("")
		result := h.Render(80)
		if result != "" {
			t.Errorf("expected empty, got '%s'", result)
		}
	})

	t.Run("IsInteractive returns false", func(t *testing.T) {
		h := NewHelpContent("help")
		if h.IsInteractive() {
			t.Error("HelpContent should not be interactive")
		}
	})

	t.Run("HandleAction never handles", func(t *testing.T) {
		h := NewHelpContent("help")

		_, handled := h.HandleAction(ActionComplete)
		if handled {
			t.Error("HelpContent should not handle any actions")
		}

		_, handled = h.HandleAction(ActionCancel)
		if handled {
			t.Error("HelpContent should not handle any actions")
		}
	})

	t.Run("SetText updates text", func(t *testing.T) {
		h := NewHelpContent("old")
		h.SetText("new")
		if h.Text() != "new" {
			t.Errorf("expected 'new', got '%s'", h.Text())
		}
	})

	t.Run("Clear removes text", func(t *testing.T) {
		h := NewHelpContent("help")
		h.Clear()
		if h.Text() != "" {
			t.Errorf("expected empty, got '%s'", h.Text())
		}
		if h.IsVisible() {
			t.Error("cleared HelpContent should not be visible")
		}
	})
}
