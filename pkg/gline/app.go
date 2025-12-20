package gline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/git"
	"github.com/atinylittleshell/gsh/internal/system"
	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	defaultExplanation  string // Shown when buffer is blank (e.g., coach tips)
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
	coachTipStyle    lipgloss.Style

	// Multiline support
	multilineState *MultilineState
	originalPrompt string
	height         int

	// LLM status indicator
	llmIndicator LLMIndicator

	// Border Status
	borderStatus BorderStatusModel

	// Idle summary tracking
	lastInputTime      time.Time
	idleSummaryShown   bool
	idleSummaryPending bool
	idleSummaryStateId int
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

// resourceMsg carries updated system resources
type resourceMsg struct {
	resources *system.Resources
}

type gitStatusMsg struct {
	status *git.RepoStatus
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

// Idle summary messages
type idleCheckMsg struct {
	stateId int
}

type setIdleSummaryMsg struct {
	stateId int
	summary string
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

	borderStatus := NewBorderStatusModel()
	borderStatus.UpdateContext(options.User, options.Host, options.CurrentDirectory)

	return appModel{
		predictor: predictor,
		explainer: explainer,
		analytics: analytics,
		logger:    logger,
		options:   options,

		textInput:          textInput,
		dirty:              false,
		prediction:         "",
		explanation:        explanation,
		defaultExplanation: explanation, // Store for restoring when buffer is blank
		historyValues:      historyValues,
		result:             "",
		appState:           Active,
		interrupted:        false, // Explicitly initialize to prevent stateful behavior

		predictionStateId: 0,

		explanationStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")),
		completionStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Red
		coachTipStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Faded gray

		// Initialize multiline state
		multilineState: NewMultilineState(),
		originalPrompt: prompt,

		llmIndicator: NewLLMIndicator(),
		borderStatus: borderStatus,

		// Initialize idle summary tracking
		lastInputTime:      time.Now(),
		idleSummaryShown:   false,
		idleSummaryPending: false,
		idleSummaryStateId: 0,
	}
}

func (m appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.llmIndicator.Tick(),
		func() tea.Msg {
			return attemptPredictionMsg{
				stateId: m.predictionStateId,
			}
		},
		m.fetchResources(),
		m.fetchGitStatus(),
	}

	// Start idle check timer if enabled
	if m.options.IdleSummaryTimeout > 0 && m.options.IdleSummaryGenerator != nil {
		cmds = append(cmds, m.scheduleIdleCheck())
	}

	return tea.Batch(cmds...)
}

func (m appModel) scheduleIdleCheck() tea.Cmd {
	stateId := m.idleSummaryStateId
	timeout := time.Duration(m.options.IdleSummaryTimeout) * time.Second
	return tea.Tick(timeout, func(t time.Time) tea.Msg {
		return idleCheckMsg{stateId: stateId}
	})
}

func (m appModel) fetchResources() tea.Cmd {
	return func() tea.Msg {
		res := system.GetResources()
		return resourceMsg{resources: res}
	}
}

func (m appModel) fetchGitStatus() tea.Cmd {
	return func() tea.Msg {
		if m.options.CurrentDirectory == "" {
			return nil
		}
		// Create a context with timeout for git status check
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		status := git.GetStatusWithContext(ctx, m.options.CurrentDirectory)
		return gitStatusMsg{status: status}
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case LLMTickMsg:
		m.llmIndicator.Update()
		if m.llmIndicator.GetStatus() == LLMStatusInFlight {
			return m, m.llmIndicator.Tick()
		}
		return m, nil

	case resourceMsg:
		m.borderStatus.UpdateResources(msg.resources)
		// Schedule next update after 1 second
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			// Instead of returning resourceMsg directly (which would block if done synchronously),
			// we trigger another fetch command which runs in a goroutine
			return "fetch_resources_trigger"
		})

	case string:
		if msg == "fetch_resources_trigger" {
			return m, m.fetchResources()
		}

	case gitStatusMsg:
		if msg.status != nil {
			m.borderStatus.UpdateGit(msg.status)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.textInput.Width = msg.Width
		m.explanationStyle = m.explanationStyle.Width(max(1, msg.Width-2))
		m.completionStyle = m.completionStyle.Width(max(1, msg.Width-2))
		m.borderStatus.SetWidth(max(0, msg.Width-2))
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

	case idleCheckMsg:
		return m.handleIdleCheck(msg)

	case setIdleSummaryMsg:
		return m.handleSetIdleSummary(msg)

	case tea.KeyMsg:
		switch msg.String() {

		// TODO: replace with custom keybindings
		case "backspace":
			if !m.textInput.InReverseSearch() {
				// if the input is already empty, we should clear prediction and restore default tip
				if m.textInput.Value() == "" {
					m.dirty = true
					m.predictionStateId++
					m.clearPredictionAndRestoreDefault()
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

			// Set result to empty string so shell doesn't try to execute it
			m.result = ""
			// Use interrupt message to indicate Ctrl+C was pressed
			// We do not reset multiline state here so that Gline() can reconstruct the full input
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

	// Track if content is pre-formatted (completion/history boxes) and should skip word wrapping
	isPreformatted := false

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
			isPreformatted = true
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
			isPreformatted = true

		} else if completionBox != "" {
			assistantContent = completionBox
			isPreformatted = true
		} else if helpBox != "" {
			assistantContent = helpBox
		} else {
			assistantContent = m.explanation
		}
	}

	// Track if this is a coach tip for styling after word wrap
	isCoachTip := m.explanation == m.defaultExplanation && m.explanation != ""

	// Render Assistant Box with custom border that includes LLM indicators
	boxWidth := max(0, m.textInput.Width-2)
	borderColor := lipgloss.Color("62")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Word wrap content to fit box width, then split into lines
	innerWidth := max(0, boxWidth-2) // Account for left/right borders
	// Content area is innerWidth minus 2 spaces for left/right padding
	contentWidth := innerWidth - 2
	// Use custom word wrapping that uses GetRuneWidth for accurate Unicode/emoji width calculation
	// This ensures coach tips with emoji render correctly in the assistant box
	// Note: Skip word wrapping for completion/history boxes as they are already formatted with proper columns
	var wrappedContent string
	if isPreformatted {
		// Completion and history boxes are pre-formatted, don't word wrap
		wrappedContent = assistantContent
	} else {
		wrappedContent = WordwrapWithRuneWidth(assistantContent, contentWidth)
	}
	lines := strings.Split(wrappedContent, "\n")

	// Apply faded style to each line of coach tips after word wrapping
	if isCoachTip {
		for i, line := range lines {
			if line != "" {
				lines[i] = m.coachTipStyle.Render(line)
			}
		}
	}
	if len(lines) > availableHeight {
		lines = lines[:availableHeight]
	}

	// Vertically center all content in the box
	if len(lines) < availableHeight {
		// Calculate padding for vertical centering
		topPadding := (availableHeight - len(lines)) / 2
		bottomPadding := availableHeight - len(lines) - topPadding

		// Add empty lines at top
		centeredLines := make([]string, 0, availableHeight)
		for i := 0; i < topPadding; i++ {
			centeredLines = append(centeredLines, "")
		}
		centeredLines = append(centeredLines, lines...)
		// Add empty lines at bottom
		for i := 0; i < bottomPadding; i++ {
			centeredLines = append(centeredLines, "")
		}
		lines = centeredLines
	}

	// Top Border Logic
	// ╭[Badge][risk]──[context]╮
	topLeft := m.borderStatus.RenderTopLeft()
	// Calculate available space for top context
	// width - 2 (corners) - len(topLeft) - 2 (padding maybe?)

	// We construct top border in pieces
	// corner + topLeft + separator + context + ... + corner
	// Actually typical lipgloss border is uniform. We need to override.

	// We manually draw the top line.
	// "╭" + [Badge][Risk] + "──" + [Context] + "──" + "╮"

	// Let's compute exact widths.
	// Use TopLeftWidth() method which accounts for terminal-specific rendering
	// of emoji characters like 🤖, rather than lipgloss.Width() which may be incorrect
	topLeftWidth := m.borderStatus.TopLeftWidth()

	// Available width for middle
	middleWidth := innerWidth

	// If middleWidth is small, we might have issues.
	if middleWidth <= 0 {
		middleWidth = 0
	}

	topContentWidth := middleWidth
	// We need some lines between topLeft and Context?
	// Spec says: "Command kind badge immediately followed by the execution risk meter."
	// "Top edge: Prompt-style context stripes ... separated by line-continuation characters"
	// So: ╭[Badge][Risk]────[Context]────╮

	// Context
	// We want to fill the remaining space with context, right aligned or distributed?
	// Spec says: "Top edge (left-to-right): Prompt-style context stripes"
	// But Top-left is Badge/Risk.
	// So Badge/Risk comes first, then Context.
	// Should we pad with lines between them?

	// If we just concatenate: Badge Risk Context
	// And pad the rest with lines?
	// Or: Badge Risk ── Context ── ?

	// Let's assume: Badge Risk [context] ──────────
	// Or: Badge Risk ── [context] ── ?

	// Let's try to put context immediately after, separated by line.
	// But context stripes are variable width.

	// Render context with available width
	// Available = topContentWidth - topLeftWidth
	contextAvailableWidth := topContentWidth - topLeftWidth - 1 // -1 for separator
	if contextAvailableWidth < 0 {
		contextAvailableWidth = 0
	}

	topContext := m.borderStatus.RenderTopContext(contextAvailableWidth)
	topContextWidth := lipgloss.Width(topContext)

	// Line filler
	fillerWidth := topContentWidth - topLeftWidth - topContextWidth
	if fillerWidth < 0 {
		fillerWidth = 0
	}

	// Construction
	// ╭ + topLeft + [filler/separator] + topContext + [filler] + ╮
	// We prefer context to be visible.
	// The prompt context stripes usually sit on the line.

	// Design choice:
	// ╭[Badge][Risk]──[Context]──────────────╮

	// Note: We need to use border style for the line parts (╭, ─, ╮)
	// But Badge/Risk/Context have their own colors.

	var topBar strings.Builder
	topBar.WriteString(borderStyle.Render("╭"))
	topBar.WriteString(topLeft)

	if topContext != "" {
		// Separator line
		// Use Divider style from borderStatus? Or just border color?
		// Spec: "separated by line-continuation characters ... degrade to icon-only"
		// "Apply subtle color to the divider"
		// borderStatus handles internal dividers in Context.
		// Here we need divider between Risk and Context.
		topBar.WriteString(m.borderStatus.styles.Divider.Render("─"))
		topBar.WriteString(topContext)

		// Remaining filler
		if fillerWidth > 1 {
			topBar.WriteString(borderStyle.Render(strings.Repeat("─", fillerWidth-1)))
		}
	} else {
		// Just fill
		if fillerWidth > 0 {
			topBar.WriteString(borderStyle.Render(strings.Repeat("─", fillerWidth)))
		}
	}
	topBar.WriteString(borderStyle.Render("╮"))

	var result strings.Builder
	result.WriteString(topBar.String())
	result.WriteString("\n")

	// Content lines with left/right borders
	// Middle content - with one space padding on each side
	// Content is already wrapped at contentWidth
	for _, line := range lines {
		// Truncate or pad line to fit content width
		// Use stringWidthWithAnsi instead of lipgloss.Width to properly handle emoji
		lineWidth := stringWidthWithAnsi(line)
		if lineWidth > contentWidth {
			line = truncateWithAnsi(line, contentWidth)
			lineWidth = stringWidthWithAnsi(line)
		}
		padding := max(0, contentWidth-lineWidth)
		result.WriteString(borderStyle.Render("│"))
		result.WriteString(" ") // Left padding
		if isCoachTip {
			// Right-justify coach tips
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(line)
		} else {
			// Left-justify other content
			result.WriteString(line)
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(" ") // Right padding
		result.WriteString(borderStyle.Render("│"))
		result.WriteString("\n")
	}

	// Bottom border with indicators: ╰[Res]──[User@Host]──[Res]── Fast:✓ Slow:○ ╯
	// Bottom-left: Resource Glance
	// Bottom-center: User@Host (centered) - suppressed if window too narrow
	// Bottom-right: LLM Indicator (preserved)

	bottomLeft := m.borderStatus.RenderBottomLeft()
	bottomCenter := m.borderStatus.RenderBottomCenter()
	bottomCenterWidth := lipgloss.Width(bottomCenter)
	bottomLeftWidth := lipgloss.Width(bottomLeft)

	indicatorStr := " " + m.llmIndicator.View() + " "
	// Use the indicator's Width() method which accounts for terminal-specific rendering
	// of the lightning bolt character, rather than lipgloss.Width() which may be incorrect
	indicatorLen := 2 + m.llmIndicator.Width() // 2 spaces + lightning bolt width

	// Calculate minimum required space for all elements
	minRequiredWidth := bottomLeftWidth + indicatorLen + 10 // 10 chars minimum for spacing

	// Use middleWidth to match the top bar's content width
	bottomContentWidth := middleWidth

	// Determine if we have enough space for user@hostname
	showUserHost := bottomContentWidth > minRequiredWidth && bottomCenter != ""

	// Calculate available space for centering
	var leftFillerWidth, rightFillerWidth, availableSpace int
	var totalUsedWidth int

	if showUserHost {
		totalUsedWidth = bottomLeftWidth + bottomCenterWidth + indicatorLen
		availableSpace = bottomContentWidth - totalUsedWidth

		if availableSpace < 0 {
			// Not enough space even with user@host, drop it
			showUserHost = false
			totalUsedWidth = bottomLeftWidth + indicatorLen
			availableSpace = bottomContentWidth - totalUsedWidth
		}

		// Distribute extra space to center the user@host
		leftFillerWidth = availableSpace / 2
		rightFillerWidth = availableSpace - leftFillerWidth
	} else {
		// User@host suppressed, just center between left and right
		totalUsedWidth = bottomLeftWidth + indicatorLen
		availableSpace = bottomContentWidth - totalUsedWidth

		leftFillerWidth = availableSpace / 2
		rightFillerWidth = availableSpace - leftFillerWidth
	}

	// Construction
	// ╰ + bottomLeft + leftFiller + center + rightFiller + indicator + ╯

	result.WriteString(borderStyle.Render("╰"))
	result.WriteString(bottomLeft)
	if showUserHost && leftFillerWidth > 0 {
		result.WriteString(borderStyle.Render(strings.Repeat("─", leftFillerWidth)))
	}
	if showUserHost && bottomCenter != "" {
		result.WriteString(bottomCenter)
	}
	if showUserHost && rightFillerWidth > 0 {
		result.WriteString(borderStyle.Render(strings.Repeat("─", rightFillerWidth)))
	}
	if !showUserHost && availableSpace > 0 {
		// User@host suppressed, just fill the space
		result.WriteString(borderStyle.Render(strings.Repeat("─", availableSpace)))
	}
	result.WriteString(indicatorStr)
	result.WriteString(borderStyle.Render("╯"))

	return inputStr + "\n" + result.String()
}

// stringWidthWithAnsi calculates the display width of a string, handling ANSI escape codes
// Uses terminal-specific probing for emoji characters to get accurate widths
func stringWidthWithAnsi(s string) int {
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		width += GetRuneWidth(r)
	}

	return width
}

// truncateWithAnsi truncates a string to maxWidth display columns, handling ANSI escape codes
// Uses terminal-specific probing for emoji characters to get accurate widths
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
		runeWidth := GetRuneWidth(r)
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

		// Update border status with new input
		m.borderStatus.UpdateInput(newVal)

		// Clear any existing error when user types
		m.lastError = nil

		// Reset idle timer and state when user types
		m.lastInputTime = time.Now()
		m.idleSummaryShown = false
		m.idleSummaryStateId++

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
			// we should clear the prediction and restore the default tip
			m.clearPredictionAndRestoreDefault()
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

// clearPredictionAndRestoreDefault clears the prediction and restores the default
// explanation (e.g., coach tips) - used when the input buffer becomes blank
func (m *appModel) clearPredictionAndRestoreDefault() {
	m.prediction = ""
	m.explanation = m.defaultExplanation
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

	// When input is blank and there's no prediction, preserve the default explanation (coach tips)
	if strings.TrimSpace(m.textInput.Value()) == "" && prediction == "" {
		m.explanation = m.defaultExplanation
		// Reset LLM status to prevent pulsing when showing coaching tips
		m.llmIndicator.SetStatus(LLMStatusSuccess)
		return m, nil
	}

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
		// Don't show indicator when buffer is empty - just return clean state
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

// handleIdleCheck checks if the user is idle and triggers summary generation
func (m appModel) handleIdleCheck(msg idleCheckMsg) (tea.Model, tea.Cmd) {
	// Ignore stale idle check messages
	if msg.stateId != m.idleSummaryStateId {
		return m, nil
	}

	// Don't generate if idle summary is disabled or already shown
	if m.options.IdleSummaryTimeout <= 0 || m.options.IdleSummaryGenerator == nil {
		return m, nil
	}

	if m.idleSummaryShown || m.idleSummaryPending {
		return m, nil
	}

	// Check if user input is empty (idle at command prompt)
	if strings.TrimSpace(m.textInput.Value()) != "" {
		// User has typed something, reschedule idle check
		return m, m.scheduleIdleCheck()
	}

	// Check if enough time has passed since last input
	idleTimeout := time.Duration(m.options.IdleSummaryTimeout) * time.Second
	if time.Since(m.lastInputTime) < idleTimeout {
		// Not idle long enough, reschedule
		return m, m.scheduleIdleCheck()
	}

	// User is idle, trigger summary generation
	m.idleSummaryPending = true
	m.logger.Debug("user idle, generating summary",
		zap.Duration("idle_duration", time.Since(m.lastInputTime)),
	)

	stateId := m.idleSummaryStateId
	return m, tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		summary, err := m.options.IdleSummaryGenerator(ctx)
		if err != nil {
			m.logger.Debug("idle summary generation failed", zap.Error(err))
			return setIdleSummaryMsg{stateId: stateId, summary: ""}
		}

		return setIdleSummaryMsg{stateId: stateId, summary: summary}
	})
}

// handleSetIdleSummary sets the idle summary in the assistant box
func (m appModel) handleSetIdleSummary(msg setIdleSummaryMsg) (tea.Model, tea.Cmd) {
	// Ignore stale messages
	if msg.stateId != m.idleSummaryStateId {
		return m, nil
	}

	m.idleSummaryPending = false

	// If no summary (generation failed or no commands), don't update
	if msg.summary == "" {
		return m, nil
	}

	// Set the summary as the default explanation
	m.idleSummaryShown = true
	m.defaultExplanation = "💭 " + msg.summary
	m.explanation = m.defaultExplanation

	m.logger.Debug("idle summary displayed",
		zap.String("summary", msg.summary),
	)

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
		// Reconstruct what was on screen so it persists
		var inputStr string
		if appModel.multilineState.IsActive() {
			lines := appModel.multilineState.GetLines()
			for i, line := range lines {
				if i == 0 {
					inputStr += appModel.originalPrompt + line + "\n"
				} else {
					inputStr += "> " + line + "\n"
				}
			}
		}
		// Append current line with ^C
		inputStr += appModel.textInput.Prompt + appModel.textInput.Value() + "^C\n"

		fmt.Print(RESET_CURSOR_COLUMN + inputStr)
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
