package predict

import (
	"context"
	"sync"

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
func (p *EventPredictionProvider) Predict(ctx context.Context, input string) (string, error) {
	prediction, err := p.emitPredictEvent(ctx, input)
	if err != nil {
		p.logger.Warn("prediction middleware returned error", zap.Error(err))
		return "", nil
	}
	return prediction, nil
}

func (p *EventPredictionProvider) emitPredictEvent(ctx context.Context, input string) (string, error) {
	if p.interp == nil {
		return "", nil
	}

	p.mu.Lock()
	// Ensure middleware sees the cancellable context used by PredictionState
	p.interp.SetContext(ctx)
	defer func() {
		p.interp.SetContext(context.Background())
		p.mu.Unlock()
	}()

	val := p.interp.EmitEvent(interpreter.EventReplPredict, interpreter.CreateReplPredictContext(input))
	prediction, err, _ := interpreter.ExtractPredictionResult(val)
	return prediction, err
}
