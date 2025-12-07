package context

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/history"
)

const (
	// DefaultHistoryConciseLimit is the default number of history entries for concise context.
	DefaultHistoryConciseLimit = 30
	// DefaultHistoryVerboseLimit is the default number of history entries for verbose context.
	DefaultHistoryVerboseLimit = 30
)

// ConciseHistoryRetriever retrieves a concise command history context.
// It shows commands grouped by directory without detailed metadata.
type ConciseHistoryRetriever struct {
	historyManager *history.HistoryManager
	limit          int
}

// NewConciseHistoryRetriever creates a new ConciseHistoryRetriever.
// If limit is 0 or negative, DefaultHistoryConciseLimit is used.
func NewConciseHistoryRetriever(hm *history.HistoryManager, limit int) *ConciseHistoryRetriever {
	if limit <= 0 {
		limit = DefaultHistoryConciseLimit
	}
	return &ConciseHistoryRetriever{
		historyManager: hm,
		limit:          limit,
	}
}

// Name returns the retriever name.
func (r *ConciseHistoryRetriever) Name() string {
	return "history_concise"
}

// GetContext returns recent command history formatted for LLM context.
// Commands are grouped by directory with minimal formatting.
func (r *ConciseHistoryRetriever) GetContext() (string, error) {
	historyEntries, err := r.historyManager.GetRecentEntries("", r.limit)
	if err != nil {
		return "", err
	}

	var commandHistory string
	var lastDirectory string
	for _, entry := range historyEntries {
		if entry.Directory != lastDirectory {
			commandHistory += fmt.Sprintf("# %s\n", entry.Directory)
			lastDirectory = entry.Directory
		}
		commandHistory += entry.Command + "\n"
	}

	return fmt.Sprintf(`<recent_commands>
%s
</recent_commands>`, strings.TrimSpace(commandHistory)), nil
}

// VerboseHistoryRetriever retrieves a verbose command history context.
// It includes sequence numbers, exit codes, and directory information.
type VerboseHistoryRetriever struct {
	historyManager *history.HistoryManager
	limit          int
}

// NewVerboseHistoryRetriever creates a new VerboseHistoryRetriever.
// If limit is 0 or negative, DefaultHistoryVerboseLimit is used.
func NewVerboseHistoryRetriever(hm *history.HistoryManager, limit int) *VerboseHistoryRetriever {
	if limit <= 0 {
		limit = DefaultHistoryVerboseLimit
	}
	return &VerboseHistoryRetriever{
		historyManager: hm,
		limit:          limit,
	}
}

// Name returns the retriever name.
func (r *VerboseHistoryRetriever) Name() string {
	return "history_verbose"
}

// GetContext returns recent command history with detailed metadata.
// Includes sequence number, exit code, and command for each entry.
func (r *VerboseHistoryRetriever) GetContext() (string, error) {
	historyEntries, err := r.historyManager.GetRecentEntries("", r.limit)
	if err != nil {
		return "", err
	}

	var commandHistory = "#sequence,exit_code,command\n"
	var lastDirectory string
	for _, entry := range historyEntries {
		if entry.Directory != lastDirectory {
			commandHistory += fmt.Sprintf("# %s\n", entry.Directory)
			lastDirectory = entry.Directory
		}
		commandHistory += fmt.Sprintf("%d,%d,%s\n",
			entry.ID,
			entry.ExitCode.Int32,
			entry.Command,
		)
	}

	return fmt.Sprintf(`<recent_commands>
%s
</recent_commands>`, strings.TrimSpace(commandHistory)), nil
}
