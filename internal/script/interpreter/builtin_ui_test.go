package interpreter

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/repl/render"
)

func TestSpinnerManager(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)

	// Test GenerateID
	id1 := manager.GenerateID()
	id2 := manager.GenerateID()
	if id1 == id2 {
		t.Errorf("Expected different IDs, got id1=%s, id2=%s", id1, id2)
	}

	// Test NewSpinnerWithID and GetSpinnerByID
	spinner1 := manager.NewSpinnerWithID(id1)
	spinner1.SetMessage("message1")

	retrieved, exists := manager.GetSpinnerByID(id1)
	if !exists {
		t.Error("Expected spinner to exist")
	}
	if retrieved != spinner1 {
		t.Error("Expected to retrieve the same spinner")
	}

	// Start spinner1 to make it active
	ctx := context.Background()
	spinner1.Start(ctx)

	// Test GetActiveSpinnerWithID returns the started spinner
	activeSpinner, activeID, exists := manager.GetActiveSpinnerWithID()
	if !exists || activeID != id1 {
		t.Errorf("Expected active spinner to be id1, got %s", activeID)
	}
	if activeSpinner != spinner1 {
		t.Error("Expected active spinner to be spinner1")
	}

	// Add and start second spinner, it becomes active
	spinner2 := manager.NewSpinnerWithID(id2)
	spinner2.SetMessage("message2")
	spinner2.Start(ctx)

	_, activeID, exists = manager.GetActiveSpinnerWithID()
	if !exists || activeID != id2 {
		t.Errorf("Expected active spinner to be id2, got %s", activeID)
	}

	// Test HasActiveSpinners
	if !manager.HasActiveSpinners() {
		t.Error("Expected HasActiveSpinners to return true")
	}

	// Test RemoveSpinner (stops and removes)
	manager.RemoveSpinner(id2)
	if !manager.HasActiveSpinners() {
		t.Error("Expected to have active spinners after removing id2")
	}

	// Active should now be id1 (spinner2 was removed)
	_, activeID, exists = manager.GetActiveSpinnerWithID()
	if !exists || activeID != id1 {
		t.Errorf("Expected active spinner to switch to id1, got %s", activeID)
	}

	// Remove last spinner
	manager.RemoveSpinner(id1)
	if manager.HasActiveSpinners() {
		t.Error("Expected HasActiveSpinners to return false after removing all spinners")
	}

	_, _, exists = manager.GetActiveSpinnerWithID()
	if exists {
		t.Error("Expected no active spinner")
	}
}

func TestManagedSpinnerBasic(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)
	spinner := manager.NewSpinnerWithID("test_id")
	spinner.SetMessage("Loading...")

	if spinner.ID() != "test_id" {
		t.Errorf("Expected id to be 'test_id', got %s", spinner.ID())
	}

	if spinner.IsRunning() {
		t.Error("Expected running to be false initially")
	}
}

func TestManagedSpinnerSetMessage(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)
	spinner := manager.NewSpinnerWithID("test_id")
	spinner.SetMessage("Initial")

	// Start and let it render
	ctx := context.Background()
	stop := spinner.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	stop()

	// Verify initial message was rendered
	output := buf.String()
	if !strings.Contains(output, "Initial") {
		t.Errorf("Expected output to contain 'Initial', got %s", output)
	}
}

func TestManagedSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)
	spinner := manager.NewSpinnerWithID("test_id")
	spinner.SetMessage("Loading...")

	// Start spinner
	ctx := context.Background()
	stop := spinner.Start(ctx)

	if !spinner.IsRunning() {
		t.Error("Expected running to be true after Start")
	}

	// Give spinner time to render
	time.Sleep(100 * time.Millisecond)

	// Stop spinner
	stop()

	if spinner.IsRunning() {
		t.Error("Expected running to be false after Stop")
	}

	// Verify something was written to the buffer
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected spinner to write output to buffer")
	}
}

// Helper to match the spinnerContext logic in the file
func getOrCreateSpinnerContext() (context.Context, context.CancelFunc) {
	// Create a background context with cancel
	return context.WithCancel(context.Background())
}

func TestSpinnerMultipleInstances(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)
	ctx := context.Background()

	id1 := manager.GenerateID()
	id2 := manager.GenerateID()
	id3 := manager.GenerateID()

	spinner1 := manager.NewSpinnerWithID(id1)
	spinner1.SetMessage("Task 1")
	spinner2 := manager.NewSpinnerWithID(id2)
	spinner2.SetMessage("Task 2")
	spinner3 := manager.NewSpinnerWithID(id3)
	spinner3.SetMessage("Task 3")

	// Start all spinners
	spinner1.Start(ctx)
	spinner2.Start(ctx)
	spinner3.Start(ctx)

	// Verify all spinners exist
	s1, exists1 := manager.GetSpinnerByID(id1)
	s2, exists2 := manager.GetSpinnerByID(id2)
	s3, exists3 := manager.GetSpinnerByID(id3)

	if !exists1 || !exists2 || !exists3 {
		t.Error("Expected all spinners to exist")
	}
	if s1 == nil || s2 == nil || s3 == nil {
		t.Error("Expected all spinners to be non-nil")
	}

	// Active should be id3 (most recently started based on creation order)
	_, activeID, _ := manager.GetActiveSpinnerWithID()
	if activeID != id3 {
		t.Errorf("Expected active to be id3, got %s", activeID)
	}

	// Remove middle spinner
	manager.RemoveSpinner(id2)

	// Spinners 1 and 3 should still exist
	_, exists1 = manager.GetSpinnerByID(id1)
	_, exists3 = manager.GetSpinnerByID(id3)
	_, exists2 = manager.GetSpinnerByID(id2)

	if !exists1 || !exists3 {
		t.Error("Expected spinners 1 and 3 to still exist")
	}

	if exists2 {
		t.Error("Expected spinner 2 to be removed")
	}

	// Active should still be id3
	_, activeID, _ = manager.GetActiveSpinnerWithID()
	if activeID != id3 {
		t.Errorf("Expected active to remain id3, got %s", activeID)
	}

	// Cleanup
	manager.RemoveSpinner(id1)
	manager.RemoveSpinner(id3)
}

// TestSpinnerCoordinatedRendering verifies that only one spinner renders at a time
// and that stopping a spinner causes the next most recent one to render
func TestSpinnerCoordinatedRendering(t *testing.T) {
	var buf bytes.Buffer
	manager := render.NewSpinnerManager(&buf)
	ctx := context.Background()

	// Create and start three spinners
	id1 := manager.GenerateID()
	id2 := manager.GenerateID()
	id3 := manager.GenerateID()

	spinner1 := manager.NewSpinnerWithID(id1)
	spinner1.SetMessage("First Task")
	spinner2 := manager.NewSpinnerWithID(id2)
	spinner2.SetMessage("Second Task")
	spinner3 := manager.NewSpinnerWithID(id3)
	spinner3.SetMessage("Third Task")

	// Start all spinners
	spinner1.Start(ctx)
	spinner2.Start(ctx)
	spinner3.Start(ctx)

	// The manager should have 3 live spinners
	if manager.LiveCount() != 3 {
		t.Errorf("Expected 3 live spinners, got %d", manager.LiveCount())
	}

	// Give time to render
	time.Sleep(100 * time.Millisecond)

	// Stop spinner3 - spinner2 should become active
	manager.RemoveSpinner(id3)

	if manager.LiveCount() != 2 {
		t.Errorf("Expected 2 live spinners after stopping spinner3, got %d", manager.LiveCount())
	}

	// Give time to render the newly active spinner
	time.Sleep(100 * time.Millisecond)

	// Stop spinner2 - spinner1 should become active
	manager.RemoveSpinner(id2)

	if manager.LiveCount() != 1 {
		t.Errorf("Expected 1 live spinner after stopping spinner2, got %d", manager.LiveCount())
	}

	// Stop spinner1 - no spinners left
	manager.RemoveSpinner(id1)

	if manager.LiveCount() != 0 {
		t.Errorf("Expected 0 live spinners after stopping all, got %d", manager.LiveCount())
	}

	// Verify output was generated (contains spinner messages)
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected spinner output to be generated")
	}

	// Should contain "Third Task" since that was the most recent and rendered first
	if !strings.Contains(output, "Third Task") {
		t.Error("Expected output to contain 'Third Task' (the most recent spinner)")
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name           string
		x              float64
		y              float64
		wantError      bool
		expectedOutput string
	}{
		{
			name:           "move up",
			x:              0,
			y:              -1,
			wantError:      false,
			expectedOutput: "\033[A",
		},
		{
			name:           "move down",
			x:              0,
			y:              1,
			wantError:      false,
			expectedOutput: "\033[B",
		},
		{
			name:           "move left",
			x:              -1,
			y:              0,
			wantError:      false,
			expectedOutput: "\033[D",
		},
		{
			name:           "move right",
			x:              1,
			y:              0,
			wantError:      false,
			expectedOutput: "\033[C",
		},
		{
			name:           "move up-left",
			x:              -2,
			y:              -3,
			wantError:      false,
			expectedOutput: "\033[A\033[A\033[A\033[D\033[D",
		},
		{
			name:           "move down-right",
			x:              2,
			y:              3,
			wantError:      false,
			expectedOutput: "\033[B\033[B\033[B\033[C\033[C",
		},
		{
			name:           "no movement",
			x:              0,
			y:              0,
			wantError:      false,
			expectedOutput: "",
		},
		{
			name:           "large movement up",
			x:              0,
			y:              -10,
			wantError:      false,
			expectedOutput: strings.Repeat("\033[A", 10),
		},
		{
			name:           "large movement right",
			x:              20,
			y:              0,
			wantError:      false,
			expectedOutput: strings.Repeat("\033[C", 20),
		},
		{
			name:           "diagonal movement",
			x:              5,
			y:              -5,
			wantError:      false,
			expectedOutput: strings.Repeat("\033[A", 5) + strings.Repeat("\033[C", 5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a mock cursor object
			cursorObj := &UICursorObjectValue{}

			// Get the moveCursor function
			moveCursorFn := cursorObj.GetProperty("moveCursor")
			if moveCursorFn == nil {
				os.Stdout = oldStdout
				t.Fatal("Expected moveCursor property to exist")
			}

			builtin, ok := moveCursorFn.(*BuiltinValue)
			if !ok {
				os.Stdout = oldStdout
				t.Fatal("Expected moveCursor to be a BuiltinValue")
			}

			// Call the function with x and y arguments
			args := []Value{
				&NumberValue{Value: tt.x},
				&NumberValue{Value: tt.y},
			}

			result, err := builtin.Fn(args)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if (err != nil) != tt.wantError {
				t.Errorf("moveCursor() error = %v, wantError %v", err, tt.wantError)
			}

			// Result should always be NullValue
			if _, ok := result.(*NullValue); !ok {
				t.Errorf("Expected NullValue result, got %T", result)
			}

			// Verify the output matches expected ANSI sequences
			if output != tt.expectedOutput {
				t.Errorf("moveCursor() output mismatch\nexpected: %q\ngot:      %q", tt.expectedOutput, output)
			}
		})
	}
}

func TestMoveCursorArgumentValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []Value
		wantError bool
		errorMsg  string
	}{
		{
			name:      "no arguments",
			args:      []Value{},
			wantError: true,
			errorMsg:  "takes 2 arguments",
		},
		{
			name:      "one argument",
			args:      []Value{&NumberValue{Value: 1}},
			wantError: true,
			errorMsg:  "takes 2 arguments",
		},
		{
			name:      "three arguments",
			args:      []Value{&NumberValue{Value: 1}, &NumberValue{Value: 2}, &NumberValue{Value: 3}},
			wantError: true,
			errorMsg:  "takes 2 arguments",
		},
		{
			name:      "first argument not a number",
			args:      []Value{&StringValue{Value: "not a number"}, &NumberValue{Value: 1}},
			wantError: true,
			errorMsg:  "first argument must be a number",
		},
		{
			name:      "second argument not a number",
			args:      []Value{&NumberValue{Value: 1}, &StringValue{Value: "not a number"}},
			wantError: true,
			errorMsg:  "second argument must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursorObj := &UICursorObjectValue{}
			moveCursorFn := cursorObj.GetProperty("moveCursor")
			builtin, _ := moveCursorFn.(*BuiltinValue)

			_, err := builtin.Fn(tt.args)

			if (err != nil) != tt.wantError {
				t.Errorf("moveCursor() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err != nil {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errorMsg)) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			}
		})
	}
}
