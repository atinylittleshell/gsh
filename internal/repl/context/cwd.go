package context

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/repl/executor"
)

// WorkingDirectoryRetriever retrieves the current working directory context.
type WorkingDirectoryRetriever struct {
	executor *executor.REPLExecutor
}

// NewWorkingDirectoryRetriever creates a new WorkingDirectoryRetriever.
func NewWorkingDirectoryRetriever(exec *executor.REPLExecutor) *WorkingDirectoryRetriever {
	return &WorkingDirectoryRetriever{
		executor: exec,
	}
}

// Name returns the retriever name.
func (r *WorkingDirectoryRetriever) Name() string {
	return "working_directory"
}

// GetContext returns the current working directory formatted for LLM context.
func (r *WorkingDirectoryRetriever) GetContext() (string, error) {
	pwd := r.executor.GetPwd()
	return fmt.Sprintf("<working_dir>%s</working_dir>", pwd), nil
}
