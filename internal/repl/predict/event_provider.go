package predict

import (
	"context"
	"sync"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
)

// EventPredictionProvider calls the repl.predict middleware chain to obtain predictions.
// If no middleware is registered or middleware returns null, no prediction is returned.
type EventPredictionProvider struct {
	interp *interpreter.Interpreter
	logger *zap.Logger

	mu sync.Mutex
}

// NewEventPredictionProvider creates a new EventPredictionProvider.
func NewEventPredictionProvider(
	interp *interpreter.Interpreter,
	logger *zap.Logger,
) *EventPredictionProvider {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &EventPredictionProvider{
		interp: interp,
		logger: logger,
	}
}

// Predict emits the repl.predict event and parses the middleware response.
// If middleware returns no prediction or an error, an empty string is returned.
func (p *EventPredictionProvider) Predict(ctx context.Context, request input.PredictionRequest) (input.PredictionResponse, error) {
	prediction, err := p.emitPredictEvent(ctx, request)
	if err != nil {
		p.logger.Warn("prediction middleware returned error", zap.Error(err))
		return input.PredictionResponse{}, nil
	}
	return prediction, nil
}

func (p *EventPredictionProvider) emitPredictEvent(ctx context.Context, request input.PredictionRequest) (input.PredictionResponse, error) {
	if p.interp == nil {
		return input.PredictionResponse{}, nil
	}

	p.mu.Lock()
	// Ensure middleware sees the cancellable context used by PredictionState
	p.interp.SetContext(ctx)
	defer func() {
		p.interp.SetContext(context.Background())
		p.mu.Unlock()
	}()

	ctxVal := interpreter.CreateReplPredictContext(
		request.Input,
		convertHistoryItems(request.History),
		request.Source.String(),
	)

	val := p.interp.EmitEvent(interpreter.EventReplPredict, ctxVal)
	prediction, err, _ := interpreter.ExtractPredictionResult(val)

	response := input.PredictionResponse{
		Prediction: prediction.Prediction,
		Source:     input.ParsePredictionSource(prediction.Source),
	}
	if response.Source == input.PredictionSourceNone && request.Source != input.PredictionSourceNone {
		response.Source = request.Source
	}

	return response, err
}

func convertHistoryItems(entries []history.HistoryEntry) []interpreter.PredictionHistoryItem {
	items := make([]interpreter.PredictionHistoryItem, 0, len(entries))
	for _, entry := range entries {
		var exitCode *int32
		if entry.ExitCode.Valid {
			val := entry.ExitCode.Int32
			exitCode = &val
		}

		items = append(items, interpreter.PredictionHistoryItem{
			Command:   entry.Command,
			Directory: entry.Directory,
			ExitCode:  exitCode,
		})
	}
	return items
}
