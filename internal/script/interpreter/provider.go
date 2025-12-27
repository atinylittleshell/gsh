package interpreter

// ModelProvider defines the interface for LLM model providers
type ModelProvider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string

	// ChatCompletion sends a chat completion request
	ChatCompletion(request ChatRequest) (*ChatResponse, error)

	// StreamingChatCompletion sends a chat completion request with streaming response.
	// The callbacks provide hooks for content chunks and tool call detection.
	// Returns the final complete response after streaming is done.
	StreamingChatCompletion(request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error)
}

// StreamCallback is called for each chunk of streamed content.
// The content parameter contains the incremental text delta.
type StreamCallback func(content string)

// StreamCallbacks provides extended callbacks for streaming responses.
// This allows for more granular control over streaming, including tool call detection.
type StreamCallbacks struct {
	// OnContent is called for each chunk of content text.
	OnContent func(content string)

	// OnToolCallStart is called when a tool call starts streaming.
	// At this point, the tool ID and name are known but arguments may still be streaming.
	OnToolCallStart func(toolCallID string, toolName string)
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	// Model configuration
	Model *ModelValue

	// Messages in the conversation
	Messages []ChatMessage

	// Tools available to the agent
	Tools []ChatTool
}

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role       string // "system", "user", "assistant", "tool"
	Content    string
	Name       string         // Optional: name of the tool or user
	ToolCallID string         // For tool result messages (required by OpenAI API)
	ToolCalls  []ChatToolCall // For assistant messages that request tool calls

	// ContentParts allows multipart content with cache control for prompt caching.
	// When set, this takes precedence over Content for providers that support it.
	// Compatible with OpenAI, Ollama, and OpenRouter (Anthropic, Gemini, etc.)
	ContentParts []ContentPart
}

// ContentPart represents a part of a multipart message content.
// Used for prompt caching with OpenRouter/Anthropic and vision content with OpenAI.
type ContentPart struct {
	Type         string        // "text", "image_url"
	Text         string        // For "text" type
	CacheControl *CacheControl // Optional cache control for prompt caching
}

// CacheControl specifies caching behavior for a content part.
// Supported by OpenRouter when using Anthropic or Gemini models.
// Ignored (but safe to send) by OpenAI direct and Ollama.
type CacheControl struct {
	Type string // "ephemeral" - marks content for caching
	TTL  string // Optional: "5m" or "1h" (Anthropic supports these TTLs)
}

// ChatTool represents a tool that can be called by the model
type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	// The generated message
	Content string

	// Finish reason ("stop", "length", "tool_calls", etc.)
	FinishReason string

	// Token usage information
	Usage *ChatUsage

	// Tool calls requested by the model
	ToolCalls []ChatToolCall
}

// ChatUsage represents token usage information
type ChatUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	// CachedTokens indicates how many prompt tokens were cache hits.
	CachedTokens int
}

// ChatToolCall represents a tool call requested by the model
type ChatToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

// ProviderRegistry manages model providers
type ProviderRegistry struct {
	providers map[string]ModelProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]ModelProvider),
	}
}

// Register registers a model provider
func (r *ProviderRegistry) Register(provider ModelProvider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (ModelProvider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}
