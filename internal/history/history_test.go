package history

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicOperations(t *testing.T) {
	historyManager, err := NewHistoryManager(":memory:")
	assert.NoError(t, err, "Failed to create history manager")

	entry, err := historyManager.StartCommand("echo hello", "/")
	if err != nil {
		t.Errorf("Failed to start command: %v", err)
	}
	assert.False(t, entry.CreatedAt.IsZero(), "Expected CreatedAt to be set")
	assert.False(t, entry.UpdatedAt.IsZero(), "Expected UpdatedAt to be set")

	_, _ = historyManager.FinishCommand(entry, 0)
	if err != nil {
		t.Errorf("Failed to finish command: %v", err)
	}

	entry, err = historyManager.StartCommand("echo world", "/")
	if err != nil {
		t.Errorf("Failed to start command: %v", err)
	}

	_, _ = historyManager.FinishCommand(entry, 0)
	if err != nil {
		t.Errorf("Failed to finish command: %v", err)
	}

	allEntries, err := historyManager.GetRecentEntries("", 3)
	if err != nil {
		t.Errorf("Failed to get recent entries: %v", err)
	}

	assert.Len(t, allEntries, 2, "Expected 2 entries")

	assert.Equal(t, "echo hello", allEntries[0].Command, "Expected most recent command to be 'echo hello'")

	targetEntries, _ := historyManager.GetRecentEntries("/", 3)
	assert.Len(t, targetEntries, 2, "Expected 2 entries")

	nonTargetEntries, _ := historyManager.GetRecentEntries("/tmp", 3)
	assert.Len(t, nonTargetEntries, 0, "Expected 0 entries")
}

func TestDeleteEntry(t *testing.T) {
	historyManager, err := NewHistoryManager(":memory:")
	assert.NoError(t, err, "Failed to create history manager")

	// Create some test entries
	entry1, err := historyManager.StartCommand("command1", "/")
	assert.NoError(t, err)
	_, _ = historyManager.FinishCommand(entry1, 0)
	assert.NoError(t, err)

	entry2, err := historyManager.StartCommand("command2", "/")
	assert.NoError(t, err)
	entry2, err = historyManager.FinishCommand(entry2, 0)
	assert.NoError(t, err)

	entry3, err := historyManager.StartCommand("command3", "/")
	assert.NoError(t, err)
	_, _ = historyManager.FinishCommand(entry3, 0)
	assert.NoError(t, err)

	// Test cases
	tests := []struct {
		name          string
		idToDelete    uint
		expectedError bool
		checkAfter    func(t *testing.T, hm *HistoryManager)
	}{
		{
			name:          "Delete existing entry",
			idToDelete:    entry2.ID,
			expectedError: false,
			checkAfter: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
				// Verify entry2 is gone
				for _, e := range entries {
					assert.NotEqual(t, entry2.ID, e.ID)
				}
			},
		},
		{
			name:          "Delete non-existent entry",
			idToDelete:    99999,
			expectedError: true,
			checkAfter: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := historyManager.DeleteEntry(tt.idToDelete)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.checkAfter != nil {
				tt.checkAfter(t, historyManager)
			}
		})
	}
}

func setupTestHistory(t *testing.T) (*HistoryManager, []uint) {
	historyManager, err := NewHistoryManager(":memory:")
	assert.NoError(t, err, "Failed to create history manager")

	var entryIDs []uint
	testEntries := []struct {
		command string
		dir     string
		exit    int
	}{
		{"command1", "/dir1", 0},
		{"command2", "/dir2", 1},
		{"command3", "/dir3", 0},
	}

	for _, e := range testEntries {
		entry, err := historyManager.StartCommand(e.command, e.dir)
		assert.NoError(t, err)
		entry, err = historyManager.FinishCommand(entry, e.exit)
		assert.NoError(t, err)
		entryIDs = append(entryIDs, entry.ID)
	}

	return historyManager, entryIDs
}

func TestHistoryCommandHandler(t *testing.T) {
	nextHandler := func(ctx context.Context, args []string) error {
		return nil
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		checkResult func(t *testing.T, hm *HistoryManager, ids []uint)
	}{
		{
			name:    "Basic history command",
			args:    []string{"history"},
			wantErr: false,
			checkResult: func(t *testing.T, hm *HistoryManager, ids []uint) {
				entries, err := hm.GetRecentEntries("", 20)
				assert.NoError(t, err)
				assert.Len(t, entries, 3)
			},
		},
		{
			name:    "History with limit",
			args:    []string{"history", "2"},
			wantErr: false,
			checkResult: func(t *testing.T, hm *HistoryManager, ids []uint) {
				entries, err := hm.GetRecentEntries("", 2)
				assert.NoError(t, err)
				assert.LessOrEqual(t, len(entries), 2)
			},
		},
		{
			name:    "Clear history",
			args:    []string{"history", "-c"},
			wantErr: false,
			checkResult: func(t *testing.T, hm *HistoryManager, ids []uint) {
				entries, err := hm.GetRecentEntries("", 20)
				assert.NoError(t, err)
				assert.Len(t, entries, 0)
			},
		},
		{
			name:    "Delete entry without number",
			args:    []string{"history", "-d"},
			wantErr: true,
		},
		{
			name:    "Delete entry with invalid number",
			args:    []string{"history", "-d", "invalid"},
			wantErr: true,
		},
		{
			name:    "Delete entry with valid number",
			args:    []string{"history", "-d", "2"}, // Delete the second entry
			wantErr: false,
			checkResult: func(t *testing.T, hm *HistoryManager, ids []uint) {
				entries, err := hm.GetRecentEntries("", 20)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
				for _, e := range entries {
					assert.NotEqual(t, "command2", e.Command)
				}
			},
		},
		{
			name:    "Show help",
			args:    []string{"history", "--help"},
			wantErr: false,
		},
		{
			name:    "Invalid limit",
			args:    []string{"history", "-999"},
			wantErr: false,
			checkResult: func(t *testing.T, hm *HistoryManager, ids []uint) {
				entries, err := hm.GetRecentEntries("", 20)
				assert.NoError(t, err)
				assert.True(t, len(entries) <= 20)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyManager, entryIDs := setupTestHistory(t)
			handler := NewHistoryCommandHandler(historyManager)
			wrappedHandler := handler(nextHandler)

			err := wrappedHandler(context.Background(), tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, historyManager, entryIDs)
			}
		})
	}
}

func TestGetRecentEntriesByPrefix(t *testing.T) {
	historyManager, err := NewHistoryManager(":memory:")
	assert.NoError(t, err, "Failed to create history manager")

	// Create test entries
	testCases := []struct {
		command   string
		directory string
		exitCode  int
	}{
		{"git status", "/", 0},
		{"git commit", "/", 0},
		{"echo hello", "/", 0},
		{"git push", "/", 0},
		{"ls -l", "/", 0},
	}

	for _, tc := range testCases {
		entry, err := historyManager.StartCommand(tc.command, tc.directory)
		assert.NoError(t, err)
		_, err = historyManager.FinishCommand(entry, tc.exitCode)
		assert.NoError(t, err)
	}

	// Test cases for prefix search
	t.Run("Find git commands", func(t *testing.T) {
		entries, err := historyManager.GetRecentEntriesByPrefix("git", 10)
		assert.NoError(t, err)
		assert.Len(t, entries, 3)
		assert.Equal(t, "git push", entries[0].Command)
		assert.Equal(t, "git commit", entries[1].Command)
		assert.Equal(t, "git status", entries[2].Command)
	})

	t.Run("Find with limit", func(t *testing.T) {
		entries, err := historyManager.GetRecentEntriesByPrefix("git", 2)
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
		assert.Equal(t, "git push", entries[0].Command)
		assert.Equal(t, "git commit", entries[1].Command)
	})

	t.Run("Find with non-matching prefix", func(t *testing.T) {
		entries, err := historyManager.GetRecentEntriesByPrefix("xyz", 10)
		assert.NoError(t, err)
		assert.Len(t, entries, 0)
	})

	t.Run("Find with empty prefix", func(t *testing.T) {
		entries, err := historyManager.GetRecentEntriesByPrefix("", 10)
		assert.NoError(t, err)
		assert.Len(t, entries, 5)
	})
}