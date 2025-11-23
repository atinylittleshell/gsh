package gline

import "github.com/atinylittleshell/gsh/pkg/shellinput"

type Options struct {
	AssistantHeight    int
	CompletionProvider shellinput.CompletionProvider
}

func NewOptions() Options {
	return Options{
		AssistantHeight: 3,
	}
}
