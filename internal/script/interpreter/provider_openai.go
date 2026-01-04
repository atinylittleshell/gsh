package interpreter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements the ModelProvider interface for OpenAI
type OpenAIProvider struct {
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		httpClient: &http.Client{},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// extractStringContent extracts string content from an interface{} that may be
// a string or an array of content parts (multipart format).
// Used when parsing API responses where content could be in either format.
func extractStringContent(content interface{}) string {
	if content == nil {
		return ""
	}
	// Most common case: content is a string
	if str, ok := content.(string); ok {
		return str
	}
	// Handle array of content parts (multipart format)
	if parts, ok := content.([]interface{}); ok {
		var result strings.Builder
		for _, part := range parts {
			if partMap, ok := part.(map[string]interface{}); ok {
				if text, ok := partMap["text"].(string); ok {
					result.WriteString(text)
				}
			}
		}
		return result.String()
	}
	return ""
}

// convertMessageContent converts a ChatMessage's content to the appropriate format.
// If ContentParts is set, it returns an array of openAIContentPart for multipart/cache control.
// Otherwise, it returns the plain string content.
func convertMessageContent(msg ChatMessage) []openAIContentPart {
	if len(msg.ContentParts) > 0 {
		parts := make([]openAIContentPart, len(msg.ContentParts))
		for i, part := range msg.ContentParts {
			parts[i] = openAIContentPart{
				Type: part.Type,
				Text: part.Text,
				// deliberately omit CacheControl because we set it later
			}
		}
		return parts
	} else {
		parts := make([]openAIContentPart, 1)
		parts[0] = openAIContentPart{
			Type: "text",
			Text: msg.Content,
		}
		return parts
	}
}

// ChatCompletion sends a chat completion request to OpenAI.
// The ctx parameter allows cancellation of the request (e.g., via Ctrl+C).
func (p *OpenAIProvider) ChatCompletion(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	if request.Model == nil {
		return nil, fmt.Errorf("OpenAI provider requires a model")
	}

	// Get API key from model config
	apiKeyVal, ok := request.Model.Config["apiKey"]
	if !ok {
		return nil, fmt.Errorf("OpenAI provider requires 'apiKey' in model config")
	}
	apiKeyStr, ok := apiKeyVal.(*StringValue)
	if !ok || apiKeyStr.Value == "" {
		return nil, fmt.Errorf("OpenAI provider requires 'apiKey' to be a non-empty string")
	}
	apiKey := apiKeyStr.Value

	// Get model ID from model config
	modelIDVal, ok := request.Model.Config["model"]
	if !ok {
		return nil, fmt.Errorf("OpenAI provider requires 'model' in model config")
	}
	modelIDStr, ok := modelIDVal.(*StringValue)
	if !ok || modelIDStr.Value == "" {
		return nil, fmt.Errorf("OpenAI provider requires 'model' to be a non-empty string")
	}
	modelID := modelIDStr.Value

	// Get base URL (default to OpenAI)
	baseURL := "https://api.openai.com/v1"
	if baseURLVal, ok := request.Model.Config["baseURL"]; ok {
		if baseURLStr, ok := baseURLVal.(*StringValue); ok && baseURLStr.Value != "" {
			baseURL = baseURLStr.Value
		}
	}

	// Append the chat completions endpoint
	apiURL := baseURL + "/chat/completions"

	// Build OpenAI-specific request
	openaiReq := openAIChatCompletionRequest{
		Model:    modelID,
		Usage:    &openAIUsageInclude{Include: true},
		Messages: make([]openAIMessage, len(request.Messages)),
	}

	// Convert messages
	for i, msg := range request.Messages {
		openaiReq.Messages[i] = openAIMessage{
			Role:    msg.Role,
			Content: convertMessageContent(msg),
		}
		if msg.Name != "" {
			openaiReq.Messages[i].Name = &msg.Name
		}
		// Include tool_call_id for tool result messages
		if msg.ToolCallID != "" {
			openaiReq.Messages[i].ToolCallID = &msg.ToolCallID
		}
		// Include tool_calls for assistant messages that requested tool calls
		if len(msg.ToolCalls) > 0 {
			openaiReq.Messages[i].ToolCalls = make([]openAIMessageToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				argsJSON, err := json.Marshal(tc.Arguments)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal tool call arguments: %w", err)
				}
				openaiReq.Messages[i].ToolCalls[j] = openAIMessageToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFunction{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}
		// Set cache control on the last content part if this is the last message
		contentParts := openaiReq.Messages[i].Content.([]openAIContentPart)
		if i == len(request.Messages)-1 && len(contentParts) > 0 {
			contentParts[len(contentParts)-1].CacheControl = &openAICacheControl{
				Type: "ephemeral",
				TTL:  "5m",
			}
		}
	}

	// Add optional parameters from model config
	if tempVal, ok := request.Model.Config["temperature"]; ok {
		if tempNum, ok := tempVal.(*NumberValue); ok {
			temp := tempNum.Value
			openaiReq.Temperature = &temp
		}
	}
	if maxTokensVal, ok := request.Model.Config["maxTokens"]; ok {
		if maxTokensNum, ok := maxTokensVal.(*NumberValue); ok {
			maxTokens := int(maxTokensNum.Value)
			openaiReq.MaxTokens = &maxTokens
		}
	}
	if topPVal, ok := request.Model.Config["topP"]; ok {
		if topPNum, ok := topPVal.(*NumberValue); ok {
			topP := topPNum.Value
			openaiReq.TopP = &topP
		}
	}

	// Convert tools if present
	if len(request.Tools) > 0 {
		openaiReq.Tools = make([]openAITool, len(request.Tools))
		for i, tool := range request.Tools {
			openaiReq.Tools[i] = openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	// Marshal request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context for cancellation support
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Apply custom headers from model config
	if headersVal, ok := request.Model.Config["headers"]; ok {
		if headersObj, ok := headersVal.(*ObjectValue); ok {
			for headerKey := range headersObj.Properties {
				headerVal := headersObj.GetPropertyValue(headerKey)
				if headerStr, ok := headerVal.(*StringValue); ok {
					httpReq.Header.Set(headerKey, headerStr.Value)
				}
			}
		}
	}

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var openaiResp openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to common response format
	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := openaiResp.Choices[0]
	response := &ChatResponse{
		Content:      extractStringContent(choice.Message.Content),
		FinishReason: choice.FinishReason,
	}

	// Add usage information if present
	if openaiResp.Usage != nil {
		response.Usage = &ChatUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		}
		// Include cached tokens if available (from OpenAI's prompt_tokens_details)
		if openaiResp.Usage.PromptTokensDetails != nil {
			response.Usage.CachedTokens = openaiResp.Usage.PromptTokensDetails.CachedTokens
		}
	}

	// Convert tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ChatToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			response.ToolCalls[i] = ChatToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
			}
			// Parse arguments JSON
			if tc.Function.Arguments != "" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
					response.ToolCalls[i].Arguments = args
				}
			}
		}
	}

	return response, nil
}

// StreamingChatCompletion sends a chat completion request with streaming response.
// The ctx parameter allows cancellation of the streaming request (e.g., via Ctrl+C).
// The callbacks provide hooks for content chunks and tool call detection.
func (p *OpenAIProvider) StreamingChatCompletion(ctx context.Context, request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error) {
	if request.Model == nil {
		return nil, fmt.Errorf("OpenAI provider requires a model")
	}

	// Get API key from model config
	apiKeyVal, ok := request.Model.Config["apiKey"]
	if !ok {
		return nil, fmt.Errorf("OpenAI provider requires 'apiKey' in model config")
	}
	apiKeyStr, ok := apiKeyVal.(*StringValue)
	if !ok || apiKeyStr.Value == "" {
		return nil, fmt.Errorf("OpenAI provider requires 'apiKey' to be a non-empty string")
	}
	apiKey := apiKeyStr.Value

	// Get model ID from model config
	modelIDVal, ok := request.Model.Config["model"]
	if !ok {
		return nil, fmt.Errorf("OpenAI provider requires 'model' in model config")
	}
	modelIDStr, ok := modelIDVal.(*StringValue)
	if !ok || modelIDStr.Value == "" {
		return nil, fmt.Errorf("OpenAI provider requires 'model' to be a non-empty string")
	}
	modelID := modelIDStr.Value

	// Get base URL (default to OpenAI)
	baseURL := "https://api.openai.com/v1"
	if baseURLVal, ok := request.Model.Config["baseURL"]; ok {
		if baseURLStr, ok := baseURLVal.(*StringValue); ok && baseURLStr.Value != "" {
			baseURL = baseURLStr.Value
		}
	}

	// Append the chat completions endpoint
	apiURL := baseURL + "/chat/completions"

	// Build OpenAI-specific request with streaming enabled
	openaiReq := openAIStreamingChatCompletionRequest{
		Model:    modelID,
		Messages: make([]openAIMessage, len(request.Messages)),
		Stream:   true,
		StreamOptions: &openAIStreamOptions{
			IncludeUsage: true,
		},
		Usage: &openAIUsageInclude{Include: true},
	}

	// Convert messages
	for i, msg := range request.Messages {
		openaiReq.Messages[i] = openAIMessage{
			Role:    msg.Role,
			Content: convertMessageContent(msg),
		}
		if msg.Name != "" {
			openaiReq.Messages[i].Name = &msg.Name
		}
		// Include tool_call_id for tool result messages
		if msg.ToolCallID != "" {
			openaiReq.Messages[i].ToolCallID = &msg.ToolCallID
		}
		// Include tool_calls for assistant messages that requested tool calls
		if len(msg.ToolCalls) > 0 {
			openaiReq.Messages[i].ToolCalls = make([]openAIMessageToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				argsJSON, err := json.Marshal(tc.Arguments)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal tool call arguments: %w", err)
				}
				openaiReq.Messages[i].ToolCalls[j] = openAIMessageToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFunction{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}
		// Set cache control on the last content part if this is the last message
		contentParts := openaiReq.Messages[i].Content.([]openAIContentPart)
		if i == len(request.Messages)-1 && len(contentParts) > 0 {
			contentParts[len(contentParts)-1].CacheControl = &openAICacheControl{
				Type: "ephemeral",
				TTL:  "5m",
			}
		}
	}

	// Add optional parameters from model config
	if tempVal, ok := request.Model.Config["temperature"]; ok {
		if tempNum, ok := tempVal.(*NumberValue); ok {
			temp := tempNum.Value
			openaiReq.Temperature = &temp
		}
	}
	if maxTokensVal, ok := request.Model.Config["maxTokens"]; ok {
		if maxTokensNum, ok := maxTokensVal.(*NumberValue); ok {
			maxTokens := int(maxTokensNum.Value)
			openaiReq.MaxTokens = &maxTokens
		}
	}
	if topPVal, ok := request.Model.Config["topP"]; ok {
		if topPNum, ok := topPVal.(*NumberValue); ok {
			topP := topPNum.Value
			openaiReq.TopP = &topP
		}
	}

	// Convert tools if present
	if len(request.Tools) > 0 {
		openaiReq.Tools = make([]openAITool, len(request.Tools))
		for i, tool := range request.Tools {
			openaiReq.Tools[i] = openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	// Marshal request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context for cancellation support
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Apply custom headers from model config
	if headersVal, ok := request.Model.Config["headers"]; ok {
		if headersObj, ok := headersVal.(*ObjectValue); ok {
			for headerKey := range headersObj.Properties {
				headerVal := headersObj.GetPropertyValue(headerKey)
				if headerStr, ok := headerVal.(*StringValue); ok {
					httpReq.Header.Set(headerKey, headerStr.Value)
				}
			}
		}
	}

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream
	var fullContent strings.Builder
	var finishReason string
	var toolCalls []ChatToolCall
	var usage *ChatUsage

	// Track which tool calls we've already notified about
	toolCallNotified := make(map[int]bool)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// Check for cancellation before processing each chunk
		if callbacks != nil && callbacks.ShouldCancel != nil && callbacks.ShouldCancel() {
			return nil, fmt.Errorf("streaming cancelled")
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// SSE format: "data: {json}" or "data: [DONE]"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse chunk
		var chunk openAIStreamingChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		// Capture usage if present (comes in the final chunk when stream_options.include_usage is true)
		if chunk.Usage != nil {
			usage = &ChatUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
			// Include cached tokens if available (from OpenAI's prompt_tokens_details)
			if chunk.Usage.PromptTokensDetails != nil {
				usage.CachedTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
		}

		// Process choices
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Accumulate content
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				// Call the callback with the delta
				if callbacks != nil && callbacks.OnContent != nil {
					callbacks.OnContent(choice.Delta.Content)
				}
			}

			// Capture finish reason
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			// Handle tool calls (accumulated across chunks)
			for _, tc := range choice.Delta.ToolCalls {
				// Find or create the tool call entry
				for len(toolCalls) <= tc.Index {
					toolCalls = append(toolCalls, ChatToolCall{})
				}
				if tc.ID != "" {
					toolCalls[tc.Index].ID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCalls[tc.Index].Name = tc.Function.Name
				}

				// Notify when we first see a tool call with both ID and Name
				if !toolCallNotified[tc.Index] &&
					toolCalls[tc.Index].ID != "" &&
					toolCalls[tc.Index].Name != "" {
					toolCallNotified[tc.Index] = true
					if callbacks != nil && callbacks.OnToolPending != nil {
						callbacks.OnToolPending(toolCalls[tc.Index].ID, toolCalls[tc.Index].Name)
					}
				}

				if tc.Function.Arguments != "" {
					// Accumulate arguments (they may be streamed in chunks)
					if toolCalls[tc.Index].Arguments == nil {
						toolCalls[tc.Index].Arguments = make(map[string]interface{})
					}
					// Store raw arguments for later parsing
					if existingArgs, ok := toolCalls[tc.Index].Arguments["__raw__"].(string); ok {
						toolCalls[tc.Index].Arguments["__raw__"] = existingArgs + tc.Function.Arguments
					} else {
						toolCalls[tc.Index].Arguments["__raw__"] = tc.Function.Arguments
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	// Parse accumulated tool call arguments
	for i := range toolCalls {
		if rawArgs, ok := toolCalls[i].Arguments["__raw__"].(string); ok {
			delete(toolCalls[i].Arguments, "__raw__")
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(rawArgs), &args); err == nil {
				toolCalls[i].Arguments = args
			}
		}
	}

	// Build final response
	response := &ChatResponse{
		Content:      fullContent.String(),
		FinishReason: finishReason,
		ToolCalls:    toolCalls,
		Usage:        usage,
	}

	return response, nil
}

// OpenAI-specific types

type openAIStreamingChatCompletionRequest struct {
	Model         string               `json:"model"`
	Messages      []openAIMessage      `json:"messages"`
	Stream        bool                 `json:"stream"`
	StreamOptions *openAIStreamOptions `json:"stream_options,omitempty"`
	Usage         *openAIUsageInclude  `json:"usage,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	MaxTokens     *int                 `json:"max_tokens,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	Tools         []openAITool         `json:"tools,omitempty"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIUsageInclude struct {
	Include bool `json:"include"`
}

type openAIStreamingChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role      string                      `json:"role,omitempty"`
	Content   string                      `json:"content,omitempty"`
	ToolCalls []openAIStreamDeltaToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamDeltaToolCall struct {
	Index    int            `json:"index"`
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type,omitempty"`
	Function openAIFunction `json:"function,omitempty"`
}

type openAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIMessage     `json:"messages"`
	Usage       *openAIUsageInclude `json:"usage,omitempty"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Tools       []openAITool        `json:"tools,omitempty"`
}

// openAIMessage represents a message in the OpenAI API format.
// Content can be either a string or an array of content parts (for multipart/cache control).
type openAIMessage struct {
	Role       string                  `json:"role"`
	Content    interface{}             `json:"content"` // string or []openAIContentPart
	Name       *string                 `json:"name,omitempty"`
	ToolCallID *string                 `json:"tool_call_id,omitempty"` // Required for tool result messages
	ToolCalls  []openAIMessageToolCall `json:"tool_calls,omitempty"`
}

// openAIContentPart represents a content part in multipart message format.
// Used for prompt caching (cache_control) and vision (image_url).
type openAIContentPart struct {
	Type         string              `json:"type"`                    // "text", "image_url"
	Text         string              `json:"text,omitempty"`          // For "text" type
	CacheControl *openAICacheControl `json:"cache_control,omitempty"` // For prompt caching
}

// MarshalJSON ensures that when Type == "text" we always include a "text" field
// even if the string is empty.
//
// Why: Ollama's OpenAI-compatible endpoint rejects content parts like {"type":"text"}
// (no text field) with "invalid message format".
func (p openAIContentPart) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type": p.Type,
	}

	// Always include "text" for text parts (even if empty string).
	if p.Type == "text" {
		m["text"] = p.Text
	} else if p.Text != "" {
		// Only include non-empty text for non-text parts.
		m["text"] = p.Text
	}

	if p.CacheControl != nil {
		m["cache_control"] = p.CacheControl
	}

	return json.Marshal(m)
}

// openAICacheControl specifies caching behavior for a content part.
// Supported by OpenRouter (Anthropic, Gemini). Ignored by OpenAI direct and Ollama.
type openAICacheControl struct {
	Type string `json:"type"`          // "ephemeral"
	TTL  string `json:"ttl,omitempty"` // "5m" or "1h"
}

type openAIMessageToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Arguments   string                 `json:"arguments,omitempty"` // Used in tool call responses
}

type openAIChatCompletionResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens        int                        `json:"prompt_tokens"`
	CompletionTokens    int                        `json:"completion_tokens"`
	TotalTokens         int                        `json:"total_tokens"`
	PromptTokensDetails *openAIPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// openAIPromptTokensDetails contains detailed prompt token information including cache hits.
type openAIPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}
