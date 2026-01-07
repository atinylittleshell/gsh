// Package input provides input handling for the gsh REPL.
package input

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
)

// PredictionResult contains the result of a prediction request.
type PredictionResult struct {
	// Prediction is the predicted command text.
	Prediction string

	// StateID is the state ID when this prediction was requested.
	// Used to discard stale predictions.
	StateID int64

	// Source indicates where the prediction came from.
	Source PredictionSource

	// Error contains any error that occurred during prediction.
	Error error
}

// PredictionSource indicates the source of a prediction.
type PredictionSource int

const (
	// PredictionSourceNone indicates no prediction was made.
	PredictionSourceNone PredictionSource = iota
	// PredictionSourceHistory indicates the prediction came from command history.
	PredictionSourceHistory
	// PredictionSourceLLM indicates the prediction came from an LLM.
	PredictionSourceLLM
)

// String returns the string representation of the prediction source.
func (ps PredictionSource) String() string {
	switch ps {
	case PredictionSourceNone:
		return "none"
	case PredictionSourceHistory:
		return "history"
	case PredictionSourceLLM:
		return "llm"
	default:
		return "unknown"
	}
}

// PredictionProvider defines the interface for making predictions.
// This abstraction allows for different prediction backends.
type PredictionProvider interface {
	// Predict returns a prediction for the given input.
	// The context can be used for cancellation.
	// The trigger indicates whether this is an "instant" (synchronous) or "debounced" (async) prediction.
	Predict(ctx context.Context, input string, trigger interpreter.PredictTrigger) (prediction string, err error)
}

// PredictionState manages the prediction lifecycle including debouncing,
// state coordination, and async prediction handling.
type PredictionState struct {
	// Current prediction text (displayed as ghost text)
	prediction string

	// The input text that produced the current prediction
	inputForPrediction string

	// State ID for coordinating async predictions
	stateID atomic.Int64

	// Whether the input has been modified since last empty state
	dirty bool

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Configuration
	debounceDelay time.Duration

	// Provider for predictions (handles both instant and debounced via trigger parameter)
	provider PredictionProvider
	logger   *zap.Logger

	// Pending prediction cancel function
	cancelPending context.CancelFunc
}

// PredictionStateConfig holds configuration for creating a PredictionState.
type PredictionStateConfig struct {
	// DebounceDelay is the delay before making a debounced prediction request.
	// Defaults to 200ms if not set.
	DebounceDelay time.Duration

	// Provider handles predictions (both instant and debounced via trigger parameter).
	Provider PredictionProvider

	// Logger for debug output.
	Logger *zap.Logger
}

// NewPredictionState creates a new PredictionState with the given configuration.
func NewPredictionState(config PredictionStateConfig) *PredictionState {
	debounceDelay := config.DebounceDelay
	if debounceDelay == 0 {
		debounceDelay = 200 * time.Millisecond
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &PredictionState{
		debounceDelay: debounceDelay,
		provider:      config.Provider,
		logger:        logger,
	}
}

// Prediction returns the current prediction text.
func (ps *PredictionState) Prediction() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.prediction
}

// HasPrediction returns true if there is a current prediction.
func (ps *PredictionState) HasPrediction() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.prediction != ""
}

// StateID returns the current state ID.
func (ps *PredictionState) StateID() int64 {
	return ps.stateID.Load()
}

// IsDirty returns whether the input has been modified.
func (ps *PredictionState) IsDirty() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.dirty
}

// Clear clears the current prediction.
func (ps *PredictionState) Clear() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.prediction = ""
	ps.inputForPrediction = ""
	ps.cancelPendingLocked()
}

// Reset clears all state including dirty flag.
func (ps *PredictionState) Reset() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.prediction = ""
	ps.inputForPrediction = ""
	ps.dirty = false
	ps.cancelPendingLocked()
}

// cancelPendingLocked cancels any pending prediction request.
// Must be called with mu held.
func (ps *PredictionState) cancelPendingLocked() {
	if ps.cancelPending != nil {
		ps.cancelPending()
		ps.cancelPending = nil
	}
}

// SetPrediction sets the prediction if the state ID matches.
// Returns true if the prediction was set.
func (ps *PredictionState) SetPrediction(stateID int64, prediction string) bool {
	if ps.stateID.Load() != stateID {
		ps.logger.Debug("discarding stale prediction",
			zap.Int64("expectedStateID", ps.stateID.Load()),
			zap.Int64("actualStateID", stateID),
		)
		return false
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.prediction = prediction
	return true
}

// OnInputChanged should be called when the input text changes.
// It returns a channel that will receive the prediction result, or nil if
// no prediction should be made (e.g., prediction already matches input).
// The caller should handle the result asynchronously.
//
// Instant predictions (via InstantProvider or deprecated HistoryProvider) are checked
// synchronously for instant feedback, while debounced predictions are delayed to avoid spam.
func (ps *PredictionState) OnInputChanged(input string) <-chan PredictionResult {
	ps.mu.Lock()

	// Mark as dirty if there's any input
	if len(input) > 0 {
		ps.dirty = true
	}

	// Cancel any pending prediction
	ps.cancelPendingLocked()

	// Increment state ID
	newStateID := ps.stateID.Add(1)

	// If input is empty and we were dirty, clear prediction
	if len(input) == 0 && ps.dirty {
		ps.prediction = ""
		ps.inputForPrediction = ""
		ps.mu.Unlock()

		// Still need to potentially get a null-state prediction (debounced only)
		return ps.startDebouncedPrediction(newStateID, input)
	}

	// If current prediction already starts with input, keep it
	if len(input) > 0 && strings.HasPrefix(ps.prediction, input) {
		ps.logger.Debug("keeping existing prediction",
			zap.String("input", input),
			zap.String("prediction", ps.prediction),
		)
		ps.mu.Unlock()
		return nil
	}

	// Clear current prediction
	ps.prediction = ""
	ps.mu.Unlock()

	// Try instant prediction synchronously
	if ps.provider != nil {
		prediction, err := ps.provider.Predict(context.Background(), input, interpreter.PredictTriggerInstant)
		if err == nil && prediction != "" {
			ps.logger.Debug("instant prediction",
				zap.String("input", input),
				zap.String("prediction", prediction),
			)

			// Set prediction immediately
			ps.mu.Lock()
			ps.prediction = prediction
			ps.inputForPrediction = input
			ps.mu.Unlock()

			// Return result synchronously via a pre-filled channel
			resultCh := make(chan PredictionResult, 1)
			resultCh <- PredictionResult{
				Prediction: prediction,
				StateID:    newStateID,
				Source:     PredictionSourceHistory, // Instant predictions are typically from history
			}
			close(resultCh)
			return resultCh
		}
	}

	// No instant match, fall back to debounced prediction
	return ps.startDebouncedPrediction(newStateID, input)
}

// startDebouncedPrediction starts a debounced async prediction request.
// Instant predictions are handled synchronously in OnInputChanged,
// so this function handles debounced predictions.
func (ps *PredictionState) startDebouncedPrediction(stateID int64, input string) <-chan PredictionResult {
	// If no provider, nothing to do
	if ps.provider == nil {
		return nil
	}

	resultCh := make(chan PredictionResult, 1)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	ps.mu.Lock()
	ps.cancelPending = cancel
	ps.mu.Unlock()

	go func() {
		defer close(resultCh)

		// Debounce calls to avoid spam
		select {
		case <-ctx.Done():
			return
		case <-time.After(ps.debounceDelay):
		}

		// Check if state is still valid
		if ps.stateID.Load() != stateID {
			return
		}

		// Make debounced prediction
		result := ps.predictDebounced(ctx, stateID, input)

		// Send result if still valid
		if ps.stateID.Load() == stateID {
			select {
			case resultCh <- result:
			case <-ctx.Done():
			}
		}
	}()

	return resultCh
}

// predictDebounced makes a prediction using the provider with debounced trigger.
// Instant predictions are handled synchronously in OnInputChanged.
func (ps *PredictionState) predictDebounced(ctx context.Context, stateID int64, input string) PredictionResult {
	result := PredictionResult{
		StateID: stateID,
		Source:  PredictionSourceNone,
	}

	// Don't predict for agent chat messages
	if strings.HasPrefix(input, "#") {
		return result
	}

	if ps.provider == nil {
		return result
	}

	prediction, err := ps.provider.Predict(ctx, input, interpreter.PredictTriggerDebounced)
	if err != nil {
		ps.logger.Debug("debounced prediction failed", zap.Error(err))
		result.Error = err
		return result
	}

	if prediction != "" {
		result.Prediction = prediction
		result.Source = PredictionSourceLLM // Debounced predictions are typically from LLM

		ps.logger.Debug("debounced prediction",
			zap.String("input", input),
			zap.String("prediction", prediction),
		)
	}

	return result
}

// PredictionSuggestion returns the prediction as a suggestion string.
// If the prediction starts with the input, only returns the suffix.
func (ps *PredictionState) PredictionSuggestion(input string) string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.prediction == "" {
		return ""
	}

	if strings.HasPrefix(ps.prediction, input) {
		return ps.prediction[len(input):]
	}

	return ps.prediction
}

// LLMPredictionProvider implements PredictionProvider using an LLM model.
type LLMPredictionProvider struct {
	model    *interpreter.ModelValue
	provider interpreter.ModelProvider
	logger   *zap.Logger

	// Context text for predictions (e.g., cwd, git status)
	contextText   string
	contextTextMu sync.RWMutex
}

// NewLLMPredictionProvider creates a new LLM prediction provider.
func NewLLMPredictionProvider(
	model *interpreter.ModelValue,
	provider interpreter.ModelProvider,
	logger *zap.Logger,
) *LLMPredictionProvider {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LLMPredictionProvider{
		model:    model,
		provider: provider,
		logger:   logger,
	}
}

// UpdateContext updates the context text used for predictions.
func (p *LLMPredictionProvider) UpdateContext(contextText string) {
	p.contextTextMu.Lock()
	defer p.contextTextMu.Unlock()
	p.contextText = contextText
}

// predictedCommandResponse is the expected JSON response from the LLM.
type predictedCommandResponse struct {
	PredictedCommand string `json:"predicted_command"`
}

// Predict implements PredictionProvider.
// Note: LLMPredictionProvider ignores the trigger parameter as it always does LLM-based prediction.
func (p *LLMPredictionProvider) Predict(ctx context.Context, input string, trigger interpreter.PredictTrigger) (string, error) {
	if p.model == nil || p.provider == nil {
		return "", nil
	}

	p.contextTextMu.RLock()
	contextText := p.contextText
	p.contextTextMu.RUnlock()

	var userMessage string
	if input == "" {
		// Null-state prediction
		userMessage = fmt.Sprintf(`You are gsh, an intelligent shell program.
You are asked to predict the next command I'm likely to want to run.

# Instructions
* Based on the context, analyze my potential intent
* Your prediction must be a valid, single-line, complete bash command

# Latest Context
%s

Respond with JSON in this format: {"predicted_command": "your prediction here"}

Now predict what my next command should be.`, contextText)
	} else {
		// Prefix-based prediction
		userMessage = fmt.Sprintf(`You are gsh, an intelligent shell program.
You will be given a partial bash command prefix entered by me, enclosed in <prefix> tags.
You are asked to predict what the complete bash command is.

# Instructions
* Based on the prefix and other context, analyze my potential intent
* Your prediction must start with the partial command as a prefix
* Your prediction must be a valid, single-line, complete bash command

# Latest Context
%s

Respond with JSON in this format: {"predicted_command": "your prediction here"}

<prefix>%s</prefix>`, contextText, input)
	}

	p.logger.Debug("predicting using LLM", zap.String("userMessage", userMessage))

	request := interpreter.ChatRequest{
		Model: p.model,
		Messages: []interpreter.ChatMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	response, err := p.provider.ChatCompletion(ctx, request)
	if err != nil {
		return "", err
	}

	// Parse JSON response
	var prediction predictedCommandResponse
	if err := json.Unmarshal([]byte(response.Content), &prediction); err != nil {
		// Try to extract from response directly if JSON parsing fails
		p.logger.Debug("failed to parse prediction JSON", zap.Error(err), zap.String("content", response.Content))
		return "", nil
	}

	p.logger.Debug("LLM prediction response", zap.String("prediction", prediction.PredictedCommand))

	return prediction.PredictedCommand, nil
}
