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

	"github.com/atinylittleshell/gsh/internal/history"
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
// This abstraction allows for different prediction backends (history, LLM, etc.)
type PredictionProvider interface {
	// Predict returns a prediction for the given input.
	// The context can be used for cancellation.
	Predict(ctx context.Context, input string) (prediction string, err error)
}

// HistoryProvider defines the interface for history-based predictions.
type HistoryProvider interface {
	// GetRecentEntriesByPrefix returns history entries matching the given prefix.
	GetRecentEntriesByPrefix(prefix string, limit int) ([]history.HistoryEntry, error)
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

	// Providers
	historyProvider HistoryProvider
	llmProvider     PredictionProvider
	logger          *zap.Logger

	// Number of history entries to check for prefix matches
	historyPrefixLimit int

	// Pending prediction cancel function
	cancelPending context.CancelFunc
}

// PredictionStateConfig holds configuration for creating a PredictionState.
type PredictionStateConfig struct {
	// DebounceDelay is the delay before making a prediction request.
	// Defaults to 200ms if not set.
	DebounceDelay time.Duration

	// HistoryProvider provides history-based predictions.
	HistoryProvider HistoryProvider

	// LLMProvider provides LLM-based predictions.
	LLMProvider PredictionProvider

	// Logger for debug output.
	Logger *zap.Logger

	// HistoryPrefixLimit is the number of history entries to check.
	// Defaults to 10 if not set.
	HistoryPrefixLimit int
}

// NewPredictionState creates a new PredictionState with the given configuration.
func NewPredictionState(config PredictionStateConfig) *PredictionState {
	debounceDelay := config.DebounceDelay
	if debounceDelay == 0 {
		debounceDelay = 200 * time.Millisecond
	}

	historyLimit := config.HistoryPrefixLimit
	if historyLimit == 0 {
		historyLimit = 10
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &PredictionState{
		debounceDelay:      debounceDelay,
		historyProvider:    config.HistoryProvider,
		llmProvider:        config.LLMProvider,
		logger:             logger,
		historyPrefixLimit: historyLimit,
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
// History-based predictions are checked synchronously for instant feedback,
// while LLM predictions are debounced to avoid API spam.
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

		// Still need to potentially get a null-state prediction (LLM only)
		return ps.startLLMPrediction(newStateID, input)
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

	// Don't predict for agent chat messages
	if strings.HasPrefix(input, "#") {
		return nil
	}

	// Try history-based prediction synchronously (instant feedback)
	if input != "" && ps.historyProvider != nil {
		entries, err := ps.historyProvider.GetRecentEntriesByPrefix(input, ps.historyPrefixLimit)
		if err == nil && len(entries) > 0 {
			// Use the most recent matching entry
			historyPrediction := entries[0].Command

			ps.logger.Debug("instant history prediction",
				zap.String("input", input),
				zap.String("prediction", historyPrediction),
			)

			// Set prediction immediately
			ps.mu.Lock()
			ps.prediction = historyPrediction
			ps.inputForPrediction = input
			ps.mu.Unlock()

			// Return result synchronously via a pre-filled channel
			resultCh := make(chan PredictionResult, 1)
			resultCh <- PredictionResult{
				Prediction: historyPrediction,
				StateID:    newStateID,
				Source:     PredictionSourceHistory,
			}
			close(resultCh)
			return resultCh
		}
	}

	// No history match, fall back to debounced LLM prediction
	return ps.startLLMPrediction(newStateID, input)
}

// startLLMPrediction starts a debounced async LLM prediction request.
// History-based predictions are handled synchronously in OnInputChanged,
// so this function only handles LLM fallback predictions.
func (ps *PredictionState) startLLMPrediction(stateID int64, input string) <-chan PredictionResult {
	// If no LLM provider, nothing to do
	if ps.llmProvider == nil {
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

		// Debounce LLM calls to avoid API spam
		select {
		case <-ctx.Done():
			return
		case <-time.After(ps.debounceDelay):
		}

		// Check if state is still valid
		if ps.stateID.Load() != stateID {
			return
		}

		// Make LLM prediction
		result := ps.predictLLM(ctx, stateID, input)

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

// predictLLM makes a prediction using the LLM provider.
// History-based predictions are handled synchronously in OnInputChanged.
func (ps *PredictionState) predictLLM(ctx context.Context, stateID int64, input string) PredictionResult {
	result := PredictionResult{
		StateID: stateID,
		Source:  PredictionSourceNone,
	}

	// Don't predict for agent chat messages
	if strings.HasPrefix(input, "#") {
		return result
	}

	if ps.llmProvider == nil {
		return result
	}

	prediction, err := ps.llmProvider.Predict(ctx, input)
	if err != nil {
		ps.logger.Debug("LLM prediction failed", zap.Error(err))
		result.Error = err
		return result
	}

	if prediction != "" {
		result.Prediction = prediction
		result.Source = PredictionSourceLLM

		ps.logger.Debug("LLM prediction",
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
func (p *LLMPredictionProvider) Predict(ctx context.Context, input string) (string, error) {
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

	response, err := p.provider.ChatCompletion(request)
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

// HistoryPredictionAdapter adapts a history.HistoryManager to the HistoryProvider interface.
type HistoryPredictionAdapter struct {
	manager *history.HistoryManager
}

// NewHistoryPredictionAdapter creates a new adapter for the history manager.
func NewHistoryPredictionAdapter(manager *history.HistoryManager) *HistoryPredictionAdapter {
	return &HistoryPredictionAdapter{manager: manager}
}

// GetRecentEntriesByPrefix implements HistoryProvider.
func (a *HistoryPredictionAdapter) GetRecentEntriesByPrefix(prefix string, limit int) ([]history.HistoryEntry, error) {
	if a.manager == nil {
		return nil, nil
	}
	return a.manager.GetRecentEntriesByPrefix(prefix, limit)
}
