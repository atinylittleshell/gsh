package repl

import (
	"context"
	"testing"

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

	state1 := &AgentState{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}
	state2 := &AgentState{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": state1,
			"agent2": state2,
		},
		currentAgentName: "agent1",
	}

	ctx := context.Background()

	// Switch to agent2
	err := repl.handleAgentCommand(ctx, "/agent agent2")
	assert.NoError(t, err)
	assert.Equal(t, "agent2", repl.currentAgentName)
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

	state1 := &AgentState{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": state1,
		},
		currentAgentName: "agent1",
	}

	ctx := context.Background()

	// Try to switch to non-existent agent
	err := repl.handleAgentCommand(ctx, "/agent nonexistent")
	assert.NoError(t, err)
	// Should stay on current agent
	assert.Equal(t, "agent1", repl.currentAgentName)
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

	state1 := &AgentState{
		Agent:    agent1,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}
	state2 := &AgentState{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": state1,
			"agent2": state2,
		},
		currentAgentName: "agent1",
	}

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
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}
	agent2 := &interpreter.AgentValue{
		Name: "agent2",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model2"},
		},
	}

	state1 := &AgentState{
		Agent:        agent1,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}
	state2 := &AgentState{
		Agent:        agent2,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": state1,
			"agent2": state2,
		},
		currentAgentName: "agent1",
	}

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

	state1 := &AgentState{
		Agent:    agent1,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "message1"},
			{Role: "assistant", Content: "response1"},
		},
	}
	state2 := &AgentState{
		Agent:    agent2,
		Provider: mockProvider,
		Conversation: []interpreter.ChatMessage{
			{Role: "user", Content: "message2"},
			{Role: "assistant", Content: "response2"},
		},
	}

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": state1,
			"agent2": state2,
		},
		currentAgentName: "agent1",
	}

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

	agent := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}

	state := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"agent1": state},
		currentAgentName: "agent1",
	}

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

	agent := &interpreter.AgentValue{
		Name: "agent1",
		Config: map[string]interpreter.Value{
			"model": &interpreter.ModelValue{Name: "model1"},
		},
	}

	state := &AgentState{
		Agent:        agent,
		Provider:     mockProvider,
		Conversation: []interpreter.ChatMessage{},
	}

	repl := &REPL{
		logger:           logger,
		agentStates:      map[string]*AgentState{"agent1": state},
		currentAgentName: "agent1",
	}

	ctx := context.Background()

	// Try /agent without name
	err := repl.handleAgentCommand(ctx, "/agent")
	assert.NoError(t, err)
	// Should still be on agent1
	assert.Equal(t, "agent1", repl.currentAgentName)
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

	repl := &REPL{
		logger: logger,
		agentStates: map[string]*AgentState{
			"agent1": {Agent: agent1, Provider: mockProvider},
			"agent2": {Agent: agent2, Provider: mockProvider},
			"agent3": {Agent: agent3, Provider: mockProvider},
		},
		currentAgentName: "agent1",
	}

	names := repl.GetAgentNames()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "agent1")
	assert.Contains(t, names, "agent2")
	assert.Contains(t, names, "agent3")
}

func TestGetAgentNames_NoAgents(t *testing.T) {
	logger := zap.NewNop()

	repl := &REPL{
		logger:      logger,
		agentStates: make(map[string]*AgentState),
	}

	names := repl.GetAgentNames()
	assert.Len(t, names, 0)
}
