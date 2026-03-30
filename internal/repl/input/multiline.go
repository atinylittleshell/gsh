package input

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// IsInputComplete checks whether the given shell input is syntactically complete.
// Returns true if the input is a complete statement (or has a hard syntax error).
// Returns false if the input is incomplete and needs more input (unclosed quotes,
// pipes, heredocs, etc.).
func IsInputComplete(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return true
	}

	// gsh agent commands (start with #) are always complete
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Append a newline before parsing to properly detect heredocs and other
	// constructs that require a newline to trigger IsIncomplete in mvdan/sh.
	p := syntax.NewParser()
	_, err := p.Parse(strings.NewReader(input+"\n"), "")
	if err == nil {
		return true
	}
	return !syntax.IsIncomplete(err)
}
