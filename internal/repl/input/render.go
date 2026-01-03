// Package input provides input handling for the gsh REPL.
package input

import (
	"strings"

	"github.com/atinylittleshell/gsh/internal/repl/render"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	config      RenderConfig
	width       int
	highlighter *Highlighter
}

// NewRenderer creates a new Renderer with the given configuration.
func NewRenderer(config RenderConfig, h *Highlighter) *Renderer {
	if h == nil {
		h = NewHighlighter(nil, nil)
	}

	return &Renderer{
		config:      config,
		width:       80, // default width
		highlighter: h,
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
// It returns the rendered string for the input line, with automatic line wrapping
// when the content exceeds the terminal width.
func (r *Renderer) RenderInputLine(prompt string, buffer *Buffer, prediction string, focused bool) string {
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

	// Build the content parts for wrapping
	// Only consider the last line of the prompt for width calculation,
	// since multi-line prompts are common and earlier lines don't affect wrapping
	promptLastLine := prompt
	if lastNewline := strings.LastIndex(prompt, "\n"); lastNewline != -1 {
		promptLastLine = prompt[lastNewline+1:]
	}
	promptWidth := ansi.StringWidth(promptLastLine)

	// Calculate what text to render (including prediction suffix if applicable)
	var predictionSuffix string
	if pos >= len(runes) && prediction != "" && strings.HasPrefix(prediction, text) && len(prediction) > len(text) {
		predictionRunes := []rune(prediction)
		predictionSuffix = string(predictionRunes[len(runes):])
	}

	// Use the wrapping renderer
	return r.renderWrappedInputLine(prompt, promptWidth, text, pos, predictionSuffix, focused)
}

// renderWrappedInputLine renders the input line with wrapping support.
// It properly highlights the full text first, then handles wrapping and cursor positioning.
func (rndr *Renderer) renderWrappedInputLine(prompt string, promptWidth int, text string, cursorPos int, predictionSuffix string, focused bool) string {
	runes := []rune(text)
	hasCursorAtEnd := cursorPos >= len(runes)

	availableWidth := rndr.width
	if availableWidth <= 0 {
		availableWidth = 80
	}

	// Build the complete rendered content with proper highlighting
	var result strings.Builder

	// Start with the styled prompt
	styledPrompt := rndr.config.PromptStyle.Render(prompt)
	result.WriteString(styledPrompt)

	currentWidth := promptWidth

	if len(runes) > 0 {
		// Highlight the full text first for proper syntax coloring context
		// Then split the highlighted text at the cursor position
		if cursorPos < len(runes) {
			// Cursor is in the middle of the text
			// We need to highlight the full text, then render with cursor inserted
			currentWidth = rndr.appendTextWithWrappingAndCursor(&result, text, cursorPos, currentWidth, availableWidth, focused)
		} else {
			// Cursor is at the end, highlight and render all text
			currentWidth = rndr.appendTextWithWrapping(&result, text, currentWidth, availableWidth)
		}
	}

	// Handle cursor at end of text
	if hasCursorAtEnd {
		predictionRunes := []rune(predictionSuffix)
		if len(predictionRunes) > 0 {
			// Cursor on first prediction character
			firstPredChar := string(predictionRunes[0])
			firstPredWidth := ansi.StringWidth(firstPredChar)

			if currentWidth+firstPredWidth > availableWidth {
				result.WriteString("\n")
				currentWidth = 0
			}

			if focused {
				result.WriteString(rndr.config.CursorStyle.
					Foreground(rndr.config.PredictionStyle.GetForeground()).
					Render(firstPredChar))
			} else {
				result.WriteString(rndr.config.PredictionStyle.Render(firstPredChar))
			}
			currentWidth += firstPredWidth

			// Rest of prediction with wrapping
			for _, pr := range predictionRunes[1:] {
				predCharStr := string(pr)
				predCharWidth := ansi.StringWidth(predCharStr)

				if currentWidth+predCharWidth > availableWidth {
					result.WriteString("\n")
					currentWidth = 0
				}
				result.WriteString(rndr.config.PredictionStyle.Render(predCharStr))
				currentWidth += predCharWidth
			}
		} else {
			// No prediction, cursor on space
			if currentWidth+1 > availableWidth {
				result.WriteString("\n")
			}
			if focused {
				result.WriteString(rndr.config.CursorStyle.Render(" "))
			} else {
				result.WriteString(" ")
			}
		}
	}

	return result.String()
}

// appendTextWithWrappingAndCursor appends highlighted text with a cursor at the specified position.
// It highlights the full text first to maintain proper syntax coloring context.
func (rndr *Renderer) appendTextWithWrappingAndCursor(result *strings.Builder, text string, cursorPos int, currentWidth, availableWidth int, focused bool) int {
	if text == "" {
		return currentWidth
	}

	runes := []rune(text)
	cursorChar := string(runes[cursorPos])
	cursorCharWidth := ansi.StringWidth(cursorChar)

	// Highlight the entire text first for proper syntax coloring context
	highlighted := rndr.highlighter.Highlight(text)

	highlightedRunes := []rune(highlighted)

	var output strings.Builder
	width := currentWidth
	textIdx := 0      // index in original runes
	highlightIdx := 0 // index in highlighted runes

	// Track the current ANSI style so we can re-apply it after line breaks
	var currentStyle strings.Builder

	for textIdx < len(runes) && highlightIdx < len(highlightedRunes) {
		// Check if we're at an ANSI escape sequence in highlighted output
		if highlightedRunes[highlightIdx] == '\x1b' {
			// Capture and output the entire escape sequence
			var escSeq strings.Builder
			for highlightIdx < len(highlightedRunes) {
				ch := highlightedRunes[highlightIdx]
				escSeq.WriteRune(ch)
				output.WriteRune(ch)
				highlightIdx++
				// ANSI sequences end with a letter
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
					break
				}
			}
			// Track the current style (reset clears it, other sequences update it)
			seq := escSeq.String()
			if seq == "\x1b[0m" || seq == "\x1b[m" {
				currentStyle.Reset()
			} else {
				currentStyle.WriteString(seq)
			}
			continue
		}

		// Get the current character from original text
		if textIdx >= len(runes) {
			break
		}

		// Check if this is the cursor position
		if textIdx == cursorPos {
			// Check if we need to wrap before the cursor character
			if width+cursorCharWidth > availableWidth && width > 0 {
				output.WriteRune('\n')
				width = 0
			}

			// Render the cursor character with cursor style (overriding syntax highlighting)
			if focused {
				output.WriteString(rndr.config.CursorStyle.Render(cursorChar))
			} else {
				// When not focused, show the character with its original highlighting
				if highlightIdx < len(highlightedRunes) {
					output.WriteRune(highlightedRunes[highlightIdx])
				}
			}
			highlightIdx++
			textIdx++
			width += cursorCharWidth

			// Re-apply the current style after cursor (since cursor style may have reset it)
			if focused && currentStyle.Len() > 0 {
				output.WriteString(currentStyle.String())
			}
			continue
		}

		origChar := runes[textIdx]
		charWidth := ansi.StringWidth(string(origChar))

		// Check if we need to wrap before this character
		if width+charWidth > availableWidth && width > 0 {
			output.WriteRune('\n')
			// Re-apply the current style after the line break
			if currentStyle.Len() > 0 {
				output.WriteString(currentStyle.String())
			}
			width = 0
		}

		// Output the character from highlighted text
		if highlightIdx < len(highlightedRunes) {
			output.WriteRune(highlightedRunes[highlightIdx])
			highlightIdx++
		}
		textIdx++
		width += charWidth
	}

	// Output any remaining ANSI codes (like reset sequences)
	for highlightIdx < len(highlightedRunes) {
		output.WriteRune(highlightedRunes[highlightIdx])
		highlightIdx++
	}

	result.WriteString(output.String())
	return width
}

// appendTextWithWrapping appends highlighted text to the result with line wrapping.
// It returns the new current width after appending.
func (rndr *Renderer) appendTextWithWrapping(result *strings.Builder, text string, currentWidth, availableWidth int) int {
	if text == "" {
		return currentWidth
	}

	// Highlight the entire text first for proper syntax coloring context
	highlighted := rndr.highlighter.Highlight(text)

	// Now we need to insert line breaks at the right visual positions
	// We'll walk through the original text to track visual width,
	// and walk through the highlighted text to output it with breaks

	runes := []rune(text)
	highlightedRunes := []rune(highlighted)

	var output strings.Builder
	width := currentWidth
	textIdx := 0      // index in original runes
	highlightIdx := 0 // index in highlighted runes

	// Track the current ANSI style so we can re-apply it after line breaks
	var currentStyle strings.Builder

	for textIdx < len(runes) && highlightIdx < len(highlightedRunes) {
		// Check if we're at an ANSI escape sequence in highlighted output
		if highlightedRunes[highlightIdx] == '\x1b' {
			// Capture and output the entire escape sequence
			var escSeq strings.Builder
			for highlightIdx < len(highlightedRunes) {
				ch := highlightedRunes[highlightIdx]
				escSeq.WriteRune(ch)
				output.WriteRune(ch)
				highlightIdx++
				// ANSI sequences end with a letter
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
					break
				}
			}
			// Track the current style (reset clears it, other sequences update it)
			seq := escSeq.String()
			if seq == "\x1b[0m" || seq == "\x1b[m" {
				currentStyle.Reset()
			} else {
				currentStyle.WriteString(seq)
			}
			continue
		}

		// Get the current character from original text
		if textIdx >= len(runes) {
			break
		}
		origChar := runes[textIdx]
		charWidth := ansi.StringWidth(string(origChar))

		// Check if we need to wrap before this character
		if width+charWidth > availableWidth && width > 0 {
			output.WriteRune('\n')
			// Re-apply the current style after the line break
			if currentStyle.Len() > 0 {
				output.WriteString(currentStyle.String())
			}
			width = 0
		}

		// Output the character from highlighted text
		// The highlighted rune at this position should correspond to the original
		if highlightIdx < len(highlightedRunes) {
			output.WriteRune(highlightedRunes[highlightIdx])
			highlightIdx++
		}
		textIdx++
		width += charWidth
	}

	// Output any remaining ANSI codes (like reset sequences)
	for highlightIdx < len(highlightedRunes) {
		output.WriteRune(highlightedRunes[highlightIdx])
		highlightIdx++
	}

	result.WriteString(output.String())
	return width
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

// RenderHistorySearchPrompt renders the prompt for history search mode.
// It shows "(history search)`query': " similar to bash's Ctrl+R style.
// The cursor is rendered at the end of the query.
func (r *Renderer) RenderHistorySearchPrompt(state *HistorySearchState, showCursor bool) string {
	if state == nil || !state.IsActive() {
		return ""
	}

	query := state.Query()

	// Style the prompt with yellow for the label (matching the UI color scheme)
	promptStyle := lipgloss.NewStyle().Foreground(render.ColorYellow)
	queryStyle := lipgloss.NewStyle().Bold(true)

	// Render cursor at end of query if focused
	cursorStr := ""
	if showCursor {
		cursorStr = r.config.CursorStyle.Render(" ")
	}

	return promptStyle.Render("(history search)") +
		"`" + queryStyle.Render(query) + cursorStr + "': "
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
	promptWidth := ansi.StringWidth(prompt)
	runes := []rune(text)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}
	textBeforeCursor := string(runes[:cursorPos])
	textWidth := ansi.StringWidth(textBeforeCursor)
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
