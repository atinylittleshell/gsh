package interpreter

import (
	"context"
	"strings"
	"testing"
)

// getEmittedEvents is a helper that extracts the emitted event names from the interpreter's emittedEvents variable
func getEmittedEvents(interp *Interpreter) []string {
	eventsVal, ok := interp.env.Get("emittedEvents")
	if !ok || eventsVal == nil {
		return nil
	}
	arrVal, ok := eventsVal.(*ArrayValue)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arrVal.Elements))
	for _, el := range arrVal.Elements {
		if strVal, ok := el.(*StringValue); ok {
			result = append(result, strVal.Value)
		}
	}
	return result
}

// TestAgentEventsEmitted tests that agent lifecycle events are properly emitted
// during agent execution through the event manager.
func TestAgentEventsEmitted(t *testing.T) {
	// Create a mock provider that returns a simple response
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Register event handlers using gsh script that track events in an array
	// Note: gsh uses simple assignment for variables, not 'let'
	_, err := interp.EvalString(`
emittedEvents = []

tool onAgentStart(ctx) {
	emittedEvents.push("agent.start")
}
tool onAgentIterationStart(ctx) {
	emittedEvents.push("agent.iteration.start")
}
tool onAgentIterationEnd(ctx) {
	emittedEvents.push("agent.iteration.end")
}
tool onAgentEnd(ctx) {
	emittedEvents.push("agent.end")
}

gsh.on("agent.start", onAgentStart)
gsh.on("agent.iteration.start", onAgentIterationStart)
gsh.on("agent.iteration.end", onAgentIterationEnd)
gsh.on("agent.end", onAgentEnd)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Create agent
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
		},
	}

	// Create conversation with the user message
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Execute agent
	_, err = interp.ExecuteAgent(context.Background(), conv, agent, false)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// Verify events were emitted in correct order
	expectedEvents := []string{
		EventAgentStart,
		EventAgentIterationStart,
		EventAgentIterationEnd,
		EventAgentEnd,
	}

	emittedEvents := getEmittedEvents(interp)

	if len(emittedEvents) != len(expectedEvents) {
		t.Errorf("Expected %d events, got %d: %v", len(expectedEvents), len(emittedEvents), emittedEvents)
	}

	for i, expected := range expectedEvents {
		if i >= len(emittedEvents) {
			t.Errorf("Missing event at index %d: expected %s", i, expected)
			continue
		}
		if emittedEvents[i] != expected {
			t.Errorf("Event at index %d: expected %s, got %s", i, expected, emittedEvents[i])
		}
	}
}

// TestAgentToolEventsEmitted tests that tool events are emitted during tool execution.
func TestAgentToolEventsEmitted(t *testing.T) {
	// Create a mock provider
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Register event handlers using gsh script that track tool events in an array
	// Note: gsh uses simple assignment for variables, not 'let'
	_, err := interp.EvalString(`
emittedEvents = []

tool onToolStart(ctx) {
	emittedEvents.push("agent.tool.start")
}
tool onToolEnd(ctx) {
	emittedEvents.push("agent.tool.end")
}

gsh.on("agent.tool.start", onToolStart)
gsh.on("agent.tool.end", onToolEnd)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Create a weather tool
	weatherTool := &ToolValue{
		Name: "get_weather",
	}

	// Create agent with the weather tool
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
			"tools": &ArrayValue{
				Elements: []Value{weatherTool},
			},
		},
	}

	// Create conversation that will trigger tool use
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
	}

	// Create callbacks with tool executor
	callbacks := &AgentCallbacks{
		ToolExecutor: func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
			return `{"weather": "sunny", "temp": 72}`, nil
		},
	}

	// Execute agent with callbacks for tool execution
	_, err = interp.ExecuteAgentWithCallbacks(context.Background(), conv, agent, false, callbacks)
	if err != nil {
		t.Fatalf("ExecuteAgentWithCallbacks failed: %v", err)
	}

	// Check that tool events were emitted
	emittedEvents := getEmittedEvents(interp)

	hasToolStart := false
	hasToolEnd := false
	for _, event := range emittedEvents {
		if event == EventAgentToolStart {
			hasToolStart = true
		}
		if event == EventAgentToolEnd {
			hasToolEnd = true
		}
	}

	if !hasToolStart {
		t.Error("Expected agent.tool.start event to be emitted")
	}
	if !hasToolEnd {
		t.Error("Expected agent.tool.end event to be emitted")
	}
}

// TestAgentChunkEventsEmitted tests that chunk events are emitted during streaming.
func TestAgentChunkEventsEmitted(t *testing.T) {
	// Create a mock provider
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Register event handler using gsh script that tracks chunk count
	_, err := interp.EvalString(`
chunkCount = 0

tool onChunk(ctx) {
	chunkCount = chunkCount + 1
}

gsh.on("agent.chunk", onChunk)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Create agent
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
		},
	}

	// Create conversation
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Execute agent with streaming enabled
	_, err = interp.ExecuteAgent(context.Background(), conv, agent, true)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// Verify that at least one chunk was emitted
	chunkCountVal, ok := interp.env.Get("chunkCount")
	if !ok || chunkCountVal == nil {
		t.Fatal("chunkCount variable not found")
	}
	numVal, ok := chunkCountVal.(*NumberValue)
	if !ok {
		t.Fatalf("chunkCount is not a number, got %s", chunkCountVal.Type())
	}
	if numVal.Value == 0 {
		t.Error("Expected at least one chunk event to be emitted")
	}
}

// TestEventConstants tests that event constant names are correct
func TestEventConstants(t *testing.T) {
	// Verify event constants match expected values
	tests := []struct {
		constant string
		expected string
	}{
		{EventAgentStart, "agent.start"},
		{EventAgentEnd, "agent.end"},
		{EventAgentIterationStart, "agent.iteration.start"},
		{EventAgentIterationEnd, "agent.iteration.end"},
		{EventAgentToolStart, "agent.tool.start"},
		{EventAgentToolEnd, "agent.tool.end"},
		{EventAgentChunk, "agent.chunk"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("Event constant: expected %s, got %s", tt.expected, tt.constant)
		}
	}
}

func TestExtractToolOverride(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected *ToolOverride
	}{
		{
			name:     "nil value returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "null value returns nil",
			input:    &NullValue{},
			expected: nil,
		},
		{
			name:     "non-object value returns nil",
			input:    &StringValue{Value: "test"},
			expected: nil,
		},
		{
			name: "object without result property returns nil",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"other": {Value: &StringValue{Value: "value"}},
				},
			},
			expected: nil,
		},
		{
			name: "object with null result returns nil",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"result": {Value: &NullValue{}},
				},
			},
			expected: nil,
		},
		{
			name: "object with non-string result returns nil",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"result": {Value: &NumberValue{Value: 123}},
				},
			},
			expected: nil,
		},
		{
			name: "object with string result returns override",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"result": {Value: &StringValue{Value: "permission denied"}},
				},
			},
			expected: &ToolOverride{Result: "permission denied", Error: ""},
		},
		{
			name: "object with result and error returns override",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"result": {Value: &StringValue{Value: "failed"}},
					"error":  {Value: &StringValue{Value: "not allowed"}},
				},
			},
			expected: &ToolOverride{Result: "failed", Error: "not allowed"},
		},
		{
			name: "object with result and non-string error ignores error",
			input: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"result": {Value: &StringValue{Value: "success"}},
					"error":  {Value: &NumberValue{Value: 500}},
				},
			},
			expected: &ToolOverride{Result: "success", Error: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolOverride(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %+v, got nil", tt.expected)
				} else if result.Result != tt.expected.Result || result.Error != tt.expected.Error {
					t.Errorf("expected %+v, got %+v", tt.expected, result)
				}
			}
		})
	}
}

func TestToolStartOverride(t *testing.T) {
	// Create a mock provider that will request a tool call
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Register an event handler that overrides tool execution on agent.tool.start
	_, err := interp.EvalString(`
toolWasExecuted = false
overrideMessage = "Tool execution blocked by permission system"

tool onToolStart(ctx) {
	# Block any tool named get_weather
	if (ctx.toolCall.name == "get_weather") {
		return { result: overrideMessage }
	}
	# No return = allow normal execution
}

gsh.on("agent.tool.start", onToolStart)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Create a weather tool (will be triggered by SmartMockProvider)
	weatherTool := &ToolValue{
		Name: "get_weather",
	}

	// Create agent with the tool
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
			"tools":        &ArrayValue{Elements: []Value{weatherTool}},
		},
	}

	// Create conversation that triggers the weather tool
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
	}

	// Execute agent with callbacks (tool executor won't be called due to override)
	callbacks := &AgentCallbacks{
		ToolExecutor: func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
			// This should NOT be called because we override in agent.tool.start
			interp.env.Set("toolWasExecuted", &BoolValue{Value: true})
			return `{"weather": "sunny"}`, nil
		},
	}

	result, err := interp.ExecuteAgentWithCallbacks(context.Background(), conv, agent, false, callbacks)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// Verify tool was NOT executed
	toolWasExecuted, _ := interp.env.Get("toolWasExecuted")
	if boolVal, ok := toolWasExecuted.(*BoolValue); ok && boolVal.Value {
		t.Error("Tool should NOT have been executed due to override")
	}

	// Verify the conversation contains the override result
	convResult, ok := result.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result)
	}

	// Find the tool message in the conversation
	found := false
	for _, msg := range convResult.Messages {
		if msg.Role == "tool" {
			found = true
			overrideMsg, _ := interp.env.Get("overrideMessage")
			expected := overrideMsg.(*StringValue).Value
			if msg.Content != expected {
				t.Errorf("Expected tool result %q, got %q", expected, msg.Content)
			}
		}
	}
	if !found {
		t.Error("Expected to find a tool message in the conversation")
	}
}

func TestToolEndOverride(t *testing.T) {
	// Create a mock provider that will request a tool call
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Define a weather tool that returns sensitive data, and an override handler
	_, err := interp.EvalString(`
tool get_weather(city: string): string {
	# This returns sensitive data that should be redacted
	return "{\"weather\": \"sunny\", \"secret\": \"api_key_123\"}"
}

tool onToolEnd(ctx) {
	# Redact sensitive information from tool output
	if (ctx.toolCall.output != null && ctx.toolCall.output.includes("secret")) {
		return { result: "weather: [REDACTED]" }
	}
	# No return = keep original output
}

gsh.on("agent.tool.end", onToolEnd)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Get the tool from the environment
	weatherTool, ok := interp.env.Get("get_weather")
	if !ok {
		t.Fatal("get_weather tool not found in environment")
	}

	// Create agent with the tool
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
			"tools":        &ArrayValue{Elements: []Value{weatherTool}},
		},
	}

	// Create conversation that triggers the weather tool
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
	}

	result, err := interp.ExecuteAgent(context.Background(), conv, agent, false)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// Verify the conversation contains the redacted result
	convResult, ok := result.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result)
	}

	// Find the tool message in the conversation
	found := false
	for _, msg := range convResult.Messages {
		if msg.Role == "tool" {
			found = true
			expected := "weather: [REDACTED]"
			if msg.Content != expected {
				t.Errorf("Expected tool result %q, got %q", expected, msg.Content)
			}
		}
	}
	if !found {
		t.Error("Expected to find a tool message in the conversation")
	}
}

func TestToolStartOverrideWithError(t *testing.T) {
	// Create a mock provider that will request a tool call
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)
	interp.providerRegistry.Register(mockProvider)

	// Register an event handler that returns an error override
	_, err := interp.EvalString(`
tool onToolStart(ctx) {
	return { result: "Permission denied", error: "Tool execution not allowed" }
}

gsh.on("agent.tool.start", onToolStart)
`, nil)
	if err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Create a weather tool (will be triggered by SmartMockProvider)
	weatherTool := &ToolValue{
		Name: "get_weather",
	}

	// Create agent with the tool
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
			"tools":        &ArrayValue{Elements: []Value{weatherTool}},
		},
	}

	// Create conversation that triggers the weather tool
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather in San Francisco?"},
		},
	}

	// Execute agent with callbacks
	callbacks := &AgentCallbacks{
		ToolExecutor: func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
			return `{"weather": "sunny"}`, nil
		},
	}

	result, err := interp.ExecuteAgentWithCallbacks(context.Background(), conv, agent, false, callbacks)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// Verify the conversation contains the error message
	convResult, ok := result.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result)
	}

	// Find the tool message in the conversation - should contain error wrapper
	found := false
	for _, msg := range convResult.Messages {
		if msg.Role == "tool" {
			found = true
			// When there's an error, the result is wrapped with "Error executing tool:"
			if !strings.Contains(msg.Content, "Error executing tool") {
				t.Errorf("Expected error message in tool result, got %q", msg.Content)
			}
		}
	}
	if !found {
		t.Error("Expected to find a tool message in the conversation")
	}
}
