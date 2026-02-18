package interpreter

import (
	"fmt"
	"go.uber.org/zap/zapcore"
)

// builtinNames contains all the names of built-in functions and objects
var builtinNames = map[string]bool{
	"print":    true,
	"input":    true,
	"JSON":     true,
	"log":      true,
	"env":      true,
	"Map":      true,
	"Set":      true,
	"exec":     true,
	"gsh":      true,
	"Math":     true,
	"DateTime": true,
	"Regexp":   true,
	"typeof":   true,
}

// isBuiltin checks if a name is a built-in function or object
func isBuiltin(name string) bool {
	return builtinNames[name]
}

// BuiltinFunction represents a built-in function
type BuiltinFunction func(args []Value) (Value, error)

// BuiltinValue represents a built-in function value
type BuiltinValue struct {
	Name string
	Fn   BuiltinFunction
}

func (b *BuiltinValue) Type() ValueType { return ValueTypeTool }
func (b *BuiltinValue) String() string {
	return fmt.Sprintf("<builtin %s>", b.Name)
}
func (b *BuiltinValue) IsTruthy() bool { return true }
func (b *BuiltinValue) Equals(other Value) bool {
	if otherBuiltin, ok := other.(*BuiltinValue); ok {
		return b.Name == otherBuiltin.Name
	}
	return false
}

// registerBuiltins registers all built-in functions and objects in the environment
func (i *Interpreter) registerBuiltins() {
	// Register print function
	i.globalEnv.Set("print", &BuiltinValue{
		Name: "print",
		Fn:   builtinPrint,
	})

	// Register JSON object with parse and stringify methods
	jsonObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"parse": {Value: &BuiltinValue{
				Name: "JSON.parse",
				Fn:   builtinJSONParse,
			}},
			"stringify": {Value: &BuiltinValue{
				Name: "JSON.stringify",
				Fn:   builtinJSONStringify,
			}},
		},
	}
	i.globalEnv.Set("JSON", jsonObj)

	// Register log object with debug, info, warn, error methods
	// These methods use the interpreter's zap logger if available
	logObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"debug": {Value: &BuiltinValue{
				Name: "log.debug",
				Fn:   i.makeLogFunc(zapcore.DebugLevel, "DEBUG"),
			}},
			"info": {Value: &BuiltinValue{
				Name: "log.info",
				Fn:   i.makeLogFunc(zapcore.InfoLevel, "INFO"),
			}},
			"warn": {Value: &BuiltinValue{
				Name: "log.warn",
				Fn:   i.makeLogFunc(zapcore.WarnLevel, "WARN"),
			}},
			"error": {Value: &BuiltinValue{
				Name: "log.error",
				Fn:   i.makeLogFunc(zapcore.ErrorLevel, "ERROR"),
			}},
		},
	}
	i.globalEnv.Set("log", logObj)

	// Register env object for environment variable access
	i.globalEnv.Set("env", &EnvValue{interp: i})

	// Register Map constructor
	i.globalEnv.Set("Map", &BuiltinValue{
		Name: "Map",
		Fn:   builtinMap,
	})

	// Register Set constructor
	i.globalEnv.Set("Set", &BuiltinValue{
		Name: "Set",
		Fn:   builtinSet,
	})

	// Register exec function for executing shell commands
	i.globalEnv.Set("exec", &BuiltinValue{
		Name: "exec",
		Fn:   i.builtinExec,
	})

	// Register input function for reading user input
	i.globalEnv.Set("input", &BuiltinValue{
		Name: "input",
		Fn:   i.builtinInput,
	})

	// Register typeof function for runtime type inspection
	i.globalEnv.Set("typeof", &BuiltinValue{
		Name: "typeof",
		Fn:   builtinTypeof,
	})
}

// builtinTypeof returns the type of a value as a string
func builtinTypeof(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("typeof() takes exactly 1 argument, got %d", len(args))
	}

	val := args[0]
	var typeName string

	switch val.(type) {
	case *StringValue:
		typeName = "string"
	case *NumberValue:
		typeName = "number"
	case *BoolValue:
		typeName = "boolean"
	case *NullValue:
		typeName = "null"
	case *ArrayValue:
		typeName = "array"
	case *ObjectValue:
		typeName = "object"
	case *MapValue:
		typeName = "map"
	case *SetValue:
		typeName = "set"
	case *ToolValue:
		typeName = "tool"
	case *BuiltinValue:
		typeName = "function"
	case *ModelValue:
		typeName = "model"
	case *AgentValue:
		typeName = "agent"
	case *MCPProxyValue:
		typeName = "mcp"
	case *MCPToolValue:
		typeName = "mcp_tool"
	case *ConversationValue:
		typeName = "conversation"
	default:
		typeName = "unknown"
	}

	return &StringValue{Value: typeName}, nil
}
