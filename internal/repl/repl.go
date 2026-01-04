// Package repl provides the main interactive shell implementation for gsh.
// It consolidates functionality from pkg/gline, pkg/shellinput, and other
// packages into a cohesive REPL that leverages the gsh script interpreter.
package repl

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"golang.org/x/term"
	shinterp "mvdan.cc/sh/v3/interp"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl/completion"
	"github.com/atinylittleshell/gsh/internal/repl/config"
	replcontext "github.com/atinylittleshell/gsh/internal/repl/context"
	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	"github.com/atinylittleshell/gsh/internal/repl/predict"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// ErrExit is returned when the user requests to exit the REPL.
var ErrExit = fmt.Errorf("exit requested")

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

	// Startup tracking
	startTime      time.Time
	startupTracker StartupTimeTracker

	sigintChannelFactory func() (chan os.Signal, func())
}

// Options holds configuration options for creating a new REPL.
// StartupTimeTracker is an interface for tracking startup time.
// This allows the REPL to report when the user actually sees the welcome screen.
type StartupTimeTracker interface {
	TrackStartupTime(durationMs int64)
}

type Options struct {
	// ConfigPath is the path to the repl.gsh configuration file.
	// If empty, the default path (~/.gsh/repl.gsh) is used.
	ConfigPath string

	// DefaultConfigContent is the embedded content of defaults/init.gsh.
	// This is loaded before the user's ~/.gsh/repl.gsh file.
	DefaultConfigContent string

	// DefaultConfigFS is the embedded filesystem containing the default config and any
	// modules it imports. If nil, imports from the default config are disabled.
	DefaultConfigFS fs.FS

	// DefaultConfigBasePath is the base path within DefaultConfigFS where the default
	// config resides (e.g., "defaults" if the config is at "defaults/init.gsh").
	DefaultConfigBasePath string

	// HistoryPath is the path to the history database file.
	// If empty, the default path is used.
	HistoryPath string

	// Logger is the logger to use. If nil, a no-op logger is used.
	Logger *zap.Logger

	// ExecMiddleware is optional middleware for command execution.
	ExecMiddleware []executor.ExecMiddleware

	// BuildVersion is the build version string (e.g., "dev" or "1.0.0").
	// Used to show [dev] indicator in prompt for development builds.
	BuildVersion string

	// Runner is the sh runner to use for bash command execution.
	// If nil, a new runner is created with default settings.
	// Passing a runner allows sharing environment variables (like SHELL)
	// that were set up during gsh initialization.
	Runner *shinterp.Runner

	// StartTime is when the application started (for accurate startup time tracking).
	// If zero, startup time tracking is skipped.
	StartTime time.Time

	// StartupTracker is called when the REPL is ready (welcome screen shown).
	// This allows accurate startup time measurement from app start to user-visible ready state.
	StartupTracker StartupTimeTracker
}

// NewREPL creates a new REPL instance.
func NewREPL(opts Options) (*REPL, error) {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create ONE interpreter that will be shared by executor, config, and renderer
	interp := interpreter.New(&interpreter.Options{
		Logger:  logger,
		Version: opts.BuildVersion,
		Runner:  opts.Runner,
	})

	// Initialize executor with the shared interpreter
	exec, err := executor.NewREPLExecutor(interp, logger, opts.ExecMiddleware...)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Load bash configuration files first (for bash/zsh compatibility)
	ctx := context.Background()
	if err := loadBashConfigs(ctx, exec, logger); err != nil {
		logger.Warn("failed to load bash configs", zap.Error(err))
	}

	// Initialize REPL context BEFORE loading config so SDK assignments like
	// gsh.models.workhorse = myModel can work during config evaluation
	replCtx := &interpreter.REPLContext{
		LastCommand: &interpreter.REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Load gsh-specific configuration into the shared interpreter
	loader := config.NewLoader(logger)
	var loadResult *config.LoadResult

	if opts.ConfigPath != "" {
		// Get absolute path for proper import resolution
		absConfigPath, err := filepath.Abs(opts.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config path: %w", err)
		}

		content, err := os.ReadFile(absConfigPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
			loadResult = &config.LoadResult{
				Config:      config.DefaultConfig(),
				Interpreter: interp,
				Errors:      []error{},
			}
		} else {
			// Evaluate with filesystem origin for import resolution
			_, evalErr := interp.EvalString(string(content), &interpreter.ScriptOrigin{
				Type:     interpreter.OriginFilesystem,
				BasePath: filepath.Dir(absConfigPath),
			})

			loadResult = &config.LoadResult{
				Config:      config.DefaultConfig(),
				Interpreter: interp,
				Errors:      []error{},
			}
			if evalErr != nil {
				loadResult.Errors = append(loadResult.Errors, evalErr)
			}
			loader.ExtractConfigFromInterpreter(interp, loadResult)
		}
	} else {
		loadResult, err = loader.LoadDefaultConfigPathInto(interp, config.EmbeddedDefaults{
			Content:  opts.DefaultConfigContent,
			FS:       opts.DefaultConfigFS,
			BasePath: opts.DefaultConfigBasePath,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Log and display any non-fatal config errors
	// These are important for debugging - show them to the user
	for _, configErr := range loadResult.Errors {
		logger.Warn("config warning", zap.Error(configErr))
		fmt.Fprintf(os.Stderr, "gsh: config error: %v\n", configErr)
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

	// Initialize prediction router using lazy model resolution.
	// Use SDKModelRef so that predictions always use the current gsh.models.lite value,
	// even if the user changes it after startup (e.g., in an event handler).
	liteModelRef := &interpreter.SDKModelRef{Tier: "lite", Models: interp.SDKConfig().GetModels()}

	// Initialize prediction router using the lite model tier with lazy resolution
	predictor := predict.NewRouterFromConfig(liteModelRef, logger)

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

	repl := &REPL{
		config:             loadResult.Config,
		executor:           exec,
		history:            historyMgr,
		predictor:          predictor,
		contextProvider:    contextProvider,
		completionProvider: completionProvider,
		logger:             logger,
		startTime:          opts.StartTime,
		startupTracker:     opts.StartupTracker,
		sigintChannelFactory: func() (chan os.Signal, func()) {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT)
			return ch, func() { signal.Stop(ch) }
		},
	}

	return repl, nil
}

func (r *REPL) newSigintChannel() (chan os.Signal, func()) {
	if r.sigintChannelFactory != nil {
		return r.sigintChannelFactory()
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	return ch, func() { signal.Stop(ch) }
}

func (r *REPL) setSigintChannelFactory(factory func() (chan os.Signal, func())) {
	r.sigintChannelFactory = factory
}

// Run starts the interactive REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	r.logger.Info("starting REPL")

	// Emit repl.ready event (welcome screen is handled by event handler in defaults/events/repl.gsh)
	r.emitREPLEvent("repl.ready")

	// Track startup time - this is when the user actually sees the welcome screen
	if r.startupTracker != nil && !r.startTime.IsZero() {
		startupMs := time.Since(r.startTime).Milliseconds()
		r.startupTracker.TrackStartupTime(startupMs)
	}

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
			r.emitREPLEvent("repl.exit")
			return ctx.Err()
		default:
		}

		// Get prompt - emits repl.prompt event internally
		prompt := r.getPrompt()

		// Get history values for navigation
		historyValues := r.getHistoryValues()

		// Reset prediction state for new input
		if predictionState != nil {
			predictionState.Reset()
		}

		// Create input model with initial terminal width
		termWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
		if termWidth <= 0 {
			termWidth = 80
		}
		inputModel := input.New(input.Config{
			Prompt:             prompt,
			HistoryValues:      historyValues,
			HistorySearchFunc:  r.createHistorySearchFunc(),
			CompletionProvider: r.completionProvider,
			AliasExistsFunc:    r.executor.AliasOrFunctionExists,
			GetEnvFunc:         r.executor.GetEnv,
			GetWorkingDirFunc:  r.executor.GetPwd,
			PredictionState:    predictionState,
			Width:              termWidth,
			Logger:             r.logger,
		})

		// Update predictor context asynchronously (don't block prompt display)
		go r.updatePredictorContext()

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
			// Print the prompt + user input so it persists in terminal history
			// We use \r to return to start of line since Bubble Tea may leave cursor mid-line
			fmt.Print("\r" + model.Prompt() + result.Value + "\n")

			// Process the command
			if err := r.processCommand(ctx, result.Value); err != nil {
				// Check if user requested exit
				if err == ErrExit {
					r.emitREPLEvent("repl.exit")
					return nil
				}
				// Log other errors but continue
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

	// Set up SIGINT handling for command execution
	// Create a cancellable context that will be cancelled on Ctrl+C
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling for SIGINT (Ctrl+C)
	sigChan, stopSigint := r.newSigintChannel()
	defer stopSigint()

	// Track if we were interrupted
	interrupted := false
	var ignoreSigint atomic.Bool

	// Start a goroutine to handle SIGINT
	go func() {
		for {
			select {
			case <-sigChan:
				// When a foreground child owns the terminal, let it handle SIGINT.
				if ignoreSigint.Load() {
					continue
				}
				// Ctrl+C received - cancel the context
				// Note: The terminal echoes ^C automatically, so we don't print it here
				interrupted = true
				cancel()
				return
			case <-cmdCtx.Done():
				// Context was cancelled for another reason, exit goroutine
				return
			}
		}
	}()

	// Set the cancellable context on the interpreter so agent execution can use it
	interp := r.executor.Interpreter()
	interp.SetContext(cmdCtx)
	defer interp.SetContext(context.Background()) // Clear context after command completes

	// Record ALL user input in history (including agent commands like "#...")
	// This is done before middleware so all user input is captured
	var historyEntry *history.HistoryEntry
	if r.history != nil {
		entry, err := r.history.StartCommand(command, r.executor.GetPwd())
		if err != nil {
			r.logger.Debug("failed to record command in history", zap.Error(err))
		} else {
			historyEntry = entry
		}
	}

	// Emit command.input event to middleware chain
	inputCtx := &interpreter.ObjectValue{
		Properties: map[string]*interpreter.PropertyDescriptor{
			"input": {Value: &interpreter.StringValue{Value: command}},
		},
	}
	result := interp.EmitEvent("command.input", inputCtx)

	// Check if we were interrupted during middleware execution
	if interrupted {
		if historyEntry != nil {
			// Record as interrupted (exit code 130 is standard for SIGINT)
			if _, finishErr := r.history.FinishCommand(historyEntry, 130); finishErr != nil {
				r.logger.Debug("failed to finish history entry", zap.Error(finishErr))
			}
		}
		return nil
	}

	// Check if middleware handled the command
	if result != nil {
		if obj, ok := result.(*interpreter.ObjectValue); ok {
			// Check for { handled: true }
			if handledVal := obj.GetPropertyValue("handled"); handledVal != nil {
				if bv, ok := handledVal.(*interpreter.BoolValue); ok && bv.Value {
					// Middleware handled the input, don't execute as shell command
					if historyEntry != nil {
						if _, finishErr := r.history.FinishCommand(historyEntry, 0); finishErr != nil {
							r.logger.Debug("failed to finish history entry", zap.Error(finishErr))
						}
					}
					return nil
				}
			}
			// Check for modified input
			if inputVal := obj.GetPropertyValue("input"); inputVal != nil {
				if sv, ok := inputVal.(*interpreter.StringValue); ok {
					command = sv.Value
				}
			}
		}
	}

	// Get potentially modified input from context (middleware may have modified it)
	if inputVal := inputCtx.GetPropertyValue("input"); inputVal != nil {
		if sv, ok := inputVal.(*interpreter.StringValue); ok {
			command = sv.Value
		}
	}

	// Handle built-in commands (like "exit")
	if handled, err := r.handleBuiltinCommand(command); handled {
		if historyEntry != nil {
			if _, finishErr := r.history.FinishCommand(historyEntry, 0); finishErr != nil {
				r.logger.Debug("failed to finish history entry", zap.Error(finishErr))
			}
		}
		return err // Will be ErrExit if user wants to exit
	}

	// Fall through: execute as shell command
	ignoreSigint.Store(true)
	exitCode := r.executeShellCommand(cmdCtx, command)
	ignoreSigint.Store(false)

	// Finish history entry with actual exit code
	if historyEntry != nil {
		_, finishErr := r.history.FinishCommand(historyEntry, exitCode)
		if finishErr != nil {
			r.logger.Debug("failed to finish history entry", zap.Error(finishErr))
		}
	}

	return nil
}

// executeShellCommand executes a command via the shell (mvdan/sh).
// This is the fall-through path when middleware doesn't handle input.
// Note: History recording is done in processCommand before this is called.
// Returns the exit code of the command.
func (r *REPL) executeShellCommand(ctx context.Context, command string) int {
	// Emit repl.command.before event with the command text
	r.emitREPLEvent("repl.command.before", &interpreter.StringValue{Value: command})

	// Execute the command
	startTime := timeNow()
	exitCode, err := r.executor.ExecuteBash(ctx, command)
	duration := timeNow().Sub(startTime)

	// Update last exit code and duration
	r.lastExitCode = exitCode
	r.lastDurationMs = duration.Milliseconds()

	// Update the interpreter's REPL context with the last command info
	r.executor.Interpreter().SDKConfig().UpdateLastCommand(exitCode, r.lastDurationMs)

	// Emit repl.command.after event with command, exit code, and duration
	r.emitREPLEvent("repl.command.after",
		&interpreter.StringValue{Value: command},
		&interpreter.NumberValue{Value: float64(exitCode)},
		&interpreter.NumberValue{Value: float64(r.lastDurationMs)})

	// Display error if execution failed (not just non-zero exit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
	}

	return exitCode
}

// handleBuiltinCommand handles built-in REPL commands.
// Returns true if the command was handled, and an error if the REPL should exit.
func (r *REPL) handleBuiltinCommand(command string) (bool, error) {
	switch command {
	case "exit":
		// Signal exit by returning ErrExit
		return true, ErrExit

	default:
		return false, nil
	}
}

// getPrompt returns the prompt string to display.
// Emits repl.prompt event to allow dynamic prompt updates (e.g., Starship integration).
// Event handlers can set gsh.prompt to customize the prompt.
func (r *REPL) getPrompt() string {
	interp := r.executor.Interpreter()

	// Emit repl.prompt event to let handlers update the prompt dynamically
	interp.EmitEvent("repl.prompt", &interpreter.NullValue{})

	// Read gsh.prompt property (may have been updated by event handler)
	replCtx := interp.SDKConfig().GetREPLContext()
	if replCtx != nil && replCtx.PromptValue != nil {
		if strVal, ok := replCtx.PromptValue.(*interpreter.StringValue); ok && strVal.Value != "" {
			return strVal.Value
		}
	}

	// Fallback to default prompt if gsh.prompt not set
	return "gsh> "
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

// createHistorySearchFunc returns a function for searching history (used by Ctrl+R).
func (r *REPL) createHistorySearchFunc() input.HistorySearchFunc {
	if r.history == nil {
		return nil
	}

	return func(query string) []string {
		entries, err := r.history.SearchHistory(query, 100)
		if err != nil {
			r.logger.Debug("failed to search history", zap.Error(err))
			return nil
		}

		// Convert to string slice (already in reverse chronological order)
		values := make([]string, 0, len(entries))
		for _, entry := range entries {
			values = append(values, entry.Command)
		}

		return values
	}
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

// emitREPLEvent emits a REPL event by calling all registered handlers for that event
func (r *REPL) emitREPLEvent(eventName string, args ...interpreter.Value) {
	interp := r.executor.Interpreter()

	// Create context object from args
	var ctx interpreter.Value
	if len(args) == 0 {
		ctx = &interpreter.NullValue{}
	} else if len(args) == 1 {
		ctx = args[0]
	} else {
		// Multiple args - wrap in an object
		props := make(map[string]*interpreter.PropertyDescriptor)
		for i, arg := range args {
			props[fmt.Sprintf("arg%d", i)] = &interpreter.PropertyDescriptor{Value: arg}
		}
		ctx = &interpreter.ObjectValue{Properties: props}
	}

	// Use the interpreter's EmitEvent which handles the middleware chain
	interp.EmitEvent(eventName, ctx)
}

// loadBashConfigs loads bash configuration files in the correct order.
// This maintains compatibility with bash/zsh configurations.
func loadBashConfigs(ctx context.Context, exec *executor.REPLExecutor, logger *zap.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Files to load in order
	configFiles := []string{
		filepath.Join(homeDir, ".gshrc"),
		filepath.Join(homeDir, ".gshenv"),
	}

	// Load each config file
	for _, configFile := range configFiles {
		if err := config.LoadBashRC(ctx, exec, configFile); err != nil {
			logger.Warn("failed to load bash config", zap.String("file", configFile), zap.Error(err))
		}
	}

	return nil
}
