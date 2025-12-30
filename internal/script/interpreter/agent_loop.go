package interpreter

import (
	"context"
	"fmt"
	"time"

	"github.com/atinylittleshell/gsh/internal/acp"
)

// DefaultMaxIterations is the default maximum number of tool call iterations
// if not specified in the agent config.
const DefaultMaxIterations = 100

// ExecuteAgent executes an agent with a conversation and returns the updated conversation.
// streaming parameter enables streaming responses.
// SDK events (agent.start, agent.end, etc.) are emitted via the interpreter's event manager.
// The user message is inferred from the conversation.
func (i *Interpreter) ExecuteAgent(
	ctx context.Context,
	conv *ConversationValue,
	agent *AgentValue,
	streaming bool,
) (Value, error) {
	return i.executeAgentInternal(ctx, conv, agent, streaming, nil)
}

// ExecuteAgentWithCallbacks executes an agent with callbacks and returns the updated conversation.
// streaming parameter enables streaming responses.
// callbacks can be used to hook into agent execution events (tool execution, streaming chunks, etc.).
// SDK events (agent.start, agent.end, etc.) are emitted via the interpreter's event manager regardless of callbacks.
// The user message is inferred from the conversation.
func (i *Interpreter) ExecuteAgentWithCallbacks(
	ctx context.Context,
	conv *ConversationValue,
	agent *AgentValue,
	streaming bool,
	callbacks *AgentCallbacks,
) (Value, error) {
	return i.executeAgentInternal(ctx, conv, agent, streaming, callbacks)
}

// executeAgentInternal is the core agentic loop implementation.
// It supports both simple execution (no callbacks) and REPL execution (with callbacks).
// Events are always emitted via the interpreter's EmitEvent method.
func (i *Interpreter) executeAgentInternal(ctx context.Context, conv *ConversationValue, agent *AgentValue, streaming bool, callbacks *AgentCallbacks) (Value, error) {
	startTime := time.Now()

	// Track token usage across all iterations
	var totalInputTokens, totalOutputTokens, totalCachedTokens int

	// Helper to emit events via the interpreter's event manager
	emitEvent := func(eventName string, eventCtx Value) {
		i.EmitEvent(eventName, eventCtx)
	}

	// Get user message for events from conversation (find the last user message)
	userMessage := ""
	if len(conv.Messages) > 0 {
		// Find the last user message in conversation
		for j := len(conv.Messages) - 1; j >= 0; j-- {
			if conv.Messages[j].Role == "user" {
				userMessage = conv.Messages[j].Content
				break
			}
		}
	}

	// Emit agent.start event
	emitEvent(EventAgentStart, createAgentStartContext(agent.Name, userMessage))

	// Helper to call OnComplete callback with ACP-aligned result
	callOnComplete := func(stopReason acp.StopReason, err error) {
		// Emit agent.end event first
		durationMs := time.Since(startTime).Milliseconds()
		emitEvent(EventAgentEnd, createAgentEndContext(agent.Name, string(stopReason), durationMs, totalInputTokens, totalOutputTokens, totalCachedTokens, err))

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
	useStreaming := streaming

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
					if callbacks != nil && callbacks.OnChunk != nil {
						callbacks.OnChunk(content)
					}
				},
			}
			if callbacks != nil && callbacks.OnToolCallStreaming != nil {
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

			// Execute the tool
			toolResult, toolErr = i.executeToolCall(agent, toolCall)

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
