package predict

import (
	"context"
	"errors"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNullStatePredictor(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		predictor := NewNullStatePredictor(NullStatePredictorConfig{})
		assert.NotNil(t, predictor)
		assert.NotNil(t, predictor.logger)
		assert.NotNil(t, predictor.formatter)
		assert.Nil(t, predictor.model)
		assert.Nil(t, predictor.provider)
	})

	t.Run("with model and provider", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		assert.Equal(t, model, predictor.model)
		assert.Equal(t, provider, predictor.provider)
	})

	t.Run("with custom formatter", func(t *testing.T) {
		formatter := &DefaultContextFormatter{}
		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Formatter: formatter,
		})
		assert.Equal(t, formatter, predictor.formatter)
	})
}

func TestNullStatePredictor_UpdateContext(t *testing.T) {
	predictor := NewNullStatePredictor(NullStatePredictorConfig{})

	predictor.UpdateContext(map[string]string{
		"cwd":     "/home/user/project",
		"history": "git status\ngit add .",
	})

	predictor.contextTextMu.RLock()
	defer predictor.contextTextMu.RUnlock()

	assert.Contains(t, predictor.contextText, "cwd")
	assert.Contains(t, predictor.contextText, "/home/user/project")
	assert.Contains(t, predictor.contextText, "history")
}

func TestNullStatePredictor_Predict(t *testing.T) {
	t.Run("non-empty input returns empty", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Len(t, provider.requests, 0) // No API call made
	})

	t.Run("nil model returns empty", func(t *testing.T) {
		predictor := NewNullStatePredictor(NullStatePredictorConfig{})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("successful prediction", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "ls -la"}`,
			},
		}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "ls -la", result)
		require.Len(t, provider.requests, 1)
	})

	t.Run("prediction with context", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "git commit -m 'fix: bug'"}`,
			},
		}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		predictor.UpdateContext(map[string]string{
			"git": "modified: main.go",
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "git commit -m 'fix: bug'", result)
		require.Len(t, provider.requests, 1)
		assert.Contains(t, provider.requests[0].Messages[0].Content, "modified: main.go")
	})

	t.Run("provider error", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			err: errors.New("API error"),
		}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		_, err := predictor.Predict(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: "not valid json",
			},
		}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})
}
