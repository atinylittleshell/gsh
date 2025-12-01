package gline

import (
	"github.com/charmbracelet/bubbles/spinner"
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

// LLMIndicator holds the state for an LLM status indicator
type LLMIndicator struct {
	spinner spinner.Model
	status  LLMStatus
	label   string
}

// NewLLMIndicator creates a new LLM indicator with the given label
func NewLLMIndicator(label string) LLMIndicator {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return LLMIndicator{
		spinner: s,
		status:  LLMStatusIdle,
		label:   label,
	}
}

// SetStatus updates the indicator status
func (i *LLMIndicator) SetStatus(status LLMStatus) {
	i.status = status
}

// GetStatus returns the current status
func (i LLMIndicator) GetStatus() LLMStatus {
	return i.status
}

// Update processes spinner tick messages
func (i *LLMIndicator) Update(msg spinner.TickMsg) {
	i.spinner, _ = i.spinner.Update(msg)
}

// View renders the indicator
func (i LLMIndicator) View() string {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))    // Red
	idleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))     // Gray
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))  // Dim

	var statusIcon string
	switch i.status {
	case LLMStatusInFlight:
		statusIcon = i.spinner.View()
	case LLMStatusSuccess:
		statusIcon = successStyle.Render("✓")
	case LLMStatusError:
		statusIcon = errorStyle.Render("✗")
	default:
		statusIcon = idleStyle.Render("○")
	}

	return labelStyle.Render(i.label+":") + statusIcon
}
