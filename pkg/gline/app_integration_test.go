package gline

import (
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
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
			"git ":   "git status",
			"docker": "docker ps",
			"ls":     "ls -la",
			"ls ":    "ls -la",
			"cd":     "cd ~/",
			"vim":    "vim .",
		},
		contexts: map[string]string{
			"git":    "git command context",
			"git ":   "git command context",
			"docker": "docker command context",
			"ls":     "list files context",
			"ls ":    "list files context",
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
			"git":        "General git command help",
			"git status": "Shows the status of the working directory",
			"docker ps":  "Lists running Docker containers",
			"ls":         "Lists files in the current directory",
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

func (p *appCompletionProvider) GetCompletions(line string, pos int) []shellinput.CompletionCandidate {
	inputLine := line[:pos]
	if completions, ok := p.completions[inputLine]; ok {
		candidates := make([]shellinput.CompletionCandidate, len(completions))
		for i, s := range completions {
			candidates[i] = shellinput.CompletionCandidate{Value: s}
		}
		return candidates
	}
	return []shellinput.CompletionCandidate{}
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

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

func TestCtrlKClearsPredictionAndExplanation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	options := NewOptions()

	model := initialModel(
		"test> ",
		[]string{},
		"",
		predictor,
		explainer,
		nil,
		logger,
		options,
	)

	model.textInput.SetValue("git")
	model.textInput.SetCursor(len("git"))

	model, _ = model.setPrediction(model.predictionStateId, "git status", "git")
	assert.NotEmpty(t, model.textInput.MatchedSuggestions())

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updatedModel.(appModel)

	assert.Empty(t, model.textInput.MatchedSuggestions(), "Ctrl+K should clear prediction-backed suggestions")
	assert.Empty(t, model.prediction, "Ctrl+K should clear the active prediction")
	assert.Empty(t, model.explanation, "Ctrl+K should clear any pending explanation when trimming predictions")
}

func TestCtrlKRerequestsPredictionWhenTextRemains(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	options := NewOptions()

	model := initialModel(
		"test> ",
		[]string{},
		"",
		predictor,
		explainer,
		nil,
		logger,
		options,
	)

	model.textInput.SetValue("git status")
	model.textInput.SetCursor(len("git"))

	model, _ = model.setPrediction(model.predictionStateId, "git status", "git")
	assert.NotEmpty(t, model.textInput.MatchedSuggestions(), "Prediction-backed suggestions should be visible before trimming")

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updatedModel.(appModel)

	require.Equal(t, "git", model.textInput.Value(), "Ctrl+K should trim text after the cursor")
	assert.Empty(t, model.textInput.MatchedSuggestions(), "Matched suggestions should clear when trimming autocompletion text")

	assert.Empty(t, model.prediction, "Prediction should remain cleared after Ctrl+K until new input arrives")
	assert.Empty(t, model.explanation, "Explanation should clear after Ctrl+K until predictions resume")
}

func TestCtrlKRefreshesPredictionWhenTextUnchanged(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	options := NewOptions()

	model := initialModel(
		"test> ",
		[]string{},
		"",
		predictor,
		explainer,
		nil,
		logger,
		options,
	)

	model.textInput.SetValue("git")
	model.textInput.SetCursor(len("git"))

	model, _ = model.setPrediction(model.predictionStateId, "git status", "git")
	assert.NotEmpty(t, model.textInput.MatchedSuggestions(), "Prediction-backed suggestions should be visible before trimming")

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updatedModel.(appModel)

	require.Equal(t, "git", model.textInput.Value(), "Ctrl+K should leave the buffer unchanged when there's no text after the cursor")
	assert.Empty(t, model.textInput.MatchedSuggestions(), "Matched suggestions should clear when trimming autocompletion text")

	assert.True(t, model.textInput.SuggestionsSuppressedUntilInput(), "Suggestions should be suppressed after trimming ghost text")

	if cmd != nil {
		if attemptMsg, ok := cmd().(attemptPredictionMsg); ok {
			predictionModel, predictionCmd := model.attemptPrediction(attemptMsg)
			model = predictionModel.(appModel)

			if predictionCmd != nil {
				if predMsg, ok := predictionCmd().(setPredictionMsg); ok {
					model, predictionCmd = model.setPrediction(predMsg.stateId, predMsg.prediction, predMsg.inputContext)

					if predictionCmd != nil {
						if attemptExplMsg, ok := predictionCmd().(attemptExplanationMsg); ok {
							explanationModel, ecmd := model.attemptExplanation(attemptExplMsg)
							model = explanationModel.(appModel)

							if ecmd != nil {
								if explMsg, ok := ecmd().(setExplanationMsg); ok {
									explanationModel, _ = model.setExplanation(explMsg)
									model = explanationModel.(appModel)
								}
							}
						}
					}
				}
			}
		}
	}

	assert.Equal(t, "General git command help", model.explanation, "Assistant help should derive from the remaining buffer while suggestions are suppressed")
	assert.Empty(t, model.textInput.MatchedSuggestions(), "Suggestions should stay hidden until the user types again")

	// Resume predictions after the user provides more input
	updatedModel, predictionCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	model = updatedModel.(appModel)

	if predictionCmd != nil {
		if attemptMsg, ok := predictionCmd().(attemptPredictionMsg); ok {
			predictionModel, pcmd := model.attemptPrediction(attemptMsg)
			model = predictionModel.(appModel)

			if pcmd != nil {
				if predMsg, ok := pcmd().(setPredictionMsg); ok {
					model, pcmd = model.setPrediction(predMsg.stateId, predMsg.prediction, predMsg.inputContext)
					if pcmd != nil {
						if attemptExplMsg, ok := pcmd().(attemptExplanationMsg); ok {
							explanationModel, ecmd := model.attemptExplanation(attemptExplMsg)
							model = explanationModel.(appModel)
							if ecmd != nil {
								if explMsg, ok := ecmd().(setExplanationMsg); ok {
									explanationModel, _ = model.setExplanation(explMsg)
									model = explanationModel.(appModel)
								}
							}
						}
					}
				}
			}
		}
	}

	assert.Equal(t, "git status", model.prediction, "Prediction should resume once the user enters new input")
	assert.Equal(t, "Shows the status of the working directory", model.explanation, "Explanation should return after suppression lifts")
}

func TestCtrlKRestoresSuggestionsWithoutNewInput(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	options := NewOptions()

	model := initialModel(
		"test> ",
		[]string{},
		"",
		predictor,
		explainer,
		nil,
		logger,
		options,
	)

	model.textInput.SetValue("ls -la")
	model.textInput.SetCursor(len("ls"))

	model, _ = model.setPrediction(model.predictionStateId, "ls -la", "ls")
	assert.NotEmpty(t, model.textInput.MatchedSuggestions(), "Prediction-backed suggestions should be visible before trimming user text")

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = updatedModel.(appModel)

	require.Equal(t, "ls", model.textInput.Value(), "Ctrl+K should trim text after the cursor while leaving the prefix intact")
	assert.Empty(t, model.textInput.MatchedSuggestions(), "Matched suggestions should clear after trimming autocompletion text")

	assert.True(t, model.textInput.SuggestionsSuppressedUntilInput(), "Suggestions should remain suppressed after Ctrl+K until new input")

	if cmd != nil {
		// Prediction attempts should refresh help text without re-enabling suggestions while suppression is active
		if attemptMsg, ok := cmd().(attemptPredictionMsg); ok {
			predictionModel, predictionCmd := model.attemptPrediction(attemptMsg)
			model = predictionModel.(appModel)

			if predictionCmd != nil {
				if predMsg, ok := predictionCmd().(setPredictionMsg); ok {
					model, predictionCmd = model.setPrediction(predMsg.stateId, predMsg.prediction, predMsg.inputContext)

					if predictionCmd != nil {
						if attemptExplMsg, ok := predictionCmd().(attemptExplanationMsg); ok {
							explanationModel, ecmd := model.attemptExplanation(attemptExplMsg)
							model = explanationModel.(appModel)

							if ecmd != nil {
								if explMsg, ok := ecmd().(setExplanationMsg); ok {
									explanationModel, _ = model.setExplanation(explMsg)
									model = explanationModel.(appModel)
								}
							}
						}
					}
				}
			}
		}
	}

	assert.NotEmpty(t, model.explanation, "Assistant help should refresh even while suggestions are suppressed")
	assert.Empty(t, model.textInput.MatchedSuggestions(), "Suppressed suggestions should stay hidden until more input arrives")

	// Once the user types again, suggestions and help should repopulate
	updatedModel, predictionCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	model = updatedModel.(appModel)

	if predictionCmd != nil {
		if attemptMsg, ok := predictionCmd().(attemptPredictionMsg); ok {
			predictionModel, pcmd := model.attemptPrediction(attemptMsg)
			model = predictionModel.(appModel)

			if pcmd != nil {
				if predMsg, ok := pcmd().(setPredictionMsg); ok {
					model, pcmd = model.setPrediction(predMsg.stateId, predMsg.prediction, predMsg.inputContext)

					if pcmd != nil {
						if attemptExplMsg, ok := pcmd().(attemptExplanationMsg); ok {
							explanationModel, ecmd := model.attemptExplanation(attemptExplMsg)
							model = explanationModel.(appModel)

							if ecmd != nil {
								if explMsg, ok := ecmd().(setExplanationMsg); ok {
									explanationModel, _ = model.setExplanation(explMsg)
									model = explanationModel.(appModel)
								}
							}
						}
					}
				}
			}
		}
	}

	assert.Equal(t, "ls -la", model.prediction, "Prediction should repopulate after new input following Ctrl+K")
	assert.NotEmpty(t, model.textInput.MatchedSuggestions(), "Suggestions should return once the user presses another key")
	assert.Equal(t, "Lists all files in long format including hidden files", model.explanation, "Explanation should reflect the refreshed suggestion after new input")
}

func TestApp_KeyHandling_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	// Verify completion inserted the common prefix
	value := model.textInput.Value()

	assert.Equal(t, "git ", value,
		"Expected value to be the common prefix \"git \" after first tab press, got %q", value)

	// Note: Can't access unexported completion field, but if value changed, completion worked
}

func TestApp_WindowResize_Integration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	predictor := newMockPredictor()
	explainer := newMockExplainer()
	analytics := newMockAnalytics()
	completionProvider := newAppCompletionProvider()

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	options := NewOptions()
	options.CompletionProvider = completionProvider

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

	options := NewOptions()
	options.CompletionProvider = completionProvider

	// Test analytics recording directly without calling Gline
	// to avoid double analytics recording (once by Gline, once manually)

	// Create a model to simulate the state
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

	// Set up prediction state to simulate what would happen after user interaction
	model.lastPredictionInput = "git"
	model.lastPrediction = "git status"
	model.result = "git status"

	// Simulate the analytics recording that would happen in Gline
	err := analytics.NewEntry(model.lastPredictionInput, model.lastPrediction, model.result)
	require.NoError(t, err)

	// Verify analytics entry was recorded correctly
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

	options := NewOptions()
	options.CompletionProvider = completionProvider
	// Increase AssistantHeight so content fits
	options.AssistantHeight = 5

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

			// Set dimensions for testing
			model.textInput.Width = 80
			model.height = 24

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

		})
	}
}
