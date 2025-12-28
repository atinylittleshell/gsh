package interpreter

import (
	"testing"
)

// TestGshReplNull tests that gsh.repl is null when no REPL context is set
func TestGshReplNull(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Without setting REPL context, gsh.repl should be null
	result, err := interp.EvalString(`gsh.repl == null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be true (gsh.repl is null)
	if boolVal, ok := result.FinalResult.(*BoolValue); ok {
		if !boolVal.Value {
			t.Errorf("expected true, got false")
		}
	} else {
		t.Errorf("expected bool, got %s", result.FinalResult.Type())
	}
}

// TestGshReplModels tests that gsh.repl.models is accessible when REPL context is set
func TestGshReplModels(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context with model tiers
	replCtx := &REPLContext{
		Models: &REPLModels{
			Lite: &ModelValue{
				Name: "lite",
				Config: map[string]Value{
					"model": &StringValue{Value: "gpt-4-mini"},
				},
			},
			Workhorse: &ModelValue{
				Name: "workhorse",
				Config: map[string]Value{
					"model": &StringValue{Value: "gpt-4"},
				},
			},
			Premium: &ModelValue{
				Name: "premium",
				Config: map[string]Value{
					"model": &StringValue{Value: "gpt-4-turbo"},
				},
			},
		},
		LastCommand: &REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test accessing gsh.repl.models.lite
	result, err := interp.EvalString(`gsh.repl.models.lite.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "lite" {
			t.Errorf("expected 'lite', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}

	// Test accessing gsh.repl.models.workhorse
	result, err = interp.EvalString(`gsh.repl.models.workhorse.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "workhorse" {
			t.Errorf("expected 'workhorse', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}

	// Test accessing gsh.repl.models.premium
	result, err = interp.EvalString(`gsh.repl.models.premium.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "premium" {
			t.Errorf("expected 'premium', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}
}

// TestGshReplLastCommand tests that gsh.repl.lastCommand is accessible
func TestGshReplLastCommand(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context
	replCtx := &REPLContext{
		Models: &REPLModels{},
		LastCommand: &REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test initial values
	result, err := interp.EvalString(`gsh.repl.lastCommand.exitCode`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 0 {
			t.Errorf("expected 0, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Update lastCommand through SDKConfig
	interp.SDKConfig().UpdateLastCommand(42, 1500)

	// Test updated values
	result, err = interp.EvalString(`gsh.repl.lastCommand.exitCode`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 42 {
			t.Errorf("expected 42, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Test durationMs
	result, err = interp.EvalString(`gsh.repl.lastCommand.durationMs`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 1500 {
			t.Errorf("expected 1500, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}
}

// TestGshEventHandlers tests that event handlers can be registered and retrieved
func TestGshEventHandlers(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Register an event handler using gsh.on
	result, err := interp.EvalString("tool myHandler() { return \"handler called\" }")
	if err != nil {
		t.Fatalf("unexpected error registering tool: %v", err)
	}

	result, err = interp.EvalString("gsh.on(\"test.event\", myHandler)")
	if err != nil {
		t.Fatalf("unexpected error calling gsh.on: %v", err)
	}

	// Result should be a string (handler ID)
	if _, ok := result.FinalResult.(*StringValue); !ok {
		t.Errorf("expected string (handler ID), got %s", result.FinalResult.Type())
	}

	// Verify the handler was registered
	handlers := interp.GetEventHandlers("test.event")
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}
}

// TestGshOnWithoutHandler tests gsh.on error handling
func TestGshOnWithoutHandler(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Try to register a non-tool as handler (should fail)
	_, err := interp.EvalString(`gsh.on("test.event", "not a tool")`)
	if err == nil {
		t.Fatal("expected error when passing non-tool to gsh.on")
	}
}

// TestGshOffRemovesAllHandlers tests that gsh.off without handlerID removes all handlers
func TestGshOffRemovesAllHandlers(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Register multiple handlers
	_, err := interp.EvalString(`
		tool handler1() { return "handler1" }
		tool handler2() { return "handler2" }
		gsh.on("test.event", handler1)
		gsh.on("test.event", handler2)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both handlers are registered
	handlers := interp.GetEventHandlers("test.event")
	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}

	// Remove all handlers
	_, err = interp.EvalString(`gsh.off("test.event")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all handlers are removed
	handlers = interp.GetEventHandlers("test.event")
	if len(handlers) != 0 {
		t.Errorf("expected 0 handlers after gsh.off without handlerID, got %d", len(handlers))
	}
}

// TestGshReplReadOnly tests that gsh.repl properties are read-only
func TestGshReplReadOnly(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context
	replCtx := &REPLContext{
		Models:      &REPLModels{},
		LastCommand: &REPLLastCommand{},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Try to modify gsh.repl (should fail)
	_, err := interp.EvalString(`gsh.repl = "something"`)
	if err == nil {
		t.Fatal("expected error when trying to assign to gsh.repl")
	}
}
