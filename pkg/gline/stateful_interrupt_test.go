package gline

import (
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// TestStatefulInterruptBehavior reproduces the bug where interrupted state
// persists across multiple gline calls, causing subsequent calls to incorrectly
// return ErrInterrupted even when the user didn't press Ctrl+C
func TestStatefulInterruptBehavior(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	options := NewOptions()

	// First call: simulate a normal successful interaction
	t.Run("First call should work normally", func(t *testing.T) {
		// Create a mock that simulates normal user input
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Verify initial state
		if model.interrupted {
			t.Error("Initial model should not be interrupted")
		}
		if model.appState != Active {
			t.Errorf("Expected Active state, got %v", model.appState)
		}
	})

	// Second call: simulate Ctrl+C interruption
	t.Run("Second call with Ctrl+C should be interrupted", func(t *testing.T) {
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Simulate Ctrl+C by sending interruptMsg
		updatedModel, _ := model.Update(interruptMsg{})
		appModel := updatedModel.(appModel)

		// Verify the model is now interrupted
		if !appModel.interrupted {
			t.Error("Model should be interrupted after interruptMsg")
		}
		if appModel.appState != Terminated {
			t.Errorf("Expected Terminated state, got %v", appModel.appState)
		}
	})

	// Third call: this should work normally but might fail due to stateful bug
	t.Run("Third call should work normally again", func(t *testing.T) {
		// This is where the bug would manifest - if there's global state,
		// this new model might incorrectly inherit the interrupted state
		model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)

		// Verify fresh state
		if model.interrupted {
			t.Error("Fresh model should not be interrupted - this indicates stateful bug")
		}
		if model.appState != Active {
			t.Errorf("Expected Active state, got %v", model.appState)
		}
	})
}

// TestMultipleGlineCallsWithInterruption tests the actual Gline function
// to see if interrupted state persists across calls
func TestMultipleGlineCallsWithInterruption(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	options := NewOptions()

	type testRun struct {
		name string
		msg  tea.Msg
	}

	cases := []testRun{
		{name: "normal run", msg: tea.KeyMsg{Type: tea.KeyCtrlD}},
		{name: "interrupted run", msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
		{name: "subsequent normal run", msg: tea.KeyMsg{Type: tea.KeyCtrlD}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each run should start from a clean, active model state.
			initial := initialModel("test> ", []string{}, "", nil, nil, nil, logger, options)
			if initial.appState != Active {
				t.Fatalf("expected initial model to be Active, got %v", initial.appState)
			}

			model, err := runProgramWithMessage(initial, tc.msg)
			if err != nil {
				t.Fatalf("program returned error: %v", err)
			}

			if model.appState != Terminated {
				t.Fatalf("expected program to terminate, got state %v", model.appState)
			}

			switch tc.msg.(type) {
			case tea.KeyMsg:
				km := tc.msg.(tea.KeyMsg)
				if km.Type == tea.KeyCtrlC {
					if !model.interrupted {
						t.Fatalf("expected interrupted state after Ctrl+C")
					}
				} else {
					if model.interrupted {
						t.Fatalf("unexpected interrupted state for %v run", tc.name)
					}
				}
			default:
				if model.interrupted {
					t.Fatalf("unexpected interrupted state for %v run", tc.name)
				}
			}
		})
	}
}

// runProgramWithMessage executes a Bubble Tea program using the provided initial model and
// sends a single message to drive it to completion.
func runProgramWithMessage(initial appModel, msg tea.Msg) (appModel, error) {
	p := tea.NewProgram(
		initial,
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		// Disable Bubble Tea's attempt to acquire a TTY for stdin.
		// Using nil keeps the input type "custom" and bypasses the
		// /dev/tty fallback that doesn't exist in CI.
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)

	var (
		model tea.Model
		err   error
	)

	done := make(chan struct{})
	go func() {
		model, err = p.Run()
		close(done)
	}()

	// Give the program a moment to initialize before sending the message.
	time.Sleep(10 * time.Millisecond)
	p.Send(msg)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		// Ensure the program exits even if the message was not processed.
		p.Quit()
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			return appModel{}, err
		}
	}

	resultModel, ok := model.(appModel)
	if !ok {
		return appModel{}, err
	}

	return resultModel, err
}
