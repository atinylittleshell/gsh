package gline

import "github.com/atinylittleshell/gsh/pkg/shellinput"

type Options struct {
	// Deprecated: use AssistantHeight instead
	MinHeight          int
	AssistantHeight    int
	CompletionProvider shellinput.CompletionProvider
	RichHistory        []shellinput.HistoryItem
}

func NewOptions() Options {
	return Options{
		AssistantHeight: 3,
	}
}
