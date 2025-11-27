package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ChatCompletion sends a chat completion request to OpenAI
func (p *OpenAIProvider) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
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
	baseURL := "https://api.openai.com/v1/chat/completions"
	if baseURLVal, ok := request.Model.Config["baseURL"]; ok {
		if baseURLStr, ok := baseURLVal.(*StringValue); ok && baseURLStr.Value != "" {
			baseURL = baseURLStr.Value
		}
	}

	// Build OpenAI-specific request
	openaiReq := openAIChatCompletionRequest{
		Model:    modelID,
		Messages: make([]openAIMessage, len(request.Messages)),
	}

	// Convert messages
	for i, msg := range request.Messages {
		openaiReq.Messages[i] = openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.Name != "" {
			openaiReq.Messages[i].Name = &msg.Name
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

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

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
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}

	// Add usage information if present
	if openaiResp.Usage != nil {
		response.Usage = &ChatUsage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
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

// OpenAI-specific types

type openAIChatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role      string                  `json:"role"`
	Content   string                  `json:"content"`
	Name      *string                 `json:"name,omitempty"`
	ToolCalls []openAIMessageToolCall `json:"tool_calls,omitempty"`
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
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
