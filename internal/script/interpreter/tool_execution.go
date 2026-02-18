package interpreter

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/acp"
)

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

	// Call the tool - use globalEnv since agent tool calls create their own scope from tool.Env
	result, err := i.CallTool(i.globalEnv, tool, valueArgs)
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
			paramType = mapGSHTypeToJSONType(typeName)
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
func mapGSHTypeToJSONType(gshType string) string {
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
