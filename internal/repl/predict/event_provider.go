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

// Predict emits the repl.predict event with the specified trigger and parses the middleware response.
// If middleware returns no prediction or an error, an empty string is returned.
// Implements PredictionProvider interface.
func (p *EventPredictionProvider) Predict(ctx context.Context, input string, trigger interpreter.PredictTrigger, existingPrediction string) (string, error) {
	p.logger.Debug("prediction request",
		zap.String("input", input),
		zap.String("trigger", string(trigger)),
		zap.String("existingPrediction", existingPrediction),
	)

	prediction, err := p.emitPredictEvent(ctx, input, trigger, existingPrediction)
	if err != nil {
		p.logger.Debug("prediction failed",
			zap.String("input", input),
			zap.String("trigger", string(trigger)),
			zap.Error(err),
		)
		return "", nil
	}

	p.logger.Debug("prediction result",
		zap.String("input", input),
		zap.String("trigger", string(trigger)),
		zap.String("prediction", prediction),
	)
	return prediction, nil
}

func (p *EventPredictionProvider) emitPredictEvent(ctx context.Context, input string, trigger interpreter.PredictTrigger, existingPrediction string) (string, error) {
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

	val := p.interp.EmitEvent(interpreter.EventReplPredict, interpreter.CreateReplPredictContext(input, trigger, existingPrediction))
	prediction, err, _ := interpreter.ExtractPredictionResult(val)
	return prediction, err
}
