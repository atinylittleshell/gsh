package gline

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestViewLayout(t *testing.T) {
	logger := zap.NewNop()
	options := NewOptions()
	options.AssistantHeight = 5

	model := initialModel("gsh> ", []string{}, "explanation", nil, nil, nil, logger, options)

	// Simulate window resize to set height
	termHeight := 20
	model.height = termHeight
	model.textInput.Width = 80

	view := model.View()

	// Assertions

	// 1. The prompt should be present
	assert.Contains(t, view, "gsh> ")

	// 2. The explanation should be present (inside assistant box)
	assert.Contains(t, view, "explanation")

	// 3. The view should contain padding (newlines) to fill the screen
	// We expect the height of the returned string to be roughly termHeight or slightly more
	// because View returns the full string to print.
	// Note: bubbletea counts actual lines.

	// Let's count newlines in the output.
	lineCount := strings.Count(view, "\n")

	// The view consists of:
	// Spacer (X lines)
	// Input (1 line usually)
	// Assistant Box (AssistantHeight lines)

	// Total should be around termHeight.
	// It might vary slightly due to how lipgloss or exact counting works,
	// but it should be at least termHeight - 1.

	assert.True(t, lineCount >= termHeight - 2, "View should return enough lines to fill the terminal. Got %d lines for %d term height", lineCount, termHeight)

	// 4. Check that spacer is at the top (starts with newlines)
	// If spacer > 0, it should start with \n
	if lineCount > options.AssistantHeight + 2 {
		assert.True(t, strings.HasPrefix(view, "\n"), "View should start with newlines (spacer)")
	}
}

func TestViewTruncation(t *testing.T) {
	logger := zap.NewNop()
	options := NewOptions()
	options.AssistantHeight = 3 // Small height

	// Long explanation
	longExplanation := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"

	model := initialModel("gsh> ", []string{}, longExplanation, nil, nil, nil, logger, options)
	model.height = 20
	model.textInput.Width = 80

	view := model.View()

	// Check that view contains truncated explanation
	// Since AssistantHeight is 3, and we have borders (2 lines), available content height is 1.
	// So only "Line 1" should be visible?
	// Or if borders are added to height, then yes.

	assert.Contains(t, view, "Line 1")
	assert.NotContains(t, view, "Line 4")
	assert.NotContains(t, view, "Line 5")
}
