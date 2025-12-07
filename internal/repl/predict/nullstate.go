package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
)

// NullStatePredictor predicts commands when the input is empty.
// It uses context (cwd, git status, history, etc.) to suggest a likely next command.
type NullStatePredictor struct {
	model     *interpreter.ModelValue
	provider  interpreter.ModelProvider
	logger    *zap.Logger
	formatter ContextFormatter

	contextText   string
	contextTextMu sync.RWMutex
}

// NullStatePredictorConfig holds configuration for creating a NullStatePredictor.
type NullStatePredictorConfig struct {
	// Model is the LLM model to use for predictions.
	Model *interpreter.ModelValue

	// Provider is the model provider (OpenAI, Anthropic, etc.)
	Provider interpreter.ModelProvider

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
		provider:  cfg.Provider,
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

	if p.model == nil || p.provider == nil {
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

	response, err := p.provider.ChatCompletion(request)
	if err != nil {
		return "", err
	}

	// Parse JSON response
	var prediction nullStatePredictionResponse
	if err := json.Unmarshal([]byte(response.Content), &prediction); err != nil {
		p.logger.Debug("failed to parse prediction JSON", zap.Error(err), zap.String("content", response.Content))
		return "", nil
	}

	p.logger.Debug("null state prediction response", zap.String("prediction", prediction.PredictedCommand))

	return prediction.PredictedCommand, nil
}
