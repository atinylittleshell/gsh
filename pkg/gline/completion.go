package gline

import (
	"context"
	"strings"

	"github.com/atinylittleshell/gsh/internal/completion"
	"mvdan.cc/sh/v3/interp"
)

// ShellCompletionProvider implements shellinput.CompletionProvider using the shell's CompletionManager
type ShellCompletionProvider struct {
	CompletionManager *completion.CompletionManager
	Runner            *interp.Runner
}

// GetCompletions returns completion suggestions for the current input line
func (p *ShellCompletionProvider) GetCompletions(line string, pos int) []string {
	// Split the line into words
	words := strings.Fields(line[:pos])
	if len(words) == 0 {
		return []string{}
	}

	// Get the command (first word)
	command := words[0]

	// Look up completion spec for this command
	spec, ok := p.CompletionManager.GetSpec(command)
	if !ok {
		return []string{}
	}

	// Execute the completion
	suggestions, err := p.CompletionManager.ExecuteCompletion(context.Background(), p.Runner, spec, words)
	if err != nil {
		return []string{}
	}

	return suggestions
}
