package shellinput

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestViewSyncWithHistoryState(t *testing.T) {
	m := New()
	// Setup history items: Newest first (index 0)
	// This mimics how production code loads it (via GetAllEntries ordering DESC)
	items := []HistoryItem{
		{Command: "command_newest", Timestamp: time.Now()},
		{Command: "command_middle", Timestamp: time.Now().Add(-1 * time.Minute)},
		{Command: "command_oldest", Timestamp: time.Now().Add(-2 * time.Minute)},
	}
	m.SetRichHistory(items)

	// Activate reverse search
	m.inReverseSearch = true
	m.reverseSearchQuery = "comm"

	// Force update to populate filteredIndices
	m.updateHistorySearch()

	// Check View output
	viewOutput := m.View()

	// Expect the first item (newest) to be selected because HistorySortRecent is default
	// and it preserves index order (0 < 1 < 2).
	expectedMatch := items[0].Command // "command_newest"
	assert.Contains(t, viewOutput, expectedMatch, "View should contain the selected history item (Newest)")
	assert.Contains(t, viewOutput, "(reverse-i-search)", "View should be in reverse search mode")
	assert.Contains(t, viewOutput, "`comm':", "View should show the query")

	// Move selection down (next item, older)
	m.historySearchDown()

	// Check View output again
	viewOutput = m.View()
	expectedMatch = items[1].Command // "command_middle"
	assert.Contains(t, viewOutput, expectedMatch, "View should update to show the new selected item")
}
