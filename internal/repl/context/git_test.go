package context

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/repl/executor"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newGitTestExecutor creates an executor with a fresh interpreter for testing.
func newGitTestExecutor(t *testing.T) *executor.REPLExecutor {
	t.Helper()
	interp := interpreter.New()
	exec, err := executor.NewREPLExecutor(interp, nil)
	require.NoError(t, err)
	return exec
}

func TestGitStatusRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		exec := newGitTestExecutor(t)
		defer exec.Close()

		retriever := NewGitStatusRetriever(exec, nil)
		assert.Equal(t, "git_status", retriever.Name())
	})

	t.Run("GetContext in git repository", func(t *testing.T) {
		// This test runs in the gsh repository itself
		exec := newGitTestExecutor(t)
		defer exec.Close()

		retriever := NewGitStatusRetriever(exec, zap.NewNop())
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		assert.Contains(t, ctx, "<git_status>")
		assert.Contains(t, ctx, "</git_status>")
		assert.Contains(t, ctx, "Project root:")
		// Should contain some git status info
		assert.Contains(t, ctx, "branch")
	})

	t.Run("GetContext outside git repository", func(t *testing.T) {
		// Create a temp directory that's not a git repo
		tmpDir, err := os.MkdirTemp("", "gsh-test-no-git-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		exec := newGitTestExecutor(t)
		defer exec.Close()

		// Change to temp directory
		_, err = exec.ExecuteBash(context.Background(), "cd "+tmpDir)
		require.NoError(t, err)

		retriever := NewGitStatusRetriever(exec, zap.NewNop())
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		assert.Equal(t, "<git_status>not in a git repository</git_status>", ctx)
	})

	t.Run("handles nil logger", func(t *testing.T) {
		exec := newGitTestExecutor(t)
		defer exec.Close()

		retriever := NewGitStatusRetriever(exec, nil)
		assert.NotNil(t, retriever.logger)

		// Should not panic when getting context
		_, err := retriever.GetContext()
		assert.NoError(t, err)
	})
}

func TestGitStatusRetrieverInNewRepo(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir, err := os.MkdirTemp("", "gsh-test-git-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	exec := newGitTestExecutor(t)
	defer exec.Close()

	// Change to temp directory (persists in the executor's runner)
	_, err = exec.ExecuteBash(context.Background(), "cd "+tmpDir)
	require.NoError(t, err)

	// Initialize git in the current directory
	_, err = exec.ExecuteBash(context.Background(), "git init")
	require.NoError(t, err)

	// Configure git user for the test repo
	_, err = exec.ExecuteBash(context.Background(), "git config user.email 'test@test.com' && git config user.name 'Test'")
	require.NoError(t, err)

	// Create and add a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	retriever := NewGitStatusRetriever(exec, zap.NewNop())
	ctx, err := retriever.GetContext()

	assert.NoError(t, err)
	assert.Contains(t, ctx, "<git_status>")
	assert.Contains(t, ctx, "Project root:")
	assert.Contains(t, ctx, tmpDir)
}
