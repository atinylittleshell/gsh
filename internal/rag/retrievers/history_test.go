package retrievers

import (
	"fmt"
	"testing"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func setupTestHistoryManager(t *testing.T) *history.HistoryManager {
	hm, err := history.NewHistoryManager(":memory:")
	assert.NoError(t, err)

	// Add some test entries
	entry1, err := hm.StartCommand("ls -l", "/home")
	assert.NoError(t, err)
	_, _ = hm.FinishCommand(entry1, 0)
	assert.NoError(t, err)

	entry2, err := hm.StartCommand("pwd", "/home")
	assert.NoError(t, err)
	_, _ = hm.FinishCommand(entry2, 0)
	assert.NoError(t, err)

	entry3, err := hm.StartCommand("cd /tmp", "/tmp")
	assert.NoError(t, err)
	_, _ = hm.FinishCommand(entry3, 0)
	assert.NoError(t, err)

	return hm
}

func TestConciseHistoryContextRetriever(t *testing.T) {
	hm := setupTestHistoryManager(t)
	logger := zap.NewNop()
	runner := &interp.Runner{}

	retriever := ConciseHistoryContextRetriever{
		Runner:         runner,
		Logger:         logger,
		HistoryManager: hm,
	}

	context, err := retriever.GetContext()
	assert.NoError(t, err)

	expected := `<recent_commands>
# /home
ls -l
pwd
# /tmp
cd /tmp
</recent_commands>`
	assert.Equal(t, expected, context)
}

func TestVerboseHistoryContextRetriever(t *testing.T) {
	hm := setupTestHistoryManager(t)
	logger := zap.NewNop()
	runner := &interp.Runner{}

	retriever := VerboseHistoryContextRetriever{
		Runner:         runner,
		Logger:         logger,
		HistoryManager: hm,
	}

	context, err := retriever.GetContext()
	assert.NoError(t, err)

	// Since the sequence numbers are auto-generated, we'll need to get them from the actual entries
	entries, err := hm.GetRecentEntries("", 10)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	expected := `<recent_commands>
#sequence,exit_code,command
# /home
%d,0,ls -l
%d,0,pwd
# /tmp
%d,0,cd /tmp
</recent_commands>`
	expected = fmt.Sprintf(expected, entries[0].ID, entries[1].ID, entries[2].ID)
	assert.Equal(t, expected, context)
}

func TestRetrieverNames(t *testing.T) {
	conciseRetriever := ConciseHistoryContextRetriever{}
	verboseRetriever := VerboseHistoryContextRetriever{}

	assert.Equal(t, "history_concise", conciseRetriever.Name())
	assert.Equal(t, "history_verbose", verboseRetriever.Name())
}

