package gline

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LLMStatus represents the current status of an LLM request
type LLMStatus int

const (
	// LLMStatusIdle means no request has been made yet
	LLMStatusIdle LLMStatus = iota
	// LLMStatusInFlight means a request is currently in progress
	LLMStatusInFlight
	// LLMStatusSuccess means the last request was successful
	LLMStatusSuccess
	// LLMStatusError means the last request encountered an error
	LLMStatusError
)

const lightning = "⚡"

// Color cycle for in-flight animation: blue → purple → orange → yellow → back
var inFlightColors = []lipgloss.Color{
	"12", "33", "57", "93", "129", "208", "214", "220",
	"214", "208", "129", "93", "57", "33",
}

// LLMTickMsg is sent to advance the color animation
type LLMTickMsg struct{}

// LLMIndicator holds the state for an LLM status indicator
type LLMIndicator struct {
	status     LLMStatus
	frameIndex int
}

// NewLLMIndicator creates a new LLM indicator
func NewLLMIndicator() LLMIndicator {
	return LLMIndicator{status: LLMStatusIdle}
}

// Tick returns a command that sends LLMTickMsg after the animation interval
func (i LLMIndicator) Tick() tea.Cmd {
	return tea.Tick(time.Second/2, func(t time.Time) tea.Msg {
		return LLMTickMsg{}
	})
}

// SetStatus updates the indicator status
func (i *LLMIndicator) SetStatus(status LLMStatus) {
	i.status = status
}

// GetStatus returns the current status
func (i LLMIndicator) GetStatus() LLMStatus {
	return i.status
}

// Update advances the animation frame
func (i *LLMIndicator) Update() {
	i.frameIndex = (i.frameIndex + 1) % len(inFlightColors)
}

// Width returns the display width of the indicator.
// Note: ⚡ (U+26A1) has ambiguous East Asian width; runewidth returns 2 but
// many western terminals render it as 1 cell. We detect the actual terminal
// behavior at runtime using cursor position probing.
func (i LLMIndicator) Width() int {
	return GetLightningBoltWidth()
}

// View renders the indicator
func (i LLMIndicator) View() string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")) // Match border color
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))     // Red
	idleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))    // Gray

	switch i.status {
	case LLMStatusInFlight:
		color := inFlightColors[i.frameIndex]
		return lipgloss.NewStyle().Foreground(color).Render(lightning)
	case LLMStatusSuccess:
		return borderStyle.Render(lightning)
	case LLMStatusError:
		return redStyle.Render(lightning)
	default:
		return idleStyle.Render(lightning)
	}
}
