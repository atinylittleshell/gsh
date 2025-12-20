package gline

import (
	"context"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
)

// IdleSummaryGenerator is a function that generates an idle summary
type IdleSummaryGenerator func(ctx context.Context) (string, error)

type Options struct {
	// Deprecated: use AssistantHeight instead
	MinHeight          int
	AssistantHeight    int
	CompletionProvider shellinput.CompletionProvider
	RichHistory        []shellinput.HistoryItem
	CurrentDirectory   string
	User               string
	Host               string

	// IdleSummaryTimeout is the number of seconds of idle time before generating a summary.
	// Set to 0 to disable idle summaries.
	IdleSummaryTimeout int
	// IdleSummaryGenerator is called when the user is idle to generate a summary
	IdleSummaryGenerator IdleSummaryGenerator
}

func NewOptions() Options {
	return Options{
		AssistantHeight: 3,
	}
}
