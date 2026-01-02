package context

import (
	"fmt"
	"testing"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHistoryManager(t *testing.T) *history.HistoryManager {
	hm, err := history.NewHistoryManager(":memory:")
	require.NoError(t, err)

	// Add some test entries
	entry1, err := hm.StartCommand("ls -l", "/home")
	require.NoError(t, err)
	_, err = hm.FinishCommand(entry1, 0)
	require.NoError(t, err)

	entry2, err := hm.StartCommand("pwd", "/home")
	require.NoError(t, err)
	_, err = hm.FinishCommand(entry2, 0)
	require.NoError(t, err)

	entry3, err := hm.StartCommand("cd /tmp", "/tmp")
	require.NoError(t, err)
	_, err = hm.FinishCommand(entry3, 0)
	require.NoError(t, err)

	return hm
}

func TestConciseHistoryRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		retriever := NewConciseHistoryRetriever(nil, 10)
		assert.Equal(t, "history_concise", retriever.Name())
	})

	t.Run("GetContext returns formatted history", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewConciseHistoryRetriever(hm, 10)

		ctx, err := retriever.GetContext()
		assert.NoError(t, err)

		expected := `<recent_commands>
# /home
ls -l
pwd
# /tmp
cd /tmp
</recent_commands>`
		assert.Equal(t, expected, ctx)
	})

	t.Run("uses default limit when zero", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewConciseHistoryRetriever(hm, 0)

		assert.Equal(t, DefaultHistoryConciseLimit, retriever.limit)
	})

	t.Run("uses default limit when negative", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewConciseHistoryRetriever(hm, -5)

		assert.Equal(t, DefaultHistoryConciseLimit, retriever.limit)
	})

	t.Run("respects limit", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewConciseHistoryRetriever(hm, 2)

		ctx, err := retriever.GetContext()
		assert.NoError(t, err)

		// Should only have the last 2 entries
		assert.Contains(t, ctx, "pwd")
		assert.Contains(t, ctx, "cd /tmp")
		assert.NotContains(t, ctx, "ls -l")
	})

	t.Run("handles empty history", func(t *testing.T) {
		hm, err := history.NewHistoryManager(":memory:")
		require.NoError(t, err)

		retriever := NewConciseHistoryRetriever(hm, 10)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		expected := `<recent_commands>

</recent_commands>`
		assert.Equal(t, expected, ctx)
	})
}

func TestVerboseHistoryRetriever(t *testing.T) {
	t.Run("Name returns correct value", func(t *testing.T) {
		retriever := NewVerboseHistoryRetriever(nil, 10)
		assert.Equal(t, "history_verbose", retriever.Name())
	})

	t.Run("GetContext returns formatted history with metadata", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewVerboseHistoryRetriever(hm, 10)

		ctx, err := retriever.GetContext()
		assert.NoError(t, err)

		// Get actual entry IDs
		entries, err := hm.GetRecentEntries("", 10)
		require.NoError(t, err)
		require.Len(t, entries, 3)

		expected := fmt.Sprintf(`<recent_commands>
#sequence,exit_code,command
# /home
%d,0,ls -l
%d,0,pwd
# /tmp
%d,0,cd /tmp
</recent_commands>`, entries[0].ID, entries[1].ID, entries[2].ID)
		assert.Equal(t, expected, ctx)
	})

	t.Run("uses default limit when zero", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewVerboseHistoryRetriever(hm, 0)

		assert.Equal(t, DefaultHistoryVerboseLimit, retriever.limit)
	})

	t.Run("uses default limit when negative", func(t *testing.T) {
		hm := setupTestHistoryManager(t)
		retriever := NewVerboseHistoryRetriever(hm, -5)

		assert.Equal(t, DefaultHistoryVerboseLimit, retriever.limit)
	})

	t.Run("includes exit codes", func(t *testing.T) {
		hm, err := history.NewHistoryManager(":memory:")
		require.NoError(t, err)

		// Add entry with non-zero exit code
		entry, err := hm.StartCommand("false", "/home")
		require.NoError(t, err)
		_, err = hm.FinishCommand(entry, 1)
		require.NoError(t, err)

		retriever := NewVerboseHistoryRetriever(hm, 10)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		assert.Contains(t, ctx, ",1,false")
	})

	t.Run("handles empty history", func(t *testing.T) {
		hm, err := history.NewHistoryManager(":memory:")
		require.NoError(t, err)

		retriever := NewVerboseHistoryRetriever(hm, 10)
		ctx, err := retriever.GetContext()

		assert.NoError(t, err)
		expected := `<recent_commands>
#sequence,exit_code,command
</recent_commands>`
		assert.Equal(t, expected, ctx)
	})
}
