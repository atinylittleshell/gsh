// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/repl/render"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// ToolExecutor is a function that executes a tool call and returns the result.
// It receives the tool name and arguments, and returns the result as a string.
type ToolExecutor func(ctx context.Context, toolName string, args map[string]interface{}) (string, error)

// State holds the state for a single agent.
type State struct {
	Agent         *interpreter.AgentValue
	Provider      interpreter.ModelProvider
	Conversation  []interpreter.ChatMessage
	Tools         []interpreter.ChatTool   // Available tools for this agent
	ToolExecutor  ToolExecutor             // Function to execute tool calls
	MaxIterations int                      // Maximum iterations for the agentic loop (0 uses default)
	Interpreter   *interpreter.Interpreter // Interpreter for executing the agent loop
}

// Manager manages multiple agents and handles messaging to the current agent.
type Manager struct {
	states           map[string]*State
	currentAgentName string
	logger           *zap.Logger
	renderer         *render.Renderer
}

// NewManager creates a new agent manager.
func NewManager(logger *zap.Logger) *Manager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Manager{
		states: make(map[string]*State),
		logger: logger,
	}
}

// SetRenderer sets the renderer for agent output.
// If not set, output will use simple fmt.Print calls.
func (m *Manager) SetRenderer(r *render.Renderer) {
	m.renderer = r
}

// AddAgent adds an agent state to the manager.
func (m *Manager) AddAgent(name string, state *State) {
	m.states[name] = state
}

// GetAgent returns the state for a named agent.
func (m *Manager) GetAgent(name string) *State {
	return m.states[name]
}

// GetAgentNames returns all configured agent names.
func (m *Manager) GetAgentNames() []string {
	names := make([]string, 0, len(m.states))
	for name := range m.states {
		names = append(names, name)
	}
	return names
}

// CurrentAgentName returns the name of the current agent.
func (m *Manager) CurrentAgentName() string {
	return m.currentAgentName
}

// SetCurrentAgent sets the current agent by name.
// Returns an error if the agent doesn't exist.
func (m *Manager) SetCurrentAgent(name string) error {
	if _, exists := m.states[name]; !exists {
		return fmt.Errorf("agent '%s' not found", name)
	}
	m.currentAgentName = name
	return nil
}

// CurrentAgent returns the current agent's state, or nil if none is set.
func (m *Manager) CurrentAgent() *State {
	if m.currentAgentName == "" {
		return nil
	}
	return m.states[m.currentAgentName]
}

// HasAgents returns true if any agents are configured.
func (m *Manager) HasAgents() bool {
	return len(m.states) > 0
}

// AgentCount returns the number of configured agents.
func (m *Manager) AgentCount() int {
	return len(m.states)
}

// AllStates returns a map of all agent states (for iteration).
func (m *Manager) AllStates() map[string]*State {
	return m.states
}

// ClearCurrentConversation clears the conversation history for the current agent.
func (m *Manager) ClearCurrentConversation() error {
	if m.currentAgentName == "" {
		return fmt.Errorf("no current agent")
	}

	state := m.states[m.currentAgentName]
	if state != nil {
		state.Conversation = []interpreter.ChatMessage{}
	}
	return nil
}

// SendMessage sends a message to the current agent and streams the response.
// The onChunk callback is called for each chunk of the response as it streams.
// This uses the interpreter's ExecuteAgentWithCallbacks for the agentic loop,
// with callbacks to drive the REPL's UI.
func (m *Manager) SendMessage(ctx context.Context, message string, onChunk func(string)) error {
	if m.currentAgentName == "" {
		return fmt.Errorf("no current agent")
	}

	state := m.states[m.currentAgentName]
	if state == nil {
		return fmt.Errorf("current agent state not found")
	}

	// Defensive checks to catch initialization bugs early
	if state.Interpreter == nil {
		return fmt.Errorf("BUG: interpreter not configured for agent '%s' - this is a programming error in agent initialization", m.currentAgentName)
	}

	if state.ToolExecutor != nil && len(state.Tools) == 0 {
		m.logger.Warn("agent has tool executor but no tools defined - tools will not be available to the LLM",
			zap.String("agent", m.currentAgentName))
	}

	if len(state.Tools) > 0 && state.ToolExecutor == nil {
		m.logger.Warn("agent has tools defined but no tool executor - tool calls will fail",
			zap.String("agent", m.currentAgentName))
	}

	// Build initial conversation from state
	conv := &interpreter.ConversationValue{
		Messages: make([]interpreter.ChatMessage, len(state.Conversation)),
	}
	copy(conv.Messages, state.Conversation)

	// Add the user message
	conv.Messages = append(conv.Messages, interpreter.ChatMessage{
		Role:    "user",
		Content: message,
	})

	// Track if we've rendered the header (only render once at the start)
	headerRendered := false

	// Spinner management - we need to handle this specially because the spinner
	// should stop on the first content chunk
	var stopSpinner func()
	var spinnerMu sync.Mutex
	firstChunkReceived := false

	// Track if we've shown a streaming tool call for the current response.
	// When the LLM returns multiple tool calls in one response, we only show
	// the streaming (pending) status for the first one to avoid display issues.
	streamingToolShown := false

	// Store the stop function for the pending spinner so we can stop it
	// when the tool call starts executing
	var stopPendingSpinner func()
	var pendingSpinnerMu sync.Mutex

	// Build callbacks to drive the REPL UI
	callbacks := &interpreter.AgentCallbacks{
		Streaming: true,
		Tools:     state.Tools, // Pass REPL built-in tools to be sent to the LLM

		OnIterationStart: func(iteration int) {
			// Render header on first iteration
			if !headerRendered && m.renderer != nil {
				m.renderer.RenderAgentHeader(m.currentAgentName)
				headerRendered = true
			}

			// Reset streaming tool flag and pending spinner for new iteration
			streamingToolShown = false
			pendingSpinnerMu.Lock()
			stopPendingSpinner = nil
			pendingSpinnerMu.Unlock()

			// Start thinking spinner for each iteration
			spinnerMu.Lock()
			firstChunkReceived = false
			if m.renderer != nil {
				stopSpinner = m.renderer.StartThinkingSpinner(ctx)
			}
			spinnerMu.Unlock()
		},

		OnChunk: func(content string) {
			// Stop spinner on first content chunk
			spinnerMu.Lock()
			if !firstChunkReceived && stopSpinner != nil {
				stopSpinner()
				stopSpinner = nil
				firstChunkReceived = true
			}
			spinnerMu.Unlock()

			// Render text through renderer if available
			if m.renderer != nil {
				m.renderer.RenderAgentText(content)
			}
			if onChunk != nil {
				onChunk(content)
			}
		},

		OnResponse: func(response *interpreter.ChatResponse) {
			// Make sure spinner is stopped even if no content was received
			spinnerMu.Lock()
			if stopSpinner != nil {
				stopSpinner()
				stopSpinner = nil
			}
			spinnerMu.Unlock()
		},

		OnToolCallStreaming: func(toolCallID string, toolName string) {
			// Called when a tool call starts streaming (before arguments are complete)
			// Show pending state to give user immediate feedback
			if m.renderer == nil {
				return
			}

			// Stop thinking spinner since we're now showing tool pending state
			spinnerMu.Lock()
			if stopSpinner != nil {
				stopSpinner()
				stopSpinner = nil
			}
			spinnerMu.Unlock()

			// Only render pending state for the first streaming tool call in this response.
			// When the LLM returns multiple tool calls, showing pending for all of them
			// causes display issues since RenderToolComplete can only replace one line.
			if streamingToolShown {
				return
			}
			streamingToolShown = true

			// Start pending spinner - it will be stopped when OnToolCallStart is called
			pendingSpinnerMu.Lock()
			stopPendingSpinner = m.renderer.StartToolPendingSpinner(ctx, toolName)
			pendingSpinnerMu.Unlock()
		},

		OnToolCallStart: func(toolCall acp.ToolCall) {
			if m.renderer == nil {
				return
			}

			// Stop pending spinner if it's running (for the first tool call in a batch)
			pendingSpinnerMu.Lock()
			if stopPendingSpinner != nil {
				stopPendingSpinner()
				stopPendingSpinner = nil
			}
			pendingSpinnerMu.Unlock()

			// Check if this is an exec tool call for special rendering
			if toolCall.Name == "exec" {
				if cmd, ok := toolCall.Arguments["command"].(string); ok && cmd != "" {
					m.renderer.RenderExecStart(cmd)
				}
			} else {
				// For non-exec tools, render executing state with args
				m.renderer.RenderToolExecuting(toolCall.Name, toolCall.Arguments)
			}
		},

		OnToolCallEnd: func(toolCall acp.ToolCall, update acp.ToolCallUpdate) {
			if m.renderer == nil {
				return
			}

			isExecTool := toolCall.Name == "exec"

			if isExecTool {
				var command string
				if cmd, ok := toolCall.Arguments["command"].(string); ok {
					command = cmd
				}

				// Parse exit code from exec result for rendering
				var execExitCode int
				if update.Error == nil {
					execExitCode = parseExecExitCode(update.Content)
				} else {
					execExitCode = 1
				}

				if command != "" {
					m.renderer.RenderExecEnd(command, update.Duration, execExitCode)
				}
			} else {
				// For non-exec tools, render completion state
				success := update.Status == acp.ToolCallStatusCompleted
				m.renderer.RenderToolComplete(toolCall.Name, toolCall.Arguments, update.Duration, success)
				// Render tool output
				m.renderer.RenderToolOutput(toolCall.Name, update.Content)
			}

			// Log tool errors
			if update.Error != nil {
				m.logger.Warn("tool execution failed",
					zap.String("tool", toolCall.Name),
					zap.Error(update.Error),
				)
			}
		},

		OnComplete: func(result acp.AgentResult) {
			// Render error if any
			if result.Error != nil && headerRendered && m.renderer != nil {
				m.renderer.RenderAgentError(result.Error)
			}

			// Render footer with stats
			if m.renderer != nil {
				usage := result.Usage
				if usage == nil {
					usage = &acp.TokenUsage{}
				}
				m.renderer.RenderAgentFooter(usage.PromptTokens, usage.CompletionTokens, usage.CachedTokens, result.Duration)
			}

			m.logger.Debug("agent interaction",
				zap.String("agent", m.currentAgentName),
				zap.String("message", message),
				zap.String("stopReason", string(result.StopReason)),
				zap.Duration("duration", result.Duration),
			)
		},

		ToolExecutor: state.ToolExecutor,
	}

	// Execute using the interpreter's agentic loop
	result, err := state.Interpreter.ExecuteAgentWithCallbacks(ctx, conv, state.Agent, callbacks)

	// Update state conversation from result
	if result != nil {
		if convResult, ok := result.(*interpreter.ConversationValue); ok {
			state.Conversation = convResult.Messages
		}
	}

	return err
}

// parseExecExitCode extracts the exit code from an exec tool result JSON.
func parseExecExitCode(result string) int {
	// Simple parsing - look for "exitCode": N pattern
	// The result format is: {"output": "...", "exitCode": N}
	const prefix = `"exitCode":`
	idx := strings.Index(result, prefix)
	if idx == -1 {
		return 0
	}

	// Skip to the number
	start := idx + len(prefix)
	// Skip whitespace
	for start < len(result) && (result[start] == ' ' || result[start] == '\t') {
		start++
	}

	// Read digits
	end := start
	for end < len(result) && result[end] >= '0' && result[end] <= '9' {
		end++
	}

	if start == end {
		return 0
	}

	// Parse the number
	var exitCode int
	_, _ = fmt.Sscanf(result[start:end], "%d", &exitCode)
	return exitCode
}

// SendMessageToCurrentAgent sends a message to the current agent with default output handling.
// This is a convenience method that prints chunks to stdout and handles errors.
// If a renderer is set, it handles all output formatting. Otherwise, it falls back to
// simple fmt.Print calls.
func (m *Manager) SendMessageToCurrentAgent(ctx context.Context, message string) error {
	// If renderer is set, it handles output - we don't need the callback
	var callback func(string)
	if m.renderer == nil {
		callback = func(content string) {
			// Print each chunk immediately without newline
			fmt.Print(content)
		}
	}

	err := m.SendMessage(ctx, message, callback)

	// Print final newline after streaming completes (only if no renderer)
	if m.renderer == nil {
		fmt.Println()
	}

	if err != nil {
		// If no renderer, print error to stderr (otherwise it's already rendered inside the agent block)
		if m.renderer == nil {
			fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
		}
		return nil // Return nil to not propagate error to caller (matches original behavior)
	}

	return nil
}
