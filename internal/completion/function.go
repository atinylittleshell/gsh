package completion

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// CompletionFunction represents a bash completion function
type CompletionFunction struct {
	Name   string
	Runner *interp.Runner
}

// NewCompletionFunction creates a new CompletionFunction
func NewCompletionFunction(name string, runner *interp.Runner) *CompletionFunction {
	return &CompletionFunction{
		Name:   name,
		Runner: runner,
	}
}

// Execute runs the completion function with the given arguments
func (f *CompletionFunction) Execute(ctx context.Context, args []string) ([]string, error) {
	script := fmt.Sprintf(`
		# Set up completion environment
		COMP_LINE=%q
		COMP_POINT=%d
		COMP_WORDS=(%s)
		COMP_CWORD=%d

		# Initialize empty COMPREPLY
		COMPREPLY=()

		# Call the completion function
		%s
	`,
		strings.Join(args, " "),
		len(strings.Join(args, " ")),
		strings.Join(args, " "),
		len(args)-1,
		f.Name,
	)

	// Parse and execute the script
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse completion script: %w", err)
	}

	if err := f.Runner.Run(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to execute completion function: %w", err)
	}

	// Get COMPREPLY from the runner's variables
	compreply, ok := f.Runner.Vars["COMPREPLY"]
	if !ok {
		return []string{}, nil
	}

	if compreply.Kind != expand.Indexed {
		return []string{}, nil
	}

	// Get all elements of the array
	results := compreply.List
	return results, nil
}
