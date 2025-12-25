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

	// Build version for dev mode indicator
	buildVersion string
}

// Options holds configuration options for creating a new REPL.
type Options struct {
	// ConfigPath is the path to the .gshrc.gsh configuration file.
	// If empty, the default path (~/.gshrc.gsh) is used.
	ConfigPath string

	// DefaultConfigContent is the embedded content of .gshrc.default.gsh.
	// This is loaded before the user's .gshrc.gsh file.
	DefaultConfigContent string

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

	// Initialize executor first (needed for loading .gshrc)
	exec, err := executor.NewREPLExecutor(logger, opts.ExecMiddleware...)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Load bash configuration files first (for bash/zsh compatibility)
	ctx := context.Background()
	if err := loadBashConfigs(ctx, exec, logger); err != nil {
		logger.Warn("failed to load bash configs", zap.Error(err))
	}

	// Load gsh-specific configuration from .gshrc.gsh
	loader := config.NewLoader(logger)
	var loadResult *config.LoadResult

	if opts.ConfigPath != "" {
		loadResult, err = loader.LoadFromFile(opts.ConfigPath)
	} else {
		loadResult, err = loader.LoadDefaultConfigPath(opts.DefaultConfigContent)
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
	agentManager := agent.NewManager(logger)

	// Always initialize the built-in default agent if a model is configured
	defaultAgentModel := loadResult.Config.GetDefaultAgentModel()
	if defaultAgentModel != nil && defaultAgentModel.Provider != nil {
		// Create the built-in default agent with a simple system prompt
		defaultAgent := &interpreter.AgentValue{
			Name: "default",
			Config: map[string]interpreter.Value{
				"model": defaultAgentModel,
				"systemPrompt": &interpreter.StringValue{
					Value: "You are gsh, an AI-powered shell program.",
				},
			},
		}

		defaultState := &agent.State{
			Agent:        defaultAgent,
			Provider:     defaultAgentModel.Provider,
			Conversation: []interpreter.ChatMessage{},
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
		buildVersion:       opts.BuildVersion,
	}

	// Set the REPL as the agent provider for completions
	completionProvider.SetAgentProvider(repl)

	return repl, nil
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

// parseAgentInput parses input after the "#" prefix.
// Returns isCommand (true if input is a command starting with "/"),
// and the command/message content.
func parseAgentInput(input string) (isCommand bool, content string) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "/") {
		return true, trimmed[1:] // Remove "/" prefix
	}
	return false, input // Keep original spacing for messages
}

// handleAgentCommand handles agent chat commands (prefixed with '#').
func (r *REPL) handleAgentCommand(ctx context.Context, input string) error {
	// Check if any agents are configured
	if !r.agentManager.HasAgents() {
		fmt.Fprintf(os.Stderr, "gsh: no agents configured. Configure defaultAgentModel in .gshrc.gsh or add custom agents\n")
		return nil
	}

	// Parse input to determine if it's a command or message
	isCommand, content := parseAgentInput(input)

	if isCommand {
		// Handle agent commands
		return r.handleAgentCommandAction(content)
	}

	// Handle empty message
	if strings.TrimSpace(content) == "" {
		fmt.Println("Agent mode: type your message after # to chat with the current agent.")
		fmt.Println("Commands:")
		fmt.Println("  # /clear        - clear current agent's conversation")
		fmt.Println("  # /agents       - list all available agents")
		fmt.Println("  # /agent <name> - switch to a different agent")
		return nil
	}

	// Send message to current agent
	return r.agentManager.SendMessageToCurrentAgent(ctx, content)
}

// handleAgentCommandAction handles agent commands (/clear, /agents, /agent).
func (r *REPL) handleAgentCommandAction(commandLine string) error {
	// Split command and arguments
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		fmt.Fprintf(os.Stderr, "gsh: empty command\n")
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "clear":
		return r.handleClearCommand()
	case "agents":
		return r.handleAgentsCommand()
	case "agent":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "gsh: /agent command requires an agent name\n")
			return nil
		}
		return r.handleSwitchAgentCommand(args[0])
	default:
		fmt.Fprintf(os.Stderr, "gsh: unknown command: /%s. Try /agents or /clear\n", cmd)
		return nil
	}
}

// handleClearCommand clears the current agent's conversation.
func (r *REPL) handleClearCommand() error {
	if err := r.agentManager.ClearCurrentConversation(); err != nil {
		fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
		return nil
	}
	fmt.Println("→ Conversation cleared")
	return nil
}

// handleAgentsCommand lists all available agents.
func (r *REPL) handleAgentsCommand() error {
	if !r.agentManager.HasAgents() {
		fmt.Println("No agents configured.")
		return nil
	}

	currentName := r.agentManager.CurrentAgentName()
	fmt.Println("Available agents:")
	for name, state := range r.agentManager.AllStates() {
		marker := " "
		if name == currentName {
			marker = "•"
		}

		msgCount := len(state.Conversation)
		status := fmt.Sprintf("(%d messages)", msgCount)
		if name == currentName {
			status = fmt.Sprintf("(current, %d messages)", msgCount)
		}

		// Try to get description from agent config
		description := ""
		if name == "default" {
			description = " - Built-in default agent"
		} else if descVal, ok := state.Agent.Config["description"]; ok {
			if descStr, ok := descVal.(*interpreter.StringValue); ok {
				description = " - " + descStr.Value
			}
		}

		fmt.Printf("  %s %-12s %s%s\n", marker, name, status, description)
	}
	return nil
}

// handleSwitchAgentCommand switches to a different agent.
func (r *REPL) handleSwitchAgentCommand(agentName string) error {
	// Check if agent exists and switch to it
	if err := r.agentManager.SetCurrentAgent(agentName); err != nil {
		fmt.Fprintf(os.Stderr, "gsh: agent '%s' not found. Use /agents to see available agents\n", agentName)
		return nil
	}

	// Get the state to show message count
	state := r.agentManager.GetAgent(agentName)
	msgCount := len(state.Conversation)
	if msgCount > 0 {
		fmt.Printf("→ Switched to agent '%s' (%d messages in history)\n", agentName, msgCount)
	} else {
		fmt.Printf("→ Switched to agent '%s'\n", agentName)
	}
	return nil
}

// ErrExit is returned when the user requests to exit the REPL.
var ErrExit = fmt.Errorf("exit requested")

// handleBuiltinCommand handles built-in REPL commands.
// Returns true if the command was handled, and an error if the REPL should exit.
func (r *REPL) handleBuiltinCommand(command string) (bool, error) {
	switch command {
	case "exit", ":exit":
		// Signal exit by returning ErrExit
		fmt.Println("exit")
		return true, ErrExit

	case ":clear":
		// Clear screen
		fmt.Print("\033[H\033[2J")
		return true, nil

	default:
		return false, nil
	}
}

// getPrompt returns the prompt string to display.
func (r *REPL) getPrompt() string {
	var prompt string

	// Check if GSH_UPDATE_PROMPT tool is defined
	promptTool := r.config.GetUpdatePromptTool()
	if promptTool != nil {
		// Call the tool with exit code and duration
		exitCodeValue := &interpreter.NumberValue{Value: float64(r.lastExitCode)}
		durationValue := &interpreter.NumberValue{Value: float64(r.lastDurationMs)}

		result, err := r.executor.Interpreter().CallTool(promptTool, []interpreter.Value{exitCodeValue, durationValue})
		if err != nil {
			r.logger.Warn("GSH_UPDATE_PROMPT tool failed, using static prompt", zap.Error(err))
			prompt = r.config.Prompt
		} else if strValue, ok := result.(*interpreter.StringValue); ok {
			prompt = strValue.Value
		} else {
			r.logger.Warn("GSH_UPDATE_PROMPT tool did not return a string, using static prompt",
				zap.String("returnType", result.Type().String()))
			prompt = r.config.Prompt
		}
	} else {
		// Fall back to static prompt
		prompt = r.config.Prompt
	}

	// Add [dev] prefix for development builds
	if r.buildVersion == "dev" {
		prompt = "[dev] " + prompt
	}

	return prompt
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

// GetAgentNames returns all configured agent names for completion.
func (r *REPL) GetAgentNames() []string {
	return r.agentManager.GetAgentNames()
}

// GetAgentCommands returns the list of valid agent commands for completion.
func (r *REPL) GetAgentCommands() []string {
	return AgentCommands
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
