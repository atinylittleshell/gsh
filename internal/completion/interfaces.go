package completion

import (
	"context"

	"mvdan.cc/sh/v3/interp"
)

// CompletionManagerInterface defines the interface for completion management
type CompletionManagerInterface interface {
	GetSpec(command string) (CompletionSpec, bool)
	ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string) ([]string, error)
}
