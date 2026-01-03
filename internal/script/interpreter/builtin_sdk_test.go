package interpreter

import (
	"testing"
)

// TestGshModelsAvailableInScriptMode tests that gsh.models is available even without REPL context
func TestGshModelsAvailableInScriptMode(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// gsh.models should be available in both REPL and script mode (not null)
	result, err := interp.EvalString(`gsh.models != null`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be true (gsh.models is always initialized)
	if boolVal, ok := result.FinalResult.(*BoolValue); ok {
		if !boolVal.Value {
			t.Errorf("expected gsh.models to be available in script mode, but it was null")
		}
	} else {
		t.Errorf("expected bool, got %s", result.FinalResult.Type())
	}

	// Model tiers should be null by default (not yet assigned)
	result, err = interp.EvalString(`gsh.models.lite == null`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); ok {
		if !boolVal.Value {
			t.Errorf("expected gsh.models.lite to be null by default")
		}
	} else {
		t.Errorf("expected bool, got %s", result.FinalResult.Type())
	}
}

// TestGshReplModels tests that gsh.models is accessible and settable when REPL context is set
func TestModelsObjectValue_PointerIdentity(t *testing.T) {
	// Test that repeated access to gsh.models.lite returns the same SDKModelRef instance
	models := &Models{
		Lite:      &ModelValue{Name: "liteModel"},
		Workhorse: &ModelValue{Name: "workhorseModel"},
	}
	modelsObj := NewModelsObjectValue(models)

	// Multiple accesses should return the same pointer
	ref1 := modelsObj.GetProperty("lite")
	ref2 := modelsObj.GetProperty("lite")

	if ref1 != ref2 {
		t.Error("expected same SDKModelRef instance on repeated access")
	}

	// Different tiers should return different pointers
	workhorseRef := modelsObj.GetProperty("workhorse")
	if ref1 == workhorseRef {
		t.Error("expected different SDKModelRef instances for different tiers")
	}

	// Verify they're SDKModelRef
	sdkRef1, ok := ref1.(*SDKModelRef)
	if !ok {
		t.Fatalf("expected *SDKModelRef, got %T", ref1)
	}
	if sdkRef1.Tier != "lite" {
		t.Errorf("expected tier 'lite', got %q", sdkRef1.Tier)
	}
}

func TestGshReplModels(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Models are initialized in SDKConfig (available in both REPL and script mode)
	// They start as nil by default
	models := interp.SDKConfig().GetModels()
	models.Lite = nil
	models.Workhorse = nil
	models.Premium = nil

	// Test that model tiers start as null
	result, err := interp.EvalString(`gsh.models.lite == null`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected gsh.models.lite to be null initially")
	}

	// Define a model and assign it to gsh.models.lite
	_, err = interp.EvalString(`
model testLite {
	provider: "openai",
	model: "gpt-4-mini",
}
gsh.models.lite = testLite
`, nil)
	if err != nil {
		t.Fatalf("unexpected error setting lite model: %v", err)
	}

	// Test accessing gsh.models.lite.name after assignment
	result, err = interp.EvalString(`gsh.models.lite.name`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "testLite" {
			t.Errorf("expected 'testLite', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}

	// Test that workhorse and premium are still null
	result, err = interp.EvalString(`gsh.models.workhorse == null`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected gsh.models.workhorse to still be null")
	}

	// Test assigning a non-model value should fail
	_, err = interp.EvalString(`gsh.models.premium = "not a model"`, nil)
	if err == nil {
		t.Fatal("expected error when assigning non-model to gsh.models.premium")
	}
}

// TestGshReplLastCommand tests that gsh.lastCommand is accessible
func TestGshReplLastCommand(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context (lastCommand is REPL-specific)
	replCtx := &REPLContext{
		LastCommand: &REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test initial values
	result, err := interp.EvalString(`gsh.lastCommand.exitCode`, nil)
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
	result, err = interp.EvalString(`gsh.lastCommand.exitCode`, nil)
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
	result, err = interp.EvalString(`gsh.lastCommand.durationMs`, nil)
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

	// Register an event handler using gsh.use (middleware signature)
	result, err := interp.EvalString("tool myHandler(ctx, next) { return next(ctx) }", nil)
	if err != nil {
		t.Fatalf("unexpected error registering tool: %v", err)
	}

	result, err = interp.EvalString(`gsh.use("test.event", myHandler)`, nil)
	if err != nil {
		t.Fatalf("unexpected error calling gsh.use: %v", err)
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
func TestGshUseWithoutHandler(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Try to register a non-tool as handler (should fail)
	_, err := interp.EvalString(`gsh.use("test.event", "not a tool")`, nil)
	if err == nil {
		t.Fatal("expected error when passing non-tool to gsh.use")
	}
}

// TestGshOffRemovesAllHandlers tests that gsh.off without handlerID removes all handlers
func TestGshRemoveHandler(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Register a handler (middleware signature)
	_, err := interp.EvalString(`
		tool handler1(ctx, next) { return next(ctx) }
		gsh.use("test.event", handler1)
	`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify handler is registered
	handlers := interp.GetEventHandlers("test.event")
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}

	// Remove handler by reference
	result, err := interp.EvalString(`gsh.remove("test.event", handler1)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return true
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected true from gsh.remove(), got %v", result.FinalResult)
	}

	// Verify handler is removed
	handlers = interp.GetEventHandlers("test.event")
	if len(handlers) != 0 {
		t.Errorf("expected 0 handlers after gsh.remove, got %d", len(handlers))
	}
}

// TestGshVersionReadOnly tests that gsh.version is read-only
func TestGshVersionReadOnly(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Try to modify gsh.version (should fail)
	_, err := interp.EvalString(`gsh.version = "hacked"`, nil)
	if err == nil {
		t.Fatal("expected error when trying to assign to gsh.version")
	}
}

func TestGshUseCommandInput(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Define a middleware tool for command.input event
	_, err := interp.EvalString(`
tool testMiddleware(ctx, next) {
	return { handled: true }
}
`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Register middleware using gsh.use("command.input", ...)
	result, err := interp.EvalString(`gsh.use("command.input", testMiddleware)`, nil)
	if err != nil {
		t.Fatalf("unexpected error calling gsh.use(): %v", err)
	}

	// Should return a string ID
	if _, ok := result.FinalResult.(*StringValue); !ok {
		t.Errorf("expected string ID from gsh.use(), got %s", result.FinalResult.Type())
	}

	// Verify middleware was registered
	handlers := interp.GetEventHandlers("command.input")
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler registered, got %d", len(handlers))
	}
}

func TestGshUseMiddlewareValidation(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Test that non-tool argument fails
	_, err := interp.EvalString(`gsh.use("command.input", "not a tool")`, nil)
	if err == nil {
		t.Fatal("expected error when passing non-tool to gsh.use()")
	}

	// Test that wrong parameter count fails
	_, err = interp.EvalString(`
tool wrongParams(ctx) {
	return { handled: true }
}
gsh.use("command.input", wrongParams)
`, nil)
	if err == nil {
		t.Fatal("expected error when middleware has wrong parameter count")
	}
}

func TestGshRemoveMiddleware(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Define and register middleware
	_, err := interp.EvalString(`
tool myMiddleware(ctx, next) {
	return { handled: true }
}
gsh.use("command.input", myMiddleware)
`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify registered
	handlers := interp.GetEventHandlers("command.input")
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler registered, got %d", len(handlers))
	}

	// Remove middleware using gsh.remove()
	result, err := interp.EvalString(`gsh.remove("command.input", myMiddleware)`, nil)
	if err != nil {
		t.Fatalf("unexpected error calling gsh.remove(): %v", err)
	}

	// Should return true
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected true from gsh.remove(), got %v", result.FinalResult)
	}

	// Verify middleware was removed
	handlers = interp.GetEventHandlers("command.input")
	if len(handlers) != 0 {
		t.Errorf("expected 0 handlers after removal, got %d", len(handlers))
	}

	// Removing again should return false
	result, err = interp.EvalString(`gsh.remove("command.input", myMiddleware)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || boolVal.Value {
		t.Errorf("expected false from second gsh.remove(), got %v", result.FinalResult)
	}
}

func TestGshMiddlewareChainExecution(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Define middleware that handles # prefix
	_, err := interp.EvalString(`
tool prefixMiddleware(ctx, next) {
	if (ctx.input.startsWith("#")) {
		return { handled: true }
	}
	return next(ctx)
}
gsh.use("command.input", prefixMiddleware)
`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test middleware chain execution via EmitEvent
	inputCtx := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"input": {Value: &StringValue{Value: "# hello"}},
		},
	}
	result := interp.EmitEvent("command.input", inputCtx)

	// Input with # should return handled: true
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if obj, ok := result.(*ObjectValue); ok {
		if handledVal := obj.GetPropertyValue("handled"); handledVal != nil {
			if bv, ok := handledVal.(*BoolValue); !ok || !bv.Value {
				t.Error("expected handled: true")
			}
		} else {
			t.Error("expected handled property in result")
		}
	} else {
		t.Errorf("expected ObjectValue result, got %T", result)
	}

	// Input without # should fall through (return nil)
	inputCtx2 := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"input": {Value: &StringValue{Value: "echo hello"}},
		},
	}
	result2 := interp.EmitEvent("command.input", inputCtx2)
	if result2 != nil {
		t.Errorf("expected nil result for fall-through, got %v", result2)
	}
}
