package interpreter

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/atinylittleshell/gsh/internal/repl/render"
)

// Global spinner manager shared across all scripts
var (
	spinnerManager *SpinnerManager
	spinnerMutex   sync.Mutex
	spinnerContext context.Context
)

// SpinnerManager tracks multiple spinners by ID
type SpinnerManager struct {
	mu              sync.Mutex
	spinners        map[string]*UISpinner
	activeSpinnerID string // Most recently started spinner
	nextID          int
}

// NewSpinnerManager creates a new spinner manager
func NewSpinnerManager() *SpinnerManager {
	return &SpinnerManager{
		spinners: make(map[string]*UISpinner),
		nextID:   0,
	}
}

// GenerateID generates a new unique spinner ID
func (sm *SpinnerManager) GenerateID() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	id := fmt.Sprintf("spinner_%d", sm.nextID)
	sm.nextID++
	return id
}

// AddSpinner adds a new spinner and returns its ID
func (sm *SpinnerManager) AddSpinner(id string, spinner *UISpinner) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.spinners[id] = spinner
	sm.activeSpinnerID = id // Most recent spinner becomes active
}

// RemoveSpinner removes a spinner by ID
func (sm *SpinnerManager) RemoveSpinner(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.spinners, id)

	// If we removed the active spinner, find another active one
	if sm.activeSpinnerID == id {
		if len(sm.spinners) > 0 {
			// Pick any remaining spinner
			for activeID := range sm.spinners {
				sm.activeSpinnerID = activeID
				break
			}
		} else {
			sm.activeSpinnerID = ""
		}
	}
}

// GetSpinner returns a spinner by ID
func (sm *SpinnerManager) GetSpinner(id string) (*UISpinner, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	spinner, exists := sm.spinners[id]
	return spinner, exists
}

// GetActiveSpinner returns the most recently started spinner
func (sm *SpinnerManager) GetActiveSpinner() (*UISpinner, string, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.activeSpinnerID == "" {
		return nil, "", false
	}
	spinner, exists := sm.spinners[sm.activeSpinnerID]
	return spinner, sm.activeSpinnerID, exists
}

// HasActiveSpinners returns true if there are any active spinners
func (sm *SpinnerManager) HasActiveSpinners() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.spinners) > 0
}

// init initializes the global spinner manager
func init() {
	spinnerManager = NewSpinnerManager()
}

// UISpinner wraps the rendering spinner with methods callable from gsh script
type UISpinner struct {
	id         string
	message    string
	frameIndex int
	stopCh     chan struct{}
	mu         sync.Mutex
	running    bool
	writer     io.Writer
}

// SpinnerFrames animation frames
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// UIStylesObject provides semantic style functions
type UIStylesObject struct{}

// UISpinnerObject provides spinner control methods
type UISpinnerObject struct{}

// UICursorObject provides cursor control methods
type UICursorObject struct{}

// createUIObject creates the gsh.ui object with spinner, styles, and cursor control
func (i *Interpreter) createUIObject() *ObjectValue {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"spinner": {Value: &UISpinnerObjectValue{interp: i}, ReadOnly: true},
			"styles":  {Value: &UIStylesObjectValue{}, ReadOnly: true},
			"cursor":  {Value: &UICursorObjectValue{}, ReadOnly: true},
		},
	}
}

// UISpinnerObjectValue represents gsh.ui.spinner
type UISpinnerObjectValue struct {
	interp *Interpreter
}

func (s *UISpinnerObjectValue) Type() ValueType { return ValueTypeObject }
func (s *UISpinnerObjectValue) String() string  { return "<gsh.ui.spinner>" }
func (s *UISpinnerObjectValue) IsTruthy() bool  { return true }
func (s *UISpinnerObjectValue) Equals(other Value) bool {
	_, ok := other.(*UISpinnerObjectValue)
	return ok
}

func (s *UISpinnerObjectValue) GetProperty(name string) Value {
	switch name {
	case "start":
		return &BuiltinValue{
			Name: "gsh.ui.spinner.start",
			Fn: func(args []Value) (Value, error) {
				if len(args) < 1 || len(args) > 2 {
					return nil, fmt.Errorf("spinner.start() takes 1 or 2 arguments (message: string, [id: string]), got %d", len(args))
				}
				msgVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("spinner.start() first argument must be a string, got %s", args[0].Type())
				}

				// Optional custom ID
				var spinnerID string
				if len(args) == 2 {
					idVal, ok := args[1].(*StringValue)
					if !ok {
						return nil, fmt.Errorf("spinner.start() second argument must be a string, got %s", args[1].Type())
					}
					spinnerID = idVal.Value
				} else {
					spinnerID = spinnerManager.GenerateID()
				}

				spinnerMutex.Lock()
				defer spinnerMutex.Unlock()

				// Initialize context if needed
				if spinnerContext == nil || spinnerContext.Err() != nil {
					var cancel context.CancelFunc
					spinnerContext, cancel = context.WithCancel(context.Background())
					_ = cancel // Context cancellation not currently used
				}

				// Create and start new spinner
				spinner := NewUISpinner(spinnerID, msgVal.Value, os.Stdout)
				spinnerManager.AddSpinner(spinnerID, spinner)
				spinner.Start(spinnerContext)

				return &StringValue{Value: spinnerID}, nil
			},
		}
	case "setMessage":
		return &BuiltinValue{
			Name: "gsh.ui.spinner.setMessage",
			Fn: func(args []Value) (Value, error) {
				if len(args) < 1 || len(args) > 2 {
					return nil, fmt.Errorf("spinner.setMessage() takes 1 or 2 arguments (message: string, [id: string]), got %d", len(args))
				}
				msgVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("spinner.setMessage() first argument must be a string, got %s", args[0].Type())
				}

				spinnerMutex.Lock()
				defer spinnerMutex.Unlock()

				// If ID specified, update that spinner; otherwise update active spinner
				if len(args) == 2 {
					idVal, ok := args[1].(*StringValue)
					if !ok {
						return nil, fmt.Errorf("spinner.setMessage() second argument must be a string, got %s", args[1].Type())
					}
					if spinner, exists := spinnerManager.GetSpinner(idVal.Value); exists && spinner.running {
						spinner.SetMessage(msgVal.Value)
					}
				} else {
					if spinner, _, exists := spinnerManager.GetActiveSpinner(); exists && spinner.running {
						spinner.SetMessage(msgVal.Value)
					}
				}
				return &NullValue{}, nil
			},
		}
	case "stop":
		return &BuiltinValue{
			Name: "gsh.ui.spinner.stop",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("spinner.stop() takes 1 argument (id: string), got %d", len(args))
				}

				idVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("spinner.stop() argument must be a string, got %s", args[0].Type())
				}

				spinnerMutex.Lock()
				defer spinnerMutex.Unlock()

				if spinner, exists := spinnerManager.GetSpinner(idVal.Value); exists && spinner.running {
					spinner.Stop()
					spinnerManager.RemoveSpinner(idVal.Value)
				}
				return &NullValue{}, nil
			},
		}
	default:
		return &NullValue{}
	}
}

func (s *UISpinnerObjectValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.ui.spinner", name)
}

// UIStylesObjectValue represents gsh.ui.styles
type UIStylesObjectValue struct{}

func (s *UIStylesObjectValue) Type() ValueType { return ValueTypeObject }
func (s *UIStylesObjectValue) String() string  { return "<gsh.ui.styles>" }
func (s *UIStylesObjectValue) IsTruthy() bool  { return true }
func (s *UIStylesObjectValue) Equals(other Value) bool {
	_, ok := other.(*UIStylesObjectValue)
	return ok
}

func (s *UIStylesObjectValue) GetProperty(name string) Value {
	switch name {
	case "primary":
		return &BuiltinValue{
			Name: "gsh.ui.styles.primary",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.primary() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.primary() argument must be a string, got %s", args[0].Type())
				}
				// Yellow (ANSI 11) - primary UI color
				style := lipgloss.NewStyle().Foreground(render.ColorYellow)
				return &StringValue{Value: style.Render(textVal.Value)}, nil
			},
		}
	case "success":
		return &BuiltinValue{
			Name: "gsh.ui.styles.success",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.success() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.success() argument must be a string, got %s", args[0].Type())
				}
				// Green (ANSI 10)
				style := lipgloss.NewStyle().Foreground(render.ColorGreen)
				return &StringValue{Value: style.Render(textVal.Value)}, nil
			},
		}
	case "error":
		return &BuiltinValue{
			Name: "gsh.ui.styles.error",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.error() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.error() argument must be a string, got %s", args[0].Type())
				}
				// Red (ANSI 9)
				style := lipgloss.NewStyle().Foreground(render.ColorRed)
				return &StringValue{Value: style.Render(textVal.Value)}, nil
			},
		}
	case "dim":
		return &BuiltinValue{
			Name: "gsh.ui.styles.dim",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.dim() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.dim() argument must be a string, got %s", args[0].Type())
				}
				// Gray (ANSI 8)
				style := lipgloss.NewStyle().Foreground(render.ColorGray)
				return &StringValue{Value: style.Render(textVal.Value)}, nil
			},
		}
	case "bold":
		return &BuiltinValue{
			Name: "gsh.ui.styles.bold",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.bold() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.bold() argument must be a string, got %s", args[0].Type())
				}
				style := lipgloss.NewStyle().Bold(true)
				return &StringValue{Value: style.Render(textVal.Value)}, nil
			},
		}
	default:
		return &NullValue{}
	}
}

func (s *UIStylesObjectValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.ui.styles", name)
}

// UICursorObjectValue represents gsh.ui.cursor
type UICursorObjectValue struct{}

func (c *UICursorObjectValue) Type() ValueType { return ValueTypeObject }
func (c *UICursorObjectValue) String() string  { return "<gsh.ui.cursor>" }
func (c *UICursorObjectValue) IsTruthy() bool  { return true }
func (c *UICursorObjectValue) Equals(other Value) bool {
	_, ok := other.(*UICursorObjectValue)
	return ok
}

func (c *UICursorObjectValue) GetProperty(name string) Value {
	switch name {
	case "clearLine":
		return &BuiltinValue{
			Name: "gsh.ui.cursor.clearLine",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("cursor.clearLine() takes no arguments, got %d", len(args))
				}
				// ANSI escape sequence: clear line
				fmt.Fprint(os.Stdout, "\033[K")
				return &NullValue{}, nil
			},
		}
	case "moveCursor":
		return &BuiltinValue{
			Name: "gsh.ui.cursor.moveCursor",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("cursor.moveCursor() takes 2 arguments (x: number, y: number), got %d", len(args))
				}
				xVal, ok := args[0].(*NumberValue)
				if !ok {
					return nil, fmt.Errorf("cursor.moveCursor() first argument must be a number, got %s", args[0].Type())
				}
				yVal, ok := args[1].(*NumberValue)
				if !ok {
					return nil, fmt.Errorf("cursor.moveCursor() second argument must be a number, got %s", args[1].Type())
				}
				x := int(xVal.Value)
				y := int(yVal.Value)

				// Move up/down
				if y > 0 {
					for i := 0; i < y; i++ {
						fmt.Fprint(os.Stdout, "\033[B") // Move down
					}
				} else if y < 0 {
					for i := 0; i < -y; i++ {
						fmt.Fprint(os.Stdout, "\033[A") // Move up
					}
				}

				// Move left/right
				if x > 0 {
					for i := 0; i < x; i++ {
						fmt.Fprint(os.Stdout, "\033[C") // Move right
					}
				} else if x < 0 {
					for i := 0; i < -x; i++ {
						fmt.Fprint(os.Stdout, "\033[D") // Move left
					}
				}

				return &NullValue{}, nil
			},
		}
	case "clearLines":
		return &BuiltinValue{
			Name: "gsh.ui.cursor.clearLines",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("cursor.clearLines() takes 1 argument (lines: number), got %d", len(args))
				}
				numVal, ok := args[0].(*NumberValue)
				if !ok {
					return nil, fmt.Errorf("cursor.clearLines() argument must be a number, got %s", args[0].Type())
				}
				lines := int(numVal.Value)
				if lines < 0 {
					return nil, fmt.Errorf("cursor.clearLines() argument must be non-negative, got %d", lines)
				}
				// Move up and clear each line
				for i := 0; i < lines; i++ {
					fmt.Fprint(os.Stdout, "\033[A\033[K")
				}
				return &NullValue{}, nil
			},
		}
	default:
		return &NullValue{}
	}
}

func (c *UICursorObjectValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.ui.cursor", name)
}

// NewUISpinner creates a new spinner
func NewUISpinner(id, message string, writer io.Writer) *UISpinner {
	return &UISpinner{
		id:      id,
		message: message,
		writer:  writer,
		stopCh:  make(chan struct{}),
	}
}

// SetMessage updates the spinner message
func (s *UISpinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Start begins the spinner animation
func (s *UISpinner) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run(ctx)
}

// Stop stops the spinner and clears the line
func (s *UISpinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	// Clear the line
	fmt.Fprint(s.writer, "\r\033[K")
}

// run the spinner animation loop
func (s *UISpinner) run(ctx context.Context) {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()
				return
			}
			frame := SpinnerFrames[s.frameIndex%len(SpinnerFrames)]
			s.frameIndex++
			message := s.message
			s.mu.Unlock()

			// Render frame with carriage return to overwrite
			fmt.Fprintf(s.writer, "\r%s %s", frame, message)
		}
	}
}
