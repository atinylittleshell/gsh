package interpreter

import (
	"context"
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
`)
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
`)
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
`)
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
