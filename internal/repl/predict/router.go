package predict

import (
	"context"

	"github.com/atinylittleshell/gsh/internal/repl/config"
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

// NewRouterFromConfig creates a Router configured from the REPL config.
// It sets up the prefix and null-state predictors using the model specified
// in GSH_CONFIG.predictModel.
// Returns nil if no prediction model is configured.
func NewRouterFromConfig(cfg *config.Config, logger *zap.Logger) *Router {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Get the prediction model from config
	model := cfg.GetPredictModel()
	if model == nil {
		logger.Debug("no prediction model configured, LLM predictions disabled (history-based predictions may still work)")
		return nil
	}

	// Verify the model has a provider
	if model.Provider == nil {
		logger.Warn("prediction model has no provider",
			zap.String("model", cfg.PredictModel))
		return nil
	}

	logger.Debug("creating prediction router",
		zap.String("model", cfg.PredictModel),
		zap.String("provider", model.Provider.Name()))

	// Create prefix predictor
	prefixPredictor := NewPrefixPredictor(PrefixPredictorConfig{
		Model:  model,
		Logger: logger,
	})

	// Create null-state predictor
	nullStatePredictor := NewNullStatePredictor(NullStatePredictorConfig{
		Model:  model,
		Logger: logger,
	})

	return NewRouter(RouterConfig{
		PrefixPredictor:    prefixPredictor,
		NullStatePredictor: nullStatePredictor,
		Logger:             logger,
	})
}
