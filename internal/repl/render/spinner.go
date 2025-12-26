package render

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// SpinnerFrames contains the braille spinner animation frames
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner manages an animated spinner display
type Spinner struct {
	writer   io.Writer
	frames   []string
	interval time.Duration
	mu       sync.Mutex
	running  bool
	message  string
	done     chan struct{} // Channel to signal spinner has fully stopped
}

// NewSpinner creates a new spinner with default frames
func NewSpinner(writer io.Writer) *Spinner {
	return &Spinner{
		writer:   writer,
		frames:   SpinnerFrames,
		interval: 80 * time.Millisecond,
	}
}

// SetMessage sets the message to display after the spinner
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Start begins the spinner animation and returns a stop function.
// The stop function blocks until the spinner has fully stopped and cleared the line.
// The spinner runs in a goroutine and updates the display using ANSI escape codes.
func (s *Spinner) Start(ctx context.Context) func() {
	ctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return func() { cancel() }
	}
	s.running = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx)

	// Return a stop function that cancels and waits for completion
	return func() {
		cancel()
		<-s.done // Wait for spinner goroutine to finish
	}
}

// run is the internal goroutine that animates the spinner
func (s *Spinner) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIndex := 0

	// Render initial frame
	s.renderFrame(frameIndex)

	for {
		select {
		case <-ctx.Done():
			s.clearLine()
			s.mu.Lock()
			s.running = false
			done := s.done
			s.mu.Unlock()
			close(done) // Signal that spinner has fully stopped
			return
		case <-ticker.C:
			frameIndex = (frameIndex + 1) % len(s.frames)
			s.renderFrame(frameIndex)
		}
	}
}

// renderFrame renders a single spinner frame with optional message
func (s *Spinner) renderFrame(frameIndex int) {
	s.mu.Lock()
	message := s.message
	s.mu.Unlock()

	frame := s.frames[frameIndex]
	styledFrame := ToolPendingStyle.Render(frame)

	// Move cursor to beginning of line, clear line, render frame
	if message != "" {
		fmt.Fprintf(s.writer, "\r\033[K%s %s", styledFrame, message)
	} else {
		fmt.Fprintf(s.writer, "\r\033[K%s", styledFrame)
	}
}

// clearLine clears the current line
func (s *Spinner) clearLine() {
	fmt.Fprintf(s.writer, "\r\033[K")
}

// GetCurrentFrame returns the current spinner frame for a given index
// This is useful for inline spinners where the caller manages the animation
func GetCurrentFrame(index int) string {
	return SpinnerFrames[index%len(SpinnerFrames)]
}

// InlineSpinner manages a spinner that updates inline without owning the whole line
type InlineSpinner struct {
	writer   io.Writer
	frames   []string
	interval time.Duration
	mu       sync.Mutex
	running  bool
	prefix   string        // Text before spinner
	suffix   string        // Text after spinner (typically empty during spinning)
	done     chan struct{} // Channel to signal spinner has fully stopped
}

// NewInlineSpinner creates a new inline spinner
func NewInlineSpinner(writer io.Writer) *InlineSpinner {
	return &InlineSpinner{
		writer:   writer,
		frames:   SpinnerFrames,
		interval: 80 * time.Millisecond,
	}
}

// SetPrefix sets the prefix text shown before the spinner
func (s *InlineSpinner) SetPrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefix = prefix
}

// SetSuffix sets the suffix text shown after the spinner
func (s *InlineSpinner) SetSuffix(suffix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suffix = suffix
}

// Start begins the inline spinner animation and returns a stop function.
// The stop function blocks until the spinner has fully stopped.
func (s *InlineSpinner) Start(ctx context.Context) func() {
	ctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return func() { cancel() }
	}
	s.running = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx)

	// Return a stop function that cancels and waits for completion
	return func() {
		cancel()
		<-s.done // Wait for spinner goroutine to finish
	}
}

// run is the internal goroutine that animates the inline spinner
func (s *InlineSpinner) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIndex := 0

	// Render initial frame
	s.renderFrame(frameIndex)

	for {
		select {
		case <-ctx.Done():
			// Don't clear - let caller handle final state
			s.mu.Lock()
			s.running = false
			done := s.done
			s.mu.Unlock()
			close(done) // Signal that spinner has fully stopped
			return
		case <-ticker.C:
			frameIndex = (frameIndex + 1) % len(s.frames)
			s.renderFrame(frameIndex)
		}
	}
}

// renderFrame renders a single spinner frame with prefix
func (s *InlineSpinner) renderFrame(frameIndex int) {
	s.mu.Lock()
	prefix := s.prefix
	suffix := s.suffix
	s.mu.Unlock()

	frame := s.frames[frameIndex]
	styledFrame := ToolPendingStyle.Render(frame)

	// Move cursor to beginning of line, clear line, render
	fmt.Fprintf(s.writer, "\r\033[K%s %s%s", prefix, styledFrame, suffix)
}

// ClearAndFinish clears the spinner line - caller should print final state
func (s *InlineSpinner) ClearAndFinish() {
	fmt.Fprintf(s.writer, "\r\033[K")
}
