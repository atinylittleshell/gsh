package gline

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// MultilineState tracks the state of multiline input
type MultilineState struct {
	buffer           strings.Builder
	isContinuation   bool
	continuationChar string
}

// NewMultilineState creates a new multiline state
func NewMultilineState() *MultilineState {
	return &MultilineState{
		buffer:           strings.Builder{},
		isContinuation:   false,
		continuationChar: ">",
	}
}

// AddLine adds a line to the multiline buffer and checks if more input is needed
//
// IMPORTANT: This method expects individual lines without embedded newlines.
// In real shell usage, users type one line at a time, and the shell determines
// whether more input is needed based on syntax analysis.
//
// For here-documents, the usage pattern is:
// 1. User types: cat <<EOF
// 2. Shell shows: > (continuation prompt)
// 3. User types: content line
// 4. Shell shows: > (continuation prompt)
// 5. User types: EOF
// 6. Command completes
func (m *MultilineState) AddLine(line string) (complete bool, prompt string) {
	// Validate input assumptions - AddLine should receive single lines without embedded newlines
	// If embedded newlines are detected, this indicates the method is being used for
	// string literal testing rather than simulating real interactive shell usage
	//
	// Note: We intentionally don't do anything with this check other than acknowledge it
	// because stripping newlines would break paste functionality.

	// Defer panic recovery to prevent shell crashes
	defer func() {
		if r := recover(); r != nil {
			// Reset to a safe state
			m.Reset()
			complete = true
			prompt = ""
		}
	}()

	// Add the line to buffer
	if m.buffer.Len() > 0 {
		m.buffer.WriteString("\n")
	}
	m.buffer.WriteString(line)

	// Check for buffer size limits to prevent memory exhaustion
	if m.buffer.Len() > 1024*1024 { // 1MB limit
		m.Reset()
		return true, ""
	}

	// Check for backslash continuation
	if strings.HasSuffix(strings.TrimRight(line, " \t"), "\\") {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// Check if we have a complete command using the bash parser
	fullInput := m.buffer.String()
	parser := syntax.NewParser()

	// Try to parse the complete input
	var stmts []*syntax.Stmt
	err := parser.Stmts(strings.NewReader(fullInput), func(stmt *syntax.Stmt) bool {
		stmts = append(stmts, stmt)
		return true
	})

	// Check for incomplete quotes or other obvious incomplete patterns
	if hasIncompleteQuotes(fullInput) || hasIncompleteConstructs(fullInput) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// If parsing succeeded and we have statements, we're complete
	if err == nil && len(stmts) > 0 {
		m.isContinuation = false
		return true, ""
	}

	// If we have a syntax error that indicates incomplete input, continue
	if err != nil && isIncompleteSyntaxError(err) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// Check for incomplete function definitions
	if hasIncompleteFunctionDef(fullInput) {
		m.isContinuation = true
		return false, m.continuationChar
	}

	// For other cases, assume we're complete
	m.isContinuation = false
	return true, ""
}

// GetCompleteCommand returns the complete command and resets the state
func (m *MultilineState) GetCompleteCommand() string {
	// Defer panic recovery to prevent shell crashes
	defer func() {
		if r := recover(); r != nil {
			// Silently handle panic and reset state
			m.Reset()
		}
	}()

	result := m.buffer.String()
	resultLen := len(result)

	// Validate result before returning
	if resultLen > 1024*1024 { // 1MB limit
		m.Reset()
		return ""
	}

	m.Reset()
	return result
}

// Reset clears the multiline state
func (m *MultilineState) Reset() {
	m.buffer.Reset()
	m.isContinuation = false
}

// IsActive returns true if we're in the middle of a multiline input
func (m *MultilineState) IsActive() bool {
	return m.isContinuation || m.buffer.Len() > 0
}

// GetAccumulatedLines returns the accumulated lines for display purposes
func (m *MultilineState) GetAccumulatedLines() string {
	return m.buffer.String()
}

// GetLines returns the individual lines that have been entered
func (m *MultilineState) GetLines() []string {
	content := m.buffer.String()
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

// hasIncompleteQuotes checks if the input has unclosed quotes
func hasIncompleteQuotes(input string) bool {
	inEscape := false
	inSingleQuote := false
	inDoubleQuote := false

	for _, ch := range input {
		if inEscape {
			inEscape = false
			continue
		}

		switch ch {
		case '\\':
			inEscape = true
		case '\'':
			// Only process if we're not inside double quotes
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			// Only process if we're not inside single quotes
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
	}

	// If we're still inside any quote, we have incomplete quotes
	return inSingleQuote || inDoubleQuote
}

// hasIncompleteConstructs checks for other incomplete constructs
func hasIncompleteConstructs(input string) bool {
	input = strings.TrimSpace(input)

	// Check for incomplete here document
	if strings.HasSuffix(input, "<<") || strings.HasSuffix(input, "<<-") {
		return true
	}

	// Check for incomplete command substitution
	if strings.Count(input, "$(") > strings.Count(input, ")") {
		return true
	}

	// Check for incomplete backticks
	if strings.Count(input, "`")%2 != 0 {
		return true
	}

	// Check for incomplete parentheses (but not command substitution)
	// Remove command substitution patterns first to avoid false positives
	cleanInput := input
	// Remove all command substitution patterns
	loopCount := 0
	maxLoops := 1000 // Safety limit to prevent infinite loops
	for {
		loopCount++
		if loopCount > maxLoops {
			// Safety break to prevent infinite loops - if we've processed 1000 times, something is wrong
			break
		}

		start := strings.Index(cleanInput, "$(")
		if start == -1 {
			break
		}
		// Find the matching closing paren for this command substitution
		end := findMatchingParen(cleanInput[start+2:], '(', ')')
		if end == -1 {
			// Incomplete command substitution, will be caught above
			break
		}
		// Remove the complete command substitution
		newCleanInput := cleanInput[:start] + cleanInput[start+2+end+1:]
		if newCleanInput == cleanInput {
			// Safety break - if no change occurs, prevent potential infinite loop
			break
		}
		cleanInput = newCleanInput
	}

	// Now check for unmatched parentheses in the cleaned input
	openParens := strings.Count(cleanInput, "(")
	closeParens := strings.Count(cleanInput, ")")

	// Simplify return boolean logic (S1008)
	return openParens > closeParens
}

// findMatchingParen finds the matching closing parenthesis for an opening one
func findMatchingParen(s string, openChar, closeChar rune) int {
	count := 0
	for i, ch := range s {
		switch ch {
		case openChar:
			count++
		case closeChar:
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

// hasIncompleteFunctionDef checks for incomplete function definitions
func hasIncompleteFunctionDef(input string) bool {
	input = strings.TrimSpace(input)

	// Check if we have an opening brace without a closing brace
	openingBrace := strings.Count(input, "{")
	closingBrace := strings.Count(input, "}")

	// If we have more opening braces than closing braces, it's likely incomplete
	if openingBrace > closingBrace {
		return true
	}

	// Also check if the last non-whitespace character is an opening brace
	return strings.HasSuffix(input, "{")
}

// isIncompleteSyntaxError checks if the error indicates incomplete input
func isIncompleteSyntaxError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Common incomplete syntax patterns
	incompletePatterns := []string{
		"unexpected end of file",
		"unexpected newline",
		"unclosed",
		"unfinished",
		"incomplete",
		"EOF",
	}

	for _, pattern := range incompletePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}
