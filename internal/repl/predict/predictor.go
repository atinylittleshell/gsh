// Package predict provides LLM-based prediction functionality for the gsh REPL.
// It includes prefix-based prediction, null-state prediction, command explanation,
// and a router to coordinate between different prediction strategies.
package predict

import (
	"context"
)

// Predictor defines the interface for making command predictions.
// Implementations can use different strategies (history, LLM, etc.)
type Predictor interface {
	// Predict returns a prediction for the given input.
	// The context can be used for cancellation.
	// Returns the predicted command and any error that occurred.
	Predict(ctx context.Context, input string) (prediction string, err error)

	// UpdateContext updates the context information used for predictions.
	// The context map contains key-value pairs like "cwd", "git", "history", etc.
	UpdateContext(contextMap map[string]string)
}

// ContextFormatter formats a context map into a string suitable for LLM prompts.
type ContextFormatter interface {
	// FormatContext formats the context map into a string.
	FormatContext(contextMap map[string]string) string
}

// DefaultContextFormatter is the default implementation of ContextFormatter.
type DefaultContextFormatter struct{}

// FormatContext formats the context map into a string with labeled sections.
func (f *DefaultContextFormatter) FormatContext(contextMap map[string]string) string {
	if len(contextMap) == 0 {
		return ""
	}

	var result string
	for key, value := range contextMap {
		if value == "" {
			continue
		}
		result += "## " + key + "\n" + value + "\n\n"
	}
	return result
}

// BestPractices contains shell command best practices used in prediction prompts.
const BestPractices = `* Git commit messages should follow conventional commit message format`
