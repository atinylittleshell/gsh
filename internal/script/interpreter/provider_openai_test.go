package interpreter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIProviderChatCompletion(t *testing.T) {
	tests := []struct {
		name           string
		request        ChatRequest
		mockResponse   string
		mockStatusCode int
		expectedError  string
		checkResponse  func(t *testing.T, resp *ChatResponse)
	}{
		{
			name: "Basic chat completion",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider": &StringValue{Value: "openai"},
						"apiKey":   &StringValue{Value: "test-key"},
						"model":    &StringValue{Value: "gpt-4"},
					},
				},
				Messages: []ChatMessage{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 8,
					"total_tokens": 18
				}
			}`,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ChatResponse) {
				if resp.Content != "Hello! How can I help you?" {
					t.Errorf("expected content 'Hello! How can I help you?', got %q", resp.Content)
				}
				if resp.FinishReason != "stop" {
					t.Errorf("expected finish reason 'stop', got %q", resp.FinishReason)
				}
				if resp.Usage == nil {
					t.Fatal("expected usage to be set")
				}
				if resp.Usage.PromptTokens != 10 {
					t.Errorf("expected prompt tokens 10, got %d", resp.Usage.PromptTokens)
				}
				if resp.Usage.CompletionTokens != 8 {
					t.Errorf("expected completion tokens 8, got %d", resp.Usage.CompletionTokens)
				}
				if resp.Usage.TotalTokens != 18 {
					t.Errorf("expected total tokens 18, got %d", resp.Usage.TotalTokens)
				}
			},
		},
		{
			name: "Chat completion with temperature",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider":    &StringValue{Value: "openai"},
						"apiKey":      &StringValue{Value: "test-key"},
						"model":       &StringValue{Value: "gpt-4"},
						"temperature": &NumberValue{Value: 0.7},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-456",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Response"
					},
					"finish_reason": "stop"
				}]
			}`,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ChatResponse) {
				if resp.Content != "Response" {
					t.Errorf("expected content 'Response', got %q", resp.Content)
				}
			},
		},
		{
			name: "Chat completion with max tokens",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider":  &StringValue{Value: "openai"},
						"apiKey":    &StringValue{Value: "test-key"},
						"model":     &StringValue{Value: "gpt-4"},
						"maxTokens": &NumberValue{Value: 100},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-789",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Short response"
					},
					"finish_reason": "length"
				}]
			}`,
			mockStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *ChatResponse) {
				if resp.FinishReason != "length" {
					t.Errorf("expected finish reason 'length', got %q", resp.FinishReason)
				}
			},
		},
		{
			name: "Missing API key",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider": &StringValue{Value: "openai"},
						"model":    &StringValue{Value: "gpt-4"},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "requires 'apiKey'",
		},
		{
			name: "API error - 401",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider": &StringValue{Value: "openai"},
						"apiKey":   &StringValue{Value: "invalid-key"},
						"model":    &StringValue{Value: "gpt-4"},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse:   `{"error": "Invalid API key"}`,
			mockStatusCode: http.StatusUnauthorized,
			expectedError:  "OpenAI API returned status 401",
		},
		{
			name: "API error - 429",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider": &StringValue{Value: "openai"},
						"apiKey":   &StringValue{Value: "test-key"},
						"model":    &StringValue{Value: "gpt-4"},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse:   `{"error": "Rate limit exceeded"}`,
			mockStatusCode: http.StatusTooManyRequests,
			expectedError:  "OpenAI API returned status 429",
		},
		{
			name: "Empty choices",
			request: ChatRequest{
				Model: &ModelValue{
					Name: "gpt4",
					Config: map[string]Value{
						"provider": &StringValue{Value: "openai"},
						"apiKey":   &StringValue{Value: "test-key"},
						"model":    &StringValue{Value: "gpt-4"},
					},
				},
				Messages: []ChatMessage{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-empty",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": []
			}`,
			mockStatusCode: http.StatusOK,
			expectedError:  "no choices in response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				authHeader := r.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Errorf("expected Authorization header with Bearer token")
				}

				// Parse request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				// Verify required fields
				if _, ok := reqBody["model"]; !ok {
					t.Errorf("request missing 'model' field")
				}
				if _, ok := reqBody["messages"]; !ok {
					t.Errorf("request missing 'messages' field")
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Override base URL in model config
			tt.request.Model.Config["baseURL"] = &StringValue{Value: server.URL}

			// Create provider and make request
			provider := NewOpenAIProvider()
			resp, err := provider.ChatCompletion(context.Background(), tt.request)

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, but got no error", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			// Check success
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestOpenAIProviderToolCallMessageFields(t *testing.T) {
	// Test that tool_call_id is included in tool result messages
	// and tool_calls are included in assistant messages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		// Verify messages
		messages, ok := reqBody["messages"].([]interface{})
		if !ok {
			t.Fatal("messages field not found or not an array")
		}

		// Check assistant message with tool_calls
		assistantMsg, ok := messages[1].(map[string]interface{})
		if !ok {
			t.Fatal("assistant message not found")
		}
		if assistantMsg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", assistantMsg["role"])
		}
		toolCalls, ok := assistantMsg["tool_calls"].([]interface{})
		if !ok {
			t.Fatal("tool_calls not found in assistant message")
		}
		if len(toolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(toolCalls))
		}
		toolCall, ok := toolCalls[0].(map[string]interface{})
		if !ok {
			t.Fatal("tool call not a map")
		}
		if toolCall["id"] != "call_abc123" {
			t.Errorf("expected tool call id 'call_abc123', got %v", toolCall["id"])
		}
		if toolCall["type"] != "function" {
			t.Errorf("expected tool call type 'function', got %v", toolCall["type"])
		}
		function, ok := toolCall["function"].(map[string]interface{})
		if !ok {
			t.Fatal("function not found in tool call")
		}
		if function["name"] != "get_weather" {
			t.Errorf("expected function name 'get_weather', got %v", function["name"])
		}

		// Check tool result message with tool_call_id
		toolMsg, ok := messages[2].(map[string]interface{})
		if !ok {
			t.Fatal("tool message not found")
		}
		if toolMsg["role"] != "tool" {
			t.Errorf("expected role 'tool', got %v", toolMsg["role"])
		}
		if toolMsg["tool_call_id"] != "call_abc123" {
			t.Errorf("expected tool_call_id 'call_abc123', got %v", toolMsg["tool_call_id"])
		}
		// In this provider implementation, content is always sent as multipart content parts.
		contentParts, ok := toolMsg["content"].([]interface{})
		if !ok {
			t.Fatalf("expected tool message content to be an array, got %T", toolMsg["content"])
		}
		if len(contentParts) != 1 {
			t.Fatalf("expected 1 tool content part, got %d", len(contentParts))
		}
		part, ok := contentParts[0].(map[string]interface{})
		if !ok {
			t.Fatalf("tool content part not a map")
		}
		if part["type"] != "text" {
			t.Errorf("expected tool content part type 'text', got %v", part["type"])
		}
		if part["text"] != `{"temperature": 72}` {
			t.Errorf("expected tool content text '{\"temperature\": 72}', got %v", part["text"])
		}

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-tooltest",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "The temperature is 72 degrees."
				},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()
	req := ChatRequest{
		Model: &ModelValue{
			Name: "gpt4",
			Config: map[string]Value{
				"provider": &StringValue{Value: "openai"},
				"apiKey":   &StringValue{Value: "test-key"},
				"model":    &StringValue{Value: "gpt-4"},
				"baseURL":  &StringValue{Value: server.URL},
			},
		},
		Messages: []ChatMessage{
			{Role: "user", Content: "What's the weather?"},
			{
				Role:    "assistant",
				Content: "",
				ToolCalls: []ChatToolCall{
					{
						ID:        "call_abc123",
						Name:      "get_weather",
						Arguments: map[string]interface{}{"location": "San Francisco"},
					},
				},
			},
			{
				Role:       "tool",
				Content:    `{"temperature": 72}`,
				ToolCallID: "call_abc123",
				Name:       "get_weather",
			},
		},
	}

	resp, err := provider.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "The temperature is 72 degrees." {
		t.Errorf("expected content 'The temperature is 72 degrees.', got %q", resp.Content)
	}
}

func TestOpenAIProviderAssistantToolCallEmptyContentHasTextField(t *testing.T) {
	// Regression test for Ollama compatibility:
	// when the assistant message has tool_calls and empty content, we must still
	// send content parts like {"type":"text","text":""}, not {"type":"text"}.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		messages, ok := reqBody["messages"].([]interface{})
		if !ok {
			t.Fatalf("messages field not found or not an array")
		}
		if len(messages) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(messages))
		}

		assistantMsg, ok := messages[1].(map[string]interface{})
		if !ok {
			t.Fatalf("assistant message not found or not an object")
		}
		if assistantMsg["role"] != "assistant" {
			t.Fatalf("expected role 'assistant', got %v", assistantMsg["role"])
		}

		contentParts, ok := assistantMsg["content"].([]interface{})
		if !ok {
			t.Fatalf("expected assistant content to be an array, got %T", assistantMsg["content"])
		}
		if len(contentParts) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(contentParts))
		}

		part, ok := contentParts[0].(map[string]interface{})
		if !ok {
			t.Fatalf("content part not an object")
		}
		if part["type"] != "text" {
			t.Fatalf("expected content part type 'text', got %v", part["type"])
		}
		// Critical assertion: text field must exist even if empty.
		if _, ok := part["text"]; !ok {
			t.Fatalf("expected content part to include 'text' field, got %v", part)
		}
		if part["text"] != "" {
			t.Fatalf("expected content part text to be empty string, got %v", part["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-tooltest-empty-content",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "ok"},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()
	req := ChatRequest{
		Model: &ModelValue{
			Name: "gpt4",
			Config: map[string]Value{
				"provider": &StringValue{Value: "openai"},
				"apiKey":   &StringValue{Value: "test-key"},
				"model":    &StringValue{Value: "gpt-4"},
				"baseURL":  &StringValue{Value: server.URL},
			},
		},
		Messages: []ChatMessage{
			{Role: "user", Content: "What is 6 times 7?"},
			{
				Role:    "assistant",
				Content: "", // the problematic case
				ToolCalls: []ChatToolCall{
					{
						ID:        "call_abc123",
						Name:      "multiply",
						Arguments: map[string]interface{}{"a": 6, "b": 7},
					},
				},
			},
		},
	}

	_, err := provider.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCachedTokens(t *testing.T) {
	// Test that cached_tokens is correctly parsed from the response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-cached",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Cached response"
				},
				"finish_reason": "stop"
			}],
			"usage": {
				"prompt_tokens": 2006,
				"completion_tokens": 100,
				"total_tokens": 2106,
				"prompt_tokens_details": {
					"cached_tokens": 1920
				}
			}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()
	req := ChatRequest{
		Model: &ModelValue{
			Name: "gpt4",
			Config: map[string]Value{
				"provider": &StringValue{Value: "openai"},
				"apiKey":   &StringValue{Value: "test-key"},
				"model":    &StringValue{Value: "gpt-4"},
				"baseURL":  &StringValue{Value: server.URL},
			},
		},
		Messages: []ChatMessage{
			{Role: "user", Content: "Test with caching"},
		},
	}

	resp, err := provider.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage to be set")
	}
	if resp.Usage.PromptTokens != 2006 {
		t.Errorf("expected prompt tokens 2006, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CachedTokens != 1920 {
		t.Errorf("expected cached tokens 1920, got %d", resp.Usage.CachedTokens)
	}
}

func TestOpenAIProviderContentParts(t *testing.T) {
	// Test that ContentParts with CacheControl is correctly serialized
	var capturedReqBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request body
		if err := json.NewDecoder(r.Body).Decode(&capturedReqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-multipart",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "claude-3-5-sonnet",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Response to cached content"
				},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()
	req := ChatRequest{
		Model: &ModelValue{
			Name: "claude",
			Config: map[string]Value{
				"provider": &StringValue{Value: "openai"},
				"apiKey":   &StringValue{Value: "test-key"},
				"model":    &StringValue{Value: "anthropic/claude-3-5-sonnet"},
				"baseURL":  &StringValue{Value: server.URL},
			},
		},
		Messages: []ChatMessage{
			{
				Role: "system",
				ContentParts: []ContentPart{
					{
						Type: "text",
						Text: "You are a helpful assistant.",
					},
					{
						Type: "text",
						Text: "Here is a very long context that should be cached...",
					},
				},
			},
			{Role: "user", Content: "What is in the context?"},
		},
	}

	resp, err := provider.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Response to cached content" {
		t.Errorf("expected content 'Response to cached content', got %q", resp.Content)
	}

	// Verify the request body structure
	messages, ok := capturedReqBody["messages"].([]interface{})
	if !ok {
		t.Fatal("messages field not found or not an array")
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	// Check system message has multipart content
	systemMsg, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatal("system message not found")
	}
	content, ok := systemMsg["content"].([]interface{})
	if !ok {
		t.Fatal("system message content should be an array for ContentParts")
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}

	// Check first part
	part1, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("first content part not a map")
	}
	if part1["type"] != "text" {
		t.Errorf("expected type 'text', got %v", part1["type"])
	}
	if part1["text"] != "You are a helpful assistant." {
		t.Errorf("unexpected text in first part: %v", part1["text"])
	}

	// Check second part
	part2, ok := content[1].(map[string]interface{})
	if !ok {
		t.Fatal("second content part not a map")
	}
	if part2["type"] != "text" {
		t.Errorf("expected type 'text', got %v", part2["type"])
	}
	// This provider does not currently propagate CacheControl from incoming ContentParts.
	if _, hasCache := part2["cache_control"]; hasCache {
		t.Error("second part should not have cache_control")
	}

	// Check user message (last message) has cache_control applied
	userMsg, ok := messages[1].(map[string]interface{})
	if !ok {
		t.Fatal("user message not found")
	}
	userParts, ok := userMsg["content"].([]interface{})
	if !ok {
		t.Fatal("user message content should be an array")
	}
	if len(userParts) != 1 {
		t.Fatalf("expected 1 user content part, got %d", len(userParts))
	}
	userPart, ok := userParts[0].(map[string]interface{})
	if !ok {
		t.Fatal("user content part not a map")
	}
	if userPart["text"] != "What is in the context?" {
		t.Errorf("expected user content 'What is in the context?', got %q", userPart["text"])
	}
	userCache, ok := userPart["cache_control"].(map[string]interface{})
	if !ok {
		t.Fatal("cache_control not found in user part")
	}
	if userCache["type"] != "ephemeral" {
		t.Errorf("expected cache_control type 'ephemeral', got %v", userCache["type"])
	}
	if userCache["ttl"] != "5m" {
		t.Errorf("expected cache_control ttl '5m', got %v", userCache["ttl"])
	}
}

func TestOpenAIProviderCustomBaseURL(t *testing.T) {
	// This test verifies the custom base URL is used correctly
	// We'll use a mock server to verify
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "Response"},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider()
	req := ChatRequest{
		Model: &ModelValue{
			Name: "gpt4",
			Config: map[string]Value{
				"provider": &StringValue{Value: "openai"},
				"apiKey":   &StringValue{Value: "test-key"},
				"model":    &StringValue{Value: "gpt-4"},
				"baseURL":  &StringValue{Value: server.URL},
			},
		},
		Messages: []ChatMessage{
			{Role: "user", Content: "Test"},
		},
	}

	_, err := provider.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
