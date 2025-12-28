package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/repl/render"
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
func createTestState(provider interpreter.ModelProvider, systemPrompt string, tools []interpreter.ChatTool, toolExecutor ToolExecutor, maxIterations int) *State {
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
		Tools:         tools,
		ToolExecutor:  toolExecutor,
		MaxIterations: maxIterations,
		Interpreter:   interp,
	}
}

// createTestStateWithName creates a test State with a custom agent name
func createTestStateWithName(provider interpreter.ModelProvider, name string, systemPrompt string, tools []interpreter.ChatTool, toolExecutor ToolExecutor) *State {
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
		Tools:        tools,
		ToolExecutor: toolExecutor,
		Interpreter:  interp,
	}
}

func TestSendMessage_NoToolCalls(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	provider.addResponse("Hello! I'm here to help.", nil)

	state := createTestState(provider, "You are helpful.", nil, nil, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	var chunks []string
	err := manager.SendMessage(context.Background(), "Hello", func(s string) {
		chunks = append(chunks, s)
	})

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Should have received the response
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	var chunks []string
	err := manager.SendMessage(context.Background(), "What's the weather in San Francisco?", func(s string) {
		chunks = append(chunks, s)
	})

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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	err := manager.SendMessage(context.Background(), "Weather in SF and NY?", nil)

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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	err := manager.SendMessage(context.Background(), "Search and analyze gsh", nil)

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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	err := manager.SendMessage(context.Background(), "Do something infinite", nil)

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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	err := manager.SendMessage(context.Background(), "Use the failing tool", nil)

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
	logger := zap.NewNop()
	manager := NewManager(logger)

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

	err := manager.SendMessage(context.Background(), "Use a tool", nil)

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

func TestSendMessage_WithRenderer(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	provider.addResponse("Hello, I'm the assistant!", nil)

	// Create a mock renderer by using the real one with a buffer
	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	state := createTestStateWithName(provider, "test-agent", "", nil, nil)

	manager.AddAgent("test-agent", state)
	manager.SetCurrentAgent("test-agent")

	err := manager.SendMessage(context.Background(), "Hello", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify header was rendered
	if !strings.Contains(output, "agent: test-agent") {
		t.Errorf("Expected header with agent name, got: %s", output)
	}

	// Verify agent text was rendered
	if !strings.Contains(output, "Hello, I'm the assistant!") {
		t.Errorf("Expected agent response text, got: %s", output)
	}

	// Verify footer was rendered (contains token counts or duration)
	// The footer contains "in" and "out" for token counts
	if !strings.Contains(output, "in") || !strings.Contains(output, "out") {
		t.Errorf("Expected footer with token stats, got: %s", output)
	}
}

func TestSendMessage_WithRenderer_TokenAccumulation(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProviderWithUsage()
	// First call returns tool call with 100 input, 50 output tokens
	provider.addResponseWithUsage("Let me help.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "test_tool", Arguments: map[string]interface{}{}},
	}, &interpreter.ChatUsage{PromptTokens: 100, CompletionTokens: 50})
	// Second call returns final response with 150 input, 75 output tokens
	provider.addResponseWithUsage("Done!", nil, &interpreter.ChatUsage{PromptTokens: 150, CompletionTokens: 75})

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "test_tool", Description: "A test tool", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"result": "ok"}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Do something", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify accumulated token counts in footer (100+150=250 in, 50+75=125 out)
	if !strings.Contains(output, "250") {
		t.Errorf("Expected accumulated input tokens (250), got: %s", output)
	}
	if !strings.Contains(output, "125") {
		t.Errorf("Expected accumulated output tokens (125), got: %s", output)
	}
}

// mockProviderWithUsage extends mockProvider to support Usage in responses
type mockProviderWithUsage struct {
	responses     []mockResponseWithUsage
	responseIndex int
	callHistory   []interpreter.ChatRequest
}

type mockResponseWithUsage struct {
	content   string
	toolCalls []interpreter.ChatToolCall
	usage     *interpreter.ChatUsage
}

func newMockProviderWithUsage() *mockProviderWithUsage {
	return &mockProviderWithUsage{
		responses:   []mockResponseWithUsage{},
		callHistory: []interpreter.ChatRequest{},
	}
}

func (m *mockProviderWithUsage) Name() string {
	return "mock-with-usage"
}

func (m *mockProviderWithUsage) addResponseWithUsage(content string, toolCalls []interpreter.ChatToolCall, usage *interpreter.ChatUsage) {
	m.responses = append(m.responses, mockResponseWithUsage{
		content:   content,
		toolCalls: toolCalls,
		usage:     usage,
	})
}

func (m *mockProviderWithUsage) ChatCompletion(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
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
		Usage:     resp.usage,
	}, nil
}

func (m *mockProviderWithUsage) StreamingChatCompletion(request interpreter.ChatRequest, callbacks *interpreter.StreamCallbacks) (*interpreter.ChatResponse, error) {
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

func TestSendMessage_WithRenderer_NoCallback(t *testing.T) {
	// Test that renderer handles output even when callback is nil
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	provider.addResponse("Response without callback", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	state := createTestState(provider, "", nil, nil, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	// Call with nil callback
	err := manager.SendMessage(context.Background(), "Hello", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Renderer should still capture the response
	if !strings.Contains(output, "Response without callback") {
		t.Errorf("Expected renderer to capture response, got: %s", output)
	}
}

func TestSendMessage_ExecToolRendering(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	// First call returns exec tool call
	provider.addResponse("Let me run a command.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "exec", Arguments: map[string]interface{}{"command": "echo hello"}},
	})
	// Second call returns final response
	provider.addResponse("Done!", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "exec", Description: "Execute a command", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"output": "hello\n", "exitCode": 0}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Run echo", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify exec start was rendered (▶ symbol)
	if !strings.Contains(output, "▶") {
		t.Errorf("Expected exec start symbol (▶), got: %s", output)
	}

	// Verify command was shown
	if !strings.Contains(output, "echo hello") {
		t.Errorf("Expected command 'echo hello' in output, got: %s", output)
	}

	// Verify exec end was rendered (✓ symbol for success)
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol (✓), got: %s", output)
	}

	// Verify the first word of command is shown in completion
	if !strings.Contains(output, "echo") {
		t.Errorf("Expected 'echo' in completion line, got: %s", output)
	}
}

func TestSendMessage_ExecToolRendering_NonZeroExit(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	// First call returns exec tool call
	provider.addResponse("Let me run a command.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "exec", Arguments: map[string]interface{}{"command": "cat /nonexistent"}},
	})
	// Second call returns final response
	provider.addResponse("The file doesn't exist.", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "exec", Description: "Execute a command", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"output": "cat: /nonexistent: No such file or directory\n", "exitCode": 1}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Cat a file", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify exec start was rendered
	if !strings.Contains(output, "▶") {
		t.Errorf("Expected exec start symbol (▶), got: %s", output)
	}

	// Verify error symbol was shown (✗ for non-zero exit)
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol (✗) for non-zero exit, got: %s", output)
	}

	// Verify exit code is shown
	if !strings.Contains(output, "exit code 1") {
		t.Errorf("Expected 'exit code 1' in output, got: %s", output)
	}
}

func TestSendMessage_ExecToolRendering_NoRenderer(t *testing.T) {
	// Test that exec tool works fine without a renderer
	logger := zap.NewNop()
	manager := NewManager(logger)
	// No renderer set

	provider := newMockProvider()
	provider.addResponse("Let me run a command.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "exec", Arguments: map[string]interface{}{"command": "echo test"}},
	})
	provider.addResponse("Done!", nil)

	tools := []interpreter.ChatTool{
		{Name: "exec", Description: "Execute a command", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"output": "test\n", "exitCode": 0}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	var chunks []string
	err := manager.SendMessage(context.Background(), "Run echo", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should work without error
	if len(chunks) == 0 {
		t.Error("Expected some output chunks")
	}
}

func TestParseExecExitCode(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected int
	}{
		{
			name:     "zero exit code",
			result:   `{"output": "hello\n", "exitCode": 0}`,
			expected: 0,
		},
		{
			name:     "non-zero exit code",
			result:   `{"output": "error\n", "exitCode": 1}`,
			expected: 1,
		},
		{
			name:     "high exit code",
			result:   `{"output": "", "exitCode": 127}`,
			expected: 127,
		},
		{
			name:     "exit code with whitespace",
			result:   `{"output": "", "exitCode":  42}`,
			expected: 42,
		},
		{
			name:     "no exit code field",
			result:   `{"output": "hello\n"}`,
			expected: 0,
		},
		{
			name:     "empty result",
			result:   ``,
			expected: 0,
		},
		{
			name:     "truncated result",
			result:   `{"output": "...", "exitCode": 5, "truncated": true}`,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExecExitCode(tt.result)
			if got != tt.expected {
				t.Errorf("parseExecExitCode(%q) = %d, want %d", tt.result, got, tt.expected)
			}
		})
	}
}

func TestSendMessage_NonExecToolRendering(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	// First call returns a non-exec tool call
	provider.addResponse("Let me read that file.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "read_file", Arguments: map[string]interface{}{
			"path": "/home/user/config.json",
		}},
	})
	// Second call returns final response
	provider.addResponse("The file contains configuration settings.", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "read_file", Description: "Read a file", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"content": "{ \"key\": \"value\" }"}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Read the config file", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify non-exec tool symbol was rendered (○ for executing, ● for complete)
	if !strings.Contains(output, "●") {
		t.Errorf("Expected non-exec tool complete symbol (●), got: %s", output)
	}

	// Verify tool name was shown
	if !strings.Contains(output, "read_file") {
		t.Errorf("Expected tool name 'read_file' in output, got: %s", output)
	}

	// Verify success symbol was shown
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success symbol (✓), got: %s", output)
	}

	// Verify arguments were shown
	if !strings.Contains(output, "path:") {
		t.Errorf("Expected argument 'path:' in output, got: %s", output)
	}

	// Verify the path value is shown
	if !strings.Contains(output, "/home/user/config.json") {
		t.Errorf("Expected path value in output, got: %s", output)
	}
}

func TestSendMessage_NonExecToolRendering_Error(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	// First call returns a non-exec tool call
	provider.addResponse("Let me search for that.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "search", Arguments: map[string]interface{}{
			"query":     "error handling",
			"directory": "/src",
		}},
	})
	// Second call returns final response
	provider.addResponse("I couldn't complete the search.", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "search", Description: "Search files", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return "", fmt.Errorf("search index not available")
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Search for error handling", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify non-exec tool complete symbol was rendered
	if !strings.Contains(output, "●") {
		t.Errorf("Expected non-exec tool complete symbol (●), got: %s", output)
	}

	// Verify tool name was shown
	if !strings.Contains(output, "search") {
		t.Errorf("Expected tool name 'search' in output, got: %s", output)
	}

	// Verify error symbol was shown (✗ for failure)
	if !strings.Contains(output, "✗") {
		t.Errorf("Expected error symbol (✗), got: %s", output)
	}
}

func TestSendMessage_NonExecToolRendering_MultipleArgs(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	provider := newMockProvider()
	// First call returns a non-exec tool call with multiple arguments
	provider.addResponse("Let me write that file.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "write_file", Arguments: map[string]interface{}{
			"path":    "/output.txt",
			"content": "Hello, World!",
			"mode":    "overwrite",
		}},
	})
	// Second call returns final response
	provider.addResponse("File written successfully.", nil)

	var buf bytes.Buffer
	renderer := render.New(nil, &buf, func() int { return 80 })
	manager.SetRenderer(renderer)

	tools := []interpreter.ChatTool{
		{Name: "write_file", Description: "Write a file", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"success": true}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "Write a file", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify tool name was shown
	if !strings.Contains(output, "write_file") {
		t.Errorf("Expected tool name 'write_file' in output, got: %s", output)
	}

	// Verify multiple arguments are shown (at least path and content)
	if !strings.Contains(output, "path:") {
		t.Errorf("Expected argument 'path:' in output, got: %s", output)
	}
	if !strings.Contains(output, "content:") {
		t.Errorf("Expected argument 'content:' in output, got: %s", output)
	}
}

func TestSendMessage_NonExecToolRendering_NoRenderer(t *testing.T) {
	// Test that non-exec tools work fine without a renderer
	logger := zap.NewNop()
	manager := NewManager(logger)
	// No renderer set

	provider := newMockProvider()
	provider.addResponse("Let me read that.", []interpreter.ChatToolCall{
		{ID: "call1", Name: "read_file", Arguments: map[string]interface{}{"path": "/test.txt"}},
	})
	provider.addResponse("Done!", nil)

	tools := []interpreter.ChatTool{
		{Name: "read_file", Description: "Read a file", Parameters: map[string]interface{}{}},
	}

	toolExecutor := func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		return `{"content": "file contents"}`, nil
	}

	state := createTestState(provider, "", tools, toolExecutor, 0)

	manager.AddAgent("test", state)
	manager.SetCurrentAgent("test")

	var chunks []string
	err := manager.SendMessage(context.Background(), "Read file", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should work without error
	if len(chunks) == 0 {
		t.Error("Expected some output chunks")
	}
}
