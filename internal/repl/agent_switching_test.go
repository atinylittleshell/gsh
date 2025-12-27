package repl

import (
	"context"
	"testing"

	"github.com/atinylittleshell/gsh/internal/repl/agent"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParseAgentInput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantIsCommand bool
		wantContent   string
	}{
		{
			name:          "regular message",
			input:         "hello world",
			wantIsCommand: false,
			wantContent:   "hello world",
		},
		{
			name:          "command /clear",
			input:         "/clear",
			wantIsCommand: true,
			wantContent:   "clear",
		},
		{
			name:          "command /agents",
			input:         "/agents",
			wantIsCommand: true,
			wantContent:   "agents",
		},
		{
			name:          "command /agent with name",
			input:         "/agent coder",
			wantIsCommand: true,
			wantContent:   "agent coder",
		},
		{
			name:          "command with leading whitespace",
			input:         "  /clear",
			wantIsCommand: true,
			wantContent:   "clear",
		},
		{
			name:          "message with leading whitespace",
			input:         "  hello world",
			wantIsCommand: false,
			wantContent:   "  hello world",
		},
		{
			name:          "empty input",
			input:         "",
			wantIsCommand: false,
			wantContent:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsCommand, gotContent := parseAgentInput(tt.input)
			assert.Equal(t, tt.wantIsCommand, gotIsCommand)
			assert.Equal(t, tt.wantContent, gotContent)
		})
	}
}

// createTestREPLWithAgents is a helper to create a REPL with agents for testing
func createTestREPLWithAgents(logger *zap.Logger, agents map[string]*agent.State, currentAgent string) *REPL {
	mgr := agent.NewManager(logger)
	for name, state := range agents {
		mgr.AddAgent(name, state)
	}
	if currentAgent != "" {
		_ = mgr.SetCurrentAgent(currentAgent)
	}
	return &REPL{
		logger:       logger,
		agentManager: mgr,
	}
}

func TestHandleAgentCommand_SwitchAgent(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	// Create multiple agents
	agent1 := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}
	agent2 := &interpreter.AgentValue{
		Name: "agent2",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model2"},
		},
	}

	state1 := &agent.State{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}
	state2 := &agent.State{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": state1,
		"agent2": state2,
	}, "agent1")

	ctx := context.Background()

	// Switch to agent2
	err := repl.handleAgentCommand(ctx, "/agent agent2")
	assert.NoError(t, err)
	assert.Equal(t, "agent2", repl.agentManager.CurrentAgentName())
}

func TestHandleAgentCommand_SwitchToInvalidAgent(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agent1 := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}

	state1 := &agent.State{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": state1,
	}, "agent1")

	ctx := context.Background()

	// Try to switch to non-existent agent
	err := repl.handleAgentCommand(ctx, "/agent nonexistent")
	assert.NoError(t, err)
	// Should stay on current agent
	assert.Equal(t, "agent1", repl.agentManager.CurrentAgentName())
}

func TestHandleAgentCommand_ListAgents(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agent1 := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model":       &interpreter.ModelValue{Name: "model1"},
			"description": &interpreter.StringValue{Value: "First agent"},
		},
	}
	agent2 := &interpreter.AgentValue{
		Name: "agent2",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model2"},
		},
	}

	state1 := &agent.State{
		Agent:    agent1,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}
	state2 := &agent.State{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": state1,
		"agent2": state2,
	}, "agent1")

	ctx := context.Background()

	// List agents
	err := repl.handleAgentCommand(ctx, "/agents")
	assert.NoError(t, err)
}

func TestHandleAgentCommand_ConversationIsolation(t *testing.T) {
	// Test that switching agents preserves separate conversation histories

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agent1 := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1", Provider: mockProvider},
		},
	}
	agent2 := &interpreter.AgentValue{
		Name: "agent2",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model2", Provider: mockProvider},
		},
	}

	// Create interpreters for each agent state
	interp1 := interpreter.New()
	interp2 := interpreter.New()

	state1 := &agent.State{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp1,
	}
	state2 := &agent.State{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp2,
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": state1,
		"agent2": state2,
	}, "agent1")

	ctx := context.Background()

	// Send message to agent1
	err := repl.handleAgentCommand(ctx, "hello from agent1")
	require.NoError(t, err)
	assert.Len(t, state1.Conversation, 2) // user + assistant

	// Switch to agent2
	err = repl.handleAgentCommand(ctx, "/agent agent2")
	require.NoError(t, err)

	// Send message to agent2
	err = repl.handleAgentCommand(ctx, "hello from agent2")
	require.NoError(t, err)
	assert.Len(t, state2.Conversation, 2) // user + assistant

	// Agent1 conversation should be unchanged
	assert.Len(t, state1.Conversation, 2)
	assert.Equal(t, "hello from agent1", state1.Conversation[0].Content)

	// Agent2 should have its own conversation
	assert.Equal(t, "hello from agent2", state2.Conversation[0].Content)

	// Switch back to agent1
	err = repl.handleAgentCommand(ctx, "/agent agent1")
	require.NoError(t, err)

	// Send another message to agent1
	err = repl.handleAgentCommand(ctx, "second message to agent1")
	require.NoError(t, err)
	assert.Len(t, state1.Conversation, 4) // 2 previous + 2 new

	// Agent2 conversation should be unchanged
	assert.Len(t, state2.Conversation, 2)
}

func TestHandleAgentCommand_ClearCurrentAgent(t *testing.T) {
	// Test that /clear only clears the current agent's conversation

	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agent1 := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}
	agent2 := &interpreter.AgentValue{
		Name: "agent2",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model2"},
		},
	}

	state1 := &agent.State{
		Agent:    agent1,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "message1"},
			{Role: "assistant", Content: "response1"},
		},
	}
	state2 := &agent.State{
		Agent:    agent2,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "message2"},
			{Role: "assistant", Content: "response2"},
		},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": state1,
		"agent2": state2,
	}, "agent1")

	ctx := context.Background()

	// Clear agent1's conversation
	err := repl.handleAgentCommand(ctx, "/clear")
	require.NoError(t, err)

	// Agent1 should be cleared
	assert.Len(t, state1.Conversation, 0)

	// Agent2 should be unchanged
	assert.Len(t, state2.Conversation, 2)
}

func TestHandleAgentCommand_UnknownCommand(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agentVal := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}

	state := &agent.State{
		Agent:        agentVal,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{"agent1": state}, "agent1")

	ctx := context.Background()

	// Try unknown command
	err := repl.handleAgentCommand(ctx, "/unknown")
	assert.NoError(t, err)
}

func TestHandleAgentCommand_AgentCommandMissingName(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agentVal := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}

	state := &agent.State{
		Agent:        agentVal,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{"agent1": state}, "agent1")

	ctx := context.Background()

	// Try /agent without name
	err := repl.handleAgentCommand(ctx, "/agent")
	assert.NoError(t, err)
	// Should still be on agent1
	assert.Equal(t, "agent1", repl.agentManager.CurrentAgentName())
}

func TestGetAgentNames(t *testing.T) {
	logger := zap.NewNop()
	mockProvider := &MockProvider{
		responseContent: "Response",
		shouldError:     false,
	}

	agent1 := &interpreter.AgentValue{Name: "agent1"}
	agent2 := &interpreter.AgentValue{Name: "agent2"}
	agent3 := &interpreter.AgentValue{Name: "agent3"}

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{
		"agent1": {Agent: agent1, Provider: mockProvider},
		"agent2": {Agent: agent2, Provider: mockProvider},
		"agent3": {Agent: agent3, Provider: mockProvider},
	}, "agent1")

	names := repl.GetAgentNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "agent1")
	assert.Contains(t, names, "agent2")
	assert.Contains(t, names, "agent3")
}

func TestGetAgentNames_NoAgents(t *testing.T) {
	logger := zap.NewNop()

	repl := createTestREPLWithAgents(logger, map[string]*agent.State{}, "")

	names := repl.GetAgentNames()
	assert.Len(t, names, 0)
}
