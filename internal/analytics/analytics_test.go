package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTotalCount(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err, "Failed to create analytics manager")

	// Test initial count
	count, err := analyticsManager.GetTotalCount()
	assert.NoError(t, err, "Failed to get count")
	assert.Equal(t, int64(0), count, "Expected initial count to be 0")

	// Add some entries
	entries := []struct{ input, prediction, actual string }{
		{"cmd1", "pred1", "act1"},
		{"cmd2", "pred2", "act2"},
		{"cmd3", "pred3", "act3"},
	}

	for _, e := range entries {
		err := analyticsManager.NewEntry(e.input, e.prediction, e.actual)
		assert.NoError(t, err, "Failed to create entry")
	}

	// Test count after adding entries
	count, err = analyticsManager.GetTotalCount()
	assert.NoError(t, err, "Failed to get count")
	assert.Equal(t, int64(len(entries)), count, "Expected count to match number of entries")

	// Test count after clearing
	err = analyticsManager.ResetAnalytics()
	assert.NoError(t, err, "Failed to reset analytics")

	count, err = analyticsManager.GetTotalCount()
	assert.NoError(t, err, "Failed to get count")
	assert.Equal(t, int64(0), count, "Expected count to be 0 after reset")
}

func TestBasicOperations(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err, "Failed to create analytics manager")

	// Test creating new entries
	err = analyticsManager.NewEntry("cd ", "cd ~/Documents", "cd /home")
	assert.NoError(t, err, "Failed to create first entry")

	err = analyticsManager.NewEntry("ls ", "ls -la", "ls -l")
	assert.NoError(t, err, "Failed to create second entry")

	// Test getting recent entries
	entries, err := analyticsManager.GetRecentEntries(3)
	assert.NoError(t, err, "Failed to get recent entries")
	assert.Len(t, entries, 2, "Expected 2 entries")

	// Verify entries are ordered by creation time (most recent first)
	assert.Equal(t, "ls ", entries[0].Input, "Expected most recent input to be 'ls '")
	assert.Equal(t, "ls -la", entries[0].Prediction, "Expected most recent prediction to be 'ls -la'")
	assert.Equal(t, "ls -l", entries[0].Actual, "Expected most recent actual to be 'ls -l'")

	// Verify timestamps are set
	assert.False(t, entries[0].CreatedAt.IsZero(), "Expected CreatedAt to be set")
	assert.False(t, entries[0].UpdatedAt.IsZero(), "Expected UpdatedAt to be set")

	// Test limit
	limitedEntries, err := analyticsManager.GetRecentEntries(1)
	assert.NoError(t, err, "Failed to get limited entries")
	assert.Len(t, limitedEntries, 1, "Expected 1 entry")
}

func TestDeleteEntry(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err, "Failed to create analytics manager")

	// Create test entries
	err = analyticsManager.NewEntry("input1", "pred1", "actual1")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // Ensure different timestamps

	err = analyticsManager.NewEntry("input2", "pred2", "actual2")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond)

	err = analyticsManager.NewEntry("input3", "pred3", "actual3")
	assert.NoError(t, err)

	// Get entries to get their IDs
	entries, err := analyticsManager.GetRecentEntries(10)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	// Test cases
	tests := []struct {
		name          string
		idToDelete    uint
		expectedError bool
		checkAfter    func(t *testing.T, am *AnalyticsManager)
	}{
		{
			name:          "Delete existing entry",
			idToDelete:    entries[1].ID, // Delete the middle entry
			expectedError: false,
			checkAfter: func(t *testing.T, am *AnalyticsManager) {
				remainingEntries, err := am.GetRecentEntries(10)
				assert.NoError(t, err)
				assert.Len(t, remainingEntries, 2)
				// Verify the middle entry is gone
				for _, e := range remainingEntries {
					assert.NotEqual(t, entries[1].ID, e.ID)
				}
			},
		},
		{
			name:          "Delete non-existent entry",
			idToDelete:    99999,
			expectedError: true,
			checkAfter: func(t *testing.T, am *AnalyticsManager) {
				remainingEntries, err := am.GetRecentEntries(10)
				assert.NoError(t, err)
				assert.Len(t, remainingEntries, 2)
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := analyticsManager.DeleteEntry(tc.idToDelete)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			tc.checkAfter(t, analyticsManager)
		})
	}
}

func TestResetAnalytics(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err, "Failed to create analytics manager")

	// Create some test entries
	err = analyticsManager.NewEntry("input1", "pred1", "actual1")
	assert.NoError(t, err)

	err = analyticsManager.NewEntry("input2", "pred2", "actual2")
	assert.NoError(t, err)

	// Verify entries exist
	entries, err := analyticsManager.GetRecentEntries(10)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Reset analytics
	err = analyticsManager.ResetAnalytics()
	assert.NoError(t, err)

	// Verify all entries are gone
	entries, err = analyticsManager.GetRecentEntries(10)
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Verify we can still add new entries after reset
	err = analyticsManager.NewEntry("input3", "pred3", "actual3")
	assert.NoError(t, err)

	entries, err = analyticsManager.GetRecentEntries(10)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
}
