package predict

import (
	"context"
	"errors"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPredictor implements Predictor for testing.
type mockPredictor struct {
	prediction string
	err        error
	callCount  int
	lastInput  string
	contextMap map[string]string
}

func (m *mockPredictor) Predict(ctx context.Context, input string) (string, error) {
	m.callCount++
	m.lastInput = input
	if m.err != nil {
		return "", m.err
	}
	return m.prediction, nil
}

func (m *mockPredictor) UpdateContext(contextMap map[string]string) {
	m.contextMap = contextMap
}

func TestNewRouter(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		router := NewRouter(RouterConfig{})
		assert.NotNil(t, router)
		assert.NotNil(t, router.logger)
		assert.Nil(t, router.prefixPredictor)
		assert.Nil(t, router.nullStatePredictor)
	})

	t.Run("with predictors", func(t *testing.T) {
		prefixPredictor := &mockPredictor{}
		nullStatePredictor := &mockPredictor{}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    prefixPredictor,
			NullStatePredictor: nullStatePredictor,
		})

		assert.Equal(t, prefixPredictor, router.prefixPredictor)
		assert.Equal(t, nullStatePredictor, router.nullStatePredictor)
	})
}

func TestRouter_Predict(t *testing.T) {
	t.Run("routes to null state predictor for empty input", func(t *testing.T) {
		nullStatePredictor := &mockPredictor{prediction: "ls -la"}
		prefixPredictor := &mockPredictor{prediction: "git status"}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    prefixPredictor,
			NullStatePredictor: nullStatePredictor,
		})

		result, err := router.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "ls -la", result)
		assert.Equal(t, 1, nullStatePredictor.callCount)
		assert.Equal(t, 0, prefixPredictor.callCount)
	})

	t.Run("routes to prefix predictor for non-empty input", func(t *testing.T) {
		nullStatePredictor := &mockPredictor{prediction: "ls -la"}
		prefixPredictor := &mockPredictor{prediction: "git status"}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    prefixPredictor,
			NullStatePredictor: nullStatePredictor,
		})

		result, err := router.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "git status", result)
		assert.Equal(t, 0, nullStatePredictor.callCount)
		assert.Equal(t, 1, prefixPredictor.callCount)
		assert.Equal(t, "git", prefixPredictor.lastInput)
	})

	t.Run("nil null state predictor returns empty for empty input", func(t *testing.T) {
		prefixPredictor := &mockPredictor{prediction: "git status"}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    prefixPredictor,
			NullStatePredictor: nil,
		})

		result, err := router.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("nil prefix predictor returns empty for non-empty input", func(t *testing.T) {
		nullStatePredictor := &mockPredictor{prediction: "ls -la"}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    nil,
			NullStatePredictor: nullStatePredictor,
		})

		result, err := router.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("propagates prefix predictor error", func(t *testing.T) {
		prefixPredictor := &mockPredictor{err: errors.New("prediction error")}

		router := NewRouter(RouterConfig{
			PrefixPredictor: prefixPredictor,
		})

		_, err := router.Predict(context.Background(), "git")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prediction error")
	})

	t.Run("propagates null state predictor error", func(t *testing.T) {
		nullStatePredictor := &mockPredictor{err: errors.New("prediction error")}

		router := NewRouter(RouterConfig{
			NullStatePredictor: nullStatePredictor,
		})

		_, err := router.Predict(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prediction error")
	})
}

func TestRouter_UpdateContext(t *testing.T) {
	t.Run("updates both predictors", func(t *testing.T) {
		prefixPredictor := &mockPredictor{}
		nullStatePredictor := &mockPredictor{}

		router := NewRouter(RouterConfig{
			PrefixPredictor:    prefixPredictor,
			NullStatePredictor: nullStatePredictor,
		})

		contextMap := map[string]string{
			"cwd": "/home/user",
			"git": "branch: main",
		}

		router.UpdateContext(contextMap)

		assert.Equal(t, contextMap, prefixPredictor.contextMap)
		assert.Equal(t, contextMap, nullStatePredictor.contextMap)
	})

	t.Run("handles nil predictors", func(t *testing.T) {
		router := NewRouter(RouterConfig{})

		// Should not panic
		router.UpdateContext(map[string]string{
			"cwd": "/home/user",
		})
	})

	t.Run("handles partial nil predictors", func(t *testing.T) {
		prefixPredictor := &mockPredictor{}

		router := NewRouter(RouterConfig{
			PrefixPredictor: prefixPredictor,
		})

		contextMap := map[string]string{
			"cwd": "/home/user",
		}

		router.UpdateContext(contextMap)
		assert.Equal(t, contextMap, prefixPredictor.contextMap)
	})
}

func TestRouter_Accessors(t *testing.T) {
	prefixPredictor := &mockPredictor{}
	nullStatePredictor := &mockPredictor{}

	router := NewRouter(RouterConfig{
		PrefixPredictor:    prefixPredictor,
		NullStatePredictor: nullStatePredictor,
	})

	assert.Equal(t, prefixPredictor, router.PrefixPredictor())
	assert.Equal(t, nullStatePredictor, router.NullStatePredictor())
}

func TestNewRouterFromConfig(t *testing.T) {
	t.Run("returns nil when model resolver is nil", func(t *testing.T) {
		router := NewRouterFromConfig(nil, nil)
		assert.Nil(t, router)
	})

	t.Run("creates router with model resolver", func(t *testing.T) {
		model := &interpreter.ModelValue{
			Name:     "testModel",
			Provider: interpreter.NewOpenAIProvider(),
			Config: map[string]interpreter.Value{
				"provider": &interpreter.StringValue{Value: "openai"},
				"model":    &interpreter.StringValue{Value: "gpt-4o-mini"},
				"apiKey":   &interpreter.StringValue{Value: "test-key"},
			},
		}

		router := NewRouterFromConfig(model, nil)
		require.NotNil(t, router)
		assert.NotNil(t, router.PrefixPredictor())
		assert.NotNil(t, router.NullStatePredictor())
	})

	t.Run("creates router with SDKModelRef", func(t *testing.T) {
		models := &interpreter.Models{
			Lite: &interpreter.ModelValue{Name: "liteModel"},
		}
		modelRef := &interpreter.SDKModelRef{Tier: "lite", Models: models}

		router := NewRouterFromConfig(modelRef, nil)
		require.NotNil(t, router)
		assert.NotNil(t, router.PrefixPredictor())
		assert.NotNil(t, router.NullStatePredictor())
	})

	t.Run("SDKModelRef enables dynamic model resolution", func(t *testing.T) {
		// Create initial model
		initialModel := &interpreter.ModelValue{
			Name:     "initialModel",
			Provider: interpreter.NewOpenAIProvider(),
			Config: map[string]interpreter.Value{
				"provider": &interpreter.StringValue{Value: "openai"},
				"model":    &interpreter.StringValue{Value: "gpt-4o-mini"},
				"apiKey":   &interpreter.StringValue{Value: "test-key"},
			},
		}

		// Set up models registry
		models := &interpreter.Models{
			Lite: initialModel,
		}

		// Create router with SDKModelRef that holds reference to the models registry
		modelRef := &interpreter.SDKModelRef{Tier: "lite", Models: models}
		router := NewRouterFromConfig(modelRef, nil)
		require.NotNil(t, router)

		// Verify the SDKModelRef resolves to the initial model
		resolved := modelRef.GetModel()
		assert.Equal(t, initialModel, resolved)

		// Change the model in the tier
		newModel := &interpreter.ModelValue{
			Name:     "newModel",
			Provider: interpreter.NewOpenAIProvider(),
			Config: map[string]interpreter.Value{
				"provider": &interpreter.StringValue{Value: "openai"},
				"model":    &interpreter.StringValue{Value: "gpt-4-turbo"},
				"apiKey":   &interpreter.StringValue{Value: "test-key"},
			},
		}
		models.Lite = newModel

		// The same SDKModelRef should now resolve to the new model
		resolvedAfterChange := modelRef.GetModel()
		assert.Equal(t, newModel, resolvedAfterChange)
		assert.NotEqual(t, resolved, resolvedAfterChange, "SDKModelRef should dynamically resolve to the new model")
	})
}
