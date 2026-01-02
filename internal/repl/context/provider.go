// Package context provides context aggregation for LLM predictions in the gsh REPL.
// It collects relevant information from various sources (working directory, git status,
// command history, system info) to provide context for more accurate predictions.
package context

import (
	"strings"

	"go.uber.org/zap"
)

// Provider aggregates context from multiple retrievers.
// It coordinates the collection of context information from various sources
// and combines them into a single map for use by the prediction system.
type Provider struct {
	logger     *zap.Logger
	retrievers []Retriever
}

// NewProvider creates a new context Provider with the given logger and retrievers.
func NewProvider(logger *zap.Logger, retrievers ...Retriever) *Provider {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Provider{
		logger:     logger,
		retrievers: retrievers,
	}
}

// AddRetriever adds a retriever to the provider.
func (p *Provider) AddRetriever(retriever Retriever) {
	p.retrievers = append(p.retrievers, retriever)
}

// GetContext collects context from all registered retrievers.
// Returns a map where keys are retriever names and values are their context strings.
// If a retriever fails, its error is logged and it is skipped.
func (p *Provider) GetContext() map[string]string {
	result := make(map[string]string)

	for _, retriever := range p.retrievers {
		output, err := retriever.GetContext()
		if err != nil {
			p.logger.Warn("error getting context from retriever",
				zap.String("retriever", retriever.Name()),
				zap.Error(err))
			continue
		}

		result[retriever.Name()] = strings.TrimSpace(output)
	}

	return result
}

// GetContextForTypes collects context only from retrievers whose names match the given types.
// This allows selective context gathering based on the use case (e.g., agent vs prediction).
func (p *Provider) GetContextForTypes(types []string) map[string]string {
	if len(types) == 0 {
		return p.GetContext()
	}

	// Create a set for O(1) lookup
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[strings.TrimSpace(t)] = true
	}

	result := make(map[string]string)

	for _, retriever := range p.retrievers {
		if !typeSet[retriever.Name()] {
			continue
		}

		output, err := retriever.GetContext()
		if err != nil {
			p.logger.Warn("error getting context from retriever",
				zap.String("retriever", retriever.Name()),
				zap.Error(err))
			continue
		}

		result[retriever.Name()] = strings.TrimSpace(output)
	}

	return result
}
