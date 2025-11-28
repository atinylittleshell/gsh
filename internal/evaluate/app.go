package evaluate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/predict"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"golang.org/x/term"
)

type evaluationResult struct {
	truth        string
	predicted    string
	score        float64
	err          error
	inputTokens  int
	outputTokens int
	duration     float64 // duration in seconds
}

type model struct {
	analyticsManager *analytics.AnalyticsManager
	entries          []analytics.AnalyticsEntry
	results          []evaluationResult
	currentIndex     int
	currentIteration int
	totalIterations  int
	progress         progress.Model
	spinner          spinner.Model
	llmClient        *openai.Client
	modelId          string
	temperature      *float64
	quitting         bool
	isWarmingUp      bool
}

func initialModel(analyticsManager *analytics.AnalyticsManager, entries []analytics.AnalyticsEntry, llmClient *openai.Client, modelId string, temperature *float64, iterations int) model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		analyticsManager: analyticsManager,
		entries:          entries,
		results:          make([]evaluationResult, 0),
		progress:         p,
		spinner:          s,
		llmClient:        llmClient,
		modelId:          modelId,
		temperature:      temperature,
		isWarmingUp:      true,
		currentIteration: 1,
		totalIterations:  iterations,
	}
}

type evaluateMsg struct {
	result evaluationResult
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.evaluateNext,
	)
}

func (m model) evaluateNext() tea.Msg {
	if m.currentIndex >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.currentIndex]
	result := evaluateEntry(m.analyticsManager, entry, m.llmClient, m.modelId, m.temperature)

	if result.err != nil && m.analyticsManager.Logger != nil {
		m.analyticsManager.Logger.Error("error evaluating entry", zap.Error(result.err))
	}

	return evaluateMsg{
		result: result,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case evaluateMsg:
		if m.isWarmingUp {
			m.isWarmingUp = false
			return m, tea.Batch(
				m.spinner.Tick,
				m.evaluateNext,
			)
		}

		m.results = append(m.results, msg.result)
		m.currentIndex++

		if m.currentIndex >= len(m.entries) {
			if m.currentIteration >= m.totalIterations {
				return m, tea.Quit
			}
			// Start next iteration
			m.currentIteration++
			m.currentIndex = 0
		}

		return m, tea.Batch(
			m.spinner.Tick,
			m.evaluateNext,
		)

	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	if m.isWarmingUp {
		s.WriteString(fmt.Sprintf("%s Warming up...\n", m.spinner.View()))
	} else if m.currentIndex < len(m.entries) {
		s.WriteString(m.spinner.View())
		s.WriteString(m.progress.ViewAs(float64(m.currentIndex+1) / float64(len(m.entries))))
		s.WriteString(fmt.Sprintf(" (Entry %d/%d, Iteration %d/%d)\n",
			m.currentIndex+1,
			len(m.entries),
			m.currentIteration,
			m.totalIterations))
	}

	perfectMatches := 0
	errors := 0
	scores := make([]float64, 0)
	totalInputTokens := 0
	totalOutputTokens := 0
	totalDuration := 0.0
	for _, result := range m.results {
		if result.truth == result.predicted && result.score == 1.0 && result.err == nil {
			perfectMatches++
		}
		if result.err != nil {
			errors++
		}
		scores = append(scores, result.score)
		totalInputTokens += result.inputTokens
		totalOutputTokens += result.outputTokens
		totalDuration += result.duration
	}

	totalEntries := len(m.results)
	avgInputTokens := float64(totalInputTokens) / float64(totalEntries)
	avgDuration := totalDuration / float64(totalEntries)
	outputTokensPerSecond := float64(totalOutputTokens) / totalDuration

	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers("Metric", "Value", "Percentage").
		Row("Model ID", m.modelId, "").
		Row("Evaluated Entries", fmt.Sprintf("%d", totalEntries), "").
		Row("Prediction Errors", fmt.Sprintf("%d", errors), fmt.Sprintf("%.1f%%", float64(errors)/float64(totalEntries)*100)).
		Row("Perfect Predictions", fmt.Sprintf("%d", perfectMatches), fmt.Sprintf("%.1f%%", float64(perfectMatches)/float64(totalEntries)*100)).
		Row("Average Similarity", fmt.Sprintf("%.2f", average(scores)), fmt.Sprintf("%.1f%%", average(scores)*100)).
		Row("Average Latency", fmt.Sprintf("%.1fs", avgDuration), "").
		Row("Input Tokens Per Request", fmt.Sprintf("%.1f", avgInputTokens), "").
		Row("Output Tokens Per Second", fmt.Sprintf("%.1f", outputTokensPerSecond), "")

	s.WriteString(t.String() + "\n")

	return gline.RESET_CURSOR_COLUMN + s.String()
}

func average(numbers []float64) float64 {
	total := 0.0
	for _, n := range numbers {
		total += n
	}
	return total / float64(len(numbers))
}

func RunEvaluation(analyticsManager *analytics.AnalyticsManager, limit int, customModelId string, iterations int) error {
	llmClient, llmModelConfig := utils.GetLLMClient(analyticsManager.Runner, utils.FastModel)

	// Use custom model ID if provided, otherwise use default
	modelId := llmModelConfig.ModelId
	if customModelId != "" {
		modelId = customModelId
	}

	// Get recent entries
	entries, err := analyticsManager.GetRecentEntries(limit)
	if err != nil {
		return err
	}

	if len(entries) < limit {
		errMsg := fmt.Sprintf("not enough entries to evaluate: requested %d but only found %d", limit, len(entries))
		fmt.Println(errMsg)
		return errors.New(errMsg)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		p := tea.NewProgram(initialModel(analyticsManager, entries, llmClient, modelId, llmModelConfig.Temperature, iterations))

		_, err = p.Run()
		return err
	}

	return nil
}

func evaluateEntry(analyticsManager *analytics.AnalyticsManager, entry analytics.AnalyticsEntry, llmClient *openai.Client, modelId string, temperature *float64) evaluationResult {
	startTime := time.Now()
	result := evaluationResult{
		truth: entry.Actual,
	}

	request := openai.ChatCompletionRequest{
		Model: modelId,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: entry.Input,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}
	if temperature != nil {
		request.Temperature = float32(*temperature)
	}

	chatCompletion, err := llmClient.CreateChatCompletion(context.Background(), request)

	if analyticsManager.Logger != nil {
		analyticsManager.Logger.Debug(
			"chat completion for evaluating entry",
			zap.Any("entry", entry),
			zap.Any("chatCompletion", chatCompletion),
		)
	}

	if err != nil {
		result.err = err
		return result
	}

	result.duration = time.Since(startTime).Seconds()
	result.inputTokens = chatCompletion.Usage.PromptTokens
	result.outputTokens = chatCompletion.Usage.CompletionTokens

	prediction := predict.PredictedCommand{}
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &prediction)
	if err != nil {
		result.err = err
		return result
	}

	result.predicted = prediction.PredictedCommand
	score, err := SimilarityScore(prediction.PredictedCommand, entry.Actual)
	if err != nil {
		result.err = err
		return result
	}

	result.score = score
	return result
}
