package bash

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Global variable to track if exit-on-error is enabled (like bash 'set -e')
var exitOnError bool = false

// SetBuiltinHandler handles the 'set' builtin command, supporting '-e' option
func SetBuiltinHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Check if this is the 'set' command
			if args[0] != "set" {
				return next(ctx, args)
			}

			// Handle 'set -e' and 'set +e'
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "-e":
					exitOnError = true
				case "+e":
					exitOnError = false
				}
			}

			return nil
		}
	}
}

// ShouldExitOnError returns true if exit-on-error is currently enabled
func ShouldExitOnError() bool {
	return exitOnError
}

func PreprocessTypesetCommands(input string) string {
	// Handle edge cases
	if input == "" {
		return ""
	}

	// Protect against extremely large inputs (potential DoS)
	const maxInputSize = 10 * 1024 * 1024 // 10MB limit
	if len(input) > maxInputSize {
		// For large inputs, only process the first portion to prevent memory issues
		// This is a safety measure against potential resource exhaustion attacks
		input = input[:maxInputSize]
	}

	// Use parsing-based approach to avoid false positives
	return preprocessWithParsing(input)
}

// preprocessWithParsing uses a state machine to parse bash syntax and only transform
// actual commands, avoiding false positives in strings, comments, and other contexts
func preprocessWithParsing(input string) string {
	var result strings.Builder
	result.Grow(len(input) + 100) // Pre-allocate with some extra space

	// Parsing states
	type ParseState int
	const (
		StateNormal ParseState = iota
		StateSingleQuote
		StateDoubleQuote
		StateComment
		StateHeredoc
		StateCommandSubstitution
		StateArray
	)

	state := StateNormal
	var delimiter string // Store the heredoc delimiter
	parenDepth := 0      // Track parentheses nesting for command substitution and arrays

	// Helper function to check if we should transform at current position
	shouldTransformAt := func(pos int) bool {
		// Don't transform inside quotes, comments, heredocs, command substitution, or arrays
		if state != StateNormal {
			return false
		}

		// Don't transform if we're inside command substitution or arrays
		if parenDepth > 0 {
			return false
		}

		// Check if we're at a command position (start of line or after delimiter)
		if pos > 0 {
			prev := input[pos-1]
			// Simplified De Morgan's Law logic:
			// Original: !(prev == '\n' || ... || isWhitespace(prev))
			// Equivalent: prev != '\n' && ... && !isWhitespace(prev)
			if prev != '\n' && prev != ';' && prev != '|' && prev != '&' && prev != '(' && prev != '{' && !isWhitespace(prev) {
				return false
			}
		}

		// Look ahead for our target patterns
		remaining := input[pos:]

		// Check for "typeset " or "declare " with optional extra spaces
		if strings.HasPrefix(remaining, "typeset ") || strings.HasPrefix(remaining, "declare ") {
			return true
		}

		// Check for patterns with extra spaces
		if len(remaining) >= 8 {
			if (remaining[:7] == "typeset" || remaining[:7] == "declare") && isWhitespace(remaining[7]) {
				return true
			}
		}

		return false
	}

	// Helper function to transform command starting at position
	transformCommandAt := func(pos int) (string, int) {
		// Log for debugging (can be disabled in production)
		fmt.Printf("Transforming at pos %d: %.30s\n", pos, input[pos:])
		// Find the command name and flag
		start := pos
		for start < len(input) && isWhitespace(input[start]) {
			start++
		}

		// Extract command name
		cmdStart := start
		for start < len(input) && !isWhitespace(input[start]) {
			start++
		}
		cmdName := input[cmdStart:start]

		// Skip whitespace
		for start < len(input) && isWhitespace(input[start]) {
			start++
		}

		// Extract flag
		flagStart := start
		for start < len(input) && !isWhitespace(input[start]) && input[start] != '\n' {
			start++
		}
		fullFlag := input[flagStart:start]

		// Check if the flag starts with one of our target patterns
		// This handles cases like "-f", "-farg", "-F", "-p", etc.
		var targetFlag string
		if strings.HasPrefix(fullFlag, "-f") {
			targetFlag = "-f"
		} else if strings.HasPrefix(fullFlag, "-F") {
			targetFlag = "-F"
		} else if strings.HasPrefix(fullFlag, "-p") {
			targetFlag = "-p"
		}

		if (cmdName == "typeset" || cmdName == "declare") && targetFlag != "" {
			// Transform the command - only consume the flag part, not the rest
			return "gsh_typeset " + targetFlag, flagStart + len(targetFlag) - pos
		}

		// Not a target pattern, return original
		return input[pos:start], start - pos
	}

	// Main parsing loop
	i := 0
	for i < len(input) {
		ch := input[i]

		switch state {
		case StateNormal:
			// Check for comment start
			if ch == '#' {
				// Check if this is at line start or after whitespace
				if i == 0 || isWhitespace(input[i-1]) || input[i-1] == '\n' {
					state = StateComment
					result.WriteByte(ch)
					i++
					continue
				}
			}

			// Check for quote start
			if ch == '\'' {
				state = StateSingleQuote
				result.WriteByte(ch)
				i++
				continue
			}
			if ch == '"' {
				state = StateDoubleQuote
				result.WriteByte(ch)
				i++
				continue
			}

			// Check for command substitution start
			if ch == '$' && i+1 < len(input) && input[i+1] == '(' {
				state = StateCommandSubstitution
				parenDepth = 1
				result.WriteByte(ch)
				result.WriteByte(input[i+1])
				i += 2
				continue
			}

			// Check for heredoc start (<< followed by delimiter)
			if ch == '<' && i+1 < len(input) && input[i+1] == '<' {
				state = StateHeredoc
				result.WriteByte(ch)
				result.WriteByte(input[i+1])
				i += 2

				// Extract the delimiter - read until whitespace or newline
				// But don't consume the delimiter characters, just note where it starts
				delimiterStart := i; _ = delimiterStart
				tempI := i

				// Skip any leading whitespace in the delimiter
				for tempI < len(input) && isWhitespace(input[tempI]) && input[tempI] != '\n' {
					tempI++
				}

				// Handle <<- (indented heredoc) - skip the dash
				if tempI < len(input) && input[tempI] == '-' {
					tempI++
					// Skip more whitespace after dash
					for tempI < len(input) && isWhitespace(input[tempI]) && input[tempI] != '\n' {
						tempI++
					}
				}

				// Now extract the actual delimiter
				// Re-assigning to tempI is sufficient, delimiterStart was just a placeholder
				delimiterStart = tempI
				for tempI < len(input) && !isWhitespace(input[tempI]) && input[tempI] != '\n' {
					tempI++
				}
				if tempI > delimiterStart {
					// Store the delimiter for later matching
					delimiter = input[delimiterStart:tempI]
				}

				// Continue processing from current position (don't advance i)
				continue
			}

			// Track parentheses for command substitution and arrays, but not function definitions
			// Use tagged switch for better readability
			switch ch {
			case '(':
				// Only track parentheses if we're in command substitution or array state
				if state == StateCommandSubstitution || state == StateArray {
					parenDepth++
				} else if i > 0 && input[i-1] == '$' {
					// This is command substitution: $(
					state = StateCommandSubstitution
					parenDepth = 1
				} else if i > 0 && input[i-1] == '=' {
					// This might be array assignment: VAR=(
					// For now, assume any ( after = is an array
					state = StateArray
					parenDepth = 1
				}
			case ')':
				if parenDepth > 0 && (state == StateCommandSubstitution || state == StateArray) {
					parenDepth--
					if parenDepth == 0 {
						// Exit the special state
						state = StateNormal
					}
				}
			}

			// Check for potential command transformation
			if shouldTransformAt(i) {
				transformed, consumed := transformCommandAt(i)
				result.WriteString(transformed)
				i += consumed
				continue
			}

			result.WriteByte(ch)
			i++

		case StateSingleQuote:
			result.WriteByte(ch)
			if ch == '\'' {
				state = StateNormal
			}
			i++

		case StateDoubleQuote:
			result.WriteByte(ch)
			if ch == '"' {
				state = StateNormal
			}
			i++

		case StateComment:
			result.WriteByte(ch)
			if ch == '\n' {
				state = StateNormal
			}
			i++

		case StateCommandSubstitution:
			// In command substitution, track parentheses nesting
			result.WriteByte(ch)
			switch ch {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
				if parenDepth == 0 {
					// Exit command substitution state
					state = StateNormal
				}
			}
			i++

		case StateArray:
			// In array, track parentheses nesting
			result.WriteByte(ch)
			switch ch {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
				if parenDepth == 0 {
					// Exit array state
					state = StateNormal
				}
			}
			i++

		case StateHeredoc:
			// In heredoc, copy everything and check for delimiter match
			result.WriteByte(ch)

			if ch == '\n' {
				if delimiter != "" {
					remaining := input[i+1:]

					// Check if the next line starts with our delimiter at the beginning
					lines := strings.Split(remaining, "\n")
					if len(lines) > 0 {
						nextLine := lines[0]

						if nextLine == delimiter {
							// This is our delimiter line - exit heredoc state
							state = StateNormal
							delimiter = "" // Reset delimiter
						} else if strings.HasPrefix(nextLine, delimiter) {
							// Check if it's followed by whitespace or end of line
							afterDelimiter := len(delimiter)
							if afterDelimiter >= len(nextLine) || isWhitespace(nextLine[afterDelimiter]) {
								state = StateNormal
								delimiter = "" // Reset delimiter
							}
						}
					}
				}
			}
			i++
		}
	}

	return result.String()
}

// isWhitespace checks if a character is whitespace
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func RunBashScriptFromReader(ctx context.Context, runner *interp.Runner, reader io.Reader, name string) error {
	// Read all input first
	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// Preprocess the input to transform typeset/declare commands
	processedContent := PreprocessTypesetCommands(string(content))

	prog, err := syntax.NewParser().Parse(strings.NewReader(processedContent), name)
	if err != nil {
		return err
	}
	return runner.Run(ctx, prog)
}

func RunBashScriptFromFile(ctx context.Context, runner *interp.Runner, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	return RunBashScriptFromReader(ctx, runner, f, filePath)
}

func RunBashCommandInSubShell(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	subShell := runner.Subshell()

	outBuf := &threadSafeBuffer{}
	outWriter := io.Writer(outBuf)
	errBuf := &threadSafeBuffer{}
	errWriter := io.Writer(errBuf)
	_ = interp.StdIO(nil, outWriter, errWriter)(subShell)

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", err
	}

	err = subShell.Run(ctx, prog)
	if err != nil {
		return "", "", err
	}

	return outBuf.String(), errBuf.String(), nil
}

func RunBashCommand(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	outBuf := &threadSafeBuffer{}
	outWriter := io.Writer(outBuf)
	errBuf := &threadSafeBuffer{}
	errWriter := io.Writer(errBuf)
	_ = interp.StdIO(nil, outWriter, errWriter)(runner)
	defer func() {
		_ = interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner)
	}()

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", err
	}

	err = runner.Run(ctx, prog)
	if err != nil {
		return "", "", err
	}

	return outBuf.String(), errBuf.String(), nil
}
