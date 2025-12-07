package predict

import (
	"context"

	"go.uber.org/zap"
)

// Router coordinates between different prediction strategies.
// It routes prediction requests to the appropriate predictor based on the input state.
type Router struct {
	prefixPredictor    Predictor
	nullStatePredictor Predictor
	logger             *zap.Logger
}

// RouterConfig holds configuration for creating a Router.
type RouterConfig struct {
	// PrefixPredictor handles predictions when there is input text.
	PrefixPredictor Predictor

	// NullStatePredictor handles predictions when input is empty.
	NullStatePredictor Predictor

	// Logger for debug output. If nil, a no-op logger is used.
	Logger *zap.Logger
}

// NewRouter creates a new Router with the given configuration.
func NewRouter(cfg RouterConfig) *Router {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Router{
		prefixPredictor:    cfg.PrefixPredictor,
		nullStatePredictor: cfg.NullStatePredictor,
		logger:             logger,
	}
}

// Predict routes the prediction request to the appropriate predictor.
// If input is empty, uses null state predictor; otherwise uses prefix predictor.
func (r *Router) Predict(ctx context.Context, input string) (string, error) {
	if input == "" {
		if r.nullStatePredictor == nil {
			return "", nil
		}
		r.logger.Debug("routing to null state predictor")
		return r.nullStatePredictor.Predict(ctx, input)
	}

	if r.prefixPredictor == nil {
		return "", nil
	}
	r.logger.Debug("routing to prefix predictor", zap.String("input", input))
	return r.prefixPredictor.Predict(ctx, input)
}

// UpdateContext updates the context for all predictors.
func (r *Router) UpdateContext(contextMap map[string]string) {
	if r.prefixPredictor != nil {
		r.prefixPredictor.UpdateContext(contextMap)
	}
	if r.nullStatePredictor != nil {
		r.nullStatePredictor.UpdateContext(contextMap)
	}
}

// PrefixPredictor returns the prefix predictor (for testing).
func (r *Router) PrefixPredictor() Predictor {
	return r.prefixPredictor
}

// NullStatePredictor returns the null state predictor (for testing).
func (r *Router) NullStatePredictor() Predictor {
	return r.nullStatePredictor
}
