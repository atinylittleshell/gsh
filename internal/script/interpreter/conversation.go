package interpreter

import (
	"context"
	"fmt"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// AgentCallbacks provides hooks for observing and customizing agent execution.
// All callbacks are optional - nil callbacks are simply not called.
// This allows the REPL to drive its UI without the interpreter knowing about rendering.
//
// The callback types are aligned with the Agent Client Protocol (ACP) specification
// for standardized communication patterns. See: https://agentclientprotocol.com/
type AgentCallbacks struct {
	// OnIterationStart is called at the start of each agentic loop iteration.
	OnIterationStart func(iteration int)

	// OnChunk is called for each streaming content chunk.
	// Only called when Streaming is true.
	// Aligned with ACP's session/update message_chunk notifications.
	OnChunk func(content string)

	// OnToolPending is called when a tool call enters pending state (starts streaming from LLM).
	// At this point, we know the tool name but arguments may be incomplete/empty.
	// This allows showing a "pending" state to the user while arguments stream in.
	// The toolName is always available; partialArgs may be empty or partial.
	OnToolPending func(toolCallID string, toolName string)

	// OnToolCallStart is called before executing a tool.
	// The ToolCall contains the tool's initial state with Status = pending.
	// Aligned with ACP's session/update tool_call notifications.
	OnToolCallStart func(toolCall acp.ToolCall)

	// OnToolCallEnd is called after a tool completes.
	// The ToolCallUpdate contains the final status, result, and duration.
	// Aligned with ACP's session/update tool_call_update notifications.
	OnToolCallEnd func(toolCall acp.ToolCall, update acp.ToolCallUpdate)

	// OnResponse is called when a complete response is received (with usage stats).
	OnResponse func(response *ChatResponse)

	// OnComplete is called when the agent finishes.
	// The AgentResult contains the stop reason, token usage, and any error.
	// Aligned with ACP's session/prompt response with StopReason.
	OnComplete func(result acp.AgentResult)

	// Tools provides additional tools to be sent to the LLM.
	// These are merged with any tools defined in the agent's config.
	// This allows the REPL to provide built-in tools (exec, grep, etc.)
	// without modifying the agent's config.
	Tools []ChatTool

	// ToolExecutor overrides the default tool execution.
	// If nil, uses the interpreter's built-in tool resolution.
	// This allows the REPL to provide its own tool implementations (exec, grep, etc.)
	ToolExecutor func(ctx context.Context, toolName string, args map[string]interface{}) (string, error)

	// EventEmitter is called to emit SDK events (agent.start, agent.end, etc.).
	// If set, the interpreter will call this function to emit events.
	// The function receives the event name and a context object.
	// Handlers that want to produce output should print directly to stdout.
	// This allows the REPL to handle event emission through the SDK system.
	EventEmitter func(eventName string, ctx Value)
}

// evalPipeExpression evaluates a pipe expression
// Handles: String | Agent, Conversation | String, Conversation | Agent
func (i *Interpreter) evalPipeExpression(node *parser.PipeExpression) (Value, error) {
	// Evaluate left side
	left, err := i.evalExpression(node.Left)
	if err != nil {
		return nil, err
	}

	// Evaluate right side
	right, err := i.evalExpression(node.Right)
	if err != nil {
		return nil, err
	}

	// Handle different pipe combinations
	leftType := left.Type()
	rightType := right.Type()

	// Case 1: String | Agent -> Create conversation and execute
	if leftType == ValueTypeString && rightType == ValueTypeAgent {
		strVal := left.(*StringValue)
		agentVal := right.(*AgentValue)
		return i.executeAgentWithString(strVal.Value, agentVal)
	}

	// Case 2: Conversation | String -> Add user message
	if leftType == ValueTypeConversation && rightType == ValueTypeString {
		convVal := left.(*ConversationValue)
		strVal := right.(*StringValue)
		return i.addMessageToConversation(convVal, strVal.Value)
	}

	// Case 3: Conversation | Agent -> Execute agent with conversation context
	if leftType == ValueTypeConversation && rightType == ValueTypeAgent {
		convVal := left.(*ConversationValue)
		agentVal := right.(*AgentValue)
		return i.executeAgentWithConversation(convVal, agentVal)
	}

	return nil, fmt.Errorf("invalid pipe operation: cannot pipe %s to %s", leftType, rightType)
}

// executeAgentWithString creates a new conversation with a user message and executes the agent
func (i *Interpreter) executeAgentWithString(message string, agent *AgentValue) (Value, error) {
	// Create new conversation with just the user message
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	// Execute agent (which will inject system prompt)
	return i.executeAgentWithConversation(conv, agent)
}

// addMessageToConversation adds a user message to an existing conversation
func (i *Interpreter) addMessageToConversation(conv *ConversationValue, message string) (Value, error) {
	// Create a copy of the conversation to avoid mutation
	newConv := &ConversationValue{
		Messages: make([]ChatMessage, len(conv.Messages)),
	}
	copy(newConv.Messages, conv.Messages)

	// Add user message
	newConv.Messages = append(newConv.Messages, ChatMessage{
		Role:    "user",
		Content: message,
	})

	return newConv, nil
}

// executeAgentWithConversation executes an agent with an existing conversation
func (i *Interpreter) executeAgentWithConversation(conv *ConversationValue, agent *AgentValue) (Value, error) {
	// Prepare messages for the agent, injecting system prompt at the beginning
	messages := []ChatMessage{}

	// Add system prompt if configured
	if systemPromptVal, ok := agent.Config["systemPrompt"]; ok {
		if systemPromptStr, ok := systemPromptVal.(*StringValue); ok {
			messages = append(messages, ChatMessage{
				Role:    "system",
				Content: systemPromptStr.Value,
			})
		}
	}

	// Add all messages from the conversation
	messages = append(messages, conv.Messages...)

	// Create a temporary conversation with system prompt for execution
	execConv := &ConversationValue{
		Messages: messages,
	}

	// Enable streaming when running in REPL mode so agent.chunk events are emitted
	// and the response is displayed to the user
	streaming := i.sdkConfig.GetREPLContext() != nil

	return i.ExecuteAgent(context.Background(), execConv, agent, streaming)
}
