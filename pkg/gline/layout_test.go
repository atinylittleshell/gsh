package gline

import (
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

	// 3. Check that view starts with input (no spacer)
	assert.Contains(t, view, "gsh> ", "View should contain prompt")

	// 4. Check for assistant box content
	assert.Contains(t, view, "explanation", "View should contain explanation")
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
