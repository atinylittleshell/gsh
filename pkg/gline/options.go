package gline

import "github.com/atinylittleshell/gsh/pkg/shellinput"

type Options struct {
	MinHeight          int
	AssistantHeight    int
	CompletionProvider shellinput.CompletionProvider
}

func NewOptions() Options {
	return Options{
		MinHeight:       1,
		AssistantHeight: 3,
	}
}
