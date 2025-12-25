package interpreter

// ModelProvider defines the interface for LLM model providers
type ModelProvider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string

	// ChatCompletion sends a chat completion request
	ChatCompletion(request ChatRequest) (*ChatResponse, error)

	// StreamingChatCompletion sends a chat completion request with streaming response.
	// The callback is called for each chunk of content received.
	// Returns the final complete response after streaming is done.
	StreamingChatCompletion(request ChatRequest, callback StreamCallback) (*ChatResponse, error)
}

// StreamCallback is called for each chunk of streamed content.
// The content parameter contains the incremental text delta.
type StreamCallback func(content string)

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
	Role    string // "system", "user", "assistant", "tool"
	Content string
	Name    string // Optional: name of the tool or user
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
