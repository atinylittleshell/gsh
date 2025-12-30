// Package repl provides the main interactive shell implementation for gsh.
// It consolidates functionality from pkg/gline, pkg/shellinput, and other
// packages into a cohesive REPL that leverages the gsh script interpreter.
package repl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"golang.org/x/term"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl/agent"
	"github.com/atinylittleshell/gsh/internal/repl/completion"
	"github.com/atinylittleshell/gsh/internal/repl/config"
	replcontext "github.com/atinylittleshell/gsh/internal/repl/context"
	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	"github.com/atinylittleshell/gsh/internal/repl/predict"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// AgentCommands is the list of valid agent commands (without the "/" prefix).
// This is the single source of truth for what commands are available.
var AgentCommands = []string{"clear", "agents", "agent"}

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

	// Agent mode support - multiple agents
	agentManager *agent.Manager

	// Track last command exit code and duration for prompt updates
	lastExitCode   int
	lastDurationMs int64
}

// Options holds configuration options for creating a new REPL.
type Options struct {
	// ConfigPath is the path to the .gshrc.gsh configuration file.
	// If empty, the default path (~/.gshrc.gsh) is used.
	ConfigPath string

	// DefaultConfigContent is the embedded content of .gshrc.default.gsh.
	// This is loaded before the user's .gshrc.gsh file.
	DefaultConfigContent string

	// StarshipConfigContent is the embedded content of .gshrc.starship.gsh.
	// This is loaded after user config if starship is detected and integration is enabled.
	StarshipConfigContent string

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

	// Load gsh-specific configuration into the shared interpreter
	loader := config.NewLoader(logger)
	var loadResult *config.LoadResult

	if opts.ConfigPath != "" {
		content, err := os.ReadFile(opts.ConfigPath)
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
			loadResult, err = loader.LoadFromStringInto(interp, string(content))
			if err != nil {
				return nil, fmt.Errorf("failed to load config: %w", err)
			}
		}
	} else {
		loadResult, err = loader.LoadDefaultConfigPathInto(interp, opts.DefaultConfigContent, opts.StarshipConfigContent)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Log any non-fatal config errors
	for _, configErr := range loadResult.Errors {
		logger.Warn("config warning", zap.Error(configErr))
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

	// Note: We'll set the agent provider after initializing agents below

	// Initialize agent manager
	agentManager := agent.NewManager()

	// Always initialize the built-in default agent if a model is configured
	defaultAgentModel := loadResult.Config.GetDefaultAgentModel()
	if defaultAgentModel != nil && defaultAgentModel.Provider != nil {
		// Create the built-in default agent with a simple system prompt
		defaultAgent := &interpreter.AgentValue{
			Name: "default",
			Config: map[string]interpreter.Value{
				"model": defaultAgentModel,
				"systemPrompt": &interpreter.StringValue{
					Value: "You are gsh (generative shell), an AI-powered shell assistant. You use tools available to you to help the user with their questions and tasks.",
				},
			},
		}

		defaultState := &agent.State{
			Agent:        defaultAgent,
			Provider:     defaultAgentModel.Provider,
			Conversation: []interpreter.ChatMessage{},
			Interpreter:  interp,
		}
		agent.SetupAgentWithDefaultTools(defaultState)
		agentManager.AddAgent("default", defaultState)
		_ = agentManager.SetCurrentAgent("default")
		logger.Info("initialized built-in default agent", zap.String("model", defaultAgentModel.Name))
	}

	// Initialize all custom agents from configuration
	if loadResult.Config != nil && len(loadResult.Config.Agents) > 0 {
		for name, agentVal := range loadResult.Config.Agents {
			// Get the provider from the agent's model (stored in Config)
			var provider interpreter.ModelProvider
			if modelVal, ok := agentVal.Config["model"]; ok {
				if model, ok := modelVal.(*interpreter.ModelValue); ok && model.Provider != nil {
					provider = model.Provider
				} else {
					logger.Warn("agent model has no provider configured", zap.String("agent", name))
					continue
				}
			} else {
				logger.Warn("agent has no model configured", zap.String("agent", name))
				continue
			}

			customState := &agent.State{
				Agent:        agentVal,
				Provider:     provider,
				Conversation: []interpreter.ChatMessage{},
				Interpreter:  interp,
			}
			agent.SetupAgentWithDefaultTools(customState)
			agentManager.AddAgent(name, customState)
			logger.Info("initialized custom agent", zap.String("agent", name))
		}
	}

	// If no current agent is set but we have agents, pick the first one as a fallback
	if agentManager.CurrentAgentName() == "" && agentManager.HasAgents() {
		for name := range agentManager.AllStates() {
			_ = agentManager.SetCurrentAgent(name)
			logger.Info("auto-selected agent as default", zap.String("agent", name))
			break
		}
	}

	repl := &REPL{
		config:             loadResult.Config,
		executor:           exec,
		history:            historyMgr,
		predictor:          predictor,
		contextProvider:    contextProvider,
		completionProvider: completionProvider,
		agentManager:       agentManager,
		logger:             logger,
	}

	// Set the REPL as the agent provider for completions
	completionProvider.SetAgentProvider(repl)

	// Initialize REPL context with model tiers for gsh.repl access
	// Model tiers start as nil and are configured by the user in .gshrc via:
	//   gsh.repl.models.lite = myLiteModel
	//   gsh.repl.models.workhorse = myWorkhorseModel
	//   gsh.repl.models.premium = myPremiumModel
	replCtx := &interpreter.REPLContext{
		Models: &interpreter.REPLModels{
			Lite:      nil,
			Workhorse: nil,
			Premium:   nil,
		},
		LastCommand: &interpreter.REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
		Agents: []*interpreter.AgentValue{},
	}

	// Populate gsh.repl.agents from the agent manager
	// agents[0] is always the default agent
	defaultState := agentManager.GetAgent("default")
	if defaultState != nil {
		// Use the AgentValue directly - no need to create a separate REPLAgent
		replCtx.Agents = append(replCtx.Agents, defaultState.Agent)
		replCtx.CurrentAgent = defaultState.Agent
	}

	// Add custom agents from config to gsh.repl.agents
	for name, state := range agentManager.AllStates() {
		if name == "default" {
			continue // Already added
		}
		// Use the AgentValue directly
		replCtx.Agents = append(replCtx.Agents, state.Agent)
	}

	// Set up callbacks for agent management from gsh script
	replCtx.OnAgentAdded = func(newAgent *interpreter.AgentValue) {
		repl.handleAgentAddedFromSDK(newAgent)
	}
	replCtx.OnAgentSwitch = func(switchedAgent *interpreter.AgentValue) {
		repl.handleAgentSwitchFromSDK(switchedAgent)
	}
	replCtx.OnAgentModified = func(modifiedAgent *interpreter.AgentValue) {
		repl.handleAgentModifiedFromSDK(modifiedAgent)
	}

	interp.SDKConfig().SetREPLContext(replCtx)

	return repl, nil
}

// Run starts the interactive REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	r.logger.Info("starting REPL")

	// Emit repl.ready event (welcome screen is handled by event handler in .gshrc.default.gsh)
	r.emitREPLEvent("repl.ready")

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
			AliasExistsFunc:    r.executor.AliasExists,
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

	// Check if this is an agent command (starts with '#')
	if strings.HasPrefix(command, "#") {
		return r.handleAgentCommand(ctx, strings.TrimSpace(command[1:]))
	}

	// Handle built-in commands
	if handled, err := r.handleBuiltinCommand(command); handled {
		return err // Will be ErrExit if user wants to exit
	}

	// Emit repl.command.before event with the command text
	r.emitREPLEvent("repl.command.before", &interpreter.StringValue{Value: command})

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

	// Update the interpreter's REPL context with the last command info
	r.executor.Interpreter().SDKConfig().UpdateLastCommand(exitCode, r.lastDurationMs)

	// Emit repl.command.after event with command, exit code, and duration
	r.emitREPLEvent("repl.command.after",
		&interpreter.StringValue{Value: command},
		&interpreter.NumberValue{Value: float64(exitCode)},
		&interpreter.NumberValue{Value: float64(r.lastDurationMs)})

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

// getPrompt returns the prompt string to display.
// Emits repl.prompt event to allow dynamic prompt updates (e.g., Starship integration).
// Event handlers can set gsh.repl.prompt to customize the prompt.
func (r *REPL) getPrompt() string {
	interp := r.executor.Interpreter()

	// Emit repl.prompt event to let handlers update the prompt dynamically
	interp.EmitEvent("repl.prompt", &interpreter.NullValue{})

	// Read gsh.repl.prompt property (may have been updated by event handler)
	replCtx := interp.SDKConfig().GetREPLContext()
	if replCtx != nil && replCtx.PromptValue != nil {
		if strVal, ok := replCtx.PromptValue.(*interpreter.StringValue); ok && strVal.Value != "" {
			return strVal.Value
		}
	}

	// Fallback to config prompt if gsh.repl.prompt not set
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
	handlers := interp.GetEventHandlers(eventName)
	for _, handler := range handlers {
		// Call each handler with the provided arguments
		// Errors are logged but don't stop other handlers
		if _, err := interp.CallTool(handler, args); err != nil {
			r.logger.Debug("error in event handler", zap.String("event", eventName), zap.Error(err))
		}
	}
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
