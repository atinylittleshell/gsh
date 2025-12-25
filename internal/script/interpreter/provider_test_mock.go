package interpreter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MockProvider is a test provider that returns predictable responses
type MockProvider struct {
	name string
	// ResponseMap maps input prompts to responses
	ResponseMap map[string]string
	// DefaultResponse is used if no mapping found
	DefaultResponse string
	// ToolCallScenarios defines when to make tool calls
	ToolCallScenarios map[string][]ChatToolCall
	// CallHistory tracks all calls made
	CallHistory []ChatRequest
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		name:              "mock",
		ResponseMap:       make(map[string]string),
		DefaultResponse:   "Mock response",
		ToolCallScenarios: make(map[string][]ChatToolCall),
		CallHistory:       []ChatRequest{},
	}
}

// Name returns the provider name
func (m *MockProvider) Name() string {
	return m.name
}

// ChatCompletion simulates an LLM response
func (m *MockProvider) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
	// Track the call
	m.CallHistory = append(m.CallHistory, request)

	// Get the last user message
	var lastUserMsg string
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if request.Messages[i].Role == "user" {
			lastUserMsg = request.Messages[i].Content
			break
		}
	}

	// Check for tool call scenarios first
	if toolCalls, ok := m.ToolCallScenarios[lastUserMsg]; ok {
		return &ChatResponse{
			Content:   "I'll use a tool to help with that.",
			ToolCalls: toolCalls,
		}, nil
	}

	// Check for mapped response
	if response, ok := m.ResponseMap[lastUserMsg]; ok {
		return &ChatResponse{
			Content:   response,
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	// Return default response
	return &ChatResponse{
		Content:   m.DefaultResponse,
		ToolCalls: []ChatToolCall{},
	}, nil
}

// SetResponse sets a specific response for a given input
func (m *MockProvider) SetResponse(input, response string) {
	m.ResponseMap[input] = response
}

// SetToolCallScenario sets up a tool call scenario
func (m *MockProvider) SetToolCallScenario(input string, toolCalls []ChatToolCall) {
	m.ToolCallScenarios[input] = toolCalls
}

// Reset clears the call history
func (m *MockProvider) Reset() {
	m.CallHistory = []ChatRequest{}
}

// GetCallCount returns the number of calls made
func (m *MockProvider) GetCallCount() int {
	return len(m.CallHistory)
}

// GetLastRequest returns the last request made
func (m *MockProvider) GetLastRequest() *ChatRequest {
	if len(m.CallHistory) == 0 {
		return nil
	}
	return &m.CallHistory[len(m.CallHistory)-1]
}

// StreamingChatCompletion simulates streaming by calling the callback with the full response
func (m *MockProvider) StreamingChatCompletion(request ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	// Get the non-streaming response
	response, err := m.ChatCompletion(request)
	if err != nil {
		return nil, err
	}

	// Simulate streaming by calling callback with full content at once
	if callback != nil && response.Content != "" {
		callback(response.Content)
	}

	return response, nil
}

// SmartMockProvider is a more intelligent mock that can handle basic math and tool calls
type SmartMockProvider struct {
	name        string
	CallHistory []ChatRequest
}

// NewSmartMockProvider creates a new smart mock provider
func NewSmartMockProvider() *SmartMockProvider {
	return &SmartMockProvider{
		name:        "smart-mock",
		CallHistory: []ChatRequest{},
	}
}

// Name returns the provider name
func (s *SmartMockProvider) Name() string {
	return s.name
}

// ChatCompletion simulates an intelligent LLM response
func (s *SmartMockProvider) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
	// Track the call
	s.CallHistory = append(s.CallHistory, request)

	// Get the last user message
	var lastUserMsg string
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if request.Messages[i].Role == "user" {
			lastUserMsg = request.Messages[i].Content
			break
		}
	}

	// Check if there are tool results in the conversation
	hasToolResults := false
	for _, msg := range request.Messages {
		if msg.Role == "tool" {
			hasToolResults = true
			break
		}
	}

	// If we have tool results, return a response that incorporates them
	if hasToolResults {
		var toolResults []string
		for _, msg := range request.Messages {
			if msg.Role == "tool" {
				toolResults = append(toolResults, msg.Content)
			}
		}
		return &ChatResponse{
			Content:   fmt.Sprintf("Based on the tool results: %s", strings.Join(toolResults, ", ")),
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	// Check if tools are available and message suggests using them
	if len(request.Tools) > 0 {
		lowerMsg := strings.ToLower(lastUserMsg)

		// Weather tool scenario
		if strings.Contains(lowerMsg, "weather") {
			for _, tool := range request.Tools {
				if strings.Contains(tool.Name, "weather") {
					// Extract city name (simple heuristic)
					var city string
					if strings.Contains(lowerMsg, "san francisco") {
						city = "San Francisco"
					} else if strings.Contains(lowerMsg, "new york") {
						city = "New York"
					} else {
						city = "Unknown City"
					}

					// Create tool call arguments
					args := map[string]interface{}{
						"city": city,
					}

					return &ChatResponse{
						Content: "I'll check the weather for you.",
						ToolCalls: []ChatToolCall{
							{
								Name:      tool.Name,
								Arguments: args,
							},
						},
					}, nil
				}
			}
		}

		// Generic tool call scenario - if message asks to "calculate" or "compute"
		if strings.Contains(lowerMsg, "calculate") || strings.Contains(lowerMsg, "compute") {
			if len(request.Tools) > 0 {
				// Use first available tool
				tool := request.Tools[0]
				args := map[string]interface{}{
					"value": lastUserMsg,
				}
				return &ChatResponse{
					Content: "I'll use a tool to help.",
					ToolCalls: []ChatToolCall{
						{
							Name:      tool.Name,
							Arguments: args,
						},
					},
				}, nil
			}
		}
	}

	// Math questions - check specific patterns first
	lowerMsg := strings.ToLower(lastUserMsg)
	if strings.Contains(lowerMsg, "what is") || strings.Contains(lowerMsg, "?") {
		// Try to extract simple math
		if strings.Contains(lastUserMsg, "2+2") || strings.Contains(lastUserMsg, "2 + 2") {
			return &ChatResponse{
				Content:   "The answer is 4",
				ToolCalls: []ChatToolCall{},
			}, nil
		}
		if strings.Contains(lastUserMsg, "5+3") || strings.Contains(lastUserMsg, "5 + 3") {
			return &ChatResponse{
				Content:   "The answer is 8",
				ToolCalls: []ChatToolCall{},
			}, nil
		}
	}

	// Simple math handling
	if strings.Contains(lastUserMsg, "+") {
		parts := strings.Split(lastUserMsg, "+")
		if len(parts) == 2 {
			// Try to extract numbers
			var a, b int
			_, _ = fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &a)
			_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &b)
			result := a + b
			return &ChatResponse{
				Content:   fmt.Sprintf("The answer is %d", result),
				ToolCalls: []ChatToolCall{},
			}, nil
		}
	}

	if strings.Contains(lastUserMsg, "*") || strings.Contains(lastUserMsg, "multiply") {
		// Simple multiplication handling
		return &ChatResponse{
			Content:   "The result is 16", // For common test cases like 8*2
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	// Default responses based on context
	if strings.Contains(strings.ToLower(lastUserMsg), "hello") ||
		strings.Contains(strings.ToLower(lastUserMsg), "hi") {
		return &ChatResponse{
			Content:   "Hello! How can I help you?",
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	if strings.Contains(strings.ToLower(lastUserMsg), "how are you") {
		return &ChatResponse{
			Content:   "I'm doing well, thank you for asking!",
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	if strings.Contains(strings.ToLower(lastUserMsg), "summary") ||
		strings.Contains(strings.ToLower(lastUserMsg), "summarize") {
		return &ChatResponse{
			Content:   "Here's a brief summary based on the context.",
			ToolCalls: []ChatToolCall{},
		}, nil
	}

	// Generic fallback
	return &ChatResponse{
		Content:   "I understand. Let me help you with that.",
		ToolCalls: []ChatToolCall{},
	}, nil
}

// Reset clears the call history
func (s *SmartMockProvider) Reset() {
	s.CallHistory = []ChatRequest{}
}

// GetCallCount returns the number of calls made
func (s *SmartMockProvider) GetCallCount() int {
	return len(s.CallHistory)
}

// GetLastRequest returns the last request made
func (s *SmartMockProvider) GetLastRequest() *ChatRequest {
	if len(s.CallHistory) == 0 {
		return nil
	}
	return &s.CallHistory[len(s.CallHistory)-1]
}

// StreamingChatCompletion simulates streaming by calling the callback with the full response
func (s *SmartMockProvider) StreamingChatCompletion(request ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	// Get the non-streaming response
	response, err := s.ChatCompletion(request)
	if err != nil {
		return nil, err
	}

	// Simulate streaming by calling callback with full content at once
	if callback != nil && response.Content != "" {
		callback(response.Content)
	}

	return response, nil
}

// MarshalJSON is needed for tool call arguments
func (tc ChatToolCall) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name:      tc.Name,
		Arguments: tc.Arguments,
	})
}
