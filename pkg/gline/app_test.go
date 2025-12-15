package gline

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// max function needed for testing (likely inlined in actual code)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Test the terminate and interrupt functions
func TestTerminate(t *testing.T) {
	msg := terminate()
	_, ok := msg.(terminateMsg)
	assert.True(t, ok, "terminate() should return terminateMsg")
}

func TestInterrupt(t *testing.T) {
	msg := interrupt()
	_, ok := msg.(interruptMsg)
	assert.True(t, ok, "interrupt() should return interruptMsg")
}

// Test ErrInterrupted constant
func TestErrInterrupted(t *testing.T) {
	assert.Error(t, ErrInterrupted)
	assert.Equal(t, "interrupted by user", ErrInterrupted.Error())
}

// Test appState constants
func TestAppStateConstants(t *testing.T) {
	assert.Equal(t, appState(0), Active)
	assert.Equal(t, appState(1), Terminated)
}

// Test model initialization with various configurations
func TestInitialModel(t *testing.T) {
	logger := zap.NewNop()
	predictor := &mockPredictor{}
	explainer := &mockExplainer{}
	analytics := &mockAnalytics{}

	tests := []struct {
		name          string
		prompt        string
		historyValues []string
		explanation   string
		options       Options
	}{
		{
			name:          "basic initialization",
			prompt:        "test> ",
			historyValues: []string{"cmd1", "cmd2"},
			explanation:   "test explanation",
			options:       NewOptions(),
		},
		{
			name:          "empty history",
			prompt:        "$ ",
			historyValues: []string{},
			explanation:   "",
			options:       NewOptions(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialModel(
				tt.prompt,
				tt.historyValues,
				tt.explanation,
				predictor,
				explainer,
				analytics,
				logger,
				tt.options,
			)

			assert.Equal(t, tt.prompt, model.textInput.Prompt)
			assert.Equal(t, tt.explanation, model.explanation)
			assert.Equal(t, tt.historyValues, model.historyValues)
			assert.Equal(t, Active, model.appState)
			assert.False(t, model.interrupted)
			assert.Equal(t, 0, model.predictionStateId)
			assert.Equal(t, "", model.result)
			assert.Equal(t, "", model.prediction)
		})
	}
}

// Test model initialization
func TestAppModelInit(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())

	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Init() now returns a batch command that includes spinner ticks and prediction attempt
	// Just verify the command is not nil - the actual message handling is tested elsewhere
}

// Test update method with different message types
func TestAppModelUpdate(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())

	tests := []struct {
		name     string
		msg      tea.Msg
		expected appState
	}{
		{
			name:     "terminate message",
			msg:      terminateMsg{},
			expected: Terminated,
		},
		{
			name:     "interrupt message",
			msg:      interruptMsg{},
			expected: Terminated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model state
			model.appState = Active
			model.interrupted = false

			updatedModel, cmd := model.Update(tt.msg)
			assert.NotNil(t, updatedModel)

			// We can't easily test internal state without type assertion issues
			// Just verify the method doesn't panic and returns something
			if tt.expected == Terminated {
				assert.Nil(t, cmd)
			}
		})
	}
}

// Test window resize handling
func TestAppModelWindowResize(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())

	// Test window resize
	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, cmd := model.Update(resizeMsg)
	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	// Test edge case with very small width
	smallResizeMsg := tea.WindowSizeMsg{Width: 1, Height: 10}
	updatedModel, cmd = model.Update(smallResizeMsg)
	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)
}

// Test view method
func TestAppModelView(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name     string
		appState appState
		expected string
	}{
		{
			name:     "terminated state",
			appState: Terminated,
			expected: "",
		},
		{
			name:     "active state",
			appState: Active,
			expected: "", // Will be non-empty but we can't easily predict exact content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())
			model.appState = tt.appState

			view := model.View()

			if tt.appState == Terminated {
				assert.Equal(t, tt.expected, view)
			} else {
				assert.NotEmpty(t, view) // Active state should produce some output
			}
		})
	}
}


// Test getFinalOutput
func TestGetFinalOutput(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())
	model.result = "test command"

	output := model.getFinalOutput()
	assert.Contains(t, output, "test command")
	// getFinalOutput operates on a copy, so original model ShowSuggestions unchanged
	// The default value is true as set in the model initialization
	assert.True(t, model.textInput.ShowSuggestions)
}

// Test clearPrediction
func TestClearPrediction(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "", nil, nil, nil, logger, NewOptions())

	// Set some prediction data
	model.prediction = "test prediction"
	model.explanation = "test explanation"
	model.textInput.SetSuggestions([]string{"suggestion1", "suggestion2"})

	// Clear prediction
	model.clearPrediction()

	assert.Equal(t, "", model.prediction)
	assert.Equal(t, "", model.explanation)
	assert.Equal(t, []string{}, model.textInput.AvailableSuggestions())
}

// Test clearPredictionAndRestoreDefault
func TestClearPredictionAndRestoreDefault(t *testing.T) {
	logger := zap.NewNop()
	defaultTip := "💡 Coach tip: Try using pipes!"
	model := initialModel("test> ", []string{}, defaultTip, nil, nil, nil, logger, NewOptions())

	// Verify defaultExplanation is set
	assert.Equal(t, defaultTip, model.defaultExplanation)
	assert.Equal(t, defaultTip, model.explanation)

	// Set some prediction data
	model.prediction = "test prediction"
	model.explanation = "test explanation"
	model.textInput.SetSuggestions([]string{"suggestion1", "suggestion2"})

	// Clear prediction and restore default
	model.clearPredictionAndRestoreDefault()

	assert.Equal(t, "", model.prediction)
	assert.Equal(t, defaultTip, model.explanation) // Should be restored to default
	assert.Equal(t, []string{}, model.textInput.AvailableSuggestions())
}

// Test handleClearScreen
func TestHandleClearScreen(t *testing.T) {
	logger := zap.NewNop()
	model := initialModel("test> ", []string{}, "test explanation", nil, nil, nil, logger, NewOptions())

	// Set some state
	model.textInput.SetValue("test input")

	updatedModel, cmd := model.handleClearScreen()
	assert.NotNil(t, updatedModel)

	// Just verify the method doesn't panic and returns something

	// Should return a command
	assert.NotNil(t, cmd)

	// Execute the command to verify it returns ClearScreen message
	msg := cmd()
	// ClearScreen() returns a clearScreenMsg (unexported), so we can't type assert
	// We just verify that the command returns something non-nil
	assert.NotNil(t, msg, "handleClearScreen should return tea.ClearScreen command")
}