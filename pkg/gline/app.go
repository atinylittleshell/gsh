package gline

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
	"go.uber.org/zap"
)

type appModel struct {
	predictor Predictor
	explainer Explainer
	analytics PredictionAnalytics
	logger    *zap.Logger
	options   Options

	textInput           shellinput.Model
	dirty               bool
	prediction          string
	explanation         string
	lastError           error
	lastPredictionInput string
	lastPrediction      string
	predictionStateId   int

	historyValues []string
	result        string
	appState      appState
	interrupted   bool

	explanationStyle lipgloss.Style
	completionStyle  lipgloss.Style
	errorStyle       lipgloss.Style

	// Multiline support
	multilineState *MultilineState
	originalPrompt string
	height         int

	// LLM status indicator
	llmIndicator LLMIndicator
}

type attemptPredictionMsg struct {
	stateId int
}

type setPredictionMsg struct {
	stateId      int
	prediction   string
	inputContext string
}

type attemptExplanationMsg struct {
	stateId    int
	prediction string
}

// errorMsg wraps an error that occurred during prediction or explanation
type errorMsg struct {
	stateId int
	err     error
}

// helpHeaderRegex matches redundant help headers like "**@name** - "
var helpHeaderRegex = regexp.MustCompile(`^\*\*[^\*]+\*\* - `)

type setExplanationMsg struct {
	stateId     int
	explanation string
}

// ErrInterrupted is returned when the user presses Ctrl+C
var ErrInterrupted = errors.New("interrupted by user")

type terminateMsg struct{}

func terminate() tea.Msg {
	return terminateMsg{}
}

type interruptMsg struct{}

func interrupt() tea.Msg {
	return interruptMsg{}
}

type appState int

const (
	Active appState = iota
	Terminated
)

func initialModel(
	prompt string,
	historyValues []string,
	explanation string,
	predictor Predictor,
	explainer Explainer,
	analytics PredictionAnalytics,
	logger *zap.Logger,
	options Options,
) appModel {
	textInput := shellinput.New()
	textInput.Prompt = prompt
	textInput.SetHistoryValues(historyValues)
	// Initialize rich history if available
	if len(options.RichHistory) > 0 {
		textInput.SetRichHistory(options.RichHistory)
	}
	if options.CurrentDirectory != "" {
		textInput.SetCurrentDirectory(options.CurrentDirectory)
	}
	textInput.Cursor.SetMode(cursor.CursorStatic)
	textInput.ShowSuggestions = true
	textInput.CompletionProvider = options.CompletionProvider
	textInput.Focus()

	return appModel{
		predictor: predictor,
		explainer: explainer,
		analytics: analytics,
		logger:    logger,
		options:   options,

		textInput:     textInput,
		dirty:         false,
		prediction:    "",
		explanation:   explanation,
		historyValues: historyValues,
		result:        "",
		appState:      Active,
		interrupted:   false, // Explicitly initialize to prevent stateful behavior

		predictionStateId: 0,

		explanationStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")),
		completionStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Red

		// Initialize multiline state
		multilineState: NewMultilineState(),
		originalPrompt: prompt,

		llmIndicator: NewLLMIndicator(),
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		m.llmIndicator.Tick(),
		func() tea.Msg {
			return attemptPredictionMsg{
				stateId: m.predictionStateId,
			}
		},
	)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case LLMTickMsg:
		m.llmIndicator.Update()
		if m.llmIndicator.GetStatus() == LLMStatusInFlight {
			return m, m.llmIndicator.Tick()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.textInput.Width = msg.Width
		m.explanationStyle = m.explanationStyle.Width(max(1, msg.Width-2))
		m.completionStyle = m.completionStyle.Width(max(1, msg.Width-2))
		return m, nil

	case terminateMsg:
		m.appState = Terminated
		return m, nil

	case interruptMsg:
		m.appState = Terminated
		m.interrupted = true
		return m, nil

	case attemptPredictionMsg:
		m.llmIndicator.SetStatus(LLMStatusInFlight)
		model, cmd := m.attemptPrediction(msg)
		return model, tea.Batch(cmd, m.llmIndicator.Tick())

	case setPredictionMsg:
		return m.setPrediction(msg.stateId, msg.prediction, msg.inputContext)

	case attemptExplanationMsg:
		return m.attemptExplanation(msg)

	case setExplanationMsg:
		return m.setExplanation(msg)

	case errorMsg:
		if msg.stateId == m.predictionStateId {
			m.lastError = msg.err
			m.llmIndicator.SetStatus(LLMStatusError)
			m.prediction = ""
			m.explanation = ""
			m.textInput.SetSuggestions([]string{})
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		// TODO: replace with custom keybindings
		case "backspace":
			if !m.textInput.InReverseSearch() {
				// if the input is already empty, we should clear prediction
				if m.textInput.Value() == "" {
					m.dirty = true
					m.predictionStateId++
					m.clearPrediction()
					return m, nil
				}
			}

		case "enter":
			if m.textInput.InReverseSearch() {
				break
			}

			input := m.textInput.Value()

			// Handle multiline input with error handling
			complete, prompt := m.multilineState.AddLine(input)
			if !complete {
				// Need more input, update prompt and continue
				m.textInput.Prompt = prompt + " "
				// Clear the text input field but preserve the multiline buffer
				m.textInput.SetValue("")
				return m, nil
			}

			// We have a complete command - add error handling for GetCompleteCommand
			result := m.multilineState.GetCompleteCommand()
			if result == "" && input != "" {
				// Only treat empty result as error if input was not empty
				// Reset the multiline state and continue
				m.multilineState.Reset()
				m.textInput.SetValue("")
				return m, nil
			}

			m.result = result
			return m, tea.Sequence(terminate, tea.Quit)

		case "ctrl+c":
			if m.textInput.InReverseSearch() {
				break
			}

			// Handle Ctrl-C: cancel current line, preserve input with "^C" appended, and present fresh prompt
			currentInput := m.textInput.Value()

			// Reset multiline state on Ctrl+C
			if m.multilineState.IsActive() {
				m.multilineState.Reset()
				m.textInput.Prompt = m.originalPrompt // Reset to original prompt
			}

			// Print the current input with "^C" appended, then move to next line
			// This works for both empty and non-empty input
			fmt.Printf("%s^C\n", currentInput)

			// Flush output to ensure it's displayed before framework cleanup
			_ = os.Stdout.Sync()

			// Set result to empty string so shell doesn't try to execute it
			m.result = ""
			// Use interrupt message to indicate Ctrl+C was pressed
			return m, tea.Sequence(interrupt, tea.Quit)
		case "ctrl+d":
			// Handle Ctrl-D: exit shell if on blank line
			currentInput := m.textInput.Value()
			if strings.TrimSpace(currentInput) == "" {
				// On blank line, exit the shell
				m.result = "exit"
				return m, tea.Sequence(terminate, tea.Quit)
			}
			// If there's content, do nothing (standard behavior)
			return m, nil
		case "ctrl+l":
			return m.handleClearScreen()
		}
	}

	return m.updateTextInput(msg)
}

func (m appModel) View() string {
	// Once terminated, render nothing
	if m.appState == Terminated {
		return ""
	}

	var inputStr string

	// If we have multiline content, show each line with its original prompt
	if m.multilineState.IsActive() {
		lines := m.multilineState.GetLines()
		for i, line := range lines {
			if i == 0 {
				// First line uses the original prompt (textInput already adds the space)
				inputStr += m.originalPrompt + line + "\n"
			} else {
				// Subsequent lines use continuation prompt
				inputStr += "> " + line + "\n"
			}
		}
	}

	// Add the current input line with appropriate prompt
	inputStr += m.textInput.View()

	// Determine assistant content
	var assistantContent string

	// We need to handle truncation manually because lipgloss Height doesn't truncate automatically
	// Use expanded height when in reverse search mode (close to full screen)
	availableHeight := m.options.AssistantHeight
	if m.textInput.InReverseSearch() && m.height > 0 {
		// Use most of terminal height, leaving room for prompt line (2) and borders (2)
		availableHeight = max(m.options.AssistantHeight, m.height-4)
	}

	// Display error if present
	if m.lastError != nil {
		errorContent := fmt.Sprintf("LLM Inference Error: %s", m.lastError.Error())
		assistantContent = m.errorStyle.Render(errorContent)
	} else {
		// Normal assistant content logic
		helpBox := m.textInput.HelpBoxView()

		// Determine available width for completion box
		completionWidth := max(0, m.textInput.Width-4)
		if helpBox != "" {
			completionWidth = completionWidth / 2
		}

		completionBox := m.textInput.CompletionBoxView(availableHeight, completionWidth)
		historyBox := m.textInput.HistorySearchBoxView(availableHeight, max(0, m.textInput.Width-2))

		if historyBox != "" {
			assistantContent = historyBox
		} else if completionBox != "" && helpBox != "" {
			// Clean up help box text to avoid redundancy
			// Remove headers like "**@name** - " or "**name** - " using regex
			// This covers patterns like "**@debug-assistant** - " or "**@!new** - "
			helpBox = helpHeaderRegex.ReplaceAllString(helpBox, "")

			// Render side-by-side
			halfWidth := completionWidth // Already calculated

			leftStyle := lipgloss.NewStyle().
				Width(halfWidth).
				Height(availableHeight).
				MaxHeight(availableHeight)

			rightStyle := lipgloss.NewStyle().
				Width(halfWidth).
				Height(availableHeight).
				MaxHeight(availableHeight).
				PaddingLeft(1) // Add some spacing between columns

			// Render completion on left, help on right
			assistantContent = lipgloss.JoinHorizontal(lipgloss.Top,
				leftStyle.Render(completionBox),
				rightStyle.Render(helpBox))

		} else if completionBox != "" {
			assistantContent = completionBox
		} else if helpBox != "" {
			assistantContent = helpBox
		} else {
			assistantContent = m.explanation
		}
	}

	// Render Assistant Box with custom border that includes LLM indicators
	boxWidth := max(0, m.textInput.Width-2)
	borderColor := lipgloss.Color("62")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Word wrap content to fit box width, then split into lines
	innerWidth := max(0, boxWidth-2) // Account for left/right borders
	wrappedContent := wordwrap.String(assistantContent, innerWidth)
	lines := strings.Split(wrappedContent, "\n")
	if len(lines) > availableHeight {
		lines = lines[:availableHeight]
	}
	// Pad to fill the available height
	for len(lines) < availableHeight {
		lines = append(lines, "")
	}

	// Render the LLM indicator
	indicatorStr := " " + m.llmIndicator.View() + " "
	indicatorLen := 2 + m.llmIndicator.Width() // spaces + indicator

	// Build the box manually
	var result strings.Builder

	// Top border: ╭───...───╮
	topBorder := borderStyle.Render("╭" + strings.Repeat("─", innerWidth) + "╮")
	result.WriteString(topBorder)
	result.WriteString("\n")

	// Content lines with left/right borders
	contentWidth := innerWidth // Width available for content
	for _, line := range lines {
		// Truncate or pad line to fit content width
		lineWidth := lipgloss.Width(line)
		if lineWidth > contentWidth {
			// Truncate the line - need to handle ANSI codes
			line = truncateWithAnsi(line, contentWidth)
			lineWidth = lipgloss.Width(line)
		}
		padding := max(0, contentWidth-lineWidth)
		result.WriteString(borderStyle.Render("│"))
		result.WriteString(line)
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(borderStyle.Render("│"))
		result.WriteString("\n")
	}

	// Bottom border with indicators: ╰───...─── Fast:✓ Slow:○ ╯
	// Calculate how much space we have for the horizontal line
	bottomLineWidth := max(0, innerWidth-indicatorLen)
	bottomBorder := borderStyle.Render("╰"+strings.Repeat("─", bottomLineWidth)) + indicatorStr + borderStyle.Render("╯")
	result.WriteString(bottomBorder)

	return inputStr + "\n" + result.String()
}

// truncateWithAnsi truncates a string to maxWidth display columns, handling ANSI escape codes
func truncateWithAnsi(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var result strings.Builder
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		// Check if adding this rune would exceed maxWidth
		runeWidth := runewidth.RuneWidth(r)
		if width+runeWidth > maxWidth {
			break
		}
		result.WriteRune(r)
		width += runeWidth
	}

	return result.String()
}

func (m appModel) getFinalOutput() string {
	m.textInput.SetValue(m.result)
	m.textInput.SetSuggestions([]string{})
	m.textInput.Blur()
	m.textInput.ShowSuggestions = false

	// Reset to original prompt for final output display
	m.textInput.Prompt = m.originalPrompt

	s := m.textInput.View()
	return s
}

func (m appModel) updateTextInput(msg tea.Msg) (appModel, tea.Cmd) {
        oldVal := m.textInput.Value()
        oldMatchedSuggestions := m.textInput.MatchedSuggestions()
        oldSuppression := m.textInput.SuggestionsSuppressedUntilInput()
        updatedTextInput, cmd := m.textInput.Update(msg)
        newVal := updatedTextInput.Value()
        newMatchedSuggestions := updatedTextInput.MatchedSuggestions()

        textUpdated := oldVal != newVal
	suggestionsCleared := len(oldMatchedSuggestions) > 0 && len(newMatchedSuggestions) == 0
	m.textInput = updatedTextInput

	// if the text input has changed, we want to attempt a prediction
	if textUpdated && m.predictor != nil {
		m.predictionStateId++

		// Clear any existing error when user types
		m.lastError = nil

		userInput := updatedTextInput.Value()

		// whenever the user has typed something, mark the model as dirty
                if len(userInput) > 0 {
                        m.dirty = true
                }

                suppressionActive := updatedTextInput.SuggestionsSuppressedUntilInput()
                suppressionLifted := !suppressionActive && oldSuppression

                switch {
                case len(userInput) == 0 && m.dirty:
                        // if the model was dirty earlier, but now the user has cleared the input,
                        // we should clear the prediction
                        m.clearPrediction()
		case suppressionActive:
			// When suppression is active (e.g., after Ctrl+K), clear stale predictions but
			// still recompute assistant help for the remaining buffer while keeping
			// autocomplete hints hidden until new input arrives.
                        m.clearPrediction()
                        if len(userInput) > 0 {
                                cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
                                        return attemptPredictionMsg{
                                                stateId: m.predictionStateId,
                                        }
                                }))
                        }
                case len(userInput) > 0 && strings.HasPrefix(m.prediction, userInput) && !suggestionsCleared && !suppressionLifted:
                        // if the prediction already starts with the user input, we don't need to predict again
                        m.logger.Debug("gline existing predicted input already starts with user input", zap.String("userInput", userInput))
                default:
                        // in other cases, we should kick off a debounced prediction after clearing the current one
                        m.clearPrediction()

			cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return attemptPredictionMsg{
					stateId: m.predictionStateId,
				}
			}))
		}
	} else if suggestionsCleared {
		// User trimmed away ghost suggestions (e.g., via Ctrl+K) without changing
		// the underlying input. Clear any pending prediction and explanation so the
		// assistant box reflects the truncated command, and re-request a prediction
		// for the remaining buffer so the assistant can refresh its help content.
		m.clearPrediction()

		if m.predictor != nil {
			m.predictionStateId++
			if len(m.textInput.Value()) > 0 {
				cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return attemptPredictionMsg{stateId: m.predictionStateId}
				}))
			}
		}
	}

	return m, cmd
}

func (m *appModel) clearPrediction() {
	m.prediction = ""
	m.explanation = ""
	m.lastError = nil
	m.textInput.SetSuggestions([]string{})
}

func (m appModel) setPrediction(stateId int, prediction string, inputContext string) (appModel, tea.Cmd) {
	if stateId != m.predictionStateId {
		m.logger.Debug(
			"gline discarding prediction",
			zap.Int("startStateId", stateId),
			zap.Int("newStateId", m.predictionStateId),
		)
		return m, nil
	}

        m.prediction = prediction
        m.lastPredictionInput = inputContext
        m.lastPrediction = prediction
        m.textInput.SetSuggestions([]string{prediction})
        m.textInput.UpdateHelpInfo()
        m.explanation = ""
        explanationTarget := prediction
        if m.textInput.SuggestionsSuppressedUntilInput() {
                explanationTarget = m.textInput.Value()
        }

        return m, tea.Cmd(func() tea.Msg {
                return attemptExplanationMsg{stateId: m.predictionStateId, prediction: explanationTarget}
        })
}

func (m appModel) attemptPrediction(msg attemptPredictionMsg) (tea.Model, tea.Cmd) {
	if m.predictor == nil {
		return m, nil
	}
	if msg.stateId != m.predictionStateId {
		return m, nil
	}
	// Skip LLM prediction for @ commands (agentic commands)
	if strings.HasPrefix(strings.TrimSpace(m.textInput.Value()), "@") {
		m.llmIndicator.SetStatus(LLMStatusIdle)
		return m, nil
	}

	return m, tea.Cmd(func() tea.Msg {
		prediction, inputContext, err := m.predictor.Predict(m.textInput.Value())
		if err != nil {
			m.logger.Error("gline prediction failed", zap.Error(err))
			return errorMsg{stateId: msg.stateId, err: err}
		}

		m.logger.Debug(
			"gline predicted input",
			zap.Int("stateId", msg.stateId),
			zap.String("prediction", prediction),
			zap.String("inputContext", inputContext),
		)
		return setPredictionMsg{stateId: msg.stateId, prediction: prediction, inputContext: inputContext}
	})
}

func (m appModel) attemptExplanation(msg attemptExplanationMsg) (tea.Model, tea.Cmd) {
	if m.explainer == nil {
		return m, nil
	}
	if msg.stateId != m.predictionStateId {
		return m, nil
	}

	return m, tea.Cmd(func() tea.Msg {
		explanation, err := m.explainer.Explain(msg.prediction)
		if err != nil {
			m.logger.Error("gline explanation failed", zap.Error(err))
			return errorMsg{stateId: msg.stateId, err: err}
		}

		m.logger.Debug(
			"gline explained prediction",
			zap.Int("stateId", msg.stateId),
			zap.String("explanation", explanation),
		)
		return setExplanationMsg{stateId: msg.stateId, explanation: explanation}
	})
}

func (m appModel) handleClearScreen() (tea.Model, tea.Cmd) {
	// Log the current state before clearing
	m.logger.Debug("gline handleClearScreen called",
		zap.String("currentInput", m.textInput.Value()),
		zap.String("explanation", m.explanation),
		zap.Bool("hasExplanation", m.explanation != ""))

	// Use Bubble Tea's built-in screen clearing command for proper rendering pipeline integration
	// This ensures that lipgloss-styled components like info boxes render correctly after clearing
	m.logger.Debug("gline using tea.ClearScreen for proper rendering pipeline integration")

	// Return the model unchanged with the ClearScreen command
	// Bubble Tea will handle the screen clear and automatic re-render
	return m, tea.Cmd(func() tea.Msg {
		return tea.ClearScreen()
	})
}

func (m appModel) setExplanation(msg setExplanationMsg) (tea.Model, tea.Cmd) {
	if msg.stateId != m.predictionStateId {
		m.logger.Debug(
			"gline discarding explanation",
			zap.Int("startStateId", msg.stateId),
			zap.Int("newStateId", m.predictionStateId),
		)
		return m, nil
	}

	m.explanation = msg.explanation
	// Mark LLM as successful since explanation is the last step
	m.llmIndicator.SetStatus(LLMStatusSuccess)
	return m, nil
}

func Gline(
	prompt string,
	historyValues []string,
	explanation string,
	predictor Predictor,
	explainer Explainer,
	analytics PredictionAnalytics,
	logger *zap.Logger,
	options Options,
) (string, error) {
	p := tea.NewProgram(
		initialModel(prompt, historyValues, explanation, predictor, explainer, analytics, logger, options),
	)

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	appModel, ok := m.(appModel)
	if !ok {
		logger.Error("Gline resulted in an unexpected app model")
		panic("Gline resulted in an unexpected app model")
	}

	// Check if the session was interrupted by Ctrl+C
	if appModel.interrupted {
		return "", ErrInterrupted
	}

	fmt.Print(RESET_CURSOR_COLUMN + appModel.getFinalOutput() + "\n")

	if analytics != nil {
		err = analytics.NewEntry(appModel.lastPredictionInput, appModel.lastPrediction, appModel.result)
		if err != nil {
			logger.Error("failed to log analytics entry", zap.Error(err))
		}
	}

	return appModel.result, nil
}
