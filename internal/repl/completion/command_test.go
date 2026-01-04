package completion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/interp"
)

func TestNewCompleteCommandHandler(t *testing.T) {
	registry := NewSpecRegistry()

	t.Run("completion specifications", func(t *testing.T) {
		handler := NewCompleteCommandHandler(registry)

		// Create a mock next handler that should not be called for "complete" commands
		nextCalled := false
		next := func(ctx context.Context, args []string) error {
			nextCalled = true
			return nil
		}

		wrappedHandler := handler(next)

		// Test adding word list completion
		err := wrappedHandler(context.Background(), []string{"complete", "-W", "add commit push", "git"})
		require.NoError(t, err)
		assert.False(t, nextCalled, "next handler should not be called for complete command")

		spec, ok := registry.GetSpec("git")
		assert.True(t, ok)
		assert.Equal(t, "git", spec.Command)
		assert.Equal(t, WordListCompletion, spec.Type)
		assert.Equal(t, "add commit push", spec.Value)

		// Test adding function completion
		err = wrappedHandler(context.Background(), []string{"complete", "-F", "_docker_completion", "docker"})
		require.NoError(t, err)

		spec, ok = registry.GetSpec("docker")
		assert.True(t, ok)
		assert.Equal(t, "docker", spec.Command)
		assert.Equal(t, FunctionCompletion, spec.Type)
		assert.Equal(t, "_docker_completion", spec.Value)

		// Test removing completion
		err = wrappedHandler(context.Background(), []string{"complete", "-r", "git"})
		require.NoError(t, err)

		_, ok = registry.GetSpec("git")
		assert.False(t, ok, "git spec should be removed")
	})

	t.Run("error cases", func(t *testing.T) {
		handler := NewCompleteCommandHandler(registry)
		next := func(ctx context.Context, args []string) error {
			return nil
		}
		wrappedHandler := handler(next)

		// Missing word list
		err := wrappedHandler(context.Background(), []string{"complete", "-W"})
		assert.Error(t, err)

		// Missing function name
		err = wrappedHandler(context.Background(), []string{"complete", "-F"})
		assert.Error(t, err)

		// Unknown option
		err = wrappedHandler(context.Background(), []string{"complete", "-X", "test"})
		assert.Error(t, err)

		// No command specified
		err = wrappedHandler(context.Background(), []string{"complete", "-W", "words"})
		assert.Error(t, err)
	})

	t.Run("pass through non-complete commands", func(t *testing.T) {
		handler := NewCompleteCommandHandler(registry)

		nextCalled := false
		next := func(ctx context.Context, args []string) error {
			nextCalled = true
			return nil
		}

		wrappedHandler := handler(next)

		// Non-complete command should pass through
		err := wrappedHandler(context.Background(), []string{"ls", "-la"})
		require.NoError(t, err)
		assert.True(t, nextCalled, "next handler should be called for non-complete commands")
	})
}

func TestHandleCompleteCommand(t *testing.T) {
	t.Run("print mode with no specs", func(t *testing.T) {
		registry := NewSpecRegistry()
		err := handleCompleteCommand(registry, []string{"-p"})
		assert.NoError(t, err)
	})

	t.Run("print mode with specific command", func(t *testing.T) {
		registry := NewSpecRegistry()
		registry.AddSpec(CompletionSpec{
			Command: "git",
			Type:    WordListCompletion,
			Value:   "add commit",
		})
		err := handleCompleteCommand(registry, []string{"-p", "git"})
		assert.NoError(t, err)
	})

	t.Run("invalid usage - no options", func(t *testing.T) {
		registry := NewSpecRegistry()
		err := handleCompleteCommand(registry, []string{"mycommand"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid complete command usage")
	})
}

func TestCompletionManagerAlias(t *testing.T) {
	// Test that CompletionManager is an alias for SpecRegistry
	manager := NewCompletionManager()
	assert.NotNil(t, manager)

	// Should be able to use it the same way as SpecRegistry
	manager.AddSpec(CompletionSpec{
		Command: "test",
		Type:    WordListCompletion,
		Value:   "a b c",
	})

	spec, ok := manager.GetSpec("test")
	assert.True(t, ok)
	assert.Equal(t, "test", spec.Command)
}

func TestNewCompleteCommandHandlerWithRunner(t *testing.T) {
	// Test that the handler works with an actual runner
	runner, err := interp.New()
	require.NoError(t, err)

	registry := NewSpecRegistry()
	handler := NewCompleteCommandHandler(registry)

	next := func(ctx context.Context, args []string) error {
		return runner.Run(ctx, nil)
	}

	wrappedHandler := handler(next)

	// Add a completion spec
	err = wrappedHandler(context.Background(), []string{"complete", "-W", "status log diff", "git"})
	require.NoError(t, err)

	spec, ok := registry.GetSpec("git")
	assert.True(t, ok)
	assert.Equal(t, "status log diff", spec.Value)
}
