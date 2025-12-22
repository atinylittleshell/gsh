// Package repl provides the main interactive shell implementation for gsh.
// It consolidates functionality from pkg/gline, pkg/shellinput, and other
// packages into a cohesive REPL that leverages the gsh script interpreter.
package repl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl/completion"
	"github.com/atinylittleshell/gsh/internal/repl/config"
	replcontext "github.com/atinylittleshell/gsh/internal/repl/context"
	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	"github.com/atinylittleshell/gsh/internal/repl/predict"
)

// timeNow is a variable that can be overridden for testing.
var timeNow = time.Now

// REPL is the main interactive shell interface.
type REPL struct {
	config             *config.Config
	executor           *executor.REPLExecutor
	history            *history.HistoryManager
	predictor          *predict.Router
	contextProvider    *replcontext.Provider
	completionProvider *completion.Provider
	logger             *zap.Logger

	// Track last command exit code and duration for prompt updates
	lastExitCode   int
	lastDurationMs int64
}

// Options holds configuration options for creating a new REPL.
type Options struct {
	// ConfigPath is the path to the .gshrc.gsh configuration file.
	// If empty, the default path (~/.gshrc.gsh) is used.
	ConfigPath string

	// HistoryPath is the path to the history database file.
	// If empty, the default path is used.
	HistoryPath string

	// Logger is the logger to use. If nil, a no-op logger is used.
	Logger *zap.Logger

	// ExecMiddleware is optional middleware for command execution.
	ExecMiddleware []executor.ExecMiddleware
}

// NewREPL creates a new REPL instance.
func NewREPL(opts Options) (*REPL, error) {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Load configuration
	loader := config.NewLoader(logger)
	var loadResult *config.LoadResult
	var err error

	if opts.ConfigPath != "" {
		loadResult, err = loader.LoadFromFile(opts.ConfigPath)
	} else {
		loadResult, err = loader.LoadDefaultConfigPath()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Log any non-fatal config errors
	for _, configErr := range loadResult.Errors {
		logger.Warn("config warning", zap.Error(configErr))
	}

	// Initialize executor
	exec, err := executor.NewREPLExecutor(logger, opts.ExecMiddleware...)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Initialize history manager
	historyPath := opts.HistoryPath
	if historyPath == "" {
		historyPath = core.HistoryFile()
	}

	historyMgr, err := history.NewHistoryManager(historyPath)
	if err != nil {
		logger.Warn("failed to initialize history, continuing without history", zap.Error(err))
		// Continue without history - not fatal
	}

	// Initialize prediction router from config
	predictor := predict.NewRouterFromConfig(loadResult.Config, logger)

	// Initialize context provider with retrievers for predictions
	contextProvider := replcontext.NewProvider(logger,
		replcontext.NewWorkingDirectoryRetriever(exec),
		replcontext.NewGitStatusRetriever(exec, logger),
		replcontext.NewSystemInfoRetriever(),
	)

	// Add history retriever if history is available
	if historyMgr != nil {
		contextProvider.AddRetriever(replcontext.NewConciseHistoryRetriever(historyMgr, 0))
	}

	// Initialize completion provider
	completionProvider := completion.NewProvider(exec)

	return &REPL{
		config:             loadResult.Config,
		executor:           exec,
		history:            historyMgr,
		predictor:          predictor,
		contextProvider:    contextProvider,
		completionProvider: completionProvider,
		logger:             logger,
	}, nil
}

// Run starts the interactive REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	r.logger.Info("starting REPL")

	// Create prediction state if history or LLM predictor is available
	// History-based prediction doesn't require an LLM model
	var predictionState *input.PredictionState
	var historyProvider input.HistoryProvider
	if r.history != nil {
		historyProvider = input.NewHistoryPredictionAdapter(r.history)
	}

	// Create prediction state if we have history or LLM predictor
	if historyProvider != nil || r.predictor != nil {
		// Only set LLMProvider if predictor is not nil to avoid nil interface issues
		var llmProvider input.PredictionProvider
		if r.predictor != nil {
			llmProvider = r.predictor
		}

		predictionState = input.NewPredictionState(input.PredictionStateConfig{
			HistoryProvider: historyProvider,
			LLMProvider:     llmProvider,
			Logger:          r.logger,
		})
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get prompt
		prompt := r.getPrompt()

		// Get history values for navigation
		historyValues := r.getHistoryValues()

		// Update predictor context before each input session
		r.updatePredictorContext()

		// Reset prediction state for new input
		if predictionState != nil {
			predictionState.Reset()
		}

		// Create input model
		inputModel := input.New(input.Config{
			Prompt:             prompt,
			HistoryValues:      historyValues,
			CompletionProvider: r.completionProvider,
			PredictionState:    predictionState,
			Logger:             r.logger,
		})

		// Run the input loop
		p := tea.NewProgram(inputModel,
			tea.WithContext(ctx),
			tea.WithOutput(os.Stderr),
		)

		finalModel, err := p.Run()
		if err != nil {
			// Check if it's a context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("input error: %w", err)
		}

		// Get result from model
		model, ok := finalModel.(input.Model)
		if !ok {
			return fmt.Errorf("unexpected model type")
		}

		result := model.Result()

		switch result.Type {
		case input.ResultEOF:
			// Ctrl+D on empty line - exit
			fmt.Println("exit")
			return nil

		case input.ResultInterrupt:
			// Ctrl+C - cancel current input, continue loop
			fmt.Println("^C")
			continue

		case input.ResultSubmit:
			// Print the prompt + user input before executing (like old REPL)
			// This shows the user what command is being executed
			fmt.Print("\r" + model.Prompt() + result.Value + "\n")

			// Process the command
			if err := r.processCommand(ctx, result.Value); err != nil {
				// Log error but continue
				r.logger.Debug("command error", zap.Error(err))
			}
		}
	}
}

// processCommand handles a submitted command.
func (r *REPL) processCommand(ctx context.Context, command string) error {
	// Trim whitespace
	command = strings.TrimSpace(command)

	// Skip empty commands
	if command == "" {
		return nil
	}

	// Handle built-in commands
	if handled := r.handleBuiltinCommand(command); handled {
		return nil
	}

	// Record command in history
	var historyEntry *history.HistoryEntry
	if r.history != nil {
		entry, err := r.history.StartCommand(command, r.executor.GetPwd())
		if err != nil {
			r.logger.Debug("failed to record command in history", zap.Error(err))
		} else {
			historyEntry = entry
		}
	}

	// Execute the command
	startTime := timeNow()
	exitCode, err := r.executor.ExecuteBash(ctx, command)
	duration := timeNow().Sub(startTime)

	// Update last exit code and duration
	r.lastExitCode = exitCode
	r.lastDurationMs = duration.Milliseconds()

	// Finish history entry
	if historyEntry != nil {
		_, finishErr := r.history.FinishCommand(historyEntry, exitCode)
		if finishErr != nil {
			r.logger.Debug("failed to finish history entry", zap.Error(finishErr))
		}
	}

	// Display error if execution failed (not just non-zero exit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
	}

	return nil
}

// handleBuiltinCommand handles built-in REPL commands.
// Returns true if the command was handled.
func (r *REPL) handleBuiltinCommand(command string) bool {
	switch command {
	case "exit", ":exit":
		// Exit is handled by returning from Run()
		os.Exit(0)
		return true

	case ":clear":
		// Clear screen
		fmt.Print("\033[H\033[2J")
		return true

	default:
		return false
	}
}

// getPrompt returns the prompt string to display.
func (r *REPL) getPrompt() string {
	// For now, use static prompt from config
	// TODO: Support GSH_UPDATE_PROMPT tool in later phases
	return r.config.Prompt
}

// updatePredictorContext updates the predictor with current context information.
func (r *REPL) updatePredictorContext() {
	if r.predictor == nil || r.contextProvider == nil {
		return
	}

	contextMap := r.contextProvider.GetContext()
	r.predictor.UpdateContext(contextMap)
}

// getHistoryValues returns recent history entries for navigation.
func (r *REPL) getHistoryValues() []string {
	if r.history == nil {
		return nil
	}

	entries, err := r.history.GetRecentEntries("", 100)
	if err != nil {
		r.logger.Debug("failed to get history entries", zap.Error(err))
		return nil
	}

	// Convert to string slice (most recent first)
	values := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		values = append(values, entries[i].Command)
	}

	return values
}

// Close cleans up REPL resources.
func (r *REPL) Close() error {
	if r.executor != nil {
		return r.executor.Close()
	}
	return nil
}

// Config returns the current configuration.
func (r *REPL) Config() *config.Config {
	return r.config
}

// Executor returns the command executor.
func (r *REPL) Executor() *executor.REPLExecutor {
	return r.executor
}

// History returns the history manager.
func (r *REPL) History() *history.HistoryManager {
	return r.history
}
