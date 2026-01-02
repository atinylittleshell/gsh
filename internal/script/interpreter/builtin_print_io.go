package interpreter

import (
	"bufio"
	"fmt"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"strings"
)

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
