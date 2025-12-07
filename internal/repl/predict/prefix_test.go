package predict

import (
	"context"
	"errors"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockModelProvider implements interpreter.ModelProvider for testing.
type mockModelProvider struct {
	response *interpreter.ChatResponse
	err      error
	requests []interpreter.ChatRequest
}

func (m *mockModelProvider) Name() string {
	return "mock"
}

func (m *mockModelProvider) ChatCompletion(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
	m.requests = append(m.requests, request)
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewPrefixPredictor(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		predictor := NewPrefixPredictor(PrefixPredictorConfig{})
		assert.NotNil(t, predictor)
		assert.NotNil(t, predictor.logger)
		assert.NotNil(t, predictor.formatter)
		assert.Nil(t, predictor.model)
		assert.Nil(t, predictor.provider)
	})

	t.Run("with model and provider", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		assert.Equal(t, model, predictor.model)
		assert.Equal(t, provider, predictor.provider)
	})

	t.Run("with custom formatter", func(t *testing.T) {
		formatter := &DefaultContextFormatter{}
		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Formatter: formatter,
		})
		assert.Equal(t, formatter, predictor.formatter)
	})
}

func TestPrefixPredictor_UpdateContext(t *testing.T) {
	predictor := NewPrefixPredictor(PrefixPredictorConfig{})

	predictor.UpdateContext(map[string]string{
		"cwd": "/home/user",
		"git": "branch: main",
	})

	predictor.contextTextMu.RLock()
	defer predictor.contextTextMu.RUnlock()

	assert.Contains(t, predictor.contextText, "cwd")
	assert.Contains(t, predictor.contextText, "/home/user")
	assert.Contains(t, predictor.contextText, "git")
	assert.Contains(t, predictor.contextText, "branch: main")
}

func TestPrefixPredictor_Predict(t *testing.T) {
	t.Run("empty input returns empty", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Len(t, provider.requests, 0) // No API call made
	})

	t.Run("agent chat prefix skipped", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "#hello")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
		assert.Len(t, provider.requests, 0) // No API call made
	})

	t.Run("nil model returns empty", func(t *testing.T) {
		predictor := NewPrefixPredictor(PrefixPredictorConfig{})

		result, err := predictor.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("successful prediction", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "git status"}`,
			},
		}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "git status", result)
		require.Len(t, provider.requests, 1)
		assert.Contains(t, provider.requests[0].Messages[0].Content, "git")
	})

	t.Run("prediction with context", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "git push origin main"}`,
			},
		}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		predictor.UpdateContext(map[string]string{
			"git": "branch: main",
		})

		result, err := predictor.Predict(context.Background(), "git push")
		assert.NoError(t, err)
		assert.Equal(t, "git push origin main", result)
		require.Len(t, provider.requests, 1)
		assert.Contains(t, provider.requests[0].Messages[0].Content, "branch: main")
	})

	t.Run("provider error", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			err: errors.New("API error"),
		}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		_, err := predictor.Predict(context.Background(), "git")
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

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("prediction does not match prefix", func(t *testing.T) {
		model := &interpreter.ModelValue{Name: "test-model"}
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "ls -la"}`, // Doesn't start with "git"
			},
		}

		predictor := NewPrefixPredictor(PrefixPredictorConfig{
			Model:    model,
			Provider: provider,
		})

		result, err := predictor.Predict(context.Background(), "git")
		assert.NoError(t, err)
		assert.Equal(t, "", result) // Discarded because doesn't match prefix
	})
}
