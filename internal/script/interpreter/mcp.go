package interpreter

import (
	"encoding/json"
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// MCPProxyValue represents a proxy object for an MCP server
// It allows calling tools via member expressions (e.g., filesystem.read_file)
type MCPProxyValue struct {
	ServerName string
	Manager    *mcp.Manager
}

func (m *MCPProxyValue) Type() ValueType { return ValueTypeObject }
func (m *MCPProxyValue) String() string {
	return fmt.Sprintf("<mcp server: %s>", m.ServerName)
}
func (m *MCPProxyValue) IsTruthy() bool { return true }
func (m *MCPProxyValue) Equals(other Value) bool {
	if otherMcp, ok := other.(*MCPProxyValue); ok {
		return m.ServerName == otherMcp.ServerName
	}
	return false
}

// GetProperty returns a tool from this MCP server
func (m *MCPProxyValue) GetProperty(name string) (Value, error) {
	// Check if the tool exists
	tool, err := m.Manager.GetTool(m.ServerName, name)
	if err != nil {
		return nil, err
	}

	// Return an MCP tool wrapper
	return &MCPToolValue{
		ServerName: m.ServerName,
		ToolName:   tool.Name,
		Manager:    m.Manager,
	}, nil
}

// MCPToolValue represents a specific MCP tool that can be called
type MCPToolValue struct {
	ServerName string
	ToolName   string
	Manager    *mcp.Manager
}

func (m *MCPToolValue) Type() ValueType { return ValueTypeTool }
func (m *MCPToolValue) String() string {
	return fmt.Sprintf("<mcp tool: %s.%s>", m.ServerName, m.ToolName)
}
func (m *MCPToolValue) IsTruthy() bool { return true }
func (m *MCPToolValue) Equals(other Value) bool {
	if otherTool, ok := other.(*MCPToolValue); ok {
		return m.ServerName == otherTool.ServerName && m.ToolName == otherTool.ToolName
	}
	return false
}

// Call invokes the MCP tool with the given arguments
func (m *MCPToolValue) Call(args map[string]interface{}) (Value, error) {
	result, err := m.Manager.CallTool(m.ServerName, m.ToolName, args)
	if err != nil {
		return nil, err
	}

	// Convert MCP result to Value
	return mcpResultToValue(result)
}

// evalMcpDeclaration evaluates an MCP server declaration
func (i *Interpreter) evalMcpDeclaration(node *parser.McpDeclaration) (Value, error) {
	serverName := node.Name.Value

	// Build the server config from the declaration
	config := mcp.ServerConfig{}

	// Evaluate each config field
	for key, expr := range node.Config {
		value, err := i.evalExpression(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate MCP config field '%s': %w", key, err)
		}

		switch key {
		case "command":
			if strVal, ok := value.(*StringValue); ok {
				config.Command = strVal.Value
			} else {
				return nil, fmt.Errorf("MCP config 'command' must be a string, got %s", value.Type())
			}

		case "args":
			if arrVal, ok := value.(*ArrayValue); ok {
				args := make([]string, len(arrVal.Elements))
				for i, elem := range arrVal.Elements {
					if strElem, ok := elem.(*StringValue); ok {
						args[i] = strElem.Value
					} else {
						return nil, fmt.Errorf("MCP config 'args' must be an array of strings, got element of type %s", elem.Type())
					}
				}
				config.Args = args
			} else {
				return nil, fmt.Errorf("MCP config 'args' must be an array, got %s", value.Type())
			}

		case "env":
			if objVal, ok := value.(*ObjectValue); ok {
				env := make(map[string]string)
				for k, v := range objVal.Properties {
					if strVal, ok := v.(*StringValue); ok {
						env[k] = strVal.Value
					} else {
						return nil, fmt.Errorf("MCP config 'env' values must be strings, got %s for key '%s'", v.Type(), k)
					}
				}
				config.Env = env
			} else {
				return nil, fmt.Errorf("MCP config 'env' must be an object, got %s", value.Type())
			}

		case "url":
			if strVal, ok := value.(*StringValue); ok {
				config.URL = strVal.Value
			} else {
				return nil, fmt.Errorf("MCP config 'url' must be a string, got %s", value.Type())
			}

		case "headers":
			if objVal, ok := value.(*ObjectValue); ok {
				headers := make(map[string]string)
				for k, v := range objVal.Properties {
					if strVal, ok := v.(*StringValue); ok {
						headers[k] = strVal.Value
					} else {
						return nil, fmt.Errorf("MCP config 'headers' values must be strings, got %s for key '%s'", v.Type(), k)
					}
				}
				config.Headers = headers
			} else {
				return nil, fmt.Errorf("MCP config 'headers' must be an object, got %s", value.Type())
			}

		default:
			return nil, fmt.Errorf("unknown MCP config field: '%s'", key)
		}
	}

	// Register the server with the MCP manager
	err := i.mcpManager.RegisterServer(serverName, config)
	if err != nil {
		return nil, fmt.Errorf("failed to register MCP server '%s': %w", serverName, err)
	}

	// Create a proxy object for the server
	proxy := &MCPProxyValue{
		ServerName: serverName,
		Manager:    i.mcpManager,
	}

	// Register the proxy in the environment
	err = i.env.Define(serverName, proxy)
	if err != nil {
		return nil, err
	}

	return proxy, nil
}

// mcpResultToValue converts an MCP tool result to a Value
func mcpResultToValue(result interface{}) (Value, error) {
	if result == nil {
		return &NullValue{}, nil
	}

	// Marshal to JSON and unmarshal to get a clean structure
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert MCP result to JSON: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse MCP result JSON: %w", err)
	}

	// Use the existing jsonToValue function from builtin.go
	return jsonToValue(data), nil
}

// callMCPTool calls an MCP tool with the given arguments
func (i *Interpreter) callMCPTool(tool *MCPToolValue, argExprs []parser.Expression) (Value, error) {
	// Evaluate all arguments
	args := make(map[string]interface{})

	// MCP tools can take arguments in two ways:
	// 1. Single object argument: tool({key: value, ...})
	// 2. Multiple positional arguments that get mapped to tool parameters

	if len(argExprs) == 0 {
		// No arguments - call with empty args
		return tool.Call(args)
	}

	// Evaluate first argument
	firstArg, err := i.evalExpression(argExprs[0])
	if err != nil {
		return nil, err
	}

	// If single object argument, use it as the arguments map
	if len(argExprs) == 1 {
		if objVal, ok := firstArg.(*ObjectValue); ok {
			// Convert object properties to map[string]interface{}
			for key, val := range objVal.Properties {
				args[key] = valueToInterface(val)
			}
			return tool.Call(args)
		}
	}

	// Otherwise, for positional arguments, we need the tool schema to map them
	// For now, we'll treat single non-object arguments as errors
	if len(argExprs) == 1 {
		// Single non-object argument - try to use it as a single parameter
		// This is a simplified approach; real implementation would need tool schema
		args["value"] = valueToInterface(firstArg)
		return tool.Call(args)
	}

	return nil, fmt.Errorf("MCP tool calls require either a single object argument or proper parameter mapping")
}

// valueToInterface converts a Value to interface{} for MCP calls
func valueToInterface(val Value) interface{} {
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
			arr[i] = valueToInterface(elem)
		}
		return arr
	case *ObjectValue:
		obj := make(map[string]interface{})
		for key, prop := range v.Properties {
			obj[key] = valueToInterface(prop)
		}
		return obj
	default:
		return v.String()
	}
}
