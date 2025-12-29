package interpreter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Agent lifecycle event names
const (
	EventAgentStart          = "agent.start"
	EventAgentEnd            = "agent.end"
	EventAgentIterationStart = "agent.iteration.start"
	EventAgentIterationEnd   = "agent.iteration.end"
	EventAgentChunk          = "agent.chunk"
	EventAgentToolStart      = "agent.tool.start"
	EventAgentToolEnd        = "agent.tool.end"
	EventAgentExecStart      = "agent.exec.start"
	EventAgentExecEnd        = "agent.exec.end"
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

	// OnToolCallStreaming is called when a tool call starts streaming from the LLM.
	// At this point, we know the tool name but arguments may be incomplete/empty.
	// This allows showing a "pending" state to the user while arguments stream in.
	// The toolName is always available; partialArgs may be empty or partial.
	OnToolCallStreaming func(toolCallID string, toolName string)

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

	// Streaming enables streaming responses.
	// When true, OnChunk will be called for each content chunk.
	Streaming bool

	// EventEmitter is called to emit SDK events (agent.start, agent.end, etc.).
	// If set, the interpreter will call this function to emit events.
	// The function receives the event name and a context object.
	// Handlers that want to produce output should print directly to stdout.
	// This allows the REPL to handle event emission through the SDK system.
	EventEmitter func(eventName string, ctx Value)

	// UserMessage is the original user message that started this agent conversation.
	// This is used for the agent.start event context.
	UserMessage string
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

	return i.executeAgent(execConv, agent)
}

// DefaultMaxIterations is the default maximum number of tool call iterations
// if not specified in the agent config.
const DefaultMaxIterations = 100

// executeAgent executes an agent with a conversation and returns the updated conversation.
// This is the simple non-streaming version used by the script interpreter.
func (i *Interpreter) executeAgent(conv *ConversationValue, agent *AgentValue) (Value, error) {
	return i.ExecuteAgentWithCallbacks(context.Background(), conv, agent, nil)
}

// Helper functions to create event context objects

// createAgentStartContext creates the context object for agent.start event
// ctx: { message: string }
func createAgentStartContext(message string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"message": {Value: &StringValue{Value: message}},
		},
	}
}

// createAgentEndContext creates the context object for agent.end event
// ctx: { result: { stopReason, durationMs, totalInputTokens, totalOutputTokens, error } }
func createAgentEndContext(stopReason string, durationMs int64, inputTokens, outputTokens int, err error) Value {
	var errorVal Value = &NullValue{}
	if err != nil {
		errorVal = &StringValue{Value: err.Error()}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"result": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"stopReason":        {Value: &StringValue{Value: stopReason}},
					"durationMs":        {Value: &NumberValue{Value: float64(durationMs)}},
					"totalInputTokens":  {Value: &NumberValue{Value: float64(inputTokens)}},
					"totalOutputTokens": {Value: &NumberValue{Value: float64(outputTokens)}},
					"error":             {Value: errorVal},
				},
			}},
		},
	}
}

// createIterationStartContext creates the context object for agent.iteration.start event
// ctx: { iteration: number }
func createIterationStartContext(iteration int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"iteration": {Value: &NumberValue{Value: float64(iteration + 1)}}, // 1-based for users
		},
	}
}

// createIterationEndContext creates the context object for agent.iteration.end event
// ctx: { iteration: number, usage: { inputTokens, outputTokens, cachedTokens } }
func createIterationEndContext(iteration int, inputTokens, outputTokens, cachedTokens int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"iteration": {Value: &NumberValue{Value: float64(iteration + 1)}}, // 1-based for users
			"usage": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"inputTokens":  {Value: &NumberValue{Value: float64(inputTokens)}},
					"outputTokens": {Value: &NumberValue{Value: float64(outputTokens)}},
					"cachedTokens": {Value: &NumberValue{Value: float64(cachedTokens)}},
				},
			}},
		},
	}
}

// createChunkContext creates the context object for agent.chunk event
// ctx: { content: string }
func createChunkContext(content string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"content": {Value: &StringValue{Value: content}},
		},
	}
}

// createToolCallContext creates the context object for agent.tool.start/end events
// ctx: { toolCall: { id, name, args, durationMs?, output?, error? } }
func createToolCallContext(id, name string, args map[string]interface{}, durationMs *int64, output *string, err error) Value {
	// Convert args to ObjectValue
	argsProps := make(map[string]*PropertyDescriptor)
	for k, v := range args {
		argsProps[k] = &PropertyDescriptor{Value: InterfaceToValue(v)}
	}

	toolCallProps := map[string]*PropertyDescriptor{
		"id":   {Value: &StringValue{Value: id}},
		"name": {Value: &StringValue{Value: name}},
		"args": {Value: &ObjectValue{Properties: argsProps}},
	}

	if durationMs != nil {
		toolCallProps["durationMs"] = &PropertyDescriptor{Value: &NumberValue{Value: float64(*durationMs)}}
	}
	if output != nil {
		toolCallProps["output"] = &PropertyDescriptor{Value: &StringValue{Value: *output}}
	}
	if err != nil {
		toolCallProps["error"] = &PropertyDescriptor{Value: &StringValue{Value: err.Error()}}
	} else if durationMs != nil {
		// Only set error to null on end events (when durationMs is present)
		toolCallProps["error"] = &PropertyDescriptor{Value: &NullValue{}}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"toolCall": {Value: &ObjectValue{Properties: toolCallProps}},
		},
	}
}

// CreateExecStartContext creates the context object for agent.exec.start event
// ctx: { exec: { command, commandFirstWord } }
func CreateExecStartContext(command string) Value {
	firstWord := command
	if idx := findFirstSpace(command); idx > 0 {
		firstWord = command[:idx]
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"exec": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"command":          {Value: &StringValue{Value: command}},
					"commandFirstWord": {Value: &StringValue{Value: firstWord}},
				},
			}},
		},
	}
}

// CreateExecEndContext creates the context object for agent.exec.end event
// ctx: { exec: { command, commandFirstWord, durationMs, exitCode } }
func CreateExecEndContext(command string, durationMs int64, exitCode int) Value {
	firstWord := command
	if idx := findFirstSpace(command); idx > 0 {
		firstWord = command[:idx]
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"exec": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"command":          {Value: &StringValue{Value: command}},
					"commandFirstWord": {Value: &StringValue{Value: firstWord}},
					"durationMs":       {Value: &NumberValue{Value: float64(durationMs)}},
					"exitCode":         {Value: &NumberValue{Value: float64(exitCode)}},
				},
			}},
		},
	}
}

// findFirstSpace returns the index of the first space in a string, or -1 if not found
func findFirstSpace(s string) int {
	for i, c := range s {
		if c == ' ' || c == '\t' {
			return i
		}
	}
	return -1
}

// InterfaceToValue converts a Go interface{} to a Value.
// This is the canonical conversion function used throughout the interpreter.
func InterfaceToValue(val interface{}) Value {
	switch v := val.(type) {
	case nil:
		return &NullValue{}
	case bool:
		return &BoolValue{Value: v}
	case float64:
		return &NumberValue{Value: v}
	case int:
		return &NumberValue{Value: float64(v)}
	case int64:
		return &NumberValue{Value: float64(v)}
	case string:
		return &StringValue{Value: v}
	case []interface{}:
		elements := make([]Value, len(v))
		for idx, elem := range v {
			elements[idx] = InterfaceToValue(elem)
		}
		return &ArrayValue{Elements: elements}
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, val := range v {
			properties[key] = &PropertyDescriptor{Value: InterfaceToValue(val)}
		}
		return &ObjectValue{Properties: properties}
	default:
		return &StringValue{Value: fmt.Sprintf("%v", v)}
	}
}

// ExecuteAgentWithCallbacks executes an agent with optional callbacks for streaming and UI hooks.
// This is the core agentic loop implementation that can be used by both the script interpreter
// and the REPL. When callbacks is nil, it behaves like the simple executeAgent.
func (i *Interpreter) ExecuteAgentWithCallbacks(ctx context.Context, conv *ConversationValue, agent *AgentValue, callbacks *AgentCallbacks) (Value, error) {
	startTime := time.Now()

	// Track token usage across all iterations
	var totalInputTokens, totalOutputTokens, totalCachedTokens int

	// Helper to emit events if EventEmitter is configured
	emitEvent := func(eventName string, eventCtx Value) {
		if callbacks != nil && callbacks.EventEmitter != nil {
			callbacks.EventEmitter(eventName, eventCtx)
		}
	}

	// Get user message for events (from callbacks or from conversation)
	userMessage := ""
	if callbacks != nil && callbacks.UserMessage != "" {
		userMessage = callbacks.UserMessage
	} else if len(conv.Messages) > 0 {
		// Find the last user message in conversation
		for j := len(conv.Messages) - 1; j >= 0; j-- {
			if conv.Messages[j].Role == "user" {
				userMessage = conv.Messages[j].Content
				break
			}
		}
	}

	// Emit agent.start event
	emitEvent(EventAgentStart, createAgentStartContext(userMessage))

	// Helper to call OnComplete callback with ACP-aligned result
	callOnComplete := func(stopReason acp.StopReason, err error) {
		// Emit agent.end event first
		durationMs := time.Since(startTime).Milliseconds()
		emitEvent(EventAgentEnd, createAgentEndContext(string(stopReason), durationMs, totalInputTokens, totalOutputTokens, err))

		if callbacks != nil && callbacks.OnComplete != nil {
			result := acp.AgentResult{
				StopReason: stopReason,
				Duration:   time.Since(startTime),
				Usage: &acp.TokenUsage{
					PromptTokens:     totalInputTokens,
					CompletionTokens: totalOutputTokens,
					CachedTokens:     totalCachedTokens,
					TotalTokens:      totalInputTokens + totalOutputTokens,
				},
				Error: err,
			}
			callbacks.OnComplete(result)
		}
	}

	// Get model from agent config
	modelVal, ok := agent.Config["model"]
	if !ok {
		err := fmt.Errorf("agent '%s' has no model configured", agent.Name)
		callOnComplete(acp.StopReasonError, err)
		return nil, err
	}
	model, ok := modelVal.(*ModelValue)
	if !ok {
		err := fmt.Errorf("agent '%s' model config is not a model", agent.Name)
		callOnComplete(acp.StopReasonError, err)
		return nil, err
	}

	// Get max iterations from agent config, or use default
	maxIterations := DefaultMaxIterations
	if maxIterVal, ok := agent.Config["maxIterations"]; ok {
		if numVal, ok := maxIterVal.(*NumberValue); ok {
			maxIterations = int(numVal.Value)
			if maxIterations <= 0 {
				maxIterations = DefaultMaxIterations
			}
		}
	}

	// Prepare tools for the agent
	// First, add tools from callbacks (e.g., REPL built-in tools)
	tools := []ChatTool{}
	if callbacks != nil && len(callbacks.Tools) > 0 {
		tools = append(tools, callbacks.Tools...)
	}

	// Then add tools from agent config
	if toolsVal, ok := agent.Config["tools"]; ok {
		if toolsArr, ok := toolsVal.(*ArrayValue); ok {
			for _, toolValInterface := range toolsArr.Elements {
				// Handle different tool types
				switch toolVal := toolValInterface.(type) {
				case *ToolValue:
					// User-defined tool
					tool := i.convertUserToolToChatTool(toolVal)
					tools = append(tools, tool)
				case *MCPToolValue:
					// MCP tool
					tool, err := i.convertMCPToolToChatTool(toolVal)
					if err != nil {
						err = fmt.Errorf("failed to convert MCP tool: %w", err)
						callOnComplete(acp.StopReasonError, err)
						return nil, err
					}
					tools = append(tools, tool)
				case *NativeToolValue:
					// Native tool (gsh.tools.*)
					tool := i.convertNativeToolToChatTool(toolVal)
					tools = append(tools, tool)
				default:
					err := fmt.Errorf("invalid tool type in agent config: %s", toolVal.Type())
					callOnComplete(acp.StopReasonError, err)
					return nil, err
				}
			}
		}
	}

	// Create new conversation with response (excluding system messages)
	// System prompts should not be stored in conversations
	userMessages := []ChatMessage{}
	for _, msg := range conv.Messages {
		if msg.Role != "system" {
			userMessages = append(userMessages, msg)
		}
	}

	newConv := &ConversationValue{
		Messages: make([]ChatMessage, len(userMessages)),
	}
	copy(newConv.Messages, userMessages)

	// Build messages for the request (include system prompt for the model)
	buildRequestMessages := func() []ChatMessage {
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
		messages = append(messages, newConv.Messages...)
		return messages
	}

	// Determine if we should use streaming
	useStreaming := callbacks != nil && callbacks.Streaming

	// Agentic loop - continue until no tool calls or max iterations reached
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			err := ctx.Err()
			callOnComplete(acp.StopReasonCancelled, err)
			return newConv, err
		}

		// Emit agent.iteration.start event
		emitEvent(EventAgentIterationStart, createIterationStartContext(iteration))

		// Call iteration start callback
		if callbacks != nil && callbacks.OnIterationStart != nil {
			callbacks.OnIterationStart(iteration)
		}

		// Create chat request
		request := ChatRequest{
			Model:    model,
			Messages: buildRequestMessages(),
			Tools:    tools,
		}

		// Call the model (streaming or non-streaming)
		var response *ChatResponse
		var err error

		if useStreaming {
			// Use streaming with tool call detection
			streamCallbacks := &StreamCallbacks{
				OnContent: func(content string) {
					// Emit agent.chunk event
					emitEvent(EventAgentChunk, createChunkContext(content))
					// Also call the original callback
					if callbacks.OnChunk != nil {
						callbacks.OnChunk(content)
					}
				},
			}
			if callbacks.OnToolCallStreaming != nil {
				streamCallbacks.OnToolCallStart = callbacks.OnToolCallStreaming
			}
			response, err = model.Provider.StreamingChatCompletion(request, streamCallbacks)
		} else {
			// Non-streaming call
			response, err = model.ChatCompletion(request)
		}

		if err != nil {
			err = fmt.Errorf("agent execution failed: %w", err)
			callOnComplete(acp.StopReasonError, err)
			return nil, err
		}

		// Accumulate token usage
		iterInputTokens, iterOutputTokens, iterCachedTokens := 0, 0, 0
		if response.Usage != nil {
			iterInputTokens = response.Usage.PromptTokens
			iterOutputTokens = response.Usage.CompletionTokens
			iterCachedTokens = response.Usage.CachedTokens
			totalInputTokens += iterInputTokens
			totalOutputTokens += iterOutputTokens
			totalCachedTokens += iterCachedTokens
		}

		// Call response callback
		if callbacks != nil && callbacks.OnResponse != nil {
			callbacks.OnResponse(response)
		}

		// If no tool calls, add final response and return
		if len(response.ToolCalls) == 0 {
			newConv.Messages = append(newConv.Messages, ChatMessage{
				Role:    "assistant",
				Content: response.Content,
			})
			// Emit agent.iteration.end event before completing
			emitEvent(EventAgentIterationEnd, createIterationEndContext(iteration, iterInputTokens, iterOutputTokens, iterCachedTokens))
			callOnComplete(acp.StopReasonEndTurn, nil)
			return newConv, nil
		}

		// Add assistant message with tool calls
		newConv.Messages = append(newConv.Messages, ChatMessage{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		// Execute tool calls and add results
		for _, toolCall := range response.ToolCalls {
			// Create ACP-aligned tool call for callbacks
			acpToolCall := acp.ToolCall{
				ID:        toolCall.ID,
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
				Status:    acp.ToolCallStatusPending,
				Kind:      classifyToolKind(toolCall.Name),
			}

			// Emit agent.tool.start event
			emitEvent(EventAgentToolStart, createToolCallContext(toolCall.ID, toolCall.Name, toolCall.Arguments, nil, nil, nil))

			// Call tool start callback
			if callbacks != nil && callbacks.OnToolCallStart != nil {
				callbacks.OnToolCallStart(acpToolCall)
			}

			toolStart := time.Now()
			var toolResult string
			var toolErr error

			// Use custom tool executor if provided, otherwise use interpreter's built-in
			if callbacks != nil && callbacks.ToolExecutor != nil {
				toolResult, toolErr = callbacks.ToolExecutor(ctx, toolCall.Name, toolCall.Arguments)
			} else {
				toolResult, toolErr = i.executeToolCall(agent, toolCall)
			}

			toolDuration := time.Since(toolStart)
			toolDurationMs := toolDuration.Milliseconds()

			// Emit agent.tool.end event
			emitEvent(EventAgentToolEnd, createToolCallContext(toolCall.ID, toolCall.Name, toolCall.Arguments, &toolDurationMs, &toolResult, toolErr))

			// Call tool end callback with ACP-aligned update
			if callbacks != nil && callbacks.OnToolCallEnd != nil {
				status := acp.ToolCallStatusCompleted
				if toolErr != nil {
					status = acp.ToolCallStatusFailed
				}
				update := acp.ToolCallUpdate{
					ID:       toolCall.ID,
					Status:   status,
					Content:  toolResult,
					Duration: toolDuration,
					Error:    toolErr,
				}
				callbacks.OnToolCallEnd(acpToolCall, update)
			}

			if toolErr != nil {
				// On error, add error message as tool result so the model can recover
				toolResult = fmt.Sprintf("Error executing tool: %v", toolErr)
			}

			// Add tool result to conversation with proper tool_call_id
			newConv.Messages = append(newConv.Messages, ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				Name:       toolCall.Name,
				ToolCallID: toolCall.ID,
			})
		}

		// Emit agent.iteration.end event
		emitEvent(EventAgentIterationEnd, createIterationEndContext(iteration, iterInputTokens, iterOutputTokens, iterCachedTokens))

		// Continue loop to make another call
	}

	// If we reach here, we hit max iterations - return what we have
	err := fmt.Errorf("agent reached maximum iterations (%d) without completing", maxIterations)
	callOnComplete(acp.StopReasonMaxIterations, err)
	return newConv, err
}

// classifyToolKind determines the ToolKind based on the tool name.
// This helps clients choose appropriate icons and UI treatment.
func classifyToolKind(toolName string) acp.ToolKind {
	switch toolName {
	case "exec", "bash", "shell":
		return acp.ToolKindExecute
	case "read_file", "view_file", "cat", "get", "fetch":
		return acp.ToolKindRead
	case "write_file", "edit_file", "create_file", "delete_file", "put", "post":
		return acp.ToolKindWrite
	case "grep", "search", "find", "list_files", "ls":
		return acp.ToolKindSearch
	default:
		return acp.ToolKindOther
	}
}

// executeToolCall executes a tool call from the agent
func (i *Interpreter) executeToolCall(agent *AgentValue, toolCall ChatToolCall) (string, error) {
	// Find the tool in the agent's tool list
	toolsVal, ok := agent.Config["tools"]
	if !ok {
		return "", fmt.Errorf("agent has no tools configured")
	}

	toolsArr, ok := toolsVal.(*ArrayValue)
	if !ok {
		return "", fmt.Errorf("agent tools config is not an array")
	}

	// Find the matching tool
	for _, toolValInterface := range toolsArr.Elements {
		switch toolVal := toolValInterface.(type) {
		case *ToolValue:
			if toolVal.Name == toolCall.Name {
				return i.executeUserToolCall(toolVal, toolCall.Arguments)
			}
		case *MCPToolValue:
			if toolVal.ToolName == toolCall.Name {
				return i.executeMCPToolCall(toolVal, toolCall.Arguments)
			}
		case *NativeToolValue:
			if toolVal.Name == toolCall.Name {
				return i.executeNativeToolCall(toolVal, toolCall.Arguments)
			}
		}
	}

	return "", fmt.Errorf("tool '%s' not found in agent configuration", toolCall.Name)
}

// executeUserToolCall executes a user-defined tool call
func (i *Interpreter) executeUserToolCall(tool *ToolValue, args map[string]interface{}) (string, error) {
	// Convert arguments to Value array in parameter order
	valueArgs := make([]Value, len(tool.Parameters))
	for idx, paramName := range tool.Parameters {
		argVal, ok := args[paramName]
		if !ok {
			return "", fmt.Errorf("missing argument '%s' for tool '%s'", paramName, tool.Name)
		}

		// Convert JSON value to Value type
		val, err := i.jsonToValue(argVal)
		if err != nil {
			return "", fmt.Errorf("failed to convert argument '%s': %w", paramName, err)
		}
		valueArgs[idx] = val
	}

	// Call the tool
	result, err := i.CallTool(tool, valueArgs)
	if err != nil {
		return "", err
	}

	// Convert result to string (JSON format for complex types)
	return i.valueToJSON(result)
}

// executeMCPToolCall executes an MCP tool call
func (i *Interpreter) executeMCPToolCall(tool *MCPToolValue, args map[string]interface{}) (string, error) {
	// Call MCP tool using the Call method
	result, err := tool.Call(args)
	if err != nil {
		return "", err
	}

	// Convert result to JSON string
	return i.valueToJSON(result)
}

// executeNativeToolCall executes a native tool call (gsh.tools.*)
func (i *Interpreter) executeNativeToolCall(tool *NativeToolValue, args map[string]interface{}) (string, error) {
	// Call the native tool's Invoke function
	result, err := tool.Invoke(args)
	if err != nil {
		return "", err
	}

	// Native tools return strings directly (already JSON formatted)
	if str, ok := result.(string); ok {
		return str, nil
	}

	// Fallback: convert to JSON string
	return i.valueToJSON(i.interfaceToValue(result))
}

// interfaceToValue converts a Go interface{} to a Value.
// This is a method wrapper around InterfaceToValue for backward compatibility.
func (i *Interpreter) interfaceToValue(val interface{}) Value {
	return InterfaceToValue(val)
}

// convertUserToolToChatTool converts a user-defined tool to ChatTool format
func (i *Interpreter) convertUserToolToChatTool(tool *ToolValue) ChatTool {
	// Build parameters schema
	params := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}

	properties := params["properties"].(map[string]interface{})
	required := []string{}

	for _, paramName := range tool.Parameters {
		paramType := "string" // default
		if typeName, ok := tool.ParamTypes[paramName]; ok {
			paramType = i.mapGSHTypeToJSONType(typeName)
		}
		properties[paramName] = map[string]interface{}{
			"type": paramType,
		}
		required = append(required, paramName)
	}
	params["required"] = required

	return ChatTool{
		Name:        tool.Name,
		Description: fmt.Sprintf("User-defined tool: %s", tool.Name),
		Parameters:  params,
	}
}

// convertMCPToolToChatTool converts an MCP tool to ChatTool format
func (i *Interpreter) convertMCPToolToChatTool(tool *MCPToolValue) (ChatTool, error) {
	// Get tool info from the MCP manager
	toolInfo, err := i.mcpManager.GetToolInfo(tool.ServerName, tool.ToolName)
	if err != nil {
		return ChatTool{}, fmt.Errorf("failed to get tool info: %w", err)
	}

	return ChatTool{
		Name:        tool.ToolName,
		Description: toolInfo.Description,
		Parameters:  toolInfo.InputSchema,
	}, nil
}

// convertNativeToolToChatTool converts a native tool to ChatTool format
func (i *Interpreter) convertNativeToolToChatTool(tool *NativeToolValue) ChatTool {
	return ChatTool{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  tool.Parameters,
	}
}

// mapGSHTypeToJSONType maps GSH type annotations to JSON schema types
func (i *Interpreter) mapGSHTypeToJSONType(gshType string) string {
	switch gshType {
	case "string":
		return "string"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string"
	}
}

// jsonToValue converts a JSON value to a GSH Value
func (i *Interpreter) jsonToValue(jsonVal interface{}) (Value, error) {
	switch v := jsonVal.(type) {
	case nil:
		return &NullValue{}, nil
	case bool:
		return &BoolValue{Value: v}, nil
	case float64:
		return &NumberValue{Value: v}, nil
	case string:
		return &StringValue{Value: v}, nil
	case []interface{}:
		elements := make([]Value, len(v))
		for idx, elem := range v {
			val, err := i.jsonToValue(elem)
			if err != nil {
				return nil, err
			}
			elements[idx] = val
		}
		return &ArrayValue{Elements: elements}, nil
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, val := range v {
			gshVal, err := i.jsonToValue(val)
			if err != nil {
				return nil, err
			}
			properties[key] = &PropertyDescriptor{Value: gshVal}
		}
		return &ObjectValue{Properties: properties}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", jsonVal)
	}
}

// valueToJSON converts a GSH Value to a JSON string.
// It uses json.Marshal for proper escaping of special characters.
func (i *Interpreter) valueToJSON(val Value) (string, error) {
	// Convert Value to interface{} and use json.Marshal for proper escaping
	jsonBytes, err := json.Marshal(i.valueToInterface(val))
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// ValueToInterface converts a Value to interface{}.
// This is the canonical conversion function used throughout the interpreter.
func ValueToInterface(val Value) interface{} {
	switch v := val.(type) {
	case *NullValue:
		return nil
	case *BoolValue:
		return v.Value
	case *NumberValue:
		return v.Value
	case *StringValue:
		return v.Value
	case *ArrayValue:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = ValueToInterface(elem)
		}
		return arr
	case *ObjectValue:
		result := make(map[string]interface{})
		for key := range v.Properties {
			result[key] = ValueToInterface(v.GetPropertyValue(key))
		}
		return result
	default:
		return val.String()
	}
}

// valueToInterface converts a Value to interface{}.
// This is a method wrapper around ValueToInterface for backward compatibility.
func (i *Interpreter) valueToInterface(val Value) interface{} {
	return ValueToInterface(val)
}
