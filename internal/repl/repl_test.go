package repl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	// Import all subpackages to verify the directory structure is correct
	_ "github.com/atinylittleshell/gsh/internal/repl/agent"
	_ "github.com/atinylittleshell/gsh/internal/repl/completion"
	_ "github.com/atinylittleshell/gsh/internal/repl/config"
	_ "github.com/atinylittleshell/gsh/internal/repl/context"
	_ "github.com/atinylittleshell/gsh/internal/repl/executor"
	_ "github.com/atinylittleshell/gsh/internal/repl/input"
	_ "github.com/atinylittleshell/gsh/internal/repl/predict"
)

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	require.NotNil(t, repl)
	defer repl.Close()

	// Verify default config was loaded
	assert.Equal(t, "gsh> ", repl.Config().Prompt)
	assert.Equal(t, "info", repl.Config().LogLevel)

	// Verify executor was created
	assert.NotNil(t, repl.Executor())

	// Verify history manager was created
	assert.NotNil(t, repl.History())
}

func TestNewREPL_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "test.gshrc.gsh")

	// Create a test config file
	configContent := `
GSH_CONFIG = {
	prompt: "test> ",
	logLevel: "debug",
}
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	require.NotNil(t, repl)
	defer repl.Close()

	// Verify custom config was loaded
	assert.Equal(t, "test> ", repl.Config().Prompt)
	assert.Equal(t, "debug", repl.Config().LogLevel)
}

func TestNewREPL_NilLogger(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Note: We can't test :exit directly as it calls os.Exit
	// But we can test that :clear is handled
	handled := repl.handleBuiltinCommand(":clear")
	assert.True(t, handled)

	// Test unhandled command
	handled = repl.handleBuiltinCommand("ls")
	assert.False(t, handled)
}

func TestREPL_ProcessCommand_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "test.gshrc.gsh")

	// Create a test config file with custom prompt
	configContent := `
GSH_CONFIG = {
	prompt: "custom> ",
}
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Verify prompt
	assert.Equal(t, "custom> ", repl.getPrompt())
}

func TestREPL_Close(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

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
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// :clear should be handled
	handled := repl.handleBuiltinCommand(":clear")
	assert.True(t, handled)
}

func TestREPL_HandleBuiltinCommand_UnknownCommand(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Unknown commands should not be handled
	handled := repl.handleBuiltinCommand("unknown")
	assert.False(t, handled)

	handled = repl.handleBuiltinCommand("ls -la")
	assert.False(t, handled)
}

func TestREPL_ContextProviderInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Verify context provider was created
	assert.NotNil(t, repl.contextProvider)

	// Get context and verify it contains expected keys
	contextMap := repl.contextProvider.GetContext()

	// Should have working_directory
	_, hasWorkingDir := contextMap["working_directory"]
	assert.True(t, hasWorkingDir, "context should include working_directory")

	// Should have system_info
	_, hasSystemInfo := contextMap["system_info"]
	assert.True(t, hasSystemInfo, "context should include system_info")

	// Should have git_status (might be "not in a git repository")
	_, hasGitStatus := contextMap["git_status"]
	assert.True(t, hasGitStatus, "context should include git_status")

	// Should have history_concise (since history manager was initialized)
	_, hasHistory := contextMap["history_concise"]
	assert.True(t, hasHistory, "context should include history_concise")
}

func TestREPL_UpdatePredictorContext_NilPredictor(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Predictor should be nil when no model is configured
	assert.Nil(t, repl.predictor)

	// updatePredictorContext should not panic with nil predictor
	repl.updatePredictorContext()
}

func TestREPL_UpdatePredictorContext_NilContextProvider(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	// Manually set contextProvider to nil to test edge case
	repl.contextProvider = nil

	// updatePredictorContext should not panic with nil contextProvider
	repl.updatePredictorContext()
}

func TestREPL_ContextProviderWithoutHistory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use an invalid history path that will cause history initialization to fail
	historyPath := filepath.Join(tmpDir, "nonexistent_dir", "subdir", "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer repl.Close()

	// Context provider should still be initialized
	assert.NotNil(t, repl.contextProvider)

	// Get context - should still have basic retrievers
	contextMap := repl.contextProvider.GetContext()

	// Should have working_directory
	_, hasWorkingDir := contextMap["working_directory"]
	assert.True(t, hasWorkingDir, "context should include working_directory")

	// Should have system_info
	_, hasSystemInfo := contextMap["system_info"]
	assert.True(t, hasSystemInfo, "context should include system_info")
}

func TestREPL_ContextContainsWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	contextMap := repl.contextProvider.GetContext()

	// Verify working_directory contains actual path
	workingDir := contextMap["working_directory"]
	assert.Contains(t, workingDir, "<working_dir>")
	assert.Contains(t, workingDir, "</working_dir>")
}

func TestREPL_ContextContainsSystemInfo(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err)
	defer repl.Close()

	contextMap := repl.contextProvider.GetContext()

	// Verify system_info contains OS and arch
	systemInfo := contextMap["system_info"]
	assert.Contains(t, systemInfo, "<system_info>")
	assert.Contains(t, systemInfo, "OS:")
	assert.Contains(t, systemInfo, "Arch:")
}
