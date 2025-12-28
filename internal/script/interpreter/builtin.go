package interpreter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/bash"
	"go.uber.org/zap/zapcore"
)

// builtinNames contains all the names of built-in functions and objects
var builtinNames = map[string]bool{
	"print": true,
	"input": true,
	"JSON":  true,
	"log":   true,
	"env":   true,
	"Map":   true,
	"Set":   true,
	"exec":  true,
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
	// These methods use the interpreter's zap logger if available
	logObj := &ObjectValue{
		Properties: map[string]Value{
			"debug": &BuiltinValue{
				Name: "log.debug",
				Fn:   i.makeLogFunc(zapcore.DebugLevel, "DEBUG"),
			},
			"info": &BuiltinValue{
				Name: "log.info",
				Fn:   i.makeLogFunc(zapcore.InfoLevel, "INFO"),
			},
			"warn": &BuiltinValue{
				Name: "log.warn",
				Fn:   i.makeLogFunc(zapcore.WarnLevel, "WARN"),
			},
			"error": &BuiltinValue{
				Name: "log.error",
				Fn:   i.makeLogFunc(zapcore.ErrorLevel, "ERROR"),
			},
		},
	}
	i.env.Set("log", logObj)

	// Register env object for environment variable access
	i.env.Set("env", &EnvValue{interp: i})

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

	// Register exec function for executing shell commands
	i.env.Set("exec", &BuiltinValue{
		Name: "exec",
		Fn:   i.builtinExec,
	})

	// Register input function for reading user input
	i.env.Set("input", &BuiltinValue{
		Name: "input",
		Fn:   i.builtinInput,
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

// builtinInput implements the input() function
// Reads a line from stdin and returns it as a string
// Optional prompt argument is printed to stdout before reading
func (i *Interpreter) builtinInput(args []Value) (Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("input() takes 0 or 1 argument (prompt?: string), got %d", len(args))
	}

	// If a prompt is provided, print it without newline
	if len(args) == 1 {
		promptValue, ok := args[0].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("input() prompt must be a string, got %s", args[0].Type())
		}
		fmt.Print(promptValue.Value)
	}

	// Read a line from stdin
	reader := bufio.NewReader(i.stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("input() failed to read: %w", err)
	}

	// Trim the trailing newline (handle both \n and \r\n)
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")

	return &StringValue{Value: line}, nil
}

// makeLogFunc creates a log function that uses the zap logger if available,
// otherwise falls back to stderr output with the given prefix.
func (i *Interpreter) makeLogFunc(level zapcore.Level, prefix string) BuiltinFunction {
	return func(args []Value) (Value, error) {
		// Build the message from all arguments
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		message := strings.Join(parts, " ")

		// Use zap logger if available, otherwise fall back to stderr
		if i.logger != nil {
			switch level {
			case zapcore.DebugLevel:
				i.logger.Debug(message)
			case zapcore.InfoLevel:
				i.logger.Info(message)
			case zapcore.WarnLevel:
				i.logger.Warn(message)
			case zapcore.ErrorLevel:
				i.logger.Error(message)
			}
		} else {
			// Fallback: output to stderr with prefix
			fmt.Fprintf(os.Stderr, "[%s] %s\n", prefix, message)
		}

		return &NullValue{}, nil
	}
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
// It holds a reference to the interpreter to access the shared sh runner
type EnvValue struct {
	interp *Interpreter
}

func (e *EnvValue) Type() ValueType { return ValueTypeObject }
func (e *EnvValue) String() string  { return "<env>" }
func (e *EnvValue) IsTruthy() bool  { return true }
func (e *EnvValue) Equals(other Value) bool {
	_, ok := other.(*EnvValue)
	return ok
}

// GetProperty gets an environment variable from the sh runner
func (e *EnvValue) GetProperty(name string) Value {
	value := e.interp.GetEnv(name)
	if value == "" {
		// Return null if environment variable is not set
		return &NullValue{}
	}
	return &StringValue{Value: value}
}

// SetProperty sets an environment variable in the sh runner
func (e *EnvValue) SetProperty(name string, value Value) error {
	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case *StringValue:
		strValue = v.Value
	case *NumberValue:
		strValue = fmt.Sprintf("%v", v.Value)
	case *BoolValue:
		if v.Value {
			strValue = "true"
		} else {
			strValue = "false"
		}
	case *NullValue:
		// Setting to null unsets the variable
		e.interp.UnsetEnv(name)
		return nil
	default:
		strValue = v.String()
	}

	e.interp.SetEnv(name, strValue)
	return nil
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

// builtinExec implements the exec() function for executing shell commands
// exec(command: string, options?: {timeout?: number}): {stdout: string, stderr: string, exitCode: number}
func (i *Interpreter) builtinExec(args []Value) (Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("exec() takes 1 or 2 arguments (command: string, options?: object), got %d", len(args))
	}

	// First argument: command (string)
	cmdValue, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("exec() first argument must be a string, got %s", args[0].Type())
	}
	command := cmdValue.Value

	// Second argument (optional): options object
	timeout := 60 * time.Second // Default timeout
	if len(args) == 2 {
		optsValue, ok := args[1].(*ObjectValue)
		if !ok {
			return nil, fmt.Errorf("exec() second argument must be an object, got %s", args[1].Type())
		}

		// Parse timeout option if provided
		if timeoutVal, ok := optsValue.Properties["timeout"]; ok {
			if timeoutNum, ok := timeoutVal.(*NumberValue); ok {
				timeout = time.Duration(timeoutNum.Value) * time.Millisecond
			} else {
				return nil, fmt.Errorf("exec() options.timeout must be a number (milliseconds), got %s", timeoutVal.Type())
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute the command in a subshell
	stdout, stderr, exitCode, err := i.executeBashInSubshell(ctx, command)

	// Check for context timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("exec() command timed out after %v", timeout)
	}

	// If there's an execution error (not just non-zero exit code), return it
	if err != nil {
		return nil, fmt.Errorf("exec() failed: %w", err)
	}

	// Return result as an object with stdout, stderr, and exitCode
	result := &ObjectValue{
		Properties: map[string]Value{
			"stdout":   &StringValue{Value: stdout},
			"stderr":   &StringValue{Value: stderr},
			"exitCode": &NumberValue{Value: float64(exitCode)},
		},
	}

	return result, nil
}

// executeBashInSubshell executes a bash command in a subshell and returns stdout, stderr, and exit code
// It uses a subshell clone of the interpreter's runner to inherit env vars and working directory
func (i *Interpreter) executeBashInSubshell(ctx context.Context, command string) (string, string, int, error) {
	i.runnerMu.RLock()
	runner := i.runner
	i.runnerMu.RUnlock()

	return bash.RunBashCommandInSubShellWithExitCode(ctx, runner, command)
}
