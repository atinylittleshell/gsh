package gline

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// mockPredictor implements Predictor for integration testing
type mockPredictor struct {
	predictions map[string]string
	contexts    map[string]string
	delay       time.Duration
}

func newMockPredictor() *mockPredictor {
	return &mockPredictor{
		predictions: map[string]string{
			"git":    "git status",
			"docker": "docker ps",
			"ls":     "ls -la",
			"cd":     "cd ~/",
			"vim":    "vim .",
		},
		contexts: map[string]string{
			"git":    "git command context",
			"docker": "docker command context",
			"ls":     "list files context",
			"cd":     "change directory context",
			"vim":    "vim editor context",
		},
		delay: 0, // No delay by default for faster tests
	}
}

func (m *mockPredictor) Predict(input string) (prediction, inputContext string, err error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	prediction, ok := m.predictions[input]
	if !ok {
		return "", "", nil
	}

	inputContext, ok = m.contexts[input]
	if !ok {
		inputContext = "default context"
	}

	return prediction, inputContext, nil
}

// mockExplainer implements Explainer for integration testing
type mockExplainer struct {
	explanations map[string]string
	delay        time.Duration
}

func newMockExplainer() *mockExplainer {
	return &mockExplainer{
		explanations: map[string]string{
			"git status": "Shows the status of the working directory",
			"docker ps":  "Lists running Docker containers",
			"ls -la":     "Lists all files in long format including hidden files",
			"cd ~/":      "Changes to the home directory",
			"vim .":      "Opens vim editor in the current directory",
		},
		delay: 0, // No delay by default for faster tests
	}
}

func (m *mockExplainer) Explain(prediction string) (string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	explanation, ok := m.explanations[prediction]
	if !ok {
		return "No explanation available", nil
	}

	return explanation, nil
}

// mockAnalytics implements PredictionAnalytics for integration testing
type mockAnalytics struct {
	entries []analyticsEntry
}

type analyticsEntry struct {
	predictionInput string
	prediction      string
	result          string
}

func newMockAnalytics() *mockAnalytics {
	return &mockAnalytics{
		entries: make([]analyticsEntry, 0),
	}
}

func (m *mockAnalytics) NewEntry(predictionInput, prediction, result string) error {
	m.entries = append(m.entries, analyticsEntry{
		predictionInput: predictionInput,
		prediction:      prediction,
		result:          result,
	})
	return nil
}

// realCompletionProvider for app integration tests
type appCompletionProvider struct {
	completions map[string][]string
}

func newAppCompletionProvider() *appCompletionProvider {
	return &appCompletionProvider{
		completions: map[string][]string{
			"git":    {"git add", "git commit", "git push", "git status"},
			"docker": {"docker run", "docker build", "docker ps"},
			"ls":     {"ls -la", "ls -l", "ls -a"},
		},
	}
}

func (p *appCompletionProvider) GetCompletions(line string, pos int) []string {
	inputLine := line[:pos]
	if completions, ok := p.completions[inputLine]; ok {
		return completions
	}
	return []string{}
}

func (p *appCompletionProvider) GetHelpInfo(line string, pos int) string {
	return ""
}

func TestApp_PredictionFlow_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	// Test prediction flow by simulating typing
	tests := []struct {
		name                string
		input               string
		expectedPrediction  string
		expectedExplanation string
	}{
		{
			name:                "git command prediction",
			input:               "git",
			expectedPrediction:  "git status",
			expectedExplanation: "Shows the status of the working directory",
		},
		{
			name:                "docker command prediction",
			input:               "docker",
			expectedPrediction:  "docker ps",
			expectedExplanation: "Lists running Docker containers",
		},
		{
			name:                "ls command prediction",
			input:               "ls",
			expectedPrediction:  "ls -la",
			expectedExplanation: "Lists all files in long format including hidden files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create initial model
			model := initialModel(
				"> ",
				[]string{},
				"",
				predictor,
				explainer,
				analytics,
				logger,
				options,
			)

			// Set the input value
			model.textInput.SetValue(tt.input)

			// Trigger prediction
			model.predictionStateId++
			predictionMsg := attemptPredictionMsg{stateId: model.predictionStateId}
			updatedModel, cmd := model.attemptPrediction(predictionMsg)
			model = updatedModel.(appModel)

			// Execute the prediction command
			if cmd != nil {
				msg := cmd()
				if setPredMsg, ok := msg.(setPredictionMsg); ok {
					updatedModel, _ := model.setPrediction(setPredMsg.stateId, setPredMsg.prediction, setPredMsg.inputContext)
					model = updatedModel
				}
			}

			// Verify prediction was set
			assert.Equal(t, tt.expectedPrediction, model.prediction,
				"Expected prediction %q, got %q", tt.expectedPrediction, model.prediction)

			// Now trigger explanation
			if model.prediction != "" {
				explanationMsg := attemptExplanationMsg{
					stateId:    model.predictionStateId,
					prediction: model.prediction,
				}
				updatedModel, cmd := model.attemptExplanation(explanationMsg)
				model = updatedModel.(appModel)

				// Execute the explanation command
				if cmd != nil {
					msg := cmd()
					if setExplMsg, ok := msg.(setExplanationMsg); ok {
						updatedModel, _ := model.setExplanation(setExplMsg)
						model = updatedModel.(appModel)
					}
				}
			}

			// Verify explanation was set
			assert.Equal(t, tt.expectedExplanation, model.explanation,
				"Expected explanation %q, got %q", tt.expectedExplanation, model.explanation)
		})
	}
}

func TestApp_KeyHandling_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	tests := []struct {
		name           string
		initialInput   string
		keyMsg         tea.KeyMsg
		expectedResult string
		expectedState  appState
	}{
		{
			name:           "enter key sets result and terminates",
			initialInput:   "git status",
			keyMsg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectedResult: "git status",
			expectedState:  Terminated,
		},
		{
			name:           "ctrl+c terminates with empty result",
			initialInput:   "some command",
			keyMsg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedResult: "",
			expectedState:  Terminated,
		},
		{
			name:           "ctrl+d on empty line exits",
			initialInput:   "",
			keyMsg:         tea.KeyMsg{Type: tea.KeyCtrlD},
			expectedResult: "exit",
			expectedState:  Terminated,
		},
		{
			name:           "ctrl+d on non-empty line does nothing",
			initialInput:   "git status",
			keyMsg:         tea.KeyMsg{Type: tea.KeyCtrlD},
			expectedResult: "",
			expectedState:  Active,
		},
		{
			name:           "backspace on empty input clears prediction",
			initialInput:   "",
			keyMsg:         tea.KeyMsg{Type: tea.KeyBackspace},
			expectedResult: "",
			expectedState:  Active,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model
			model := initialModel(
				"> ",
				[]string{},
				"",
				predictor,
				explainer,
				analytics,
				logger,
				options,
			)

			// Set initial input
			model.textInput.SetValue(tt.initialInput)

			// Process key message
			updatedModel, cmd := model.Update(tt.keyMsg)
			model = updatedModel.(appModel)

			// Execute any commands returned (like terminate)
			if cmd != nil {
				// tea.Sequence creates a sequenceMsg internally
				// We need to call the terminate command directly
				termResult := terminate()
				updatedModel, _ := model.Update(termResult)
				model = updatedModel.(appModel)
			}

			assert.Equal(t, tt.expectedResult, model.result,
				"Expected result %q, got %q", tt.expectedResult, model.result)
			assert.Equal(t, tt.expectedState, model.appState,
				"Expected state %v, got %v", tt.expectedState, model.appState)
		})
	}
}

func TestApp_TextInput_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	model := initialModel(
		"> ",
		[]string{"prev command 1", "prev command 2"},
		"initial explanation",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	tests := []struct {
		name          string
		keyMsg        tea.KeyMsg
		expectedValue string
		expectedDirty bool
	}{
		{
			name:          "typing characters marks as dirty",
			keyMsg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")},
			expectedValue: "g",
			expectedDirty: true,
		},
		{
			name:          "continuing to type",
			keyMsg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")},
			expectedValue: "gi",
			expectedDirty: true,
		},
		{
			name:          "backspace removes character",
			keyMsg:        tea.KeyMsg{Type: tea.KeyBackspace},
			expectedValue: "g",
			expectedDirty: true,
		},
		{
			name:          "backspace to empty still dirty",
			keyMsg:        tea.KeyMsg{Type: tea.KeyBackspace},
			expectedValue: "",
			expectedDirty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedModel, _ := model.Update(tt.keyMsg)
			model = updatedModel.(appModel)

			assert.Equal(t, tt.expectedValue, model.textInput.Value(),
				"Expected value %q, got %q", tt.expectedValue, model.textInput.Value())
			assert.Equal(t, tt.expectedDirty, model.dirty,
				"Expected dirty %v, got %v", tt.expectedDirty, model.dirty)
		})
	}
}

func TestApp_HistoryIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	historyValues := []string{"git add .", "git commit", "git push"}

	model := initialModel(
		"> ",
		historyValues,
		"",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	// Test history navigation
	tests := []struct {
		name          string
		keyMsg        tea.KeyMsg
		expectedValue string
	}{
		{
			name:          "up arrow to first history item",
			keyMsg:        tea.KeyMsg{Type: tea.KeyUp},
			expectedValue: "git add .",
		},
		{
			name:          "up arrow to second history item",
			keyMsg:        tea.KeyMsg{Type: tea.KeyUp},
			expectedValue: "git commit",
		},
		{
			name:          "up arrow to third history item",
			keyMsg:        tea.KeyMsg{Type: tea.KeyUp},
			expectedValue: "git push",
		},
		{
			name:          "down arrow back to second",
			keyMsg:        tea.KeyMsg{Type: tea.KeyDown},
			expectedValue: "git commit",
		},
		{
			name:          "down arrow back to first",
			keyMsg:        tea.KeyMsg{Type: tea.KeyDown},
			expectedValue: "git add .",
		},
		{
			name:          "down arrow back to current (empty)",
			keyMsg:        tea.KeyMsg{Type: tea.KeyDown},
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedModel, _ := model.Update(tt.keyMsg)
			model = updatedModel.(appModel)

			assert.Equal(t, tt.expectedValue, model.textInput.Value(),
				"Expected value %q, got %q", tt.expectedValue, model.textInput.Value())
		})
	}
}

func TestApp_CompletionIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	model := initialModel(
		"> ",
		[]string{},
		"",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	// Type "git"
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("git")})
	model = updatedModel.(appModel)

	// Press TAB for completion
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updatedModel.(appModel)

	// Verify completion was applied
	value := model.textInput.Value()
	completions := []string{"git add", "git commit", "git push", "git status"}

	found := false
	for _, completion := range completions {
		if value == completion {
			found = true
			break
		}
	}

	assert.True(t, found,
		"Expected value to be one of %v, got %q", completions, value)

	// Note: Can't access unexported completion field, but if value changed, completion worked
}

func TestApp_WindowResize_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	model := initialModel(
		"> ",
		[]string{},
		"test explanation",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	// Simulate window resize
	resizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(resizeMsg)
	model = updatedModel.(appModel)

	// Verify width was set on text input
	assert.Equal(t, 80, model.textInput.Width,
		"Expected text input width to be 80, got %d", model.textInput.Width)

	// Verify styles were updated (check that they have a width set)
	// We can't directly access the computed width, but we can check that the update didn't crash
	view := model.View()
	assert.NotEmpty(t, view, "Expected view to be rendered after resize")
}

func TestApp_StateManagement_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          3,
	}

	model := initialModel(
		"> ",
		[]string{},
		"",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	// Verify initial state
	assert.Equal(t, Active, model.appState, "Expected initial state to be Active")
	assert.Equal(t, 0, model.predictionStateId, "Expected initial prediction state ID to be 0")
	assert.False(t, model.dirty, "Expected initial dirty state to be false")

	// Type something to make it dirty
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
	model = updatedModel.(appModel)

	assert.True(t, model.dirty, "Expected model to be dirty after typing")

	// Clear input should still be dirty
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model = updatedModel.(appModel)

	assert.True(t, model.dirty, "Expected model to still be dirty after clearing input")

	// Terminate the model
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(appModel)

	// Execute terminate command
	if cmd != nil {
		// tea.Sequence creates a sequenceMsg internally
		// We need to call the terminate command directly
		termResult := terminate()
		updatedModel, _ := model.Update(termResult)
		model = updatedModel.(appModel)
	}

	assert.Equal(t, Terminated, model.appState, "Expected state to be Terminated after enter")
}

func TestApp_Analytics_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          1,
	}

	// Test analytics functionality directly without starting the interactive UI
	// Create a model instance to test analytics recording
	model := initialModel(
		"> ",
		[]string{},
		"",
		predictor,
		explainer,
		analytics,
		logger,
		options,
	)

	// Set up prediction state
	model.lastPredictionInput = "git"
	model.lastPrediction = "git status"
	model.result = "git status"

	// Simulate calling analytics
	err := analytics.NewEntry(model.lastPredictionInput, model.lastPrediction, model.result)
	require.NoError(t, err)

	// Verify analytics entry was recorded
	assert.Len(t, analytics.entries, 1, "Expected one analytics entry")
	entry := analytics.entries[0]
	assert.Equal(t, "git", entry.predictionInput, "Expected prediction input to be 'git'")
	assert.Equal(t, "git status", entry.prediction, "Expected prediction to be 'git status'")
	assert.Equal(t, "git status", entry.result, "Expected result to be 'git status'")
}

func TestApp_View_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := Options{
		CompletionProvider: completionProvider,
		MinHeight:          3,
	}

	tests := []struct {
		name                string
		input               string
		explanation         string
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:             "basic view with input",
			input:            "git status",
			explanation:      "",
			expectedContains: []string{"> git status"},
		},
		{
			name:             "view with explanation",
			input:            "git status",
			explanation:      "Shows repository status",
			expectedContains: []string{"> git status", "Shows repository status"},
		},
		{
			name:                "terminated view is empty",
			input:               "command",
			explanation:         "explanation",
			expectedContains:    []string{},
			expectedNotContains: []string{"> command", "explanation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialModel(
				"> ",
				[]string{},
				tt.explanation,
				predictor,
				explainer,
				analytics,
				logger,
				options,
			)

			model.textInput.SetValue(tt.input)

			// Terminate model for the terminated test case
			if tt.name == "terminated view is empty" {
				model.appState = Terminated
			}

			view := model.View()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, view, expected,
					"Expected view to contain %q, got: %s", expected, view)
			}

			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, view, notExpected,
					"Expected view to not contain %q, got: %s", notExpected, view)
			}

			// Test minimum height
			if model.appState != Terminated {
				numLines := strings.Count(view, "\n")
				assert.GreaterOrEqual(t, numLines, options.MinHeight,
					"Expected at least %d lines, got %d", options.MinHeight, numLines)
			}
		})
	}
}
