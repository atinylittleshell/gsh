package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"go.uber.org/zap"
)

// GitStatusRetriever retrieves git repository status context.
type GitStatusRetriever struct {
	executor *executor.REPLExecutor
	logger   *zap.Logger
}

// NewGitStatusRetriever creates a new GitStatusRetriever.
func NewGitStatusRetriever(exec *executor.REPLExecutor, logger *zap.Logger) *GitStatusRetriever {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GitStatusRetriever{
		executor: exec,
		logger:   logger,
	}
}

// Name returns the retriever name.
func (r *GitStatusRetriever) Name() string {
	return "git_status"
}

// GetContext returns the git status formatted for LLM context.
// Returns a message indicating not in a git repository if git commands fail.
func (r *GitStatusRetriever) GetContext() (string, error) {
	ctx := context.Background()

	// Check if we're in a git repository
	revParseOut, _, err := r.executor.ExecuteBashInSubshell(ctx, "git rev-parse --show-toplevel")
	if err != nil {
		r.logger.Debug("error running `git rev-parse --show-toplevel`", zap.Error(err))
		return "<git_status>not in a git repository</git_status>", nil
	}

	// Get git status
	statusOut, _, err := r.executor.ExecuteBashInSubshell(ctx, "git status")
	if err != nil {
		r.logger.Debug("error running `git status`", zap.Error(err))
		return "", nil
	}

	return fmt.Sprintf("<git_status>Project root: %s\n%s</git_status>",
		strings.TrimSpace(revParseOut), statusOut), nil
}
