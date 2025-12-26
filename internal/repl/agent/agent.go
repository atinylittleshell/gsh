// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/repl/render"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// DefaultMaxIterations is the default maximum number of tool call iterations
// if not specified in the agent state.
const DefaultMaxIterations = 100

// timeNow is a variable that can be overridden for testing.
var timeNow = time.Now

// ToolExecutor is a function that executes a tool call and returns the result.
// It receives the tool name and arguments, and returns the result as a string.
type ToolExecutor func(ctx context.Context, toolName string, args map[string]interface{}) (string, error)

// State holds the state for a single agent.
type State struct {
	Agent         *interpreter.AgentValue
	Provider      interpreter.ModelProvider
	Conversation  []interpreter.ChatMessage
	Tools         []interpreter.ChatTool // Available tools for this agent
	ToolExecutor  ToolExecutor           // Function to execute tool calls
	MaxIterations int                    // Maximum iterations for the agentic loop (0 uses default)
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
// This implements an agentic loop that continues until no tool calls are returned
// or the maximum number of iterations is reached.
func (m *Manager) SendMessage(ctx context.Context, message string, onChunk func(string)) error {
	if m.currentAgentName == "" {
		return fmt.Errorf("no current agent")
	}

	state := m.states[m.currentAgentName]
	if state == nil {
		return fmt.Errorf("current agent state not found")
	}

	// Get the model from agent config
	var model *interpreter.ModelValue
	if modelVal, ok := state.Agent.Config["model"]; ok {
		model, _ = modelVal.(*interpreter.ModelValue)
	}

	// Get max iterations from state, or use default
	maxIterations := state.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}

	startTime := timeNow()

	// Track token usage across all iterations
	var totalInputTokens, totalOutputTokens, totalCachedTokens int

	// Track if we've added the user message (only add on first successful iteration)
	userMessageAdded := false

	// Track if we've rendered the header (only render once at the start)
	headerRendered := false

	// Agentic loop - continue until no tool calls or max iterations reached
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Render header on first iteration
		if !headerRendered && m.renderer != nil {
			m.renderer.RenderAgentHeader(m.currentAgentName)
			headerRendered = true
		}

		// Build messages for the provider
		messages := m.buildMessagesWithPendingUser(state, message, userMessageAdded)

		// Create request with tools if available
		request := interpreter.ChatRequest{
			Model:    model,
			Messages: messages,
			Tools:    state.Tools,
		}

		// Start thinking spinner while waiting for LLM
		var stopSpinner func()
		if m.renderer != nil {
			stopSpinner = m.renderer.StartThinkingSpinner(ctx)
		}

		// Track if we've received any content (to know when to stop spinner)
		firstChunkReceived := false

		// Call provider with streaming to display response in real-time
		response, err := state.Provider.StreamingChatCompletion(
			request,
			func(content string) {
				// Stop spinner on first content chunk (blocks until spinner is fully stopped)
				if !firstChunkReceived && stopSpinner != nil {
					stopSpinner()
					stopSpinner = nil
					firstChunkReceived = true
				}

				// Render text through renderer if available, otherwise use callback
				if m.renderer != nil {
					m.renderer.RenderAgentText(content)
				}
				if onChunk != nil {
					onChunk(content)
				}
			},
		)

		// Make sure spinner is stopped even if no content was received
		if stopSpinner != nil {
			stopSpinner()
		}

		if err != nil {
			// Render error and footer if we rendered a header
			if headerRendered && m.renderer != nil {
				m.renderer.RenderAgentError(err)
				duration := timeNow().Sub(startTime)
				m.renderer.RenderAgentFooter(totalInputTokens, totalOutputTokens, totalCachedTokens, duration)
			}
			return fmt.Errorf("agent error: %w", err)
		}

		// Accumulate token usage
		if response.Usage != nil {
			totalInputTokens += response.Usage.PromptTokens
			totalOutputTokens += response.Usage.CompletionTokens
			totalCachedTokens += response.Usage.CachedTokens
		}

		// On first successful response, add the user message to conversation history
		if !userMessageAdded {
			state.Conversation = append(state.Conversation, interpreter.ChatMessage{
				Role:    "user",
				Content: message,
			})
			userMessageAdded = true
		}

		// If no tool calls, add final response and return
		if len(response.ToolCalls) == 0 {
			state.Conversation = append(state.Conversation, interpreter.ChatMessage{
				Role:    "assistant",
				Content: response.Content,
			})

			duration := timeNow().Sub(startTime)
			m.logger.Debug("agent interaction",
				zap.String("agent", m.currentAgentName),
				zap.String("message", message),
				zap.String("response", response.Content),
				zap.Duration("duration", duration),
			)

			// Render footer with stats
			if m.renderer != nil {
				m.renderer.RenderAgentFooter(totalInputTokens, totalOutputTokens, totalCachedTokens, duration)
			}

			return nil
		}

		// Add assistant message with tool calls to conversation
		state.Conversation = append(state.Conversation, interpreter.ChatMessage{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		// Execute tool calls and add results to conversation
		if err := m.executeToolCalls(ctx, state, response.ToolCalls, onChunk); err != nil {
			// Render footer even on error
			if m.renderer != nil {
				duration := timeNow().Sub(startTime)
				m.renderer.RenderAgentFooter(totalInputTokens, totalOutputTokens, totalCachedTokens, duration)
			}
			return fmt.Errorf("tool execution error: %w", err)
		}

		// Continue loop to make another call with tool results
	}

	// Render footer before returning max iterations error
	if m.renderer != nil {
		duration := timeNow().Sub(startTime)
		m.renderer.RenderAgentFooter(totalInputTokens, totalOutputTokens, totalCachedTokens, duration)
	}

	// If we reach here, we hit max iterations
	return fmt.Errorf("agent reached maximum iterations (%d) without completing", maxIterations)
}

// buildMessages constructs the message array for the provider, including system prompt.
func (m *Manager) buildMessages(state *State) []interpreter.ChatMessage {
	messages := make([]interpreter.ChatMessage, 0, len(state.Conversation)+1)

	// Add system prompt if configured (from agent Config)
	if systemPromptVal, ok := state.Agent.Config["systemPrompt"]; ok {
		if systemPrompt, ok := systemPromptVal.(*interpreter.StringValue); ok && systemPrompt.Value != "" {
			messages = append(messages, interpreter.ChatMessage{
				Role:    "system",
				Content: systemPrompt.Value,
			})
		}
	}

	// Add conversation history
	messages = append(messages, state.Conversation...)

	return messages
}

// buildMessagesWithPendingUser constructs the message array including a pending user message
// that hasn't been added to conversation history yet.
func (m *Manager) buildMessagesWithPendingUser(state *State, userMessage string, userMessageAdded bool) []interpreter.ChatMessage {
	// Start with base messages
	messages := m.buildMessages(state)

	// If user message hasn't been added to conversation yet, add it to the request
	if !userMessageAdded {
		messages = append(messages, interpreter.ChatMessage{
			Role:    "user",
			Content: userMessage,
		})
	}

	return messages
}

// executeToolCalls executes all tool calls and adds results to the conversation.
func (m *Manager) executeToolCalls(ctx context.Context, state *State, toolCalls []interpreter.ChatToolCall, onChunk func(string)) error {
	for _, toolCall := range toolCalls {
		var result string
		var err error
		var execExitCode int
		var execDuration time.Duration

		// Check if this is an exec tool call for special rendering
		isExecTool := toolCall.Name == "exec"
		var command string
		if isExecTool {
			if cmd, ok := toolCall.Arguments["command"].(string); ok {
				command = cmd
			}
		}

		// Render tool start based on type
		if m.renderer != nil {
			if isExecTool && command != "" {
				m.renderer.RenderExecStart(command)
			} else if !isExecTool {
				// For non-exec tools, render executing state with args
				// (args are already complete when we receive the tool call)
				m.renderer.RenderToolExecuting(toolCall.Name, toolCall.Arguments)
			}
		}

		execStart := timeNow()

		if state.ToolExecutor != nil {
			// Use custom tool executor if provided
			result, err = state.ToolExecutor(ctx, toolCall.Name, toolCall.Arguments)
		} else {
			// Default: return error indicating no executor
			err = fmt.Errorf("no tool executor configured for tool '%s'", toolCall.Name)
		}

		execDuration = timeNow().Sub(execStart)

		// Parse exit code from exec result for rendering
		if isExecTool && err == nil {
			execExitCode = parseExecExitCode(result)
		}

		if err != nil {
			// On error, add error message as tool result so the model can recover
			result = fmt.Sprintf("Error executing tool: %v", err)
			m.logger.Warn("tool execution failed",
				zap.String("tool", toolCall.Name),
				zap.Error(err),
			)
			// For exec tools, set exit code to indicate error
			if isExecTool {
				execExitCode = 1
			}
		}

		// Render tool completion based on type
		if m.renderer != nil {
			if isExecTool && command != "" {
				m.renderer.RenderExecEnd(command, execDuration, execExitCode)
			} else if !isExecTool {
				// For non-exec tools, render completion state
				success := err == nil
				m.renderer.RenderToolComplete(toolCall.Name, toolCall.Arguments, execDuration, success)
				// Render tool output if the hook returns non-empty
				m.renderer.RenderToolOutput(toolCall.Name, result)
			}
		}

		// Add tool result to conversation
		state.Conversation = append(state.Conversation, interpreter.ChatMessage{
			Role:       "tool",
			Content:    result,
			Name:       toolCall.Name,
			ToolCallID: toolCall.ID,
		})
	}

	return nil
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
