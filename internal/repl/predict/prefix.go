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

// PrefixPredictor predicts command completions based on a partial command prefix.
// It uses an LLM to generate predictions that start with the given prefix.
type PrefixPredictor struct {
	model     *interpreter.ModelValue
	logger    *zap.Logger
	formatter ContextFormatter

	contextText   string
	contextTextMu sync.RWMutex
}

// PrefixPredictorConfig holds configuration for creating a PrefixPredictor.
type PrefixPredictorConfig struct {
	// Model is the LLM model to use for predictions (must have Provider set).
	Model *interpreter.ModelValue

	// Logger for debug output. If nil, a no-op logger is used.
	Logger *zap.Logger

	// Formatter for context text. If nil, DefaultContextFormatter is used.
	Formatter ContextFormatter
}

// NewPrefixPredictor creates a new PrefixPredictor with the given configuration.
func NewPrefixPredictor(cfg PrefixPredictorConfig) *PrefixPredictor {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	formatter := cfg.Formatter
	if formatter == nil {
		formatter = &DefaultContextFormatter{}
	}

	return &PrefixPredictor{
		model:     cfg.Model,
		logger:    logger,
		formatter: formatter,
	}
}

// UpdateContext updates the context information used for predictions.
func (p *PrefixPredictor) UpdateContext(contextMap map[string]string) {
	contextText := p.formatter.FormatContext(contextMap)

	p.contextTextMu.Lock()
	defer p.contextTextMu.Unlock()
	p.contextText = contextText
}

// prefixPredictionResponse is the expected JSON response from the LLM.
type prefixPredictionResponse struct {
	PredictedCommand string `json:"predicted_command"`
}

// Predict returns a prediction for the given input prefix.
// The prediction will start with the input prefix.
func (p *PrefixPredictor) Predict(ctx context.Context, input string) (string, error) {
	// Only handle non-empty input (prefix prediction)
	if input == "" {
		return "", nil
	}

	// Don't predict for agent chat messages
	if strings.HasPrefix(input, "#") {
		return "", nil
	}

	if p.model == nil || p.model.Provider == nil {
		return "", nil
	}

	p.contextTextMu.RLock()
	contextText := p.contextText
	p.contextTextMu.RUnlock()

	userMessage := fmt.Sprintf(`You are gsh, an intelligent shell program.
You will be given a partial bash command prefix entered by me, enclosed in <prefix> tags.
You are asked to predict what the complete bash command is.

# Instructions
* Based on the prefix and other context, analyze my potential intent
* Your prediction must start with the partial command as a prefix
* Your prediction must be a valid, single-line, complete bash command

# Best Practices
%s

# Latest Context
%s

Respond with JSON in this format: {"predicted_command": "your prediction here"}

<prefix>%s</prefix>`, BestPractices, contextText, input)

	p.logger.Debug("prefix prediction request", zap.String("input", input), zap.String("userMessage", userMessage))

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

	// Parse JSON response
	var prediction prefixPredictionResponse
	if err := json.Unmarshal([]byte(response.Content), &prediction); err != nil {
		p.logger.Debug("failed to parse prediction JSON", zap.Error(err), zap.String("content", response.Content))
		return "", nil
	}

	p.logger.Debug("prefix prediction response", zap.String("prediction", prediction.PredictedCommand))

	// Verify the prediction starts with the input
	if !strings.HasPrefix(prediction.PredictedCommand, input) {
		p.logger.Debug("prediction does not start with input, discarding",
			zap.String("input", input),
			zap.String("prediction", prediction.PredictedCommand))
		return "", nil
	}

	return prediction.PredictedCommand, nil
}
