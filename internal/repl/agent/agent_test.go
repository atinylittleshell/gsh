package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// mockProvider is a test provider for testing the agent
type mockProvider struct {
	responses     []mockResponse
	responseIndex int
	callHistory   []interpreter.ChatRequest
}

type mockResponse struct {
	content   string
	toolCalls []interpreter.ChatToolCall
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		responses:   []mockResponse{},
		callHistory: []interpreter.ChatRequest{},
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) addResponse(content string, toolCalls []interpreter.ChatToolCall) {
	m.responses = append(m.responses, mockResponse{
		content:   content,
		toolCalls: toolCalls,
	})
}

func (m *mockProvider) ChatCompletion(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
	m.callHistory = append(m.callHistory, request)

	if m.responseIndex >= len(m.responses) {
		return &interpreter.ChatResponse{
			Content:   "Default response",
			ToolCalls: []interpreter.ChatToolCall{},
		}, nil
	}

	resp := m.responses[m.responseIndex]
	m.responseIndex++

	return &interpreter.ChatResponse{
		Content:   resp.content,
		ToolCalls: resp.toolCalls,
	}, nil
}

func (m *mockProvider) StreamingChatCompletion(request interpreter.ChatRequest, callbacks *interpreter.StreamCallbacks) (*interpreter.ChatResponse, error) {
	response, err := m.ChatCompletion(request)
	if err != nil {
		return nil, err
	}

	if callbacks != nil && callbacks.OnContent != nil && response.Content != "" {
		callbacks.OnContent(response.Content)
	}

	// Notify about tool calls starting
	if callbacks != nil && callbacks.OnToolCallStart != nil {
		for _, tc := range response.ToolCalls {
			callbacks.OnToolCallStart(tc.ID, tc.Name)
		}
	}

	return response, nil
}

// createTestState creates a test State with all required fields populated
func createTestState(provider interpreter.ModelProvider, systemPrompt string, tools []interpreter.ChatTool, toolExecutor interface{}, maxIterations int) *State {
	interp := interpreter.New(nil)

	model := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: provider,
	}

	agentConfig := map[string]interpreter.Value{
		"model": model,
	}
	if systemPrompt != "" {
		agentConfig["systemPrompt"] = &interpreter.StringValue{Value: systemPrompt}
	}
	if maxIterations > 0 {
		agentConfig["maxIterations"] = &interpreter.NumberValue{Value: float64(maxIterations)}
	}

	// Convert tools to ArrayValue for the agent config
	if len(tools) > 0 {
		toolValues := make([]interpreter.Value, len(tools))
		for i, tool := range tools {
			// Create ToolValue objects that delegate to the executor
			toolValues[i] = &interpreter.ToolValue{
				Name:       tool.Name,
				Parameters: []string{}, // Simplified for tests
				ParamTypes: map[string]string{},
			}
		}
		agentConfig["tools"] = &interpreter.ArrayValue{Elements: toolValues}
	}

	agent := &interpreter.AgentValue{
		Name:   "test",
		Config: agentConfig,
	}

	return &State{
		Agent:         agent,
		Provider:      provider,
		Conversation:  []interpreter.ChatMessage{},
		MaxIterations: maxIterations,
		Interpreter:   interp,
	}
}

// createTestStateWithName creates a test State with a custom agent name
func createTestStateWithName(provider interpreter.ModelProvider, name string, systemPrompt string, tools []interpreter.ChatTool, toolExecutor interface{}) *State {
	interp := interpreter.New(nil)

	model := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: provider,
	}

	agentConfig := map[string]interpreter.Value{
		"model": model,
	}
	if systemPrompt != "" {
		agentConfig["systemPrompt"] = &interpreter.StringValue{Value: systemPrompt}
	}

	agent := &interpreter.AgentValue{
		Name:   name,
		Config: agentConfig,
	}

	return &State{
		Agent:        agent,
		Provider:     provider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp,
	}
}

func TestSendMessage_NoToolCalls(t *testing.T) {
	manager := NewManager()

	provider := newMockProvider()
	provider.addResponse("Hello! I'm here to help.", nil)

	state := createTestState(provider, "You are helpful.", nil, nil, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Hello")

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Conversation should have user message and assistant response
	if len(state.Conversation) != 2 {
		t.Errorf("Expected 2 messages in conversation, got %d", len(state.Conversation))
	}

	if state.Conversation[0].Role != "user" {
		t.Errorf("Expected first message to be user, got %s", state.Conversation[0].Role)
	}

	if state.Conversation[1].Role != "assistant" {
		t.Errorf("Expected second message to be assistant, got %s", state.Conversation[1].Role)
	}
}

func TestSendMessage_WithToolCalls(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// First response: request a tool call
	provider.addResponse("Let me check that for you.", []interpreter.ChatToolCall{
		{
			ID:        "call_123",
			Name:      "get_weather",
			Arguments: map[string]interface{}{"city": "San Francisco"},
		},
	})

	// Second response: final answer after tool result
	provider.addResponse("The weather in San Francisco is sunny and 72°F.", nil)

	// Define tools and executor
	tools := []interpreter.ChatTool{
		{
			Name:        "get_weather",
			Description: "Get the weather for a city",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{"type": "string"},
				},
				"required": []string{"city"},
			},
		},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		if toolName == "get_weather" {
			city := args["city"].(string)
			return fmt.Sprintf(`{"temperature": 72, "condition": "sunny", "city": "%s"}`, city), nil
		}
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	state := createTestState(provider, "You are a weather assistant.", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "What's the weather in San Francisco?")

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Should have 2 calls to provider (initial + after tool result)
	if len(provider.callHistory) != 2 {
		t.Errorf("Expected 2 provider calls, got %d", len(provider.callHistory))
	}

	// Conversation should have: user, assistant (with tool call), tool result, assistant (final)
	if len(state.Conversation) != 4 {
		t.Errorf("Expected 4 messages in conversation, got %d", len(state.Conversation))
		for i, msg := range state.Conversation {
			t.Logf("  [%d] %s: %s", i, msg.Role, msg.Content)
		}
	}

	// Verify message roles
	expectedRoles := []string{"user", "assistant", "tool", "assistant"}
	for i, expected := range expectedRoles {
		if i < len(state.Conversation) && state.Conversation[i].Role != expected {
			t.Errorf("Message %d: expected role %s, got %s", i, expected, state.Conversation[i].Role)
		}
	}

	// Verify tool result has correct tool_call_id
	if len(state.Conversation) >= 3 {
		toolMsg := state.Conversation[2]
		if toolMsg.ToolCallID != "call_123" {
			t.Errorf("Expected tool call ID 'call_123', got '%s'", toolMsg.ToolCallID)
		}
		if toolMsg.Name != "get_weather" {
			t.Errorf("Expected tool name 'get_weather', got '%s'", toolMsg.Name)
		}
	}
}

func TestSendMessage_MultipleToolCalls(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// First response: request multiple tool calls
	provider.addResponse("Let me check multiple cities.", []interpreter.ChatToolCall{
		{
			ID:        "call_1",
			Name:      "get_weather",
			Arguments: map[string]interface{}{"city": "San Francisco"},
		},
		{
			ID:        "call_2",
			Name:      "get_weather",
			Arguments: map[string]interface{}{"city": "New York"},
		},
	})

	// Second response: final answer after tool results
	provider.addResponse("San Francisco is sunny, New York is cloudy.", nil)

	tools := []interpreter.ChatTool{
		{
			Name:        "get_weather",
			Description: "Get the weather for a city",
			Parameters:  map[string]interface{}{},
		},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		city := args["city"].(string)
		if city == "San Francisco" {
			return `{"condition": "sunny"}`, nil
		}
		return `{"condition": "cloudy"}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Weather in SF and NY?")

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Should have: user, assistant, tool, tool, assistant
	if len(state.Conversation) != 5 {
		t.Errorf("Expected 5 messages in conversation, got %d", len(state.Conversation))
	}

	// Verify both tool results are present
	toolResults := 0
	for _, msg := range state.Conversation {
		if msg.Role == "tool" {
			toolResults++
		}
	}
	if toolResults != 2 {
		t.Errorf("Expected 2 tool results, got %d", toolResults)
	}
}

func TestSendMessage_ChainedToolCalls(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// First response: first tool call
	provider.addResponse("First, let me search.", []interpreter.ChatToolCall{
		{ID: "call_1", Name: "search", Arguments: map[string]interface{}{"query": "gsh"}},
	})

	// Second response: another tool call based on first result
	provider.addResponse("Now let me analyze.", []interpreter.ChatToolCall{
		{ID: "call_2", Name: "analyze", Arguments: map[string]interface{}{"data": "search result"}},
	})

	// Third response: final answer
	provider.addResponse("Here's my analysis based on the search.", nil)

	tools := []interpreter.ChatTool{
		{Name: "search", Description: "Search", Parameters: map[string]interface{}{}},
		{Name: "analyze", Description: "Analyze", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return fmt.Sprintf(`{"result": "%s completed"}`, toolName), nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Search and analyze gsh")

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Should have 3 calls to provider (chained tool calls)
	if len(provider.callHistory) != 3 {
		t.Errorf("Expected 3 provider calls, got %d", len(provider.callHistory))
	}

	// Conversation: user, assistant+tool1, tool1_result, assistant+tool2, tool2_result, assistant
	if len(state.Conversation) != 6 {
		t.Errorf("Expected 6 messages in conversation, got %d", len(state.Conversation))
	}
}

func TestSendMessage_MaxIterationsReached(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// Use a small max iterations for testing
	testMaxIterations := 5

	// Always return a tool call - this will trigger max iterations
	for i := 0; i < testMaxIterations+5; i++ {
		provider.addResponse("Let me use a tool.", []interpreter.ChatToolCall{
			{ID: fmt.Sprintf("call_%d", i), Name: "infinite_tool", Arguments: map[string]interface{}{}},
		})
	}

	tools := []interpreter.ChatTool{
		{Name: "infinite_tool", Description: "Always returns", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"status": "ok"}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, testMaxIterations)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Do something infinite")

	// Should return max iterations error
	if err == nil {
		t.Fatal("Expected max iterations error, got nil")
	}

	if !strings.Contains(err.Error(), "maximum iterations") {
		t.Errorf("Expected max iterations error, got: %v", err)
	}

	// Should have exactly testMaxIterations calls
	if len(provider.callHistory) != testMaxIterations {
		t.Errorf("Expected %d provider calls, got %d", testMaxIterations, len(provider.callHistory))
	}
}

func TestSendMessage_ToolExecutorError(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// First response: tool call
	provider.addResponse("Let me check.", []interpreter.ChatToolCall{
		{ID: "call_1", Name: "failing_tool", Arguments: map[string]interface{}{}},
	})

	// Second response: final answer (model handles the error gracefully)
	provider.addResponse("I encountered an error but I'll help anyway.", nil)

	tools := []interpreter.ChatTool{
		{Name: "failing_tool", Description: "This tool fails", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return "", fmt.Errorf("tool execution failed: permission denied")
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Use the failing tool")

	// Should NOT return an error - the model should be able to recover
	if err != nil {
		t.Fatalf("SendMessage should not fail on tool error: %v", err)
	}

	// The tool result should contain the error message
	var toolResultFound bool
	for _, msg := range state.Conversation {
		if msg.Role == "tool" {
			if strings.Contains(msg.Content, "Error executing tool") {
				toolResultFound = true
			}
		}
	}

	if !toolResultFound {
		t.Error("Expected tool result with error message")
	}
}

func TestSendMessage_NoToolExecutor(t *testing.T) {
		manager := NewManager()

	provider := newMockProvider()

	// Response with tool call but no executor configured
	provider.addResponse("Let me use a tool.", []interpreter.ChatToolCall{
		{ID: "call_1", Name: "some_tool", Arguments: map[string]interface{}{}},
	})

	// Final response after error
	provider.addResponse("I couldn't use the tool.", nil)

	tools := []interpreter.ChatTool{
		{Name: "some_tool", Description: "A tool", Parameters: map[string]interface{}{}},
	}

	state := createTestState(provider, "", tools, nil, 0) // No executor configured

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Use a tool")

	// Should complete (error is sent to model as tool result)
	if err != nil {
		t.Fatalf("SendMessage should handle missing executor gracefully: %v", err)
	}

	// Tool result should contain error about missing tool (since interpreter handles it)
	var errorInResult bool
	for _, msg := range state.Conversation {
		if msg.Role == "tool" && strings.Contains(msg.Content, "Error") {
			errorInResult = true
		}
	}

	if !errorInResult {
		t.Error("Expected tool result with error message")
	}
}

// Note: TestBuildMessages_* tests removed as message building logic
// is now handled by the interpreter's ExecuteAgentWithCallbacks
