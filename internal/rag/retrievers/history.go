package retrievers

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type ConciseHistoryContextRetriever struct {
	Runner         *interp.Runner
	Logger         *zap.Logger
	HistoryManager *history.HistoryManager
}

type VerboseHistoryContextRetriever struct {
	Runner         *interp.Runner
	Logger         *zap.Logger
	HistoryManager *history.HistoryManager
}

func (r ConciseHistoryContextRetriever) Name() string {
	return "history_concise"
}

func (r VerboseHistoryContextRetriever) Name() string {
	return "history_verbose"
}

func (r ConciseHistoryContextRetriever) GetContext() (string, error) {
	historyEntries, err := r.HistoryManager.GetRecentEntries("", environment.GetContextNumHistoryConcise(r.Runner, r.Logger))
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

func (r VerboseHistoryContextRetriever) GetContext() (string, error) {
	historyEntries, err := r.HistoryManager.GetRecentEntries("", environment.GetContextNumHistoryVerbose(r.Runner, r.Logger))
	if err != nil {
		return "", err
	}

	commandHistory := "#sequence,exit_code,command\n"
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
