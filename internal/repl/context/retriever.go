// Package context provides context aggregation for LLM predictions in the gsh REPL.
// It collects relevant information from various sources (working directory, git status,
// command history, system info) to provide context for more accurate predictions.
package context

// Retriever is the interface that all context retrievers must implement.
// Each retriever is responsible for collecting a specific type of context
// information that can be used by the LLM for predictions.
type Retriever interface {
	// Name returns the unique identifier for this retriever.
	// This is used as the key in the context map returned by Provider.
	Name() string

	// GetContext returns the context string for this retriever.
	// The returned string should be formatted appropriately for LLM consumption.
	GetContext() (string, error)
}
