package interpreter

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSpinnerManager(t *testing.T) {
	manager := NewSpinnerManager()

	// Test GenerateID
	id1 := manager.GenerateID()
	id2 := manager.GenerateID()
	if id1 == id2 {
		t.Errorf("Expected different IDs, got id1=%s, id2=%s", id1, id2)
	}

	// Test AddSpinner and GetSpinner
	var buf bytes.Buffer
	spinner1 := NewUISpinner(id1, "message1", &buf)
	manager.AddSpinner(id1, spinner1)

	retrieved, exists := manager.GetSpinner(id1)
	if !exists {
		t.Error("Expected spinner to exist")
	}
	if retrieved != spinner1 {
		t.Error("Expected to retrieve the same spinner")
	}

	// Test AddSpinner sets active spinner
	_, activeID, exists := manager.GetActiveSpinner()
	if !exists || activeID != id1 {
		t.Errorf("Expected active spinner to be id1, got %s", activeID)
	}

	// Add second spinner, it becomes active
	spinner2 := NewUISpinner(id2, "message2", &buf)
	manager.AddSpinner(id2, spinner2)

	_, activeID, exists = manager.GetActiveSpinner()
	if !exists || activeID != id2 {
		t.Errorf("Expected active spinner to be id2, got %s", activeID)
	}

	// Test HasActiveSpinners
	if !manager.HasActiveSpinners() {
		t.Error("Expected HasActiveSpinners to return true")
	}

	// Test RemoveSpinner
	manager.RemoveSpinner(id2)
	if !manager.HasActiveSpinners() {
		t.Error("Expected to have active spinners after removing id2")
	}

	// Active should now be id1
	_, activeID, exists = manager.GetActiveSpinner()
	if !exists || activeID != id1 {
		t.Errorf("Expected active spinner to switch to id1, got %s", activeID)
	}

	// Remove last spinner
	manager.RemoveSpinner(id1)
	if manager.HasActiveSpinners() {
		t.Error("Expected HasActiveSpinners to return false after removing all spinners")
	}

	_, _, exists = manager.GetActiveSpinner()
	if exists {
		t.Error("Expected no active spinner")
	}
}

func TestUISpinnerBasic(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewUISpinner("test_id", "Loading...", &buf)

	if spinner.id != "test_id" {
		t.Errorf("Expected id to be 'test_id', got %s", spinner.id)
	}

	if spinner.message != "Loading..." {
		t.Errorf("Expected message to be 'Loading...', got %s", spinner.message)
	}

	if spinner.running {
		t.Error("Expected running to be false initially")
	}
}

func TestUISpinnerSetMessage(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewUISpinner("test_id", "Initial", &buf)

	spinner.SetMessage("Updated")
	if spinner.message != "Updated" {
		t.Errorf("Expected message to be 'Updated', got %s", spinner.message)
	}
}

func TestUISpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewUISpinner("test_id", "Loading...", &buf)

	// Start spinner
	spinnerContext, _ := getOrCreateSpinnerContext()
	spinner.Start(spinnerContext)

	if !spinner.running {
		t.Error("Expected running to be true after Start")
	}

	// Give spinner time to render
	time.Sleep(100 * time.Millisecond)

	// Stop spinner
	spinner.Stop()
	time.Sleep(50 * time.Millisecond)

	if spinner.running {
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
	manager := NewSpinnerManager()
	var buf1, buf2, buf3 bytes.Buffer

	id1 := manager.GenerateID()
	id2 := manager.GenerateID()
	id3 := manager.GenerateID()

	spinner1 := NewUISpinner(id1, "Task 1", &buf1)
	spinner2 := NewUISpinner(id2, "Task 2", &buf2)
	spinner3 := NewUISpinner(id3, "Task 3", &buf3)

	manager.AddSpinner(id1, spinner1)
	manager.AddSpinner(id2, spinner2)
	manager.AddSpinner(id3, spinner3)

	// Verify all spinners exist
	s1, _ := manager.GetSpinner(id1)
	s2, _ := manager.GetSpinner(id2)
	s3, _ := manager.GetSpinner(id3)

	if s1 == nil || s2 == nil || s3 == nil {
		t.Error("Expected all spinners to exist")
	}

	// Active should be id3 (most recently added)
	_, activeID, _ := manager.GetActiveSpinner()
	if activeID != id3 {
		t.Errorf("Expected active to be id3, got %s", activeID)
	}

	// Remove middle spinner
	manager.RemoveSpinner(id2)

	// Spinners 1 and 3 should still exist
	s1, exists1 := manager.GetSpinner(id1)
	s3, exists3 := manager.GetSpinner(id3)
	s2, exists2 := manager.GetSpinner(id2)

	if !exists1 || !exists3 {
		t.Error("Expected spinners 1 and 3 to still exist")
	}

	if exists2 {
		t.Error("Expected spinner 2 to be removed")
	}

	// Active should still be id3
	_, activeID, _ = manager.GetActiveSpinner()
	if activeID != id3 {
		t.Errorf("Expected active to remain id3, got %s", activeID)
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
