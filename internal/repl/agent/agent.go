// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// State holds the state for a single agent.
type State struct {
	Agent         *interpreter.AgentValue
	Provider      interpreter.ModelProvider
	Conversation  []interpreter.ChatMessage
	MaxIterations int                      // Maximum iterations for the agentic loop (0 uses default)
	Interpreter   *interpreter.Interpreter // Interpreter for executing the agent loop
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
// All rendering (spinner, text output) is handled via gsh script event handlers.
// This uses the interpreter's ExecuteAgentWithCallbacks for the agentic loop,
// with the EventEmitter callback to bridge interpreter events to gsh.on() handlers.
func (m *Manager) SendMessage(ctx context.Context, message string) error {
	if m.currentAgentName == "" {
		return fmt.Errorf("no current agent")
	}

	state := m.states[m.currentAgentName]
	if state == nil {
		return fmt.Errorf("current agent state not found")
	}

	// Defensive checks to catch initialization bugs early
	if state.Interpreter == nil {
		return fmt.Errorf("BUG: interpreter not configured for agent '%s' - this is an internal error in gsh", m.currentAgentName)
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

	// Build callbacks to drive the REPL UI via events
	// All rendering is handled by gsh script event handlers (agent.chunk, agent.iteration.start, etc.)
	// Event emission is handled entirely by the interpreter's agentic loop
	callbacks := &interpreter.AgentCallbacks{
		Streaming:   true,
		UserMessage: message, // Pass user message for agent.start event

		// EventEmitter allows the interpreter to emit SDK events through gsh.on() handlers
		EventEmitter: func(eventName string, ctx interpreter.Value) {
			state.Interpreter.EmitEvent(eventName, ctx)
		},

		OnToolCallEnd: func(toolCall acp.ToolCall, update acp.ToolCallUpdate) {
			// Log tool errors
			if update.Error != nil {
				m.logger.Warn("tool execution failed",
					zap.String("tool", toolCall.Name),
					zap.Error(update.Error),
				)
			}
		},

		OnComplete: func(result acp.AgentResult) {
			// Log agent completion
			m.logger.Debug("agent interaction",
				zap.String("agent", m.currentAgentName),
				zap.String("message", message),
				zap.String("stopReason", string(result.StopReason)),
				zap.Duration("duration", result.Duration),
			)
		},
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

// SendMessageToCurrentAgent sends a message to the current agent.
// All rendering is handled via events (gsh.ui.* and event handlers in gsh script).
func (m *Manager) SendMessageToCurrentAgent(ctx context.Context, message string) error {
	err := m.SendMessage(ctx, message)

	if err != nil {
		fmt.Printf("gsh: %v\n", err)
		return nil // Return nil to not propagate error to caller (matches original behavior)
	}

	return nil
}
