package context

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockRetriever is a simple mock implementation of Retriever for testing.
type mockRetriever struct {
	name    string
	context string
	err     error
}

func (m *mockRetriever) Name() string {
	return m.name
}

func (m *mockRetriever) GetContext() (string, error) {
	return m.context, m.err
}

func TestNewProvider(t *testing.T) {
	t.Run("with nil logger", func(t *testing.T) {
		provider := NewProvider(nil)
		assert.NotNil(t, provider)
		assert.NotNil(t, provider.logger)
	})

	t.Run("with logger and retrievers", func(t *testing.T) {
		logger := zap.NewNop()
		r1 := &mockRetriever{name: "test1", context: "ctx1"}
		r2 := &mockRetriever{name: "test2", context: "ctx2"}

		provider := NewProvider(logger, r1, r2)
		assert.NotNil(t, provider)
		assert.Len(t, provider.retrievers, 2)
	})
}

func TestProviderAddRetriever(t *testing.T) {
	provider := NewProvider(nil)
	assert.Len(t, provider.retrievers, 0)

	r1 := &mockRetriever{name: "test1", context: "ctx1"}
	provider.AddRetriever(r1)
	assert.Len(t, provider.retrievers, 1)

	r2 := &mockRetriever{name: "test2", context: "ctx2"}
	provider.AddRetriever(r2)
	assert.Len(t, provider.retrievers, 2)
}

func TestProviderGetContext(t *testing.T) {
	t.Run("with multiple retrievers", func(t *testing.T) {
		r1 := &mockRetriever{name: "cwd", context: "<cwd>/home/user</cwd>"}
		r2 := &mockRetriever{name: "git", context: "<git>main branch</git>"}

		provider := NewProvider(nil, r1, r2)
		result := provider.GetContext()

		assert.Len(t, result, 2)
		assert.Equal(t, "<cwd>/home/user</cwd>", result["cwd"])
		assert.Equal(t, "<git>main branch</git>", result["git"])
	})

	t.Run("with whitespace trimming", func(t *testing.T) {
		r := &mockRetriever{name: "test", context: "  context with spaces  \n"}
		provider := NewProvider(nil, r)
		result := provider.GetContext()

		assert.Equal(t, "context with spaces", result["test"])
	})

	t.Run("with failing retriever", func(t *testing.T) {
		r1 := &mockRetriever{name: "good", context: "good context"}
		r2 := &mockRetriever{name: "bad", err: errors.New("retrieval failed")}
		r3 := &mockRetriever{name: "also_good", context: "also good"}

		provider := NewProvider(zap.NewNop(), r1, r2, r3)
		result := provider.GetContext()

		// Should have 2 results, skipping the failed one
		assert.Len(t, result, 2)
		assert.Equal(t, "good context", result["good"])
		assert.Equal(t, "also good", result["also_good"])
		_, exists := result["bad"]
		assert.False(t, exists)
	})

	t.Run("with no retrievers", func(t *testing.T) {
		provider := NewProvider(nil)
		result := provider.GetContext()

		assert.Empty(t, result)
		assert.NotNil(t, result)
	})
}

func TestProviderGetContextForTypes(t *testing.T) {
	r1 := &mockRetriever{name: "cwd", context: "cwd context"}
	r2 := &mockRetriever{name: "git", context: "git context"}
	r3 := &mockRetriever{name: "system", context: "system context"}
	r4 := &mockRetriever{name: "history", context: "history context"}

	provider := NewProvider(nil, r1, r2, r3, r4)

	t.Run("with specific types", func(t *testing.T) {
		result := provider.GetContextForTypes([]string{"cwd", "git"})

		assert.Len(t, result, 2)
		assert.Equal(t, "cwd context", result["cwd"])
		assert.Equal(t, "git context", result["git"])
		_, exists := result["system"]
		assert.False(t, exists)
	})

	t.Run("with empty types returns all", func(t *testing.T) {
		result := provider.GetContextForTypes([]string{})

		assert.Len(t, result, 4)
	})

	t.Run("with nil types returns all", func(t *testing.T) {
		result := provider.GetContextForTypes(nil)

		assert.Len(t, result, 4)
	})

	t.Run("with types containing whitespace", func(t *testing.T) {
		result := provider.GetContextForTypes([]string{" cwd ", "  git"})

		assert.Len(t, result, 2)
		assert.Equal(t, "cwd context", result["cwd"])
		assert.Equal(t, "git context", result["git"])
	})

	t.Run("with non-existent types", func(t *testing.T) {
		result := provider.GetContextForTypes([]string{"nonexistent", "cwd"})

		assert.Len(t, result, 1)
		assert.Equal(t, "cwd context", result["cwd"])
	})

	t.Run("with failing retriever in types", func(t *testing.T) {
		badRetriever := &mockRetriever{name: "bad", err: errors.New("failed")}
		provider := NewProvider(zap.NewNop(), r1, badRetriever)

		result := provider.GetContextForTypes([]string{"cwd", "bad"})

		assert.Len(t, result, 1)
		assert.Equal(t, "cwd context", result["cwd"])
	})
}
