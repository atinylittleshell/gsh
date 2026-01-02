package interpreter

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/atinylittleshell/gsh/internal/repl/render"
)

// Global spinner manager shared across all scripts
var (
	spinnerManager *render.SpinnerManager
	spinnerMutex   sync.Mutex
	spinnerContext context.Context
)

// init initializes the global spinner manager
func init() {
	spinnerManager = render.NewSpinnerManager(os.Stdout)
}

// GetSpinnerManager returns the global spinner manager (for testing)
func GetSpinnerManager() *render.SpinnerManager {
	return spinnerManager
}

// SetSpinnerManager sets the global spinner manager (for testing)
func SetSpinnerManager(manager *render.SpinnerManager) {
	spinnerManager = manager
}

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
			"write": {Value: &BuiltinValue{
				Name: "gsh.ui.write",
				Fn: func(args []Value) (Value, error) {
					// Write text to stdout without trailing newline
					for _, arg := range args {
						fmt.Fprint(os.Stdout, arg.String())
					}
					return &NullValue{}, nil
				},
			}, ReadOnly: true},
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

				// Create and start new spinner with the string ID
				spinner := spinnerManager.NewSpinnerWithID(spinnerID)
				spinner.SetMessage(msgVal.Value)
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
					if spinner, exists := spinnerManager.GetSpinnerByID(idVal.Value); exists && spinner.IsRunning() {
						spinner.SetMessage(msgVal.Value)
					}
				} else {
					if spinner, _, exists := spinnerManager.GetActiveSpinnerWithID(); exists && spinner.IsRunning() {
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

				spinnerManager.RemoveSpinner(idVal.Value)
				return &NullValue{}, nil
			},
		}
	case "stopAll":
		return &BuiltinValue{
			Name: "gsh.ui.spinner.stopAll",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("spinner.stopAll() takes no arguments, got %d", len(args))
				}

				spinnerMutex.Lock()
				defer spinnerMutex.Unlock()

				spinnerManager.StopAll()
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
	case "italic":
		return &BuiltinValue{
			Name: "gsh.ui.styles.italic",
			Fn: func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("styles.italic() takes 1 argument (text: string), got %d", len(args))
				}
				textVal, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("styles.italic() argument must be a string, got %s", args[0].Type())
				}
				style := lipgloss.NewStyle().Italic(true)
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
