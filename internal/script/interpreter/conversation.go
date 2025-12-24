package interpreter

import (
	"encoding/json"
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/parser"
)

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

// executeAgent executes an agent with a conversation and returns the updated conversation
func (i *Interpreter) executeAgent(conv *ConversationValue, agent *AgentValue) (Value, error) {
	// Get model from agent config
	modelVal, ok := agent.Config["model"]
	if !ok {
		return nil, fmt.Errorf("agent '%s' has no model configured", agent.Name)
	}
	model, ok := modelVal.(*ModelValue)
	if !ok {
		return nil, fmt.Errorf("agent '%s' model config is not a model", agent.Name)
	}

	// Prepare tools for the agent
	tools := []ChatTool{}
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
						return nil, fmt.Errorf("failed to convert MCP tool: %w", err)
					}
					tools = append(tools, tool)
				default:
					return nil, fmt.Errorf("invalid tool type in agent config: %s", toolVal.Type())
				}
			}
		}
	}

	// Create chat request
	request := ChatRequest{
		Model:    model,
		Messages: conv.Messages,
		Tools:    tools,
	}

	// Call the model directly (provider is resolved at model creation time)
	response, err := model.ChatCompletion(request)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
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

	// Handle tool calls if present
	if len(response.ToolCalls) > 0 {
		// Add assistant message with tool calls
		newConv.Messages = append(newConv.Messages, ChatMessage{
			Role:    "assistant",
			Content: response.Content,
		})

		// Execute tool calls
		for _, toolCall := range response.ToolCalls {
			toolResult, err := i.executeToolCall(agent, toolCall)
			if err != nil {
				return nil, fmt.Errorf("tool call failed: %w", err)
			}

			// Add tool result to conversation
			newConv.Messages = append(newConv.Messages, ChatMessage{
				Role:    "tool",
				Content: toolResult,
				Name:    toolCall.Name,
			})
		}

		// Make another call to get final response after tool execution
		request.Messages = newConv.Messages
		response, err = model.ChatCompletion(request)
		if err != nil {
			return nil, fmt.Errorf("agent execution after tool calls failed: %w", err)
		}
	}

	// Add assistant response to conversation
	newConv.Messages = append(newConv.Messages, ChatMessage{
		Role:    "assistant",
		Content: response.Content,
	})

	return newConv, nil
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
		properties := make(map[string]Value)
		for key, val := range v {
			gshVal, err := i.jsonToValue(val)
			if err != nil {
				return nil, err
			}
			properties[key] = gshVal
		}
		return &ObjectValue{Properties: properties}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", jsonVal)
	}
}

// valueToJSON converts a GSH Value to a JSON string
func (i *Interpreter) valueToJSON(val Value) (string, error) {
	switch v := val.(type) {
	case *NullValue:
		return "null", nil
	case *BoolValue:
		if v.Value {
			return "true", nil
		}
		return "false", nil
	case *NumberValue:
		return v.String(), nil
	case *StringValue:
		return fmt.Sprintf(`"%s"`, v.Value), nil
	case *ArrayValue:
		jsonBytes, err := json.Marshal(i.valueArrayToInterface(v.Elements))
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	case *ObjectValue:
		jsonBytes, err := json.Marshal(i.valueMapToInterface(v.Properties))
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	default:
		return val.String(), nil
	}
}

// valueArrayToInterface converts []Value to []interface{}
func (i *Interpreter) valueArrayToInterface(values []Value) []interface{} {
	result := make([]interface{}, len(values))
	for idx, val := range values {
		result[idx] = i.valueToInterface(val)
	}
	return result
}

// valueMapToInterface converts map[string]Value to map[string]interface{}
func (i *Interpreter) valueMapToInterface(values map[string]Value) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range values {
		result[key] = i.valueToInterface(val)
	}
	return result
}

// valueToInterface converts a Value to interface{}
func (i *Interpreter) valueToInterface(val Value) interface{} {
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
		return i.valueArrayToInterface(v.Elements)
	case *ObjectValue:
		return i.valueMapToInterface(v.Properties)
	default:
		return val.String()
	}
}
