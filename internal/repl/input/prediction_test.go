package input

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements PredictionProvider for testing.
type mockProvider struct {
	instantPredictions   map[string]string
	debouncedPredictions map[string]string
	err                  error
}

func (m *mockProvider) Predict(ctx context.Context, input string, trigger interpreter.PredictTrigger) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if trigger == interpreter.PredictTriggerInstant {
		prediction, ok := m.instantPredictions[input]
		if !ok {
			return "", nil
		}
		return prediction, nil
	}
	// Debounced trigger
	prediction, ok := m.debouncedPredictions[input]
	if !ok {
		return "", nil
	}
	return prediction, nil
}

func TestPredictionSource_String(t *testing.T) {
	tests := []struct {
		source   PredictionSource
		expected string
	}{
		{PredictionSourceNone, "none"},
		{PredictionSourceHistory, "history"},
		{PredictionSourceLLM, "llm"},
		{PredictionSource(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}

func TestNewPredictionState(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{})

		assert.Equal(t, 200*time.Millisecond, ps.debounceDelay)
		assert.NotNil(t, ps.logger)
		assert.Nil(t, ps.provider)
	})

	t.Run("custom values", func(t *testing.T) {
		provider := &mockProvider{}

		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 100 * time.Millisecond,
			Provider:      provider,
		})

		assert.Equal(t, 100*time.Millisecond, ps.debounceDelay)
		assert.Equal(t, provider, ps.provider)
	})
}

func TestPredictionState_BasicOperations(t *testing.T) {
	ps := NewPredictionState(PredictionStateConfig{})

	t.Run("initial state", func(t *testing.T) {
		assert.Equal(t, "", ps.Prediction())
		assert.False(t, ps.HasPrediction())
		assert.False(t, ps.IsDirty())
		assert.Equal(t, int64(0), ps.StateID())
	})

	t.Run("set prediction", func(t *testing.T) {
		stateID := ps.StateID()
		ok := ps.SetPrediction(stateID, "ls -la")
		assert.True(t, ok)
		assert.Equal(t, "ls -la", ps.Prediction())
		assert.True(t, ps.HasPrediction())
	})

	t.Run("set prediction with wrong state ID", func(t *testing.T) {
		ok := ps.SetPrediction(999, "wrong")
		assert.False(t, ok)
		assert.Equal(t, "ls -la", ps.Prediction()) // unchanged
	})

	t.Run("clear", func(t *testing.T) {
		ps.Clear()
		assert.Equal(t, "", ps.Prediction())
		assert.False(t, ps.HasPrediction())
	})

	t.Run("reset", func(t *testing.T) {
		ps.SetPrediction(ps.StateID(), "test")
		ps.mu.Lock()
		ps.dirty = true
		ps.mu.Unlock()

		ps.Reset()
		assert.Equal(t, "", ps.Prediction())
		assert.False(t, ps.IsDirty())
	})
}

func TestPredictionState_PredictionSuggestion(t *testing.T) {
	ps := NewPredictionState(PredictionStateConfig{})

	t.Run("no prediction", func(t *testing.T) {
		assert.Equal(t, "", ps.PredictionSuggestion("git"))
	})

	t.Run("prediction matches prefix", func(t *testing.T) {
		ps.SetPrediction(ps.StateID(), "git status")
		assert.Equal(t, " status", ps.PredictionSuggestion("git"))
	})

	t.Run("prediction does not match prefix", func(t *testing.T) {
		ps.SetPrediction(ps.StateID(), "ls -la")
		assert.Equal(t, "ls -la", ps.PredictionSuggestion("git"))
	})

	t.Run("empty input", func(t *testing.T) {
		ps.SetPrediction(ps.StateID(), "ls -la")
		assert.Equal(t, "ls -la", ps.PredictionSuggestion(""))
	})
}

func TestPredictionState_OnInputChanged(t *testing.T) {
	t.Run("marks dirty on input", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
		})

		assert.False(t, ps.IsDirty())
		ps.OnInputChanged("git")
		assert.True(t, ps.IsDirty())
	})

	t.Run("increments state ID", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
		})

		initialID := ps.StateID()
		ps.OnInputChanged("git")
		assert.Equal(t, initialID+1, ps.StateID())
	})

	t.Run("keeps prediction when input matches prefix", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
		})

		ps.SetPrediction(ps.StateID(), "git status")
		ch := ps.OnInputChanged("git")
		assert.Nil(t, ch) // no new prediction needed
		assert.Equal(t, "git status", ps.Prediction())
	})

	t.Run("clears prediction when input does not match", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
		})

		ps.SetPrediction(ps.StateID(), "git status")
		ps.OnInputChanged("ls")
		assert.Equal(t, "", ps.Prediction())
	})

	t.Run("clears when dirty and empty", func(t *testing.T) {
		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
		})

		ps.OnInputChanged("git") // marks dirty
		ps.SetPrediction(ps.StateID(), "git status")

		ps.OnInputChanged("") // clear because dirty
		assert.Equal(t, "", ps.Prediction())
	})
}

func TestPredictionState_InstantPrediction(t *testing.T) {
	provider := &mockProvider{
		instantPredictions: map[string]string{
			"git": "git status",
		},
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	ch := ps.OnInputChanged("git")
	require.NotNil(t, ch)

	// Wait for result
	select {
	case result := <-ch:
		assert.Equal(t, "git status", result.Prediction)
		assert.Equal(t, PredictionSourceHistory, result.Source)
		assert.NoError(t, result.Error)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestPredictionState_InstantPredictionIsInstant(t *testing.T) {
	// This test verifies that instant predictions bypass the debounce delay
	// and return immediately (synchronously).

	provider := &mockProvider{
		instantPredictions: map[string]string{
			"git": "git status",
		},
	}

	// Use a very long debounce delay to make the test obvious
	// If instant predictions were debounced, this test would timeout
	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Second, // Very long debounce
		Provider:      provider,
	})

	start := time.Now()
	ch := ps.OnInputChanged("git")
	require.NotNil(t, ch)

	// The channel should already have a result (synchronous)
	select {
	case result := <-ch:
		elapsed := time.Since(start)

		// Should complete in under 10ms (well under the 10s debounce)
		assert.Less(t, elapsed, 10*time.Millisecond,
			"instant prediction should be instant, not debounced")

		assert.Equal(t, "git status", result.Prediction)
		assert.Equal(t, PredictionSourceHistory, result.Source)

		// Verify prediction was also set synchronously on the state
		assert.Equal(t, "git status", ps.Prediction())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("instant prediction should be instant, but timed out")
	}
}

func TestPredictionState_DebouncedPrediction(t *testing.T) {
	// This test verifies that debounced predictions (when no instant match) ARE debounced

	// Provider with no instant matches but debounced match
	provider := &mockProvider{
		instantPredictions: map[string]string{},
		debouncedPredictions: map[string]string{
			"docker": "docker ps",
		},
	}

	debounceDelay := 50 * time.Millisecond
	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: debounceDelay,
		Provider:      provider,
	})

	start := time.Now()
	ch := ps.OnInputChanged("docker")
	require.NotNil(t, ch)

	// Wait for result
	select {
	case result := <-ch:
		elapsed := time.Since(start)

		// Should take at least the debounce delay
		assert.GreaterOrEqual(t, elapsed, debounceDelay,
			"LLM prediction should be debounced")

		assert.Equal(t, "docker ps", result.Prediction)
		assert.Equal(t, PredictionSourceLLM, result.Source)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for LLM prediction")
	}
}

func TestPredictionState_DebouncedFallback(t *testing.T) {
	// No instant matches, should fall back to debounced prediction
	provider := &mockProvider{
		instantPredictions: map[string]string{},
		debouncedPredictions: map[string]string{
			"docker": "docker ps",
		},
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	ch := ps.OnInputChanged("docker")
	require.NotNil(t, ch)

	// Wait for result
	select {
	case result := <-ch:
		assert.Equal(t, "docker ps", result.Prediction)
		assert.Equal(t, PredictionSourceLLM, result.Source)
		assert.NoError(t, result.Error)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestPredictionState_NullStatePrediction(t *testing.T) {
	provider := &mockProvider{
		debouncedPredictions: map[string]string{
			"": "ls -la",
		},
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	// Mark dirty first, then clear
	ps.OnInputChanged("x")
	ch := ps.OnInputChanged("")
	require.NotNil(t, ch)

	// Wait for result
	select {
	case result := <-ch:
		assert.Equal(t, "ls -la", result.Prediction)
		assert.Equal(t, PredictionSourceLLM, result.Source)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestPredictionState_AgentChatSkipped(t *testing.T) {
	t.Run("debounced prediction skipped for agent commands", func(t *testing.T) {
		provider := &mockProvider{
			debouncedPredictions: map[string]string{
				"#hello": "should not appear",
			},
		}

		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
			Provider:      provider,
		})

		ch := ps.OnInputChanged("#hello")
		// Debounced prediction is started but will return empty for agent chat messages
		if ch != nil {
			select {
			case result := <-ch:
				// Should return empty prediction for agent commands
				assert.Equal(t, "", result.Prediction)
				assert.Equal(t, PredictionSourceNone, result.Source)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("timeout waiting for prediction result")
			}
		}

		// Verify no prediction was set
		assert.Equal(t, "", ps.Prediction())

		// Verify LLM was called but returned empty (due to # prefix check in predictLLM)
		// Note: LLM is called but skips prediction internally for # commands
	})

	t.Run("instant prediction works for agent commands", func(t *testing.T) {
		provider := &mockProvider{
			instantPredictions: map[string]string{
				"#": "#explain this code",
			},
		}

		ps := NewPredictionState(PredictionStateConfig{
			DebounceDelay: 10 * time.Millisecond,
			Provider:      provider,
		})

		ch := ps.OnInputChanged("#")
		require.NotNil(t, ch, "instant prediction should return a channel for agent commands")

		select {
		case result := <-ch:
			// Instant prediction should work for agent commands
			assert.Equal(t, "#explain this code", result.Prediction)
			assert.Equal(t, PredictionSourceHistory, result.Source)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for history prediction")
		}

		// Verify prediction was set
		assert.Equal(t, "#explain this code", ps.Prediction())
	})
}

func TestPredictionState_Debouncing(t *testing.T) {
	provider := &mockProvider{
		debouncedPredictions: map[string]string{
			"final": "final command",
		},
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 50 * time.Millisecond,
		Provider:      provider,
	})

	// Rapid input changes
	ps.OnInputChanged("f")
	ps.OnInputChanged("fi")
	ps.OnInputChanged("fin")
	ch := ps.OnInputChanged("final")

	require.NotNil(t, ch)

	// Wait for result
	select {
	case result := <-ch:
		assert.Equal(t, "final command", result.Prediction)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestPredictionState_CancellationOnNewInput(t *testing.T) {
	provider := &mockProvider{
		debouncedPredictions: map[string]string{
			"slow": "slow result",
			"fast": "fast result",
		},
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	// Start first prediction
	ch1 := ps.OnInputChanged("slow")
	require.NotNil(t, ch1)

	// Immediately start another
	ch2 := ps.OnInputChanged("fast")
	require.NotNil(t, ch2)

	// The second one should get a result
	select {
	case result := <-ch2:
		assert.Equal(t, "fast result", result.Prediction)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestPredictionState_ProviderError(t *testing.T) {
	provider := &mockProvider{
		err: errors.New("provider error"),
	}

	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 10 * time.Millisecond,
		Provider:      provider,
	})

	ch := ps.OnInputChanged("test")
	require.NotNil(t, ch)

	select {
	case result := <-ch:
		assert.Error(t, result.Error)
		assert.Equal(t, "", result.Prediction)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for prediction")
	}
}

func TestLLMPredictionProvider(t *testing.T) {
	t.Run("nil model returns empty", func(t *testing.T) {
		provider := NewLLMPredictionProvider(nil, nil, nil)
		prediction, err := provider.Predict(context.Background(), "test", interpreter.PredictTriggerDebounced)
		assert.NoError(t, err)
		assert.Equal(t, "", prediction)
	})

	t.Run("update context", func(t *testing.T) {
		provider := NewLLMPredictionProvider(nil, nil, nil)
		provider.UpdateContext("cwd: /home/user")

		provider.contextTextMu.RLock()
		assert.Equal(t, "cwd: /home/user", provider.contextText)
		provider.contextTextMu.RUnlock()
	})
}

// mockModelProvider implements interpreter.ModelProvider for testing.
type mockModelProvider struct {
	response *interpreter.ChatResponse
	err      error
}

func (m *mockModelProvider) Name() string {
	return "mock"
}

func (m *mockModelProvider) ChatCompletion(ctx context.Context, request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockModelProvider) StreamingChatCompletion(ctx context.Context, request interpreter.ChatRequest, callbacks *interpreter.StreamCallbacks) (*interpreter.ChatResponse, error) {
	response, err := m.ChatCompletion(ctx, request)
	if err != nil {
		return nil, err
	}
	if callbacks != nil && callbacks.OnContent != nil && response.Content != "" {
		callbacks.OnContent(response.Content)
	}
	return response, nil
}

func TestLLMPredictionProvider_WithMockProvider(t *testing.T) {
	model := &interpreter.ModelValue{
		Name:   "test-model",
		Config: map[string]interpreter.Value{},
	}

	t.Run("prefix prediction", func(t *testing.T) {
		mockProvider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "git status"}`,
			},
		}

		provider := NewLLMPredictionProvider(model, mockProvider, nil)
		prediction, err := provider.Predict(context.Background(), "git", interpreter.PredictTriggerDebounced)

		assert.NoError(t, err)
		assert.Equal(t, "git status", prediction)
	})

	t.Run("null state prediction", func(t *testing.T) {
		mockProvider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: `{"predicted_command": "ls -la"}`,
			},
		}

		provider := NewLLMPredictionProvider(model, mockProvider, nil)
		prediction, err := provider.Predict(context.Background(), "", interpreter.PredictTriggerDebounced)

		assert.NoError(t, err)
		assert.Equal(t, "ls -la", prediction)
	})

	t.Run("provider error", func(t *testing.T) {
		mockProvider := &mockModelProvider{
			err: errors.New("API error"),
		}

		provider := NewLLMPredictionProvider(model, mockProvider, nil)
		_, err := provider.Predict(context.Background(), "test", interpreter.PredictTriggerDebounced)

		assert.Error(t, err)
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockProvider := &mockModelProvider{
			response: &interpreter.ChatResponse{
				Content: "not valid json",
			},
		}

		provider := NewLLMPredictionProvider(model, mockProvider, nil)
		prediction, err := provider.Predict(context.Background(), "test", interpreter.PredictTriggerDebounced)

		assert.NoError(t, err)
		assert.Equal(t, "", prediction)
	})
}

func TestPredictionState_Concurrency(t *testing.T) {
	ps := NewPredictionState(PredictionStateConfig{
		DebounceDelay: 5 * time.Millisecond,
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.OnInputChanged("test")
			ps.Prediction()
			ps.StateID()
			ps.IsDirty()
			ps.HasPrediction()
			ps.PredictionSuggestion("t")
		}(i)
	}
	wg.Wait()

	// No panic means success
}
