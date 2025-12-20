package coach

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const lightning = "⚡"

// Color cycle for animation: blue → purple → orange → yellow → back
// Matches the colors in pkg/gline/llm_status.go
var animationColors = []lipgloss.Color{
	"12", "33", "57", "93", "129", "208", "214", "220",
	"214", "208", "129", "93", "57", "33",
}

// ProgressIndicator displays an animated lightning bolt with a message
type ProgressIndicator struct {
	message    string
	frameIndex int
	wordCount  int
	stopChan   chan struct{}
	doneChan   chan struct{}
	mu         sync.Mutex
	stopOnce   sync.Once
	running    bool
}

// NewProgressIndicator creates a new progress indicator with the given message
func NewProgressIndicator(message string) *ProgressIndicator {
	return &ProgressIndicator{
		message:  message,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start begins the animation
func (p *ProgressIndicator) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	go func() {
		defer close(p.doneChan)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		// Initial render
		p.render()

		for {
			select {
			case <-p.stopChan:
				return
			case <-ticker.C:
				p.frameIndex = (p.frameIndex + 1) % len(animationColors)
				p.render()
			}
		}
	}()
}

// render displays the current frame
func (p *ProgressIndicator) render() {
	p.mu.Lock()
	wordCount := p.wordCount
	message := p.message
	p.mu.Unlock()

	color := animationColors[p.frameIndex]
	bolt := lipgloss.NewStyle().Foreground(color).Render(lightning)

	// Use carriage return to overwrite the line, no newline
	if wordCount > 0 {
		// Word count in static gray, only bolt pulses
		countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(fmt.Sprintf("(%d)", wordCount))
		fmt.Printf("\r  %s %s%s", message, countStyle, bolt)
	} else {
		fmt.Printf("\r  %s %s", message, bolt)
	}
}

// shutdown performs the shared shutdown sequence (called via stopOnce)
func (p *ProgressIndicator) shutdown() {
	p.mu.Lock()
	wasRunning := p.running
	p.running = false
	p.mu.Unlock()

	if wasRunning {
		close(p.stopChan)
		<-p.doneChan
	}

	// Clear the line and move cursor back
	fmt.Print("\r\033[K")
}

// Stop stops the animation and clears the line
func (p *ProgressIndicator) Stop() {
	p.stopOnce.Do(p.shutdown)
}

// StopWithMessage stops the animation and prints a final message
func (p *ProgressIndicator) StopWithMessage(message string) {
	p.stopOnce.Do(p.shutdown)
	fmt.Println(message)
}

// UpdateMessage updates the message being displayed
func (p *ProgressIndicator) UpdateMessage(message string) {
	p.mu.Lock()
	p.message = message
	p.mu.Unlock()
}

// UpdateWordCount updates the word count being displayed
func (p *ProgressIndicator) UpdateWordCount(count int) {
	p.mu.Lock()
	p.wordCount = count
	p.mu.Unlock()
}

// AddWords adds to the word count
func (p *ProgressIndicator) AddWords(count int) {
	p.mu.Lock()
	p.wordCount += count
	p.mu.Unlock()
}
