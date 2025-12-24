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
	})

	t.Run("with model", func(t *testing.T) {
		provider := &mockModelProvider{}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		assert.Equal(t, model, predictor.model)
		assert.Equal(t, provider, predictor.model.Provider)
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
		provider := &mockModelProvider{}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
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
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "ls -la"}`,
			},
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "ls -la", result)
		require.Len(t, provider.requests, 1)
	})

	t.Run("prediction with context", func(t *testing.T) {
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "git commit -m 'fix: bug'"}`,
			},
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
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
		provider := &mockModelProvider{
			err: errors.New("API error"),
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		_, err := predictor.Predict(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: "not valid json",
			},
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("JSON wrapped in markdown code block", func(t *testing.T) {
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: "```json\n{\"predicted_command\": \"git commit -m \\\"Fix: Update .gitignore\\\"\"}\n```\n",
			},
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "git commit -m \"Fix: Update .gitignore\"", result)
	})

	t.Run("JSON wrapped in plain code block", func(t *testing.T) {
		provider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: "```\n{\"predicted_command\": \"ls -la\"}\n```",
			},
		}
		model := &interpreter.ModelValue{Name: "test-model", Provider: provider}

		predictor := NewNullStatePredictor(NullStatePredictorConfig{
			Model: model,
		})

		result, err := predictor.Predict(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, "ls -la", result)
	})
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"predicted_command": "ls -la"}`,
			expected: `{"predicted_command": "ls -la"}`,
		},
		{
			name:     "JSON with whitespace",
			input:    "  \n  {\"predicted_command\": \"ls -la\"}  \n  ",
			expected: `{"predicted_command": "ls -la"}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"predicted_command\": \"git status\"}\n```",
			expected: `{"predicted_command": "git status"}`,
		},
		{
			name:     "JSON in plain code block",
			input:    "```\n{\"predicted_command\": \"ls -la\"}\n```",
			expected: `{"predicted_command": "ls -la"}`,
		},
		{
			name:     "JSON in markdown code block with trailing text",
			input:    "```json\n{\"predicted_command\": \"git commit\"}\n```\nSome explanation text",
			expected: `{"predicted_command": "git commit"}`,
		},
		{
			name:     "JSON in code block with escaped quotes",
			input:    "```json\n{\"predicted_command\": \"git commit -m \\\"Fix: Update\\\"\"}\n```\n",
			expected: `{"predicted_command": "git commit -m \"Fix: Update\""}`,
		},
		{
			name:     "multiline JSON in code block",
			input:    "```json\n{\n  \"predicted_command\": \"ls -la\"\n}\n```",
			expected: "{\n  \"predicted_command\": \"ls -la\"\n}",
		},
		{
			name:     "JSON containing triple backticks in string value",
			input:    "```json\n{\"predicted_command\": \"echo \\\"code: ``` example\\\"\"}\n```",
			expected: "{\"predicted_command\": \"echo \\\"code: ``` example\\\"\"}",
		},
		{
			name:     "JSON with triple backticks on same line (edge case)",
			input:    "```json\n{\"predicted_command\": \"ls```-la\"}\n```",
			expected: "{\"predicted_command\": \"ls```-la\"}",
		},
		{
			name:     "code block without closing backticks",
			input:    "```json\n{\"predicted_command\": \"ls -la\"}",
			expected: `{"predicted_command": "ls -la"}`,
		},
		{
			name:     "code block with closing backticks at end without newline",
			input:    "```json\n{\"predicted_command\": \"ls -la\"}```",
			expected: `{"predicted_command": "ls -la"}`,
		},
		{
			name:     "code block with CRLF line endings",
			input:    "```json\r\n{\"predicted_command\": \"git status\"}\r\n```",
			expected: `{"predicted_command": "git status"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
