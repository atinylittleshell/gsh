package context

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestExecutor creates an executor with a fresh interpreter for testing.
func newTestExecutor(t *testing.T) *executor.REPLExecutor {
	t.Helper()
	interp := interpreter.New()
	exec, err := executor.NewREPLExecutor(interp, nil)
	require.NoError(t, err)
	return exec
}

func TestWorkingDirectoryRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		exec := newTestExecutor(t)
		defer exec.Close()

		retriever := NewWorkingDirectoryRetriever(exec)
		assert.Equal(t, "working_directory", retriever.Name())
	})

	t.Run("GetContext returns formatted working directory", func(t *testing.T) {
		exec := newTestExecutor(t)
		defer exec.Close()

		retriever := NewWorkingDirectoryRetriever(exec)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		assert.Contains(t, ctx, "<working_dir>")
		assert.Contains(t, ctx, "</working_dir>")

		// The pwd should be non-empty
		pwd := exec.GetPwd()
		assert.Contains(t, ctx, pwd)
	})

	t.Run("GetContext format is correct", func(t *testing.T) {
		exec := newTestExecutor(t)
		defer exec.Close()

		retriever := NewWorkingDirectoryRetriever(exec)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		pwd := exec.GetPwd()
		expected := "<working_dir>" + pwd + "</working_dir>"
		assert.Equal(t, expected, ctx)
	})
}
