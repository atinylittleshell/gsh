package predict

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
