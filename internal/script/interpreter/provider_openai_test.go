package interpreter

import (
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
			resp, err := provider.ChatCompletion(tt.request)

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

	_, err := provider.ChatCompletion(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
