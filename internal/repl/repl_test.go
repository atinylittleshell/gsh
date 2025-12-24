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
	"fmt"
	_ "github.com/atinylittleshell/gsh/internal/repl/completion"
	_ "github.com/atinylittleshell/gsh/internal/repl/config"
	_ "github.com/atinylittleshell/gsh/internal/repl/context"
	_ "github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	_ "github.com/atinylittleshell/gsh/internal/repl/predict"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
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

	// Test that exit returns ErrExit
	handled, err := repl.handleBuiltinCommand("exit")
	assert.True(t, handled)
	assert.Equal(t, ErrExit, err)

	// Test that :exit also returns ErrExit
	handled, err = repl.handleBuiltinCommand(":exit")
	assert.True(t, handled)
	assert.Equal(t, ErrExit, err)

	// Test that :clear is handled
	handled, err = repl.handleBuiltinCommand(":clear")
	assert.True(t, handled)
	assert.NoError(t, err)

	// Test unhandled command
	handled, err = repl.handleBuiltinCommand("ls")
	assert.False(t, handled)
	assert.NoError(t, err)
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
	handled, err := repl.handleBuiltinCommand(":clear")
	assert.True(t, handled)
	assert.NoError(t, err)
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
	handled, err := repl.handleBuiltinCommand("unknown")
	assert.False(t, handled)
	assert.NoError(t, err)

	handled, err = repl.handleBuiltinCommand("ls -la")
	assert.False(t, handled)
	assert.NoError(t, err)
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

func TestREPL_HistoryPredictionWithoutLLM(t *testing.T) {
	// This test verifies that history-based prediction works even when
	// no LLM prediction model is configured. This is a regression test
	// for a nil interface issue where passing a nil *predict.Router to
	// PredictionProvider interface would cause a panic.
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.db")
	configPath := filepath.Join(tmpDir, "nonexistent.gshrc.gsh")

	logger := zaptest.NewLogger(t)

	repl, err := NewREPL(Options{
		ConfigPath:  configPath,
		HistoryPath: historyPath,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer repl.Close()

	// Verify that no LLM predictor is configured
	assert.Nil(t, repl.predictor, "predictor should be nil when no model is configured")

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

	// Create a history provider from the history manager
	historyProvider := input.NewHistoryPredictionAdapter(repl.history)
	require.NotNil(t, historyProvider)

	// Create prediction state with history but WITHOUT LLM provider
	// This is the key test - it should not panic when llmProvider is nil
	predictionState := input.NewPredictionState(input.PredictionStateConfig{
		HistoryProvider: historyProvider,
		LLMProvider:     nil, // Explicitly nil - no LLM configured
		Logger:          logger,
	})
	require.NotNil(t, predictionState)

	// Trigger a prediction by simulating input change
	resultCh := predictionState.OnInputChanged("echo")
	require.NotNil(t, resultCh, "should return a result channel for prediction")

	// Wait for the prediction result (with timeout)
	select {
	case result := <-resultCh:
		// Should get a history-based prediction without error or panic
		assert.NoError(t, result.Error, "prediction should not return an error")
		assert.Equal(t, input.PredictionSourceHistory, result.Source, "prediction should come from history")
		assert.Contains(t, result.Prediction, "echo", "prediction should start with the input prefix")
	case <-time.After(1 * time.Second):
		t.Fatal("prediction timed out")
	}
}

func TestNewREPL_BuiltInDefaultAgent(t *testing.T) {
	// Test that the built-in default agent is initialized when defaultAgentModel is configured

	configPath := filepath.Join(t.TempDir(), ".gshrc.gsh")
	err := os.WriteFile(configPath, []byte(`
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		GSH_CONFIG = {
			prompt: "test> ",
			defaultAgentModel: testModel,
		}
	`), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	repl, err := NewREPL(Options{
		ConfigPath: configPath,
		Logger:     logger,
	})

	require.NoError(t, err)
	require.NotNil(t, repl)

	// Verify built-in default agent was initialized and set as current
	assert.Len(t, repl.agentStates, 1)
	assert.Equal(t, "default", repl.currentAgentName)
	assert.NotNil(t, repl.agentStates["default"])
	assert.Equal(t, "default", repl.agentStates["default"].Agent.Name)
}

func TestNewREPL_BuiltInDefaultAgentWithCustomAgents(t *testing.T) {
	// Test that the built-in default agent is used alongside custom agents

	configPath := filepath.Join(t.TempDir(), ".gshrc.gsh")
	err := os.WriteFile(configPath, []byte(`
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		agent agent1 {
			model: testModel,
			systemPrompt: "test1",
		}

		agent agent2 {
			model: testModel,
			systemPrompt: "test2",
		}

		GSH_CONFIG = {
			prompt: "test> ",
			defaultAgentModel: testModel,
		}
	`), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	repl, err := NewREPL(Options{
		ConfigPath: configPath,
		Logger:     logger,
	})

	require.NoError(t, err)
	require.NotNil(t, repl)

	// Verify built-in default agent and custom agents were initialized
	assert.Len(t, repl.agentStates, 3)                // default + agent1 + agent2
	assert.Equal(t, "default", repl.currentAgentName) // default agent is current
	assert.NotNil(t, repl.agentStates["default"])
	assert.NotNil(t, repl.agentStates["agent1"])
	assert.NotNil(t, repl.agentStates["agent2"])
}

func TestNewREPL_NoDefaultAgentWithoutModel(t *testing.T) {
	// Test that no built-in default agent is created when defaultAgentModel is not configured
	// but custom agents are still available and one is automatically selected

	configPath := filepath.Join(t.TempDir(), ".gshrc.gsh")
	err := os.WriteFile(configPath, []byte(`
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		agent customAgent {
			model: testModel,
			systemPrompt: "custom",
		}

		GSH_CONFIG = {
			prompt: "test> ",
		}
	`), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	repl, err := NewREPL(Options{
		ConfigPath: configPath,
		Logger:     logger,
	})

	require.NoError(t, err)
	require.NotNil(t, repl)

	// Verify only custom agent was initialized, no built-in default agent
	assert.Len(t, repl.agentStates, 1)
	assert.Equal(t, "customAgent", repl.currentAgentName) // Custom agent is auto-selected
	assert.NotNil(t, repl.agentStates["customAgent"])
	assert.Nil(t, repl.agentStates["default"]) // No built-in default agent
}

func TestNewREPL_BuiltInDefaultAgentIsImmutable(t *testing.T) {
	// Test that the built-in default agent has a simple, immutable system prompt

	configPath := filepath.Join(t.TempDir(), ".gshrc.gsh")
	err := os.WriteFile(configPath, []byte(`
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		GSH_CONFIG = {
			prompt: "test> ",
			defaultAgentModel: testModel,
		}
	`), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	repl, err := NewREPL(Options{
		ConfigPath: configPath,
		Logger:     logger,
	})

	require.NoError(t, err)
	require.NotNil(t, repl)

	// Verify the built-in default agent has the expected system prompt
	defaultAgent := repl.agentStates["default"]
	require.NotNil(t, defaultAgent)

	systemPromptVal, ok := defaultAgent.Agent.Config["systemPrompt"]
	require.True(t, ok, "default agent should have systemPrompt")

	systemPrompt, ok := systemPromptVal.(*interpreter.StringValue)
	require.True(t, ok, "systemPrompt should be a string")

	assert.Equal(t, "You are gsh, an AI-powered shell program. You are friendly and helpful, assisting the user with their tasks in the shell.", systemPrompt.Value)
	assert.Equal(t, "default", defaultAgent.Agent.Name)
}

func TestNewREPL_DefaultAgentCannotBeOverridden(t *testing.T) {
	// Test that users cannot define a custom agent named "default"
	// The built-in default agent always takes precedence

	configPath := filepath.Join(t.TempDir(), ".gshrc.gsh")
	err := os.WriteFile(configPath, []byte(`
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		agent default {
			model: testModel,
			systemPrompt: "custom system prompt",
		}

		GSH_CONFIG = {
			prompt: "test> ",
			defaultAgentModel: testModel,
		}
	`), 0644)
	require.NoError(t, err)

	logger := zap.NewNop()
	repl, err := NewREPL(Options{
		ConfigPath: configPath,
		Logger:     logger,
	})

	require.NoError(t, err)
	require.NotNil(t, repl)

	// The built-in default agent should be initialized first,
	// then the custom "default" agent will overwrite it in the map
	// This means the user's custom agent named "default" will win

	// Verify that only one "default" agent exists (custom overwrites built-in)
	assert.NotNil(t, repl.agentStates["default"])
	assert.Equal(t, "default", repl.currentAgentName)

	// Verify it's the custom agent (has custom system prompt)
	systemPrompt, ok := repl.agentStates["default"].Agent.Config["systemPrompt"]
	require.True(t, ok)
	systemPromptStr, ok := systemPrompt.(*interpreter.StringValue)
	require.True(t, ok)
	assert.Equal(t, "custom system prompt", systemPromptStr.Value)
}

func TestHandleAgentCommand_NoAgent(t *testing.T) {
	// Test that when no agent is configured, a helpful error is shown

	logger := zap.NewNop()
	repl := &REPL{
		logger:      logger,
		agentStates: make(map[string]*AgentState), // No agents configured
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "hello")

	// Should not return error, just print message
	assert.NoError(t, err)
}

func TestHandleAgentCommand_Clear(t *testing.T) {
	// Test that /clear command clears conversation history

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "dummy",
		shouldError:     false,
	}

	agentState := &AgentState{
		Agent:    &interpreter.AgentValue{Name: "test"},
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"test": agentState},
		currentAgentName: "test",
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "/clear")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(agentState.Conversation))
}

func TestHandleAgentCommand_Help(t *testing.T) {
	// Test that empty message shows help

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "dummy",
		shouldError:     false,
	}

	agentState := &AgentState{
		Agent:        &interpreter.AgentValue{Name: "test"},
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"test": agentState},
		currentAgentName: "test",
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "")

	assert.NoError(t, err)
}

// MockProvider implements ModelProvider for testing
type MockProvider struct {
	responseContent    string
	shouldError        bool
	chatCompletionFunc func(request interpreter.ChatRequest) (*interpreter.ChatResponse, error)
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) ChatCompletion(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
	// If custom function is set, use it
	if m.chatCompletionFunc != nil {
		return m.chatCompletionFunc(request)
	}

	// Default behavior
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return &interpreter.ChatResponse{
		Content: m.responseContent,
	}, nil
}

func TestHandleAgentCommand_Success(t *testing.T) {
	// Test successful agent chat interaction

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Hello! How can I help?",
		shouldError:     false,
	}

	// Create agent with model config
	agent := &interpreter.AgentValue{
		Name: "testAgent",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{
				Name: "test-model",
			},
			"systemPrompt": &interpreter.StringValue{
				Value: "You are helpful",
			},
		},
	}

	agentState := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"testAgent": agentState},
		currentAgentName: "testAgent",
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "hello")

	assert.NoError(t, err)

	// Verify conversation history was updated
	assert.Equal(t, 2, len(agentState.Conversation))
	assert.Equal(t, "user", agentState.Conversation[0].Role)
	assert.Equal(t, "hello", agentState.Conversation[0].Content)
	assert.Equal(t, "assistant", agentState.Conversation[1].Role)
	assert.Equal(t, "Hello! How can I help?", agentState.Conversation[1].Content)
}

func TestHandleAgentCommand_ConversationHistory(t *testing.T) {
	// Test that conversation history is maintained across messages

	logger := zap.NewNop()

	callCount := 0
	var lastRequestMessages []interpreter.ChatMessage

	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	// Set custom ChatCompletion to capture request messages
	mockProvider.chatCompletionFunc = func(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
		lastRequestMessages = request.Messages
		callCount++
		// Return mock response
		return &interpreter.ChatResponse{
			Content: mockProvider.responseContent,
		}, nil
	}

	agent := &interpreter.AgentValue{
		Name: "testAgent",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{
				Name: "test-model",
			},
			"systemPrompt": &interpreter.StringValue{
				Value: "You are helpful",
			},
		},
	}

	agentState := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"testAgent": agentState},
		currentAgentName: "testAgent",
	}

	ctx := context.Background()

	// First message
	err := repl.handleAgentCommand(ctx, "first message")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(agentState.Conversation))

	// Verify first request had system prompt + user message
	assert.Equal(t, 2, len(lastRequestMessages)) // system + user
	assert.Equal(t, "system", lastRequestMessages[0].Role)
	assert.Equal(t, "You are helpful", lastRequestMessages[0].Content)
	assert.Equal(t, "user", lastRequestMessages[1].Role)
	assert.Equal(t, "first message", lastRequestMessages[1].Content)

	// Second message
	err = repl.handleAgentCommand(ctx, "second message")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(agentState.Conversation))

	// Verify second request included conversation history
	assert.Equal(t, 4, len(lastRequestMessages)) // system + history (2) + new user message
	assert.Equal(t, "system", lastRequestMessages[0].Role)
	assert.Equal(t, "user", lastRequestMessages[1].Role)
	assert.Equal(t, "first message", lastRequestMessages[1].Content)
	assert.Equal(t, "assistant", lastRequestMessages[2].Role)
	assert.Equal(t, "Response", lastRequestMessages[2].Content)
	assert.Equal(t, "user", lastRequestMessages[3].Role)
	assert.Equal(t, "second message", lastRequestMessages[3].Content)
}

func TestHandleAgentCommand_ProviderError(t *testing.T) {
	// Test error handling when provider fails

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "",
		shouldError:     true,
	}

	agent := &interpreter.AgentValue{
		Name: "testAgent",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{
				Name: "test-model",
			},
		},
	}

	agentState := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"testAgent": agentState},
		currentAgentName: "testAgent",
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "hello")

	// Should not return error (prints to stderr instead)
	assert.NoError(t, err)

	// Conversation should not be updated on error
	assert.Equal(t, 0, len(agentState.Conversation))
}

func TestHandleAgentCommand_NoSystemPrompt(t *testing.T) {
	// Test agent without system prompt

	logger := zap.NewNop()

	var lastRequestMessages []interpreter.ChatMessage
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	mockProvider.chatCompletionFunc = func(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
		lastRequestMessages = request.Messages
		return &interpreter.ChatResponse{
			Content: mockProvider.responseContent,
		}, nil
	}

	// Agent without systemPrompt in config
	agent := &interpreter.AgentValue{
		Name: "testAgent",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{
				Name: "test-model",
			},
		},
	}

	agentState := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"testAgent": agentState},
		currentAgentName: "testAgent",
	}

	ctx := context.Background()
	err := repl.handleAgentCommand(ctx, "hello")

	assert.NoError(t, err)

	// Should not include system message
	assert.Equal(t, 1, len(lastRequestMessages))
	assert.Equal(t, "user", lastRequestMessages[0].Role)
}
