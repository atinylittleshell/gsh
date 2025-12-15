package gline

import (
	"context"
	"strings"

	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"mvdan.cc/sh/v3/interp"
)

// ShellCompletionProvider implements shellinput.CompletionProvider using the shell's CompletionManager
type ShellCompletionProvider struct {
	CompletionManager *completion.CompletionManager
	Runner            *interp.Runner
}

// GetCompletions returns completion suggestions for the current input line
func (p *ShellCompletionProvider) GetCompletions(line string, pos int) []shellinput.CompletionCandidate {
	// Skip completions for agentic commands (starting with @)
	trimmedLine := strings.TrimSpace(line[:pos])
	if strings.HasPrefix(trimmedLine, "@") {
		return []shellinput.CompletionCandidate{}
	}

	// Split the line into words
	words := strings.Fields(line[:pos])
	if len(words) == 0 {
		return []shellinput.CompletionCandidate{}
	}

	// Get the command (first word)
	command := words[0]

	// Look up completion spec for this command
	spec, ok := p.CompletionManager.GetSpec(command)
	if !ok {
		return []shellinput.CompletionCandidate{}
	}

	// Execute the completion
	suggestions, err := p.CompletionManager.ExecuteCompletion(context.Background(), p.Runner, spec, words, line, pos)
	if err != nil {
		return []shellinput.CompletionCandidate{}
	}

	return suggestions
}

// GetHelpInfo returns help information for special commands like #! and #/
// Returns empty string if no help is available
func (p *ShellCompletionProvider) GetHelpInfo(line string, pos int) string {
	// Not implemented in this basic provider, use internal/completion/provider.go's ShellCompletionProvider for full features
	return ""
}
