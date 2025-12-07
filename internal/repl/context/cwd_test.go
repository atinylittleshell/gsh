package context

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkingDirectoryRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		exec, err := executor.NewREPLExecutor(nil)
		require.NoError(t, err)
		defer exec.Close()

		retriever := NewWorkingDirectoryRetriever(exec)
		assert.Equal(t, "working_directory", retriever.Name())
	})

	t.Run("GetContext returns formatted working directory", func(t *testing.T) {
		exec, err := executor.NewREPLExecutor(nil)
		require.NoError(t, err)
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
		exec, err := executor.NewREPLExecutor(nil)
		require.NoError(t, err)
		defer exec.Close()

		retriever := NewWorkingDirectoryRetriever(exec)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		pwd := exec.GetPwd()
		expected := "<working_dir>" + pwd + "</working_dir>"
		assert.Equal(t, expected, ctx)
	})
}
