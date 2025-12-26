// Package input provides input handling for the gsh REPL.
package input

import (
	"strings"

	"github.com/atinylittleshell/gsh/internal/repl/render"
	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

// RenderConfig holds styling configuration for rendering input components.
type RenderConfig struct {
	// PromptStyle is the style applied to the prompt string.
	PromptStyle lipgloss.Style

	// TextStyle is the style applied to the input text.
	TextStyle lipgloss.Style

	// CursorStyle is the style applied to the cursor character.
	CursorStyle lipgloss.Style

	// PredictionStyle is the style for ghost text predictions.
	PredictionStyle lipgloss.Style

	// InfoPanelStyle is the style for the info panel border/container.
	InfoPanelStyle lipgloss.Style

	// CompletionPanelStyle is the style for the completion panel border/container.
	CompletionPanelStyle lipgloss.Style

	// SelectedStyle is the style for selected items in lists.
	SelectedStyle lipgloss.Style
}

// DefaultRenderConfig returns a RenderConfig with sensible default styles.
func DefaultRenderConfig() RenderConfig {
	return RenderConfig{
		PromptStyle:     lipgloss.NewStyle(),
		TextStyle:       lipgloss.NewStyle(),
		CursorStyle:     lipgloss.NewStyle().Reverse(true),
		PredictionStyle: lipgloss.NewStyle().Foreground(render.ColorGray),
		InfoPanelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(render.ColorYellow),
		CompletionPanelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(render.ColorYellow),
		SelectedStyle: lipgloss.NewStyle().Bold(true),
	}
}

// Renderer handles rendering of input components.
type Renderer struct {
	config RenderConfig
	width  int
}

// NewRenderer creates a new Renderer with the given configuration.
func NewRenderer(config RenderConfig) *Renderer {
	return &Renderer{
		config: config,
		width:  80, // default width
	}
}

// SetWidth sets the terminal width for rendering.
func (r *Renderer) SetWidth(width int) {
	if width > 0 {
		r.width = width
	}
}

// Width returns the current terminal width.
func (r *Renderer) Width() int {
	return r.width
}

// Config returns the current render configuration.
func (r *Renderer) Config() RenderConfig {
	return r.config
}

// SetConfig updates the render configuration.
func (r *Renderer) SetConfig(config RenderConfig) {
	r.config = config
}

// RenderInputLine renders the input line with prompt, text, cursor, and prediction.
// It returns the rendered string for the input line.
func (r *Renderer) RenderInputLine(prompt string, buffer *Buffer, prediction string, focused bool) string {
	// Render the prompt
	result := r.config.PromptStyle.Render(prompt)

	text := buffer.Text()
	pos := buffer.Pos()

	runes := []rune(text)

	// Ensure position is within bounds
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}

	// Text before cursor
	if pos > 0 {
		result += r.config.TextStyle.Inline(true).Render(string(runes[:pos]))
	}

	// Cursor and text after cursor
	if pos < len(runes) {
		// Cursor is on a character
		cursorChar := string(runes[pos])
		if focused {
			result += r.config.CursorStyle.Render(cursorChar)
		} else {
			result += r.config.TextStyle.Inline(true).Render(cursorChar)
		}

		// Text after cursor
		if pos+1 < len(runes) {
			result += r.config.TextStyle.Inline(true).Render(string(runes[pos+1:]))
		}

		// Prediction ghost text (only shown when cursor is after the text)
		// In this case, there's no prediction to show since cursor is in the middle
	} else {
		// Cursor is at end of text
		// Check if we have a prediction that extends the input
		if prediction != "" && strings.HasPrefix(prediction, text) && len(prediction) > len(text) {
			// Show cursor on first prediction character
			predictionRunes := []rune(prediction)
			if focused {
				// Cursor on first prediction character with prediction style
				result += r.config.CursorStyle.
					Foreground(r.config.PredictionStyle.GetForeground()).
					Render(string(predictionRunes[len(runes)]))
			} else {
				result += r.config.PredictionStyle.Render(string(predictionRunes[len(runes)]))
			}

			// Rest of prediction as ghost text
			if len(predictionRunes) > len(runes)+1 {
				result += r.config.PredictionStyle.Render(string(predictionRunes[len(runes)+1:]))
			}
		} else {
			// No prediction, just show cursor on space
			if focused {
				result += r.config.CursorStyle.Render(" ")
			} else {
				result += " "
			}
		}
	}

	return result
}

// RenderCompletionBox renders the completion suggestions in a box format.
// maxVisible controls how many items are visible at once (scrolling window).
func (r *Renderer) RenderCompletionBox(cs *CompletionState, maxVisible int) string {
	if !cs.IsVisible() {
		return ""
	}

	if maxVisible <= 0 {
		maxVisible = 4 // default
	}

	suggestions := cs.Suggestions()
	totalItems := len(suggestions)
	if totalItems == 0 {
		return ""
	}

	selected := cs.Selected()
	if selected < 0 {
		selected = 0
	}

	// Calculate visible window
	startIdx, endIdx := calculateVisibleWindow(selected, totalItems, maxVisible)

	var content strings.Builder

	for i := startIdx; i < endIdx; i++ {
		if i > startIdx {
			content.WriteString("\n")
		}

		suggestion := suggestions[i]
		var prefix string

		// Position within visible window
		posInWindow := i - startIdx

		// Scroll indicators
		if posInWindow == 0 && startIdx > 0 {
			// First line with "more above" indicator
			prefix = formatScrollIndicator("↑", startIdx)
		} else if posInWindow == maxVisible-1 && endIdx < totalItems {
			// Last line with "more below" indicator
			prefix = formatScrollIndicator("↓", totalItems-endIdx)
		} else {
			// Regular line with spacing to align with indicators
			prefix = "     "
		}

		// Selection indicator
		if i == cs.Selected() {
			prefix += "> "
			content.WriteString(prefix)
			content.WriteString(r.config.SelectedStyle.Render(suggestion))
		} else {
			prefix += "  "
			content.WriteString(prefix)
			content.WriteString(suggestion)
		}
	}

	return r.config.CompletionPanelStyle.
		Width(maxInt(1, r.width-2)).
		Render(content.String())
}

// RenderInfoPanel renders an info panel with the given content.
func (r *Renderer) RenderInfoPanel(content InfoPanelContent) string {
	if content == nil || !content.IsVisible() {
		return ""
	}

	rendered := content.Render(maxInt(1, r.width-4)) // Account for border
	if rendered == "" {
		return ""
	}

	return r.config.InfoPanelStyle.
		Width(maxInt(1, r.width-2)).
		Render(rendered)
}

// RenderHelpBox renders help text in an info panel.
func (r *Renderer) RenderHelpBox(text string) string {
	if text == "" {
		return ""
	}

	return r.config.InfoPanelStyle.
		Width(maxInt(1, r.width-2)).
		Render(text)
}

// RenderFullView renders the complete input view including:
// - Input line with prompt, text, cursor, and prediction
// - Completion box (if active)
// - Info/help panel (if available)
func (r *Renderer) RenderFullView(
	prompt string,
	buffer *Buffer,
	prediction string,
	focused bool,
	completion *CompletionState,
	infoContent InfoPanelContent,
	minHeight int,
) string {
	var result strings.Builder

	// Start with carriage return and clear line to ensure we start at column 0
	// This handles cases where log output may have left the cursor mid-line
	result.WriteString("\r\033[K")

	// Render input line
	result.WriteString(r.RenderInputLine(prompt, buffer, prediction, focused))

	// Render completion box if active
	if completion != nil && completion.IsVisible() {
		result.WriteString("\n")
		result.WriteString(r.RenderCompletionBox(completion, 4))
	}

	// Render info panel content
	if infoContent != nil && infoContent.IsVisible() {
		result.WriteString("\n")
		result.WriteString(r.RenderInfoPanel(infoContent))
	}

	// Ensure minimum height
	output := result.String()
	numLines := strings.Count(output, "\n")
	if numLines < minHeight {
		output += strings.Repeat("\n", minHeight-numLines)
	}

	return output
}

// GetPredictionSuffix returns the portion of the prediction that extends beyond
// the current input text. Returns empty string if no valid prediction.
func GetPredictionSuffix(text, prediction string) string {
	if prediction == "" || !strings.HasPrefix(prediction, text) {
		return ""
	}
	if len(prediction) <= len(text) {
		return ""
	}
	return prediction[len(text):]
}

// CalculateCursorPosition calculates the visual cursor position in the rendered line.
// This accounts for the prompt width and any multi-width characters.
func CalculateCursorPosition(prompt string, text string, cursorPos int) int {
	promptWidth := uniseg.StringWidth(prompt)
	runes := []rune(text)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}
	textBeforeCursor := string(runes[:cursorPos])
	textWidth := uniseg.StringWidth(textBeforeCursor)
	return promptWidth + textWidth
}

// calculateVisibleWindow determines the start and end indices for a scrolling window.
func calculateVisibleWindow(selected, total, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}

	// Try to keep selection roughly in the middle
	if selected < 2 {
		start = 0
	} else if selected >= total-2 {
		start = total - maxVisible
	} else {
		start = selected - 1
	}

	end = start + maxVisible

	// Ensure bounds
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	return start, end
}

// formatScrollIndicator formats a scroll indicator with count.
func formatScrollIndicator(arrow string, count int) string {
	return arrow + " " + padLeft(itoa(count), 3)
}

// padLeft pads a string on the left with spaces to reach the desired width.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// itoa converts an integer to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// maxInt returns the larger of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
