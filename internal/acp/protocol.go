// Package acp provides types and client implementation for the Agent Client Protocol (ACP).
// ACP standardizes communication between clients and AI agents.
// See: https://agentclientprotocol.com/

package acp

import "encoding/json"

// JSON-RPC 2.0 protocol types

// JSONRPCVersion is the JSON-RPC version used by ACP.
const JSONRPCVersion = "2.0"

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID).
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ACP Method names
const (
	MethodInitialize    = "initialize"
	MethodSessionNew    = "session/new"
	MethodSessionLoad   = "session/load"
	MethodSessionPrompt = "session/prompt"
	MethodSessionUpdate = "session/update"
)

// InitializeParams represents the parameters for the initialize request.
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities,omitempty"`
}

// ClientCapabilities represents the client's capabilities.
type ClientCapabilities struct {
	FS       *FSCapabilities `json:"fs,omitempty"`
	Terminal bool            `json:"terminal,omitempty"`
}

// FSCapabilities represents file system capabilities.
type FSCapabilities struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

// InitializeResult represents the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion   int               `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities,omitempty"`
	AuthMethods       []string          `json:"authMethods,omitempty"`
}

// AgentCapabilities represents the agent's capabilities.
type AgentCapabilities struct {
	LoadSession        bool                `json:"loadSession,omitempty"`
	PromptCapabilities *PromptCapabilities `json:"promptCapabilities,omitempty"`
	MCP                *MCPCapabilities    `json:"mcp,omitempty"`
}

// PromptCapabilities represents capabilities for prompts.
type PromptCapabilities struct {
	Image           bool `json:"image,omitempty"`
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
}

// MCPCapabilities represents MCP-related capabilities.
type MCPCapabilities struct {
	HTTP bool `json:"http,omitempty"`
	SSE  bool `json:"sse,omitempty"`
}

// SessionNewParams represents the parameters for session/new request.
// Both Cwd and MCPServers are required fields per the ACP protocol.
type SessionNewParams struct {
	Cwd        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers"` // Required, can be empty array
}

// MCPServer represents an MCP server configuration for the agent.
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// SessionNewResult represents the result of session/new request.
type SessionNewResult struct {
	SessionID string `json:"sessionId"`
}

// SessionPromptParams represents the parameters for session/prompt request.
type SessionPromptParams struct {
	SessionID string          `json:"sessionId"`
	Prompt    []PromptContent `json:"prompt"`
}

// PromptContent represents a content block in a prompt.
type PromptContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Resource *PromptResource `json:"resource,omitempty"`
}

// PromptResource represents a resource in a prompt.
type PromptResource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// SessionPromptResult represents the result of session/prompt request.
type SessionPromptResult struct {
	StopReason string `json:"stopReason"`
}

// SessionUpdateParams represents the parameters for session/update notification.
type SessionUpdateParams struct {
	SessionID string               `json:"sessionId"`
	Update    SessionUpdatePayload `json:"update"`
}

// SessionUpdatePayload represents the update payload in a session/update notification.
// The Content field is polymorphic based on SessionUpdate type:
// - For "agent_message_chunk": Content is a MessageContent object
// - For "tool_call_update": Content is an array of ToolCallContent objects
type SessionUpdatePayload struct {
	SessionUpdate string `json:"sessionUpdate"`

	// For agent_message_chunk - Content is *MessageContent
	// For tool_call_update - Content is []ToolCallContent (parsed via ContentArray)
	Content json.RawMessage `json:"content,omitempty"`

	// For tool_call
	ToolCallID string                 `json:"toolCallId,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`

	// For tool_call_update
	Status    string      `json:"status,omitempty"`
	RawInput  interface{} `json:"rawInput,omitempty"`
	RawOutput interface{} `json:"rawOutput,omitempty"`
}

// GetMessageContent parses Content as a MessageContent (for agent_message_chunk).
func (p *SessionUpdatePayload) GetMessageContent() *MessageContent {
	if len(p.Content) == 0 {
		return nil
	}
	var content MessageContent
	if err := json.Unmarshal(p.Content, &content); err != nil {
		return nil
	}
	return &content
}

// GetToolCallContent parses Content as []ToolCallContent (for tool_call_update).
func (p *SessionUpdatePayload) GetToolCallContent() []ToolCallContent {
	if len(p.Content) == 0 {
		return nil
	}
	var content []ToolCallContent
	if err := json.Unmarshal(p.Content, &content); err != nil {
		return nil
	}
	return content
}

// GetToolName returns the tool name, preferring Name but falling back to Title.
// Some ACP agents (like Rovo Dev) use Title for human-readable tool names.
func (p *SessionUpdatePayload) GetToolName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.Title
}

// MessageContent represents content in a message.
type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolCallContent represents content produced by a tool call.
type ToolCallContent struct {
	Type    string          `json:"type"`
	Content *MessageContent `json:"content,omitempty"`
	// For diff type
	Path    string `json:"path,omitempty"`
	OldText string `json:"oldText,omitempty"`
	NewText string `json:"newText,omitempty"`
}

// SessionUpdateType constants for the sessionUpdate field.
const (
	SessionUpdateAgentMessageChunk = "agent_message_chunk"
	SessionUpdateToolCall          = "tool_call"
	SessionUpdateToolCallUpdate    = "tool_call_update"
)

// NewInitializeRequest creates a new initialize request.
func NewInitializeRequest(id int) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodInitialize,
		Params: InitializeParams{
			ProtocolVersion: 1,
			ClientCapabilities: ClientCapabilities{
				FS: &FSCapabilities{
					ReadTextFile:  true,
					WriteTextFile: true,
				},
				Terminal: true,
			},
		},
	}
}

// NewSessionNewRequest creates a new session/new request.
func NewSessionNewRequest(id int, cwd string, mcpServers []MCPServer) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodSessionNew,
		Params: SessionNewParams{
			Cwd:        cwd,
			MCPServers: mcpServers,
		},
	}
}

// NewSessionPromptRequest creates a new session/prompt request.
func NewSessionPromptRequest(id int, sessionID string, text string) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  MethodSessionPrompt,
		Params: SessionPromptParams{
			SessionID: sessionID,
			Prompt: []PromptContent{
				{
					Type: "text",
					Text: text,
				},
			},
		},
	}
}
