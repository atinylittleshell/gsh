package render

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// SpinnerFrames contains the braille spinner animation frames
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerIDCounter is used to assign unique IDs to spinners for ordering
var spinnerIDCounter uint64

// SpinnerManager coordinates multiple spinners to ensure only one renders at a time.
// When multiple spinners are live, only the most recently created one actually renders.
// When that spinner stops, the next most recent one takes over.
// It also supports string ID mapping for script API compatibility.
type SpinnerManager struct {
	writer   io.Writer
	frames   []string
	interval time.Duration

	mu            sync.Mutex
	liveSpinners  map[uint64]*ManagedSpinner // All currently live spinners (by internal ID)
	spinnersByID  map[string]*ManagedSpinner // Spinners indexed by string ID (for script API)
	activeID      uint64                     // Internal ID of the currently rendering spinner (0 if none)
	frameIndex    int                        // Current animation frame
	ticker        *time.Ticker               // Shared ticker for animation
	stopAnimation chan struct{}              // Signal to stop animation goroutine
	animationDone chan struct{}              // Signal that animation goroutine has stopped
	nextStringID  int                        // Counter for generating string IDs
}

// ManagedSpinner represents a spinner managed by SpinnerManager
type ManagedSpinner struct {
	id       uint64
	stringID string // Optional string ID for script API
	message  string
	manager  *SpinnerManager
	done     chan struct{} // Closed when this spinner is stopped
	stopped  bool
	running  bool // Whether Start() has been called
}

// NewSpinnerManager creates a new spinner manager
func NewSpinnerManager(writer io.Writer) *SpinnerManager {
	return &SpinnerManager{
		writer:       writer,
		frames:       SpinnerFrames,
		interval:     80 * time.Millisecond,
		liveSpinners: make(map[uint64]*ManagedSpinner),
		spinnersByID: make(map[string]*ManagedSpinner),
	}
}

// GenerateID generates a new unique string spinner ID
func (m *SpinnerManager) GenerateID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("spinner_%d", m.nextStringID)
	m.nextStringID++
	return id
}

// NewSpinner creates a new managed spinner without a string ID.
// The spinner doesn't start rendering until Start is called.
func (m *SpinnerManager) NewSpinner() *ManagedSpinner {
	return m.NewSpinnerWithID("")
}

// NewSpinnerWithID creates a new managed spinner with an optional string ID.
// If id is empty, no string ID mapping is created.
// The spinner doesn't start rendering until Start is called.
func (m *SpinnerManager) NewSpinnerWithID(id string) *ManagedSpinner {
	internalID := atomic.AddUint64(&spinnerIDCounter, 1)
	spinner := &ManagedSpinner{
		id:       internalID,
		stringID: id,
		manager:  m,
		done:     make(chan struct{}),
	}

	// Register in string ID map if ID provided
	if id != "" {
		m.mu.Lock()
		m.spinnersByID[id] = spinner
		m.mu.Unlock()
	}

	return spinner
}

// GetSpinnerByID returns a spinner by its string ID
func (m *SpinnerManager) GetSpinnerByID(id string) (*ManagedSpinner, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	spinner, exists := m.spinnersByID[id]
	return spinner, exists
}

// GetActiveSpinnerWithID returns the most recently started spinner with its string ID
func (m *SpinnerManager) GetActiveSpinnerWithID() (*ManagedSpinner, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeID == 0 {
		return nil, "", false
	}
	spinner, exists := m.liveSpinners[m.activeID]
	if !exists {
		return nil, "", false
	}
	return spinner, spinner.stringID, true
}

// HasActiveSpinners returns true if there are any active spinners
func (m *SpinnerManager) HasActiveSpinners() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.liveSpinners) > 0
}

// ID returns the string ID of this spinner (empty if not set)
func (s *ManagedSpinner) ID() string {
	return s.stringID
}

// IsRunning returns whether the spinner has been started and not stopped
func (s *ManagedSpinner) IsRunning() bool {
	s.manager.mu.Lock()
	defer s.manager.mu.Unlock()
	return s.running && !s.stopped
}

// SetMessage sets the message to display after the spinner
func (s *ManagedSpinner) SetMessage(message string) {
	s.manager.mu.Lock()
	defer s.manager.mu.Unlock()
	s.message = message
}

// Start registers this spinner as live and potentially starts it rendering.
// Returns a stop function that removes this spinner and waits for cleanup.
func (s *ManagedSpinner) Start(ctx context.Context) func() {
	s.manager.mu.Lock()
	if s.stopped {
		// Already stopped, return no-op
		s.manager.mu.Unlock()
		return func() {}
	}

	if s.running {
		// Already running, return stop function
		s.manager.mu.Unlock()
		return func() {
			s.manager.stopSpinner(s)
			<-s.done
		}
	}

	s.running = true

	// Register this spinner as live
	s.manager.liveSpinners[s.id] = s

	// Determine if this should become the active spinner (it's the most recent)
	shouldActivate := s.id > s.manager.activeID
	if shouldActivate {
		s.manager.activeID = s.id
	}

	// Start animation if not already running
	needsAnimation := s.manager.ticker == nil
	if needsAnimation {
		s.manager.startAnimation()
	}
	s.manager.mu.Unlock()

	// Return stop function
	return func() {
		s.manager.stopSpinner(s)
		<-s.done
	}
}

// startAnimation starts the shared animation goroutine (must be called with lock held)
func (m *SpinnerManager) startAnimation() {
	m.ticker = time.NewTicker(m.interval)
	m.stopAnimation = make(chan struct{})
	m.animationDone = make(chan struct{})
	m.frameIndex = 0

	// Pass the ticker channel to avoid race conditions
	tickerC := m.ticker.C
	go m.runAnimation(tickerC)
}

// runAnimation is the shared animation goroutine
func (m *SpinnerManager) runAnimation(tickerC <-chan time.Time) {
	defer close(m.animationDone)

	// Render initial frame
	m.renderActiveSpinner()

	for {
		select {
		case <-m.stopAnimation:
			m.clearLine()
			return
		case <-tickerC:
			m.mu.Lock()
			m.frameIndex = (m.frameIndex + 1) % len(m.frames)
			m.mu.Unlock()
			m.renderActiveSpinner()
		}
	}
}

// renderActiveSpinner renders the currently active spinner
func (m *SpinnerManager) renderActiveSpinner() {
	m.mu.Lock()
	activeSpinner := m.liveSpinners[m.activeID]
	frameIndex := m.frameIndex
	m.mu.Unlock()

	if activeSpinner == nil {
		return
	}

	m.mu.Lock()
	message := activeSpinner.message
	m.mu.Unlock()

	frame := m.frames[frameIndex]
	styledFrame := ToolPendingStyle.Render(frame)

	if message != "" {
		fmt.Fprintf(m.writer, "\r\033[K%s %s", styledFrame, message)
	} else {
		fmt.Fprintf(m.writer, "\r\033[K%s", styledFrame)
	}
}

// stopSpinner removes a spinner and potentially activates another
func (m *SpinnerManager) stopSpinner(s *ManagedSpinner) {
	m.mu.Lock()

	if s.stopped {
		m.mu.Unlock()
		return
	}
	s.stopped = true
	s.running = false

	// Remove from live spinners
	delete(m.liveSpinners, s.id)

	// Remove from string ID map if it has a string ID
	if s.stringID != "" {
		delete(m.spinnersByID, s.stringID)
	}

	// If this was the active spinner, find the next most recent one
	if m.activeID == s.id {
		m.activeID = m.findMostRecentSpinnerID()
	}

	// If no more live spinners, stop the animation
	if len(m.liveSpinners) == 0 && m.ticker != nil {
		m.ticker.Stop()
		m.ticker = nil
		close(m.stopAnimation)
		// Wait for animation to finish (release lock temporarily)
		animationDone := m.animationDone
		m.mu.Unlock()
		<-animationDone
		// Signal done after animation cleanup
		close(s.done)
		return
	}

	m.mu.Unlock()
	close(s.done)
}

// RemoveSpinner stops and removes a spinner by its string ID
func (m *SpinnerManager) RemoveSpinner(id string) {
	spinner, exists := m.GetSpinnerByID(id)
	if exists {
		m.stopSpinner(spinner)
		<-spinner.done
	}
}

// StopAll stops all active spinners
func (m *SpinnerManager) StopAll() {
	m.mu.Lock()
	// Collect all spinners to stop (to avoid modifying map while iterating)
	spinnersToStop := make([]*ManagedSpinner, 0, len(m.liveSpinners))
	for _, spinner := range m.liveSpinners {
		spinnersToStop = append(spinnersToStop, spinner)
	}
	m.mu.Unlock()

	// Stop each spinner
	for _, spinner := range spinnersToStop {
		m.stopSpinner(spinner)
		<-spinner.done
	}
}

// findMostRecentSpinnerID finds the spinner with the highest ID (most recent)
// Must be called with lock held
func (m *SpinnerManager) findMostRecentSpinnerID() uint64 {
	var maxID uint64
	for id := range m.liveSpinners {
		if id > maxID {
			maxID = id
		}
	}
	return maxID
}

// LiveCount returns the number of live spinners (for testing)
func (m *SpinnerManager) LiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.liveSpinners)
}

// ActiveID returns the ID of the currently active spinner (for testing)
func (m *SpinnerManager) ActiveID() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeID
}

// clearLine clears the current line
func (m *SpinnerManager) clearLine() {
	fmt.Fprintf(m.writer, "\r\033[K")
}

// Spinner manages an animated spinner display (standalone, not managed)
// Deprecated: Use SpinnerManager.NewSpinner() for coordinated spinners
type Spinner struct {
	writer   io.Writer
	frames   []string
	interval time.Duration
	mu       sync.Mutex
	running  bool
	message  string
	done     chan struct{} // Channel to signal spinner has fully stopped
}

// NewSpinner creates a new standalone spinner with default frames.
// Note: For coordinated spinners, use NewSpinnerManager().NewSpinner() instead.
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
