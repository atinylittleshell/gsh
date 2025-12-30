package interpreter

import (
	"context"
	"testing"
)

// TestAgentEventsEmitted tests that agent lifecycle events are properly emitted
// during agent execution through the event manager.
func TestAgentEventsEmitted(t *testing.T) {
	// Create a mock provider that returns a simple response
	// SmartMockProvider auto-responds based on message content
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)

	// Track emitted events
	emittedEvents := []string{}
	eventContexts := make(map[string]Value)

	// Register test event callbacks to capture events
	eventNames := []string{
		EventAgentStart,
		EventAgentIterationStart,
		EventAgentIterationEnd,
		EventAgentEnd,
		EventAgentChunk,
		EventAgentToolStart,
		EventAgentToolEnd,
	}
	for _, eventName := range eventNames {
		// Capture each event name and context
		capturedEventName := eventName
		interp.RegisterTestEventCallback(eventName, func(ctx Value) {
			emittedEvents = append(emittedEvents, capturedEventName)
			eventContexts[capturedEventName] = ctx
		})
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

	// Execute agent
	_, err := interp.ExecuteAgent(context.Background(), conv, agent, false)
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

	// Verify agent.start context has message
	if startCtx, ok := eventContexts[EventAgentStart].(*ObjectValue); ok {
		if msgProp, ok := startCtx.Properties["message"]; ok {
			if msgStr, ok := msgProp.Value.(*StringValue); ok {
				if msgStr.Value != "Hello" {
					t.Errorf("agent.start message: expected 'Hello', got '%s'", msgStr.Value)
				}
			} else {
				t.Error("agent.start message is not a StringValue")
			}
		} else {
			t.Error("agent.start context missing 'message' property")
		}
	} else {
		t.Error("agent.start context is not an ObjectValue")
	}

	// Verify agent.iteration.start context has iteration number
	if iterStartCtx, ok := eventContexts[EventAgentIterationStart].(*ObjectValue); ok {
		if iterProp, ok := iterStartCtx.Properties["iteration"]; ok {
			if iterNum, ok := iterProp.Value.(*NumberValue); ok {
				if iterNum.Value != 1 {
					t.Errorf("agent.iteration.start iteration: expected 1, got %v", iterNum.Value)
				}
			} else {
				t.Error("agent.iteration.start iteration is not a NumberValue")
			}
		} else {
			t.Error("agent.iteration.start context missing 'iteration' property")
		}
	} else {
		t.Error("agent.iteration.start context is not an ObjectValue")
	}

	// Verify agent.end context has result with stopReason
	if endCtx, ok := eventContexts[EventAgentEnd].(*ObjectValue); ok {
		if resultProp, ok := endCtx.Properties["result"]; ok {
			if resultObj, ok := resultProp.Value.(*ObjectValue); ok {
				if stopReasonProp, ok := resultObj.Properties["stopReason"]; ok {
					if stopReasonStr, ok := stopReasonProp.Value.(*StringValue); ok {
						if stopReasonStr.Value != "end_turn" {
							t.Errorf("agent.end stopReason: expected 'end_turn', got '%s'", stopReasonStr.Value)
						}
					}
				}
			}
		}
	}
}

// TestAgentToolEventsEmitted tests that tool events are emitted during tool execution.
func TestAgentToolEventsEmitted(t *testing.T) {
	// Create a mock provider - SmartMockProvider will use weather tool when message contains "weather"
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)

	// Track emitted events
	emittedEvents := []string{}
	eventContexts := make(map[string]Value)

	// Register test event callbacks to capture events
	eventNames := []string{
		EventAgentStart,
		EventAgentIterationStart,
		EventAgentIterationEnd,
		EventAgentEnd,
		EventAgentChunk,
		EventAgentToolStart,
		EventAgentToolEnd,
	}
	for _, eventName := range eventNames {
		// Capture each event name and context
		capturedEventName := eventName
		interp.RegisterTestEventCallback(eventName, func(ctx Value) {
			emittedEvents = append(emittedEvents, capturedEventName)
			eventContexts[capturedEventName] = ctx
		})
	}

	// Create a weather tool that SmartMockProvider knows how to use
	weatherTool := &ToolValue{
		Name:       "get_weather",
		Parameters: []string{"city"},
		ParamTypes: map[string]string{"city": "string"},
		Body:       nil, // We'll use a custom executor
	}

	// Create agent with the tool
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
			"tools": &ArrayValue{Elements: []Value{weatherTool}},
		},
	}

	// Create conversation - SmartMockProvider triggers tool call for "weather" messages
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "What is the weather in San Francisco?"},
		},
	}

	// Create callbacks with just the ToolExecutor
	callbacks := &AgentCallbacks{
		ToolExecutor: func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
			return `{\"weather\": \"sunny\", \"temp\": 72}`, nil
		},
	}

	// Execute agent with callbacks for tool execution
	_, err := interp.executeAgentInternal(context.Background(), conv, agent, false, callbacks)
	if err != nil {
		t.Fatalf("executeAgentInternal failed: %v", err)
	}

	// Check that tool events were emitted
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

	// Verify tool start context
	if toolStartCtx, ok := eventContexts[EventAgentToolStart].(*ObjectValue); ok {
		if toolCallProp, ok := toolStartCtx.Properties["toolCall"]; ok {
			if toolCallObj, ok := toolCallProp.Value.(*ObjectValue); ok {
				if nameProp, ok := toolCallObj.Properties["name"]; ok {
					if nameStr, ok := nameProp.Value.(*StringValue); ok {
						if nameStr.Value != "get_weather" {
							t.Errorf("tool.start name: expected 'get_weather', got '%s'", nameStr.Value)
						}
					}
				}
			}
		}
	}

	// Verify tool end context has durationMs
	if toolEndCtx, ok := eventContexts[EventAgentToolEnd].(*ObjectValue); ok {
		if toolCallProp, ok := toolEndCtx.Properties["toolCall"]; ok {
			if toolCallObj, ok := toolCallProp.Value.(*ObjectValue); ok {
				if _, ok := toolCallObj.Properties["durationMs"]; !ok {
					t.Error("tool.end context missing 'durationMs' property")
				}
			}
		}
	}
}

// TestAgentChunkEventsEmitted tests that chunk events are emitted during streaming.
func TestAgentChunkEventsEmitted(t *testing.T) {
	// Create a mock provider - SmartMockProvider implements streaming by calling OnContent
	// with the full response content at once
	mockProvider := NewSmartMockProvider()

	// Create interpreter
	interp := New(nil)

	// Track emitted events and chunk contents
	chunkContents := []string{}
	emittedEvents := []string{}
	eventContexts := make(map[string]Value)

	// Register test event callback for chunk events to capture chunk contents
	interp.RegisterTestEventCallback(EventAgentChunk, func(ctx Value) {
		emittedEvents = append(emittedEvents, EventAgentChunk)
		if chunkCtx, ok := ctx.(*ObjectValue); ok {
			if contentProp, ok := chunkCtx.Properties["content"]; ok {
				if contentStr, ok := contentProp.Value.(*StringValue); ok {
					chunkContents = append(chunkContents, contentStr.Value)
				}
			}
			eventContexts[EventAgentChunk] = ctx
		}
	})

	// Create agent
	agent := &AgentValue{
		Name: "testAgent",
		Config: map[string]Value{
			"model": &ModelValue{
				Provider: mockProvider,
			},
		},
	}

	// Create conversation
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Execute agent with streaming enabled
	_, err := interp.ExecuteAgent(context.Background(), conv, agent, true)
	if err != nil {
		t.Fatalf("ExecuteAgent failed: %v", err)
	}

	// SmartMockProvider calls OnContent once with the full response
	// Verify at least one chunk was emitted
	if len(chunkContents) == 0 {
		t.Error("Expected at least one chunk event to be emitted")
	}

	// The chunk should contain some response content (SmartMockProvider returns "Hello! How can I help you?")
	if len(chunkContents) > 0 && chunkContents[0] == "" {
		t.Error("Chunk content should not be empty")
	}
}

// TestEventConstants verifies the event constant values match the spec
func TestEventConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{EventAgentStart, "agent.start"},
		{EventAgentEnd, "agent.end"},
		{EventAgentIterationStart, "agent.iteration.start"},
		{EventAgentIterationEnd, "agent.iteration.end"},
		{EventAgentChunk, "agent.chunk"},
		{EventAgentToolStart, "agent.tool.start"},
		{EventAgentToolEnd, "agent.tool.end"},
		{EventAgentExecStart, "agent.exec.start"},
		{EventAgentExecEnd, "agent.exec.end"},
	}

	for _, tc := range tests {
		if tc.constant != tc.expected {
			t.Errorf("Event constant mismatch: got %s, expected %s", tc.constant, tc.expected)
		}
	}
}
