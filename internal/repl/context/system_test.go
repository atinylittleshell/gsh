package context

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemInfoRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		retriever := NewSystemInfoRetriever()
		assert.Equal(t, "system_info", retriever.Name())
	})

	t.Run("GetContext returns system info", func(t *testing.T) {
		retriever := NewSystemInfoRetriever()
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		assert.Contains(t, ctx, "<system_info>")
		assert.Contains(t, ctx, "</system_info>")
		assert.Contains(t, ctx, runtime.GOOS)
		assert.Contains(t, ctx, runtime.GOARCH)
	})

	t.Run("GetContext format is correct", func(t *testing.T) {
		retriever := NewSystemInfoRetriever()
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		expected := "<system_info>OS: " + runtime.GOOS + ", Arch: " + runtime.GOARCH + "</system_info>"
		assert.Equal(t, expected, ctx)
	})
}
