// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

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

	// Track if we've added the user message (only add on first successful iteration)
	userMessageAdded := false

	// Agentic loop - continue until no tool calls or max iterations reached
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Build messages for the provider
		messages := m.buildMessagesWithPendingUser(state, message, userMessageAdded)

		// Create request with tools if available
		request := interpreter.ChatRequest{
			Model:    model,
			Messages: messages,
			Tools:    state.Tools,
		}

		// Call provider with streaming to display response in real-time
		response, err := state.Provider.StreamingChatCompletion(
			request,
			func(content string) {
				if onChunk != nil {
					onChunk(content)
				}
			},
		)

		if err != nil {
			return fmt.Errorf("agent error: %w", err)
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
			return fmt.Errorf("tool execution error: %w", err)
		}

		// Continue loop to make another call with tool results
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
		// Notify about tool execution start
		if onChunk != nil {
			onChunk(fmt.Sprintf("\n[Executing tool: %s]\n", toolCall.Name))
		}

		var result string
		var err error

		if state.ToolExecutor != nil {
			// Use custom tool executor if provided
			result, err = state.ToolExecutor(ctx, toolCall.Name, toolCall.Arguments)
		} else {
			// Default: return error indicating no executor
			err = fmt.Errorf("no tool executor configured for tool '%s'", toolCall.Name)
		}

		if err != nil {
			// On error, add error message as tool result so the model can recover
			result = fmt.Sprintf("Error executing tool: %v", err)
			m.logger.Warn("tool execution failed",
				zap.String("tool", toolCall.Name),
				zap.Error(err),
			)
		}

		// Add tool result to conversation
		state.Conversation = append(state.Conversation, interpreter.ChatMessage{
			Role:       "tool",
			Content:    result,
			Name:       toolCall.Name,
			ToolCallID: toolCall.ID,
		})

		// Notify about tool result
		if onChunk != nil {
			// Truncate long results for display
			displayResult := result
			if len(displayResult) > 500 {
				displayResult = displayResult[:500] + "... (truncated)"
			}
			onChunk(fmt.Sprintf("[Tool result: %s]\n", displayResult))
		}
	}

	return nil
}

// SendMessageToCurrentAgent sends a message to the current agent with default output handling.
// This is a convenience method that prints chunks to stdout and handles errors.
func (m *Manager) SendMessageToCurrentAgent(ctx context.Context, message string) error {
	err := m.SendMessage(ctx, message, func(content string) {
		// Print each chunk immediately without newline
		fmt.Print(content)
	})

	// Print final newline after streaming completes
	fmt.Println()

	if err != nil {
		fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
		return nil // Return nil to not propagate error to caller (matches original behavior)
	}

	return nil
}
