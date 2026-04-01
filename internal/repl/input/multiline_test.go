package input

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kunchenguid/gsh/internal/script/interpreter"
	"github.com/muesli/termenv"
)

type countingPredictionProvider struct {
	mu     sync.Mutex
	inputs []string
}

func (p *countingPredictionProvider) Predict(ctx context.Context, input string, trigger interpreter.PredictTrigger, existingPrediction string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inputs = append(p.inputs, input)
	return "", nil
}

func (p *countingPredictionProvider) Inputs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.inputs))
	copy(out, p.inputs)
	return out
}

func TestIsInputComplete(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Complete: empty/whitespace
		{"empty string", "", true},
		{"whitespace only", "   ", true},

		// Complete: simple commands
		{"simple command", "ls", true},
		{"command with args", "echo hello", true},
		{"multiple commands", "foo; bar", true},
		{"command with newline", "echo hello\n", true},

		// Incomplete: unclosed quotes
		{"unclosed double quote", `echo "test`, false},
		{"unclosed single quote", "echo 'test", false},

		// Incomplete: heredoc
		{"heredoc not terminated", "cat <<EOF", false},
		{"heredoc terminated", "cat <<EOF\nhello\nEOF", true},

		// Incomplete: trailing operators
		{"trailing pipe", "ls |", false},
		{"trailing and", "foo &&", false},
		{"trailing or", "foo ||", false},

		// Incomplete: control structures
		{"if without fi", "if true; then", false},
		{"complete if block", "if true; then\necho hi\nfi", true},
		{"while without done", "while true; do", false},
		{"complete while block", "while true; do\necho hi\ndone", true},

		// Trailing backslash: mvdan/sh treats \ at EOF as a literal backslash
		// (complete command), not as a line continuation. This differs from
		// bash but is consistent with the parser's behavior.
		{"trailing backslash", "echo hello \\", true},
		{"escaped backslash", "echo \\\\", true},

		// Syntax errors (not incomplete — should submit and let shell report)
		{"stray close paren", "foo )", true},

		// gsh agent commands (always complete)
		{"agent command", "#hello", true},
		{"agent command with space", "# send a message", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInputComplete(tt.input)
			if result != tt.expected {
				t.Errorf("IsInputComplete(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSubmitIncompleteInput(t *testing.T) {
	m := New(Config{})
	m.SetValue(`echo "test`)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Should NOT submit — input is incomplete
	if m.result.Type != ResultNone {
		t.Errorf("expected ResultNone for incomplete input, got %v", m.result.Type)
	}
	// Should not quit
	if cmd != nil {
		t.Error("expected no quit command for incomplete input")
	}
	// Should have inserted a newline into the buffer
	text := m.buffer.Text()
	if text != "echo \"test\n" {
		t.Errorf("expected buffer to contain newline, got %q", text)
	}
}

func TestSubmitCompleteInput(t *testing.T) {
	m := New(Config{})
	m.SetValue("echo hello")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Should submit — input is complete
	if m.result.Type != ResultSubmit {
		t.Errorf("expected ResultSubmit for complete input, got %v", m.result.Type)
	}
	if m.result.Value != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", m.result.Value)
	}
	if cmd == nil {
		t.Error("expected quit command for complete input")
	}
}

func TestSubmitMultiLineCompleteInput(t *testing.T) {
	m := New(Config{})
	// Simulate: user typed "if true; then", got continuation, typed "echo hi", got continuation, typed "fi"
	m.SetValue("if true; then\necho hi\nfi")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Should submit — multi-line input is now complete
	if m.result.Type != ResultSubmit {
		t.Errorf("expected ResultSubmit for complete multi-line input, got %v", m.result.Type)
	}
	if m.result.Value != "if true; then\necho hi\nfi" {
		t.Errorf("expected complete if block, got %q", m.result.Value)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestInsertNewlineAlwaysInsertsNewline(t *testing.T) {
	// Alt+Enter should always insert a newline, even for complete input
	m := New(Config{})
	m.SetValue("ls")

	msg := tea.KeyMsg{Type: tea.KeyEnter, Alt: true}
	newModel, cmd := m.Update(msg)
	m = newModel.(Model)

	// Should NOT submit
	if m.result.Type != ResultNone {
		t.Errorf("expected ResultNone for alt+enter, got %v", m.result.Type)
	}
	if cmd != nil {
		// cmd may be a prediction request, that's ok — just ensure it's not tea.Quit
		// We check result.Type instead
	}
	// Should have inserted a newline
	text := m.buffer.Text()
	if text != "ls\n" {
		t.Errorf("expected 'ls\\n', got %q", text)
	}
}

func TestContinuationPromptDefault(t *testing.T) {
	m := New(Config{})
	if m.ContinuationPrompt() != "> " {
		t.Errorf("expected default continuation prompt '> ', got %q", m.ContinuationPrompt())
	}
}

func TestContinuationPromptCustom(t *testing.T) {
	m := New(Config{ContinuationPrompt: "... "})
	if m.ContinuationPrompt() != "... " {
		t.Errorf("expected custom continuation prompt '... ', got %q", m.ContinuationPrompt())
	}
}

func TestRenderMultiLineInput(t *testing.T) {
	config := DefaultRenderConfig()
	// Use unstyled config for predictable output
	config.PromptStyle = lipgloss.NewStyle()
	config.TextStyle = lipgloss.NewStyle()
	config.CursorStyle = lipgloss.NewStyle()
	config.PredictionStyle = lipgloss.NewStyle()

	renderer := NewRenderer(config, NewHighlighter(nil, nil, nil))
	renderer.SetWidth(80)
	renderer.SetContinuationPrompt("> ")

	// Multi-line input: "echo \"test\nsecond line"
	buffer := NewBufferWithText("echo \"test\nsecond line")

	result := renderer.RenderInputLine("$ ", buffer, "", false)
	lines := strings.Split(result, "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), result)
	}

	// First line should start with the main prompt
	if !strings.Contains(lines[0], "$ ") {
		t.Errorf("first line should contain main prompt '$ ', got %q", lines[0])
	}
	if !strings.Contains(lines[0], "echo") {
		t.Errorf("first line should contain 'echo', got %q", lines[0])
	}

	// Second line should start with continuation prompt
	if !strings.Contains(lines[1], "> ") {
		t.Errorf("second line should contain continuation prompt '> ', got %q", lines[1])
	}
	if !strings.Contains(lines[1], "second line") {
		t.Errorf("second line should contain 'second line', got %q", lines[1])
	}
}

func TestPredictionWithNewlineDiscarded(t *testing.T) {
	// When a prediction suffix contains a newline, it should be discarded
	// to prevent the cursor from landing on an invisible newline character.
	config := DefaultRenderConfig()
	config.PromptStyle = lipgloss.NewStyle()
	config.CursorStyle = lipgloss.NewStyle().Reverse(true)

	renderer := NewRenderer(config, NewHighlighter(nil, nil, nil))
	renderer.SetWidth(80)

	// Single-line input with a prediction that contains a newline
	buffer := NewBufferWithText("echo \"test")
	prediction := "echo \"test\nmulti line input\""

	result := renderer.RenderInputLine("$ ", buffer, prediction, true)

	// The prediction suffix should NOT appear (it contains \n)
	if strings.Contains(result, "multi line") {
		t.Errorf("prediction with newline should be discarded, got %q", result)
	}
	// The result should end with a space (cursor placeholder at end of text)
	if !strings.HasSuffix(result, " ") {
		t.Errorf("should have cursor space at end, got %q", result)
	}
}

func TestRenderMultiLineNoPrediction(t *testing.T) {
	// Predictions should not appear for multi-line input
	config := DefaultRenderConfig()
	config.PromptStyle = lipgloss.NewStyle()
	config.TextStyle = lipgloss.NewStyle()
	config.CursorStyle = lipgloss.NewStyle().Reverse(true)
	config.PredictionStyle = lipgloss.NewStyle()

	renderer := NewRenderer(config, NewHighlighter(nil, nil, nil))
	renderer.SetWidth(80)
	renderer.SetContinuationPrompt("> ")

	buffer := NewBufferWithText("echo \"test\nsecond line")

	// Pass a prediction that starts with the full text
	prediction := "echo \"test\nsecond line prediction"
	result := renderer.RenderInputLine("$ ", buffer, prediction, true)

	// The prediction suffix should NOT appear in multi-line mode
	if strings.Contains(result, "prediction") {
		t.Errorf("predictions should not appear in multi-line mode, got %q", result)
	}
}

func TestMultiLineInputDoesNotStartHiddenPrediction(t *testing.T) {
	provider := &countingPredictionProvider{}
	predictionState := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	// Simulate normal single-line editing that marks prediction state dirty.
	predictionState.OnInputChanged(`echo "test`)

	m := New(Config{
		PredictionState: predictionState,
	})
	m.SetValue(`echo "test`)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	time.Sleep(30 * time.Millisecond)

	for _, input := range provider.Inputs() {
		if input == "" {
			t.Fatalf("multiline editing should cancel prediction without starting an empty-input prediction")
		}
	}
}

func TestSplitHighlightedByNewlines(t *testing.T) {
	// Force ANSI output so highlighting produces escape codes in tests
	oldProfile := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(oldProfile)

	h := NewHighlighter(nil, nil, nil)
	text := "echo \"test\nanother line"
	highlighted := h.Highlight(text)
	t.Logf("Original:    %q", text)
	t.Logf("Highlighted: %q", highlighted)

	lines := splitHighlightedByNewlines(text, highlighted)
	t.Logf("Split into %d lines:", len(lines))
	for i, line := range lines {
		t.Logf("  line %d: %q", i, line)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	// Second line should contain ANSI codes (string highlighting carried over)
	if !strings.Contains(lines[1], "\x1b[") {
		t.Errorf("second line should have ANSI highlighting, got %q", lines[1])
	}
}

func TestRenderMultiLineSyntaxHighlightingCrossLine(t *testing.T) {
	// Force ANSI output so highlighting produces escape codes in tests
	oldProfile := lipgloss.DefaultRenderer().ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(oldProfile)

	config := DefaultRenderConfig()
	config.PromptStyle = lipgloss.NewStyle()
	config.TextStyle = lipgloss.NewStyle()
	config.CursorStyle = lipgloss.NewStyle()
	config.PredictionStyle = lipgloss.NewStyle()

	renderer := NewRenderer(config, NewHighlighter(nil, nil, nil))
	renderer.SetWidth(80)
	renderer.SetContinuationPrompt("> ")

	// "echo \"test\nanother line" — the second line is inside the unclosed string
	buffer := NewBufferWithText("echo \"test\nanother line")
	result := renderer.RenderInputLine("$ ", buffer, "", false)

	resultLines := strings.Split(result, "\n")
	if len(resultLines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(resultLines))
	}

	// The second line should have ANSI escape codes (syntax highlighting)
	// If highlighting is working correctly across lines, "another line" will
	// be colored as a string (it's inside unclosed quotes).
	// A broken implementation would have no ANSI codes on line 2.
	if !strings.Contains(resultLines[1], "\x1b[") {
		t.Errorf("second line should have ANSI highlighting (inside unclosed string), got %q", resultLines[1])
	}
}

func TestRenderSingleLineUnchanged(t *testing.T) {
	config := DefaultRenderConfig()
	renderer := NewRenderer(config, NewHighlighter(nil, nil, nil))
	renderer.SetWidth(80)
	renderer.SetContinuationPrompt("> ")

	buffer := NewBufferWithText("echo hello")

	result := renderer.RenderInputLine("$ ", buffer, "", false)

	// Single-line should not contain continuation prompt
	if strings.Contains(result, "> ") {
		t.Errorf("single-line input should not contain continuation prompt, got %q", result)
	}
}

func TestInsertNewlineKeyBinding(t *testing.T) {
	km := DefaultKeyMap()
	msg := tea.KeyMsg{Type: tea.KeyEnter, Alt: true}
	action := km.Lookup(msg)
	if action != ActionInsertNewline {
		t.Errorf("expected ActionInsertNewline for alt+enter, got %v", action)
	}
}
