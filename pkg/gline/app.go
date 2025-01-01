package gline

import (
	"fmt"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

type appModel struct {
	predictor Predictor
	logger    *zap.Logger

	textInput         shellinput.Model
	dirty             bool
	prediction        string
	preview           string
	predictionStateId int

	result     string
	terminated bool

	previewStyle lipgloss.Style
}

type attemptPredictionMsg struct {
	stateId int
}

type setPredictionMsg struct {
	stateId    int
	prediction string
	preview    string
}

type terminateMsg struct{}

func terminate() tea.Msg {
	return terminateMsg{}
}

func initialModel(prompt string, preview string, predictor Predictor, logger *zap.Logger) appModel {
	textInput := shellinput.New()
	textInput.Prompt = prompt
	textInput.Cursor.SetMode(cursor.CursorStatic)
	textInput.ShowSuggestions = true
	textInput.Focus()

	return appModel{
		predictor: predictor,
		logger:    logger,

		textInput:  textInput,
		dirty:      false,
		prediction: "",
		preview:    preview,
		result:     "",
		terminated: false,

		predictionStateId: 0,

		previewStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")),
	}
}

func (m appModel) Init() tea.Cmd {
	return func() tea.Msg {
		return attemptPredictionMsg{
			stateId: m.predictionStateId,
		}
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.textInput.Width = msg.Width
		m.previewStyle = m.previewStyle.Width(max(1, msg.Width-2))
		return m, nil

	case terminateMsg:
		m.terminated = true
		return m, nil

	case attemptPredictionMsg:
		return m.attemptPrediction(msg)

	case setPredictionMsg:
		m.setPrediction(msg.stateId, msg.prediction, msg.preview)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			m.result = m.textInput.Value()
			return m, tea.Sequence(terminate, tea.Quit)

		case "ctrl+c":
			return m, tea.Sequence(terminate, tea.Quit)
		}
	}

	return m.updateTextInput(msg)
}

func (m appModel) View() string {
	if m.terminated {
		return ""
	}

	s := m.textInput.View()
	if m.preview != "" {
		s += "\n"
		s += m.previewStyle.Render(m.preview)
	}
	return s
}

func (m appModel) getFinalOutput() string {
	m.textInput.SetValue(m.result)
	m.textInput.SetSuggestions([]string{})
	m.textInput.Blur()
	m.textInput.ShowSuggestions = false

	s := m.textInput.View()
	return s
}

func (m appModel) updateTextInput(msg tea.Msg) (appModel, tea.Cmd) {
	updatedTextInput, cmd := m.textInput.Update(msg)

	textUpdated := updatedTextInput.Value() != m.textInput.Value()
	m.textInput = updatedTextInput

	// if the text input has changed, we want to attempt a prediction
	if textUpdated && m.predictor != nil {
		m.predictionStateId++

		userInput := updatedTextInput.Value()

		// whenever the user has typed something, mark the model as dirty
		if len(userInput) > 0 {
			m.dirty = true
		}

		if len(userInput) == 0 && m.dirty {
			// if the model was dirty earlier, but now the user has cleared the input,
			// we should clear the prediction
			m.clearPrediction()
		} else if len(userInput) > 0 && strings.HasPrefix(m.prediction, userInput) {
			// if the prediction already starts with the user input, we don't need to predict again
			m.logger.Debug("gline existing predicted input already starts with user input", zap.String("userInput", userInput))
		} else {
			// in other cases, we should kick off a debounced prediction after clearing the current one
			m.clearPrediction()

			cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return attemptPredictionMsg{
					stateId: m.predictionStateId,
				}
			}))
		}
	}

	return m, cmd
}

func (m *appModel) clearPrediction() {
	m.prediction = ""
	m.preview = ""
	m.textInput.SetSuggestions([]string{})
}

func (m *appModel) setPrediction(stateId int, prediction string, preview string) {
	if stateId != m.predictionStateId {
		m.logger.Debug(
			"gline discarding prediction",
			zap.Int("startStateId", stateId),
			zap.Int("newStateId", m.predictionStateId),
		)
		return
	}
	m.prediction = prediction
	m.preview = preview
	m.textInput.SetSuggestions([]string{prediction})
}

func (m appModel) attemptPrediction(msg attemptPredictionMsg) (tea.Model, tea.Cmd) {
	if m.predictor == nil {
		return m, nil
	}
	if msg.stateId != m.predictionStateId {
		return m, nil
	}

	return m, tea.Cmd(func() tea.Msg {
		prediction, preview, err := m.predictor.Predict(m.textInput.Value())
		if err != nil {
			m.logger.Error("gline prediction failed", zap.Error(err))
			return nil
		}

		m.logger.Debug(
			"gline predicted input",
			zap.Int("stateId", msg.stateId),
			zap.String("prediction", prediction),
			zap.String("preview", preview),
		)
		return setPredictionMsg{stateId: msg.stateId, prediction: prediction, preview: preview}
	})
}

func Gline(prompt string, preview string, predictor Predictor, logger *zap.Logger) (string, error) {
	p := tea.NewProgram(initialModel(prompt, preview, predictor, logger))

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	appModel, ok := m.(appModel)
	if !ok {
		logger.Error("Gline resulted in an unexpected app model")
		panic("Gline resulted in an unexpected app model")
	}

	fmt.Println(appModel.getFinalOutput())

	return appModel.result, nil
}
