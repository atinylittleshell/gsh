// Package acp provides types aligned with the Agent Client Protocol (ACP) specification.
// ACP standardizes communication between clients and AI agents.
// See: https://agentclientprotocol.com/
//
// These types are used internally by gsh to provide consistent interfaces
// across the interpreter and REPL, without requiring full ACP protocol compliance.
package acp

import "time"

// ToolCallStatus represents the execution status of a tool call.
// Aligned with ACP's ToolCallStatus enum.
type ToolCallStatus string

const (
	// ToolCallStatusPending indicates the tool call is waiting to be executed.
	// In ACP: the tool call has been created but not yet started.
	ToolCallStatusPending ToolCallStatus = "pending"

	// ToolCallStatusInProgress indicates the tool is currently executing.
	// In ACP: the tool call is actively running.
	ToolCallStatusInProgress ToolCallStatus = "in_progress"

	// ToolCallStatusCompleted indicates the tool finished successfully.
	// In ACP: the tool call completed without errors.
	ToolCallStatusCompleted ToolCallStatus = "completed"

	// ToolCallStatusFailed indicates the tool encountered an error.
	// In ACP: the tool call failed during execution.
	ToolCallStatusFailed ToolCallStatus = "failed"
)

// StopReason represents why an agent stopped processing a prompt turn.
// Aligned with ACP's StopReason enum.
type StopReason string

const (
	// StopReasonEndTurn indicates the agent finished its turn successfully.
	// The language model completed responding without requesting more tools.
	StopReasonEndTurn StopReason = "end_turn"

	// StopReasonMaxTokens indicates the maximum token limit was reached.
	StopReasonMaxTokens StopReason = "max_tokens"

	// StopReasonMaxIterations indicates the maximum number of agentic loop
	// iterations was exceeded. This is gsh-specific (ACP uses max_turn_requests).
	StopReasonMaxIterations StopReason = "max_iterations"

	// StopReasonRefusal indicates the agent refused to continue.
	// The agent declined to process the request.
	StopReasonRefusal StopReason = "refusal"

	// StopReasonCancelled indicates the turn was cancelled by the client.
	// This is returned when context cancellation occurs.
	StopReasonCancelled StopReason = "cancelled"

	// StopReasonError indicates an error occurred during processing.
	// This is gsh-specific for general errors.
	StopReasonError StopReason = "error"
)

// ToolKind represents the category of tool being invoked.
// Helps clients choose appropriate icons and UI treatment.
// Aligned with ACP's ToolKind enum.
type ToolKind string

const (
	// ToolKindRead indicates a tool that reads data (files, APIs, etc.)
	ToolKindRead ToolKind = "read"

	// ToolKindWrite indicates a tool that writes/modifies data
	ToolKindWrite ToolKind = "write"

	// ToolKindExecute indicates a tool that executes commands
	ToolKindExecute ToolKind = "execute"

	// ToolKindSearch indicates a tool that searches for information
	ToolKindSearch ToolKind = "search"

	// ToolKindOther indicates a tool that doesn't fit other categories
	ToolKindOther ToolKind = "other"
)

// ToolCall represents a tool call that the language model has requested.
// This is a simplified version of ACP's ToolCall for internal use.
type ToolCall struct {
	// ID is a unique identifier for this tool call within the session.
	// Maps to ACP's toolCallId.
	ID string `json:"id"`

	// Name is the name of the tool being invoked.
	Name string `json:"name"`

	// Title is a human-readable description of what the tool is doing.
	// Optional - if empty, clients should use the tool name.
	Title string `json:"title,omitempty"`

	// Kind is the category of tool being invoked.
	Kind ToolKind `json:"kind,omitempty"`

	// Arguments are the parameters passed to the tool.
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// Status is the current execution status.
	Status ToolCallStatus `json:"status"`
}

// ToolCallUpdate represents an update to an ongoing tool call.
// Sent during tool execution to report progress.
type ToolCallUpdate struct {
	// ID identifies which tool call this update is for.
	ID string `json:"id"`

	// Status is the new execution status, if changed.
	Status ToolCallStatus `json:"status,omitempty"`

	// Content is any output produced by the tool so far.
	Content string `json:"content,omitempty"`

	// Duration is how long the tool has been running.
	Duration time.Duration `json:"duration,omitempty"`

	// Error contains error information if the tool failed.
	Error error `json:"error,omitempty"`
}

// SessionUpdate represents a real-time update during agent execution.
// This is a simplified version of ACP's session/update notification.
type SessionUpdate struct {
	// Type indicates what kind of update this is.
	Type SessionUpdateType `json:"type"`

	// Content is the text content (for message chunks).
	Content string `json:"content,omitempty"`

	// ToolCall is set for tool_call updates.
	ToolCall *ToolCall `json:"toolCall,omitempty"`

	// ToolCallUpdate is set for tool_call_update updates.
	ToolCallUpdate *ToolCallUpdate `json:"toolCallUpdate,omitempty"`

	// StopReason is set when the agent stops.
	StopReason StopReason `json:"stopReason,omitempty"`
}

// SessionUpdateType indicates the type of session update.
type SessionUpdateType string

const (
	// SessionUpdateTypeMessageChunk is a streaming content chunk from the agent.
	SessionUpdateTypeMessageChunk SessionUpdateType = "message_chunk"

	// SessionUpdateTypeToolCall indicates a new tool call is being made.
	SessionUpdateTypeToolCall SessionUpdateType = "tool_call"

	// SessionUpdateTypeToolCallUpdate is an update to an existing tool call.
	SessionUpdateTypeToolCallUpdate SessionUpdateType = "tool_call_update"

	// SessionUpdateTypePlan indicates the agent's execution plan (if supported).
	SessionUpdateTypePlan SessionUpdateType = "plan"

	// SessionUpdateTypeComplete indicates the agent has finished.
	SessionUpdateTypeComplete SessionUpdateType = "complete"
)

// TokenUsage tracks token consumption during agent execution.
type TokenUsage struct {
	// PromptTokens is the number of tokens in the input/prompt.
	PromptTokens int `json:"promptTokens"`

	// CompletionTokens is the number of tokens in the output/completion.
	CompletionTokens int `json:"completionTokens"`

	// CachedTokens is the number of prompt tokens that were cache hits.
	CachedTokens int `json:"cachedTokens"`

	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int `json:"totalTokens"`
}

// AgentResult represents the result of an agent execution.
type AgentResult struct {
	// StopReason indicates why the agent stopped.
	StopReason StopReason `json:"stopReason"`

	// Content is the final response content from the agent.
	Content string `json:"content,omitempty"`

	// Usage contains token usage statistics.
	Usage *TokenUsage `json:"usage,omitempty"`

	// Duration is the total execution time.
	Duration time.Duration `json:"duration"`

	// Error contains any error that occurred (when StopReason is error).
	Error error `json:"error,omitempty"`
}
