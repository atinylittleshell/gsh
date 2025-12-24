package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
)

// NullStatePredictor predicts commands when the input is empty.
// It uses context (cwd, git status, history, etc.) to suggest a likely next command.
type NullStatePredictor struct {
	model     *interpreter.ModelValue
	logger    *zap.Logger
	formatter ContextFormatter

	contextText   string
	contextTextMu sync.RWMutex
}

// NullStatePredictorConfig holds configuration for creating a NullStatePredictor.
type NullStatePredictorConfig struct {
	// Model is the LLM model to use for predictions (must have Provider set).
	Model *interpreter.ModelValue

	// Logger for debug output. If nil, a no-op logger is used.
	Logger *zap.Logger

	// Formatter for context text. If nil, DefaultContextFormatter is used.
	Formatter ContextFormatter
}

// NewNullStatePredictor creates a new NullStatePredictor with the given configuration.
func NewNullStatePredictor(cfg NullStatePredictorConfig) *NullStatePredictor {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	formatter := cfg.Formatter
	if formatter == nil {
		formatter = &DefaultContextFormatter{}
	}

	return &NullStatePredictor{
		model:     cfg.Model,
		logger:    logger,
		formatter: formatter,
	}
}

// UpdateContext updates the context information used for predictions.
func (p *NullStatePredictor) UpdateContext(contextMap map[string]string) {
	contextText := p.formatter.FormatContext(contextMap)

	p.contextTextMu.Lock()
	defer p.contextTextMu.Unlock()
	p.contextText = contextText
}

// nullStatePredictionResponse is the expected JSON response from the LLM.
type nullStatePredictionResponse struct {
	PredictedCommand string `json:"predicted_command"`
}

// Predict returns a prediction for the next command when input is empty.
func (p *NullStatePredictor) Predict(ctx context.Context, input string) (string, error) {
	// Only handle empty input (null state prediction)
	if input != "" {
		return "", nil
	}

	if p.model == nil || p.model.Provider == nil {
		return "", nil
	}

	p.contextTextMu.RLock()
	contextText := p.contextText
	p.contextTextMu.RUnlock()

	userMessage := fmt.Sprintf(`You are gsh, an intelligent shell program.
You are asked to predict the next command I'm likely to want to run.

# Instructions
* Based on the context, analyze my potential intent
* Your prediction must be a valid, single-line, complete bash command

# Best Practices
%s

# Latest Context
%s

Respond with JSON in this format: {"predicted_command": "your prediction here"}

Now predict what my next command should be.`, BestPractices, contextText)

	p.logger.Debug("null state prediction request", zap.String("userMessage", userMessage))

	request := interpreter.ChatRequest{
		Model: p.model,
		Messages: []interpreter.ChatMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	response, err := p.model.ChatCompletion(request)
	if err != nil {
		return "", err
	}

	// Extract JSON from response (may be wrapped in markdown code blocks)
	jsonContent := extractJSON(response.Content)

	// Parse JSON response
	var prediction nullStatePredictionResponse
	if err := json.Unmarshal([]byte(jsonContent), &prediction); err != nil {
		p.logger.Debug("failed to parse prediction JSON", zap.Error(err), zap.String("content", response.Content))
		return "", nil
	}

	p.logger.Debug("null state prediction response", zap.String("prediction", prediction.PredictedCommand))

	return prediction.PredictedCommand, nil
}

// extractJSON extracts JSON content from a string, handling markdown code blocks.
// If the content is wrapped in ```json ... ``` or ``` ... ```, it extracts the inner content.
// Otherwise, it returns the content as-is after trimming whitespace.
func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	// Check for ```json ... ``` format
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimLeft(content, "\n\r")

		// Find the last occurrence of ``` (closing marker)
		// Use LastIndex to find the rightmost ``` which should be the closing marker
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}

		return strings.TrimSpace(content)
	}

	// Check for ``` ... ``` format (without language specifier)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimLeft(content, "\n\r")

		// Find the last occurrence of ``` (closing marker)
		// Use LastIndex to find the rightmost ``` which should be the closing marker
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}

		return strings.TrimSpace(content)
	}

	return content
}
