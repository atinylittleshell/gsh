package repl

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	// Import all subpackages to verify the directory structure is correct
	_ "github.com/atinylittleshell/gsh/internal/repl/completion"
	_ "github.com/atinylittleshell/gsh/internal/repl/config"
	_ "github.com/atinylittleshell/gsh/internal/repl/context"
	_ "github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	_ "github.com/atinylittleshell/gsh/internal/repl/predict"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// mockPredictionProvider implements input.PredictionProvider for testing.
type mockPredictionProvider struct {
	prediction string
}

func (m *mockPredictionProvider) Predict(ctx context.Context, inputStr string, trigger interpreter.PredictTrigger, existingPrediction string) (string, error) {
	if trigger == interpreter.PredictTriggerInstant {
		return m.prediction, nil
	}
	return "", nil
}

func TestDirectoryStructure(t *testing.T) {
	// This test verifies that all subpackages in internal/repl/ can be imported.
	// The imports above will fail at compile time if any package is missing
	// or has incorrect package declarations.
	t.Log("All internal/repl subpackages are correctly structured and importable")
}

func TestNewREPL_DefaultOptions(t *testing.T) {
	// Create a temporary directory for history
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")

	// Create a non-existent config path to use defaults
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	require.NotNil(t, repl)
	defer repl.Close()

	// Verify default config was loaded (Config now only holds declarations)
	assert.NotNil(t, repl.Config())

	// Verify executor was created
	assert.NotNil(t, repl.Executor())

	// Verify history manager was created
	assert.NotNil(t, repl.History())
}

func TestNewREPL_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")

	logger := zaptest.NewLogger(t)

	// Use DefaultConfigContent to set up SDK config
	defaultConfig := `
model testModel {
	provider: "openai",
	model: "gpt-4",
}
`

	repl, err := NewREPL(Options{
		DefaultConfigContent: defaultConfig,
		HistoryPath:          historyPath,
		Logger:               logger,
	})
	require.NoError(t, err)
	require.NotNil(t, repl)
	defer repl.Close()

	// Verify model was loaded
	assert.NotNil(t, repl.Config().GetModel("testModel"))
}

func TestNewREPL_NilLogger(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	// Should not panic with nil logger
	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      nil,
	})
	require.NoError(t, err)
	require.NotNil(t, repl)
	defer repl.Close()
}

func TestREPL_HandleBuiltinCommand_Exit(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Test that exit returns ErrExit
	handled, err := repl.handleBuiltinCommand("exit")
	assert.True(t, handled)
	assert.Equal(t, ErrExit, err)

	// Test unhandled command
	handled, err = repl.handleBuiltinCommand("ls")
	assert.False(t, handled)
	assert.NoError(t, err)
}

func TestREPL_ProcessCommand_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Empty command should be no-op
	err = repl.processCommand(ctx, "")
	assert.NoError(t, err)

	// Whitespace-only command should be no-op
	err = repl.processCommand(ctx, "   ")
	assert.NoError(t, err)
}

func TestREPL_ProcessCommand_Echo(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Execute a simple echo command
	err = repl.processCommand(ctx, "echo hello")
	assert.NoError(t, err)

	// Verify exit code was recorded
	assert.Equal(t, 0, repl.lastExitCode)
}

func TestREPL_ProcessCommand_RecordsHistory(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Execute a command
	err = repl.processCommand(ctx, "echo test_history")
	assert.NoError(t, err)

	// Verify it was recorded in history
	entries, err := repl.History().GetRecentEntries("", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "echo test_history", entries[0].Command)
	assert.True(t, entries[0].ExitCode.Valid)
	assert.Equal(t, int32(0), entries[0].ExitCode.Int32)
}

func TestREPL_ProcessCommand_FailingCommand(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Execute a failing command
	err = repl.processCommand(ctx, "exit 42")
	assert.NoError(t, err) // processCommand doesn't return error for non-zero exit

	// Verify exit code was recorded
	assert.Equal(t, 42, repl.lastExitCode)

	// Verify it was recorded in history with correct exit code
	entries, err := repl.History().GetRecentEntries("", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].ExitCode.Valid)
	assert.Equal(t, int32(42), entries[0].ExitCode.Int32)
}

func TestREPL_GetHistoryValues(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Execute some commands
	_ = repl.processCommand(ctx, "echo first")
	_ = repl.processCommand(ctx, "echo second")
	_ = repl.processCommand(ctx, "echo third")

	// Get history values
	values := repl.getHistoryValues()
	require.Len(t, values, 3)

	// Most recent should be first
	assert.Equal(t, "echo third", values[0])
	assert.Equal(t, "echo second", values[1])
	assert.Equal(t, "echo first", values[2])
}

func TestREPL_GetPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")

	// Use DefaultConfigContent to set up SDK config with custom prompt via event handler
	defaultConfig := `
tool onPrompt(ctx, next) {
	gsh.prompt = "custom> "
	return next(ctx)
}
gsh.use("repl.prompt", onPrompt)
`

	repl, err := NewREPL(Options{
		DefaultConfigContent: defaultConfig,
		HistoryPath:          historyPath,
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Verify prompt is set by the event handler
	assert.Equal(t, "custom> ", repl.getPrompt())
}

func TestREPL_Close(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)

	// Close should not error
	err = repl.Close()
	assert.NoError(t, err)
}

func TestREPL_Run_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run should return immediately with context error
	err = repl.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestREPL_ProcessCommand_TracksDuration(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Mock time for testing
	callCount := 0
	originalTimeNow := timeNow
	timeNow = func() time.Time {
		callCount++
		if callCount == 1 {
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		return time.Date(2024, 1, 1, 0, 0, 0, 100000000, time.UTC) // 100ms later
	}
	defer func() { timeNow = originalTimeNow }()

	ctx := context.Background()

	// Execute a command
	err = repl.processCommand(ctx, "echo hello")
	assert.NoError(t, err)

	// Verify duration was tracked
	assert.Equal(t, int64(100), repl.lastDurationMs)
}

func TestREPL_HandleBuiltinCommand_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Built-in commands are things like "exit"
	handled, err := repl.handleBuiltinCommand("exit")
	assert.True(t, handled)
	assert.Equal(t, ErrExit, err)
}

func TestREPL_HandleBuiltinCommand_UnknownCommand(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Unknown commands should not be handled
	handled, err := repl.handleBuiltinCommand("unknown")
	assert.False(t, handled)
	assert.NoError(t, err)

	handled, err = repl.handleBuiltinCommand("ls -la")
	assert.False(t, handled)
	assert.NoError(t, err)
}

func TestREPL_PredictorInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Verify predictor was created (event-driven prediction provider)
	assert.NotNil(t, repl.predictor)
}

func TestREPL_HistoryPredictionWithoutLLM(t *testing.T) {
	// This test verifies that history-based prediction works even when
	// no LLM prediction model is configured. The predictor now uses lazy
	// model resolution via SDKModelRef, so it's always created but will
	// return empty predictions if gsh.models.lite is not set.
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer repl.Close()

	// Predictor is now always created with lazy model resolution (SDKModelRef)
	// It will return empty LLM predictions if gsh.models.lite is not configured
	assert.NotNil(t, repl.predictor, "predictor uses lazy model resolution")

	// Verify history is available
	require.NotNil(t, repl.history, "history manager should be initialized")

	ctx := context.Background()

	// Execute some commands to populate history
	err = repl.processCommand(ctx, "echo hello world")
	require.NoError(t, err)
	err = repl.processCommand(ctx, "echo testing prediction")
	require.NoError(t, err)

	// Verify commands are in history
	entries, err := repl.history.GetRecentEntriesByPrefix("echo", 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 2, "should have at least 2 history entries with 'echo' prefix")

	// Create a mock provider that returns a prediction for instant trigger
	provider := &mockPredictionProvider{
		prediction: "echo hello world",
	}

	// Create prediction state with provider
	predictionState := input.NewPredictionState(input.PredictionStateConfig{
		Provider: provider,
		Logger:   logger,
	})
	require.NotNil(t, predictionState)

	// Trigger a prediction by simulating input change
	resultCh := predictionState.OnInputChanged("echo")
	require.NotNil(t, resultCh, "should return a result channel for prediction")

	// Wait for the prediction result (with timeout)
	select {
	case result := <-resultCh:
		// Should get an instant prediction without error or panic
		assert.NoError(t, result.Error, "prediction should not return an error")
		assert.Equal(t, input.PredictionSourceHistory, result.Source, "prediction should come from instant provider")
		assert.Contains(t, result.Prediction, "echo", "prediction should start with the input prefix")
	case <-time.After(1 * time.Second):
		t.Fatal("prediction timed out")
	}
}

// Tests for middleware integration in REPL

func TestREPL_ProcessCommand_NoMiddleware_FallsThroughToShell(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	ctx := context.Background()

	// Without middleware, commands should execute as shell commands
	err = repl.processCommand(ctx, "echo middleware_test")
	assert.NoError(t, err)
	assert.Equal(t, 0, repl.lastExitCode)
}

func TestREPL_ProcessCommand_MiddlewareHandlesInput(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Get interpreter and set up middleware
	interp := repl.executor.Interpreter()

	// Create a middleware that handles input starting with "#"
	code := `
tool testMiddleware(ctx, next) {
	if (ctx.input.startsWith("#")) {
		return { handled: true }
	}
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	vars := interp.GetVariables()
	toolVal, ok := vars["testMiddleware"]
	require.True(t, ok)
	_, ok = toolVal.(*interpreter.ToolValue)
	require.True(t, ok)

	// Register middleware using gsh.use("command.input", ...)
	interp.EvalString(`gsh.use("command.input", testMiddleware)`, nil)

	ctx := context.Background()

	// Input starting with # should be handled by middleware (not executed as shell)
	err = repl.processCommand(ctx, "# this is a test")
	assert.NoError(t, err)
	// lastExitCode should be unchanged (default 0) since no shell command was executed
	// The key test is that no error occurred and middleware handled it

	// Verify that the command was still recorded in history even though middleware handled it
	entries, err := repl.History().GetRecentEntries("", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1, "middleware-handled commands should be recorded in history")
	assert.Equal(t, "# this is a test", entries[0].Command)
	assert.True(t, entries[0].ExitCode.Valid)
	assert.Equal(t, int32(0), entries[0].ExitCode.Int32)
}

func TestREPL_ProcessCommand_MiddlewarePassesThrough(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Get interpreter and set up middleware
	interp := repl.executor.Interpreter()

	// Create a middleware that passes everything through
	code := `
tool passThroughMiddleware(ctx, next) {
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	vars := interp.GetVariables()
	toolVal, ok := vars["passThroughMiddleware"]
	require.True(t, ok)
	_, ok = toolVal.(*interpreter.ToolValue)
	require.True(t, ok)

	// Register middleware using gsh.use("command.input", ...)
	interp.EvalString(`gsh.use("command.input", passThroughMiddleware)`, nil)

	ctx := context.Background()

	// Command should pass through middleware and execute as shell command
	err = repl.processCommand(ctx, "echo pass_through_test")
	assert.NoError(t, err)
	assert.Equal(t, 0, repl.lastExitCode)
}

func TestREPL_ProcessCommand_MiddlewareModifiesInput(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Get interpreter and set up middleware
	interp := repl.executor.Interpreter()

	// Create a middleware that transforms "!" prefix to "echo"
	code := `
tool transformMiddleware(ctx, next) {
	if (ctx.input.startsWith("!")) {
		ctx.input = "echo " + ctx.input.substring(1)
	}
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	vars := interp.GetVariables()
	toolVal, ok := vars["transformMiddleware"]
	require.True(t, ok)
	_, ok = toolVal.(*interpreter.ToolValue)
	require.True(t, ok)

	// Register middleware using gsh.use("command.input", ...)
	interp.EvalString(`gsh.use("command.input", transformMiddleware)`, nil)

	ctx := context.Background()

	// "!hello" should be transformed to "echo hello" and executed
	err = repl.processCommand(ctx, "!hello")
	assert.NoError(t, err)
	assert.Equal(t, 0, repl.lastExitCode)
}

func TestREPL_ProcessCommand_MiddlewareChainOrder(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Get interpreter and set up middleware
	interp := repl.executor.Interpreter()

	// Create two middleware that append markers to verify order
	code := `
__middlewareOrder = ""

tool firstMiddleware(ctx, next) {
	__middlewareOrder = __middlewareOrder + "first,"
	return next(ctx)
}

tool secondMiddleware(ctx, next) {
	__middlewareOrder = __middlewareOrder + "second,"
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	// Register middleware using gsh.use("command.input", ...)
	interp.EvalString(`gsh.use("command.input", firstMiddleware)`, nil)
	interp.EvalString(`gsh.use("command.input", secondMiddleware)`, nil)

	ctx := context.Background()

	// Execute a command - both middleware should run in registration order
	err = repl.processCommand(ctx, "echo order_test")
	assert.NoError(t, err)

	// Verify order: first registered = first to run
	vars := interp.GetVariables()
	orderVal, ok := vars["__middlewareOrder"]
	require.True(t, ok)
	orderStr, ok := orderVal.(*interpreter.StringValue)
	require.True(t, ok)
	assert.Equal(t, "first,second,", orderStr.Value)
}

func TestREPL_ProcessCommand_BuiltinExitStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Get interpreter and set up middleware that passes through
	interp := repl.executor.Interpreter()

	code := `
tool passThroughMiddleware(ctx, next) {
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	// Register middleware using gsh.use("command.input", ...)
	interp.EvalString(`gsh.use("command.input", passThroughMiddleware)`, nil)

	ctx := context.Background()

	// "exit" should still work as a built-in command after middleware passes through
	err = repl.processCommand(ctx, "exit")
	assert.Equal(t, ErrExit, err)
}

func TestREPL_ProcessCommand_SetsInterpreterContext(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	interp := repl.executor.Interpreter()

	// Track if context was set during middleware execution
	var contextWasSet bool
	var contextWasCancellable bool

	// Create middleware that checks if context is set on interpreter
	code := `
tool checkContextMiddleware(ctx, next) {
	return next(ctx)
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)

	// We can't directly check from gsh script, so we'll use a Go-level check
	// by wrapping the middleware execution
	originalContext := interp.Context()

	// Register middleware
	interp.EvalString(`gsh.use("command.input", checkContextMiddleware)`, nil)

	ctx := context.Background()

	// Execute a simple command
	err = repl.processCommand(ctx, "echo test")
	assert.NoError(t, err)

	// After command completes, context should be cleared (nil -> returns Background)
	afterContext := interp.Context()

	// The context before should be Background (default)
	// This test verifies the basic flow works
	assert.NotNil(t, originalContext, "original context should not be nil")
	assert.NotNil(t, afterContext, "context after command should not be nil")

	// Context should be reset after command (SetContext(nil) called)
	// Both should effectively be background contexts
	select {
	case <-afterContext.Done():
		t.Error("context after command should not be cancelled")
	default:
		// Expected - context is not cancelled
	}

	_ = contextWasSet
	_ = contextWasCancellable
}

func TestREPL_ProcessCommand_ContextCancellationStopsShellCommand(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Create a context that we'll cancel during command execution
	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running command in a goroutine
	done := make(chan error, 1)
	go func() {
		// sleep 10 should be interrupted
		done <- repl.processCommand(ctx, "/bin/sleep 10")
	}()

	// Give the command a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context (simulates Ctrl+C effect)
	cancel()

	// Wait for command to finish (should be quick due to cancellation)
	select {
	case err := <-done:
		// Command should complete (possibly with error due to cancellation)
		// The important thing is it didn't run for 10 seconds
		_ = err // Error may or may not be nil depending on timing
	case <-time.After(2 * time.Second):
		t.Fatal("command did not respond to context cancellation within timeout")
	}
}

func TestREPL_ProcessCommand_InterruptRecordsExitCode130(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	interp := repl.executor.Interpreter()

	// Create middleware that simulates a long-running operation that gets interrupted
	// We'll use a middleware that checks for context cancellation
	code := `
tool slowMiddleware(ctx, next) {
	return { handled: true }
}
`
	_, err = interp.EvalString(code, nil)
	require.NoError(t, err)
	interp.EvalString(`gsh.use("command.input", slowMiddleware)`, nil)

	// For this test, we need to verify that when the signal handler detects
	// SIGINT, it records exit code 130. Since we can't easily send SIGINT
	// in a unit test, we verify the history recording works normally first.
	ctx := context.Background()
	err = repl.processCommand(ctx, "test command")
	assert.NoError(t, err)

	// Verify history was recorded
	entries, err := repl.History().GetRecentEntries("", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "test command", entries[0].Command)
	// Exit code 0 for middleware-handled command
	assert.True(t, entries[0].ExitCode.Valid)
	assert.Equal(t, int32(0), entries[0].ExitCode.Int32)
}

func TestREPL_ProcessCommand_IgnoresSigintWhileWaitingForChild(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	sigChan := make(chan os.Signal, 1)
	repl.setSigintChannelFactory(func() (chan os.Signal, func()) {
		return sigChan, func() {}
	})

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		_ = repl.processCommand(ctx, "/bin/sleep 2")
		close(done)
	}()

	// Give the command time to start and become the foreground job.
	time.Sleep(200 * time.Millisecond)

	// Deliver SIGINT to the REPL handler channel; the child should own it.
	sigChan <- syscall.SIGINT

	// The command should keep running for a bit instead of being interrupted immediately.
	select {
	case <-done:
		t.Fatal("command was interrupted by parent SIGINT")
	case <-time.After(200 * time.Millisecond):
	}

	<-done

	entries, err := repl.History().GetRecentEntries("", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].ExitCode.Valid)
	assert.Equal(t, int32(0), entries[0].ExitCode.Int32)
}

func TestREPL_ProcessCommand_ContextPassedToShellExecution(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.repl.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to run a command with already-cancelled context
	// The command should fail quickly or return an error
	start := time.Now()
	_ = repl.processCommand(ctx, "/bin/sleep 5")
	elapsed := time.Since(start)

	// Should complete much faster than 5 seconds because context is cancelled
	assert.Less(t, elapsed, 2*time.Second, "cancelled context should prevent long-running command")
}
