package history

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureOutput(f func() error) (string, error) {
	// Save the original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	// Run the function
	err = f()

	// Close the write end of the pipe to flush it
	if err := w.Close(); err != nil {
		return "", err
	}

	// Restore the original stdout
	os.Stdout = oldStdout

	// Read the output from the pipe
	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}

func TestHistoryCommand(t *testing.T) {
	historyManager, err := NewHistoryManager(":memory:")
	assert.NoError(t, err)

	handler := NewHistoryCommandHandler(historyManager)
	nextHandler := func(ctx context.Context, args []string) error {
		return nil
	}
	wrappedHandler := handler(nextHandler)

	// Test non-history command passes through
	err = wrappedHandler(context.Background(), []string{"echo", "hello"})
	assert.NoError(t, err)

	tests := []struct {
		name             string
		args             []string
		expectedError    bool
		setupFn          func() uint
		verify           func(t *testing.T, hm *HistoryManager)
		expectedOutputFn func(entries []HistoryEntry) string
	}{
		{
			name:          "Show help",
			args:          []string{"history", "--help"},
			expectedError: false,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				entry2, _ := historyManager.StartCommand("test2", "")
				_, _ = historyManager.FinishCommand(entry2, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				return strings.Join([]string{
					"Usage: history [option] [n]",
					"Display or manipulate the history list.",
					"",
					"Options:",
					"  -c, --clear    clear the history list",
					"  -d, --delete   delete history entry at offset",
					"  -h, --help     display this help message",
					"",
					"If n is given, display only the last n entries.",
					"If no options are given, display the history list with line numbers.",
					"",
				}, "\n")
			},
		},
		{
			name:          "List with default limit",
			args:          []string{"history"},
			expectedError: false,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				entry2, _ := historyManager.StartCommand("test2", "")
				_, _ = historyManager.FinishCommand(entry2, 0)
				entry3, _ := historyManager.StartCommand("test3", "")
				_, _ = historyManager.FinishCommand(entry3, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 20)
				assert.NoError(t, err)
				assert.Len(t, entries, 3)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				var lines []string
				for _, entry := range entries {
					lines = append(lines, fmt.Sprintf("%d %s", entry.ID, entry.Command))
				}
				return strings.Join(lines, "\n") + "\n"
			},
		},
		{
			name:          "List with custom limit",
			args:          []string{"history", "2"},
			expectedError: false,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				entry2, _ := historyManager.StartCommand("test2", "")
				_, _ = historyManager.FinishCommand(entry2, 0)
				entry3, _ := historyManager.StartCommand("test3", "")
				_, _ = historyManager.FinishCommand(entry3, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 2)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				var lines []string
				// Only take the last 2 entries
				for _, entry := range entries[len(entries)-2:] {
					lines = append(lines, fmt.Sprintf("%d %s", entry.ID, entry.Command))
				}
				return strings.Join(lines, "\n") + "\n"
			},
		},
		{
			name:          "Clear history",
			args:          []string{"history", "-c"},
			expectedError: false,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				entry2, _ := historyManager.StartCommand("test2", "")
				_, _ = historyManager.FinishCommand(entry2, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 0)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				return ""
			},
		},
		{
			name:          "Delete specific entry",
			args:          []string{"history", "-d", "%d"},
			expectedError: false,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				entry2, _ := historyManager.StartCommand("test2", "")
				_, _ = historyManager.FinishCommand(entry2, 0)
				entry3, _ := historyManager.StartCommand("test3", "")
				_, _ = historyManager.FinishCommand(entry3, 0)
				entries, _ := historyManager.GetRecentEntries("", 10)
				return entries[0].ID
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				return ""
			},
		},
		{
			name:          "Delete entry without number",
			args:          []string{"history", "-d"},
			expectedError: true,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 1)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				return ""
			},
		},
		{
			name:          "Delete entry with invalid number",
			args:          []string{"history", "-d", "invalid"},
			expectedError: true,
			setupFn: func() uint {
				_ = historyManager.ResetHistory()
				entry1, _ := historyManager.StartCommand("test1", "")
				_, _ = historyManager.FinishCommand(entry1, 0)
				return 0
			},
			verify: func(t *testing.T, hm *HistoryManager) {
				entries, err := hm.GetRecentEntries("", 10)
				assert.NoError(t, err)
				assert.Len(t, entries, 1)
			},
			expectedOutputFn: func(entries []HistoryEntry) string {
				return ""
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var id uint = 0
			if tc.setupFn != nil {
				id = tc.setupFn()
			}

			args := tc.args
			if id > 0 {
				args = make([]string, len(tc.args))
				copy(args, tc.args)
				args[2] = fmt.Sprintf("%d", id)
			}

			// Get current entries for output verification
			entries, err := historyManager.GetRecentEntries("", 20)
			assert.NoError(t, err)

			// Capture output while running the command
			output, err := captureOutput(func() error {
				return wrappedHandler(context.Background(), args)
			})

			// Verify error state
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify the output
			if tc.expectedOutputFn != nil {
				expectedOutput := tc.expectedOutputFn(entries)
				assert.Equal(t, expectedOutput, output, "Output mismatch")
			}

			// Verify the state changes
			if tc.verify != nil {
				tc.verify(t, historyManager)
			}
		})
	}
}
