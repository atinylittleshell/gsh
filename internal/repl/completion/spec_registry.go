// Package completion provides tab completion functionality for the gsh REPL.
// It manages completion specifications, integrates with bash's compgen,
// and provides file and word completion capabilities.
package completion

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// CompletionType represents the type of completion.
type CompletionType string

const (
	// WordListCompletion represents word list based completion (-W option).
	WordListCompletion CompletionType = "W"
	// FunctionCompletion represents function based completion (-F option).
	FunctionCompletion CompletionType = "F"
)

// CompletionSpec represents a completion specification for a command.
type CompletionSpec struct {
	Command string
	Type    CompletionType
	Value   string   // function name or wordlist
	Options []string // additional options like -o dirname
}

// SpecRegistry stores and executes command completion specifications.
// It manages specs like "git completes with add/commit/push" and executes them.
type SpecRegistry struct {
	specs map[string]CompletionSpec
}

// NewSpecRegistry creates a new SpecRegistry.
func NewSpecRegistry() *SpecRegistry {
	return &SpecRegistry{
		specs: make(map[string]CompletionSpec),
	}
}

// AddSpec adds or updates a completion specification.
func (r *SpecRegistry) AddSpec(spec CompletionSpec) {
	r.specs[spec.Command] = spec
}

// RemoveSpec removes a completion specification.
func (r *SpecRegistry) RemoveSpec(command string) {
	delete(r.specs, command)
}

// GetSpec retrieves a completion specification.
func (r *SpecRegistry) GetSpec(command string) (CompletionSpec, bool) {
	spec, ok := r.specs[command]
	return spec, ok
}

// ListSpecs returns all completion specifications.
func (r *SpecRegistry) ListSpecs() []CompletionSpec {
	specs := make([]CompletionSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		specs = append(specs, spec)
	}
	return specs
}

// ExecuteCompletion executes a completion specification for a given command line
// and returns the list of possible completions.
func (r *SpecRegistry) ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string) ([]string, error) {
	switch spec.Type {
	case WordListCompletion:
		words := strings.Fields(spec.Value)
		completions := make([]string, 0)
		word := ""
		if len(args) > 0 {
			word = args[len(args)-1]
		}
		for _, w := range words {
			if word == "" || strings.HasPrefix(w, word) {
				completions = append(completions, w)
			}
		}
		return completions, nil

	case FunctionCompletion:
		fn := NewCompletionFunction(spec.Value, runner)
		return fn.Execute(ctx, args)

	default:
		return nil, fmt.Errorf("unsupported completion type: %s", spec.Type)
	}
}
