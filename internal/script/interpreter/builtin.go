package interpreter

import (
	"encoding/json"
	"fmt"
	"os"
)

// builtinNames contains all the names of built-in functions and objects
var builtinNames = map[string]bool{
	"print": true,
	"JSON":  true,
	"log":   true,
	"env":   true,
	"Map":   true,
	"Set":   true,
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
	i.env.Set("print", &BuiltinValue{
		Name: "print",
		Fn:   builtinPrint,
	})

	// Register JSON object with parse and stringify methods
	jsonObj := &ObjectValue{
		Properties: map[string]Value{
			"parse": &BuiltinValue{
				Name: "JSON.parse",
				Fn:   builtinJSONParse,
			},
			"stringify": &BuiltinValue{
				Name: "JSON.stringify",
				Fn:   builtinJSONStringify,
			},
		},
	}
	i.env.Set("JSON", jsonObj)

	// Register log object with debug, info, warn, error methods
	logObj := &ObjectValue{
		Properties: map[string]Value{
			"debug": &BuiltinValue{
				Name: "log.debug",
				Fn:   builtinLogDebug,
			},
			"info": &BuiltinValue{
				Name: "log.info",
				Fn:   builtinLogInfo,
			},
			"warn": &BuiltinValue{
				Name: "log.warn",
				Fn:   builtinLogWarn,
			},
			"error": &BuiltinValue{
				Name: "log.error",
				Fn:   builtinLogError,
			},
		},
	}
	i.env.Set("log", logObj)

	// Register env object for environment variable access
	i.env.Set("env", &EnvValue{})

	// Register Map constructor
	i.env.Set("Map", &BuiltinValue{
		Name: "Map",
		Fn:   builtinMap,
	})

	// Register Set constructor
	i.env.Set("Set", &BuiltinValue{
		Name: "Set",
		Fn:   builtinSet,
	})
}

// builtinPrint implements the print() function
// Outputs to stdout for user-facing messages
func builtinPrint(args []Value) (Value, error) {
	for i, arg := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(arg.String())
	}
	fmt.Println()
	return &NullValue{}, nil
}

// builtinLogDebug implements log.debug()
// Currently outputs to stderr with [DEBUG] prefix
// TODO: Integrate with zap logger when interpreter is integrated with main gsh app
func builtinLogDebug(args []Value) (Value, error) {
	fmt.Fprint(os.Stderr, "[DEBUG] ")
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(os.Stderr, " ")
		}
		fmt.Fprint(os.Stderr, arg.String())
	}
	fmt.Fprintln(os.Stderr)
	return &NullValue{}, nil
}

// builtinLogInfo implements log.info()
// Currently outputs to stderr with [INFO] prefix
// TODO: Integrate with zap logger when interpreter is integrated with main gsh app
func builtinLogInfo(args []Value) (Value, error) {
	fmt.Fprint(os.Stderr, "[INFO] ")
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(os.Stderr, " ")
		}
		fmt.Fprint(os.Stderr, arg.String())
	}
	fmt.Fprintln(os.Stderr)
	return &NullValue{}, nil
}

// builtinLogWarn implements log.warn()
// Currently outputs to stderr with [WARN] prefix
// TODO: Integrate with zap logger when interpreter is integrated with main gsh app
func builtinLogWarn(args []Value) (Value, error) {
	fmt.Fprint(os.Stderr, "[WARN] ")
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(os.Stderr, " ")
		}
		fmt.Fprint(os.Stderr, arg.String())
	}
	fmt.Fprintln(os.Stderr)
	return &NullValue{}, nil
}

// builtinLogError implements log.error()
// Currently outputs to stderr with [ERROR] prefix
// TODO: Integrate with zap logger when interpreter is integrated with main gsh app
func builtinLogError(args []Value) (Value, error) {
	fmt.Fprint(os.Stderr, "[ERROR] ")
	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(os.Stderr, " ")
		}
		fmt.Fprint(os.Stderr, arg.String())
	}
	fmt.Fprintln(os.Stderr)
	return &NullValue{}, nil
}

// builtinJSONParse implements JSON.parse()
func builtinJSONParse(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("JSON.parse expects 1 argument, got %d", len(args))
	}

	str, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("JSON.parse expects a string argument, got %s", args[0].Type())
	}

	var result interface{}
	if err := json.Unmarshal([]byte(str.Value), &result); err != nil {
		return nil, fmt.Errorf("JSON.parse error: %v", err)
	}

	return jsonToValue(result), nil
}

// builtinJSONStringify implements JSON.stringify()
func builtinJSONStringify(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("JSON.stringify expects 1 argument, got %d", len(args))
	}

	jsonValue := valueToJSON(args[0])
	bytes, err := json.Marshal(jsonValue)
	if err != nil {
		return nil, fmt.Errorf("JSON.stringify error: %v", err)
	}

	return &StringValue{Value: string(bytes)}, nil
}

// jsonToValue converts a Go interface{} from json.Unmarshal to a Value
func jsonToValue(v interface{}) Value {
	if v == nil {
		return &NullValue{}
	}

	switch val := v.(type) {
	case bool:
		return &BoolValue{Value: val}
	case float64:
		return &NumberValue{Value: val}
	case string:
		return &StringValue{Value: val}
	case []interface{}:
		elements := make([]Value, len(val))
		for i, elem := range val {
			elements[i] = jsonToValue(elem)
		}
		return &ArrayValue{Elements: elements}
	case map[string]interface{}:
		properties := make(map[string]Value)
		for key, value := range val {
			properties[key] = jsonToValue(value)
		}
		return &ObjectValue{Properties: properties}
	default:
		return &NullValue{}
	}
}

// valueToJSON converts a Value to a Go interface{} for json.Marshal
func valueToJSON(v Value) interface{} {
	switch val := v.(type) {
	case *NullValue:
		return nil
	case *BoolValue:
		return val.Value
	case *NumberValue:
		return val.Value
	case *StringValue:
		return val.Value
	case *ArrayValue:
		result := make([]interface{}, len(val.Elements))
		for i, elem := range val.Elements {
			result[i] = valueToJSON(elem)
		}
		return result
	case *ObjectValue:
		result := make(map[string]interface{})
		for key, value := range val.Properties {
			result[key] = valueToJSON(value)
		}
		return result
	default:
		return nil
	}
}

// EnvValue represents the env object for environment variable access
type EnvValue struct{}

func (e *EnvValue) Type() ValueType { return ValueTypeObject }
func (e *EnvValue) String() string  { return "<env>" }
func (e *EnvValue) IsTruthy() bool  { return true }
func (e *EnvValue) Equals(other Value) bool {
	_, ok := other.(*EnvValue)
	return ok
}

// GetProperty gets an environment variable by name
func (e *EnvValue) GetProperty(name string) Value {
	value := os.Getenv(name)
	if value == "" {
		// Return null if environment variable is not set
		return &NullValue{}
	}
	return &StringValue{Value: value}
}

// builtinMap implements the Map() constructor
// Map() creates an empty map
// Map([[key1, val1], [key2, val2]]) creates a map from array of key-value pairs
func builtinMap(args []Value) (Value, error) {
	if len(args) == 0 {
		// Empty map
		return &MapValue{Entries: make(map[string]Value)}, nil
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("Map() takes 0 or 1 arguments, got %d", len(args))
	}

	// Expect an array of [key, value] pairs
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("Map() argument must be an array of [key, value] pairs")
	}

	entries := make(map[string]Value)
	for i, elem := range arr.Elements {
		pair, ok := elem.(*ArrayValue)
		if !ok || len(pair.Elements) != 2 {
			return nil, fmt.Errorf("Map() entry %d must be a [key, value] pair", i)
		}

		// Key must be a string
		key, ok := pair.Elements[0].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("Map() entry %d key must be a string", i)
		}

		entries[key.Value] = pair.Elements[1]
	}

	return &MapValue{Entries: entries}, nil
}

// builtinSet implements the Set() constructor
// Set() creates an empty set
// Set([val1, val2, val3]) creates a set from array of values
func builtinSet(args []Value) (Value, error) {
	if len(args) == 0 {
		// Empty set
		return &SetValue{Elements: make(map[string]Value)}, nil
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("Set() takes 0 or 1 arguments, got %d", len(args))
	}

	// Expect an array of values
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("Set() argument must be an array of values")
	}

	elements := make(map[string]Value)
	for _, elem := range arr.Elements {
		// Use string representation as key for uniqueness
		key := elem.String()
		elements[key] = elem
	}

	return &SetValue{Elements: elements}, nil
}
