package completion

import (
	"context"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"mvdan.cc/sh/v3/interp"
)

// CompletionManagerInterface defines the interface for completion management
type CompletionManagerInterface interface {
	GetSpec(command string) (CompletionSpec, bool)
	ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string, line string, pos int) ([]shellinput.CompletionCandidate, error)
}
