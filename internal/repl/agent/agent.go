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

// timeNow is a variable that can be overridden for testing.
var timeNow = time.Now

// State holds the state for a single agent.
type State struct {
	Agent        *interpreter.AgentValue
	Provider     interpreter.ModelProvider
	Conversation []interpreter.ChatMessage
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
func (m *Manager) SendMessage(ctx context.Context, message string, onChunk func(string)) error {
	if m.currentAgentName == "" {
		return fmt.Errorf("no current agent")
	}

	state := m.states[m.currentAgentName]
	if state == nil {
		return fmt.Errorf("current agent state not found")
	}

	// Build messages for the provider
	messages := make([]interpreter.ChatMessage, 0, len(state.Conversation)+2)

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

	// Add new user message
	messages = append(messages, interpreter.ChatMessage{
		Role:    "user",
		Content: message,
	})

	// Get the model from agent config
	var model *interpreter.ModelValue
	if modelVal, ok := state.Agent.Config["model"]; ok {
		model, _ = modelVal.(*interpreter.ModelValue)
	}

	// Call provider with streaming to display response in real-time
	startTime := timeNow()
	response, err := state.Provider.StreamingChatCompletion(
		interpreter.ChatRequest{
			Model:    model,
			Messages: messages,
		},
		func(content string) {
			if onChunk != nil {
				onChunk(content)
			}
		},
	)
	duration := timeNow().Sub(startTime)

	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	// Update conversation history (don't include system prompt in history)
	state.Conversation = append(state.Conversation,
		interpreter.ChatMessage{Role: "user", Content: message},
		interpreter.ChatMessage{Role: "assistant", Content: response.Content},
	)

	// Log interaction
	m.logger.Debug("agent interaction",
		zap.String("agent", m.currentAgentName),
		zap.String("message", message),
		zap.String("response", response.Content),
		zap.Duration("duration", duration),
	)

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
