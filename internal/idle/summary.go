package idle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// SummaryGenerator generates idle summaries using the slow LLM model
type SummaryGenerator struct {
	runner         *interp.Runner
	historyManager *history.HistoryManager
	logger         *zap.Logger
}

// NewSummaryGenerator creates a new idle summary generator
func NewSummaryGenerator(runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger) *SummaryGenerator {
	return &SummaryGenerator{
		runner:         runner,
		historyManager: historyManager,
		logger:         logger,
	}
}

// GenerateSummary generates a 1-sentence summary of what the user was doing
// based on commands from the last 5 minutes
func (g *SummaryGenerator) GenerateSummary(ctx context.Context) (string, error) {
	// Get commands from the last 5 minutes
	since := time.Now().Add(-5 * time.Minute)
	entries, err := g.historyManager.GetEntriesSince(since)
	if err != nil {
		return "", fmt.Errorf("failed to get recent commands: %w", err)
	}

	// If no commands in the last 5 minutes, return empty
	if len(entries) == 0 {
		g.logger.Debug("no commands in last 5 minutes for idle summary")
		return "", nil
	}

	// Format commands for the LLM
	var commandList strings.Builder
	for _, entry := range entries {
		exitStatus := "✓"
		if entry.ExitCode.Valid && entry.ExitCode.Int32 != 0 {
			exitStatus = fmt.Sprintf("✗(%d)", entry.ExitCode.Int32)
		}
		commandList.WriteString(fmt.Sprintf("[%s] %s %s\n",
			entry.CreatedAt.Format("15:04:05"),
			exitStatus,
			entry.Command,
		))
	}

	// Get the slow model client
	client, modelConfig := utils.GetLLMClient(g.runner, utils.SlowModel)

	// Build the prompt
	systemPrompt := `You are a helpful assistant that summarizes shell activity.
Given a list of recent shell commands, provide a single concise sentence describing what the user was doing or trying to accomplish.
Be specific about the task, not generic. Focus on the goal, not the individual commands.
Keep your response to ONE sentence only, no more than 15 words.
Do not start with "The user" or "You were". Just describe the activity directly.
Examples of good responses:
- "Setting up a new Go project with git version control"
- "Debugging test failures in the authentication module"
- "Exploring directory structure to find configuration files"`

	userPrompt := fmt.Sprintf("Recent commands from the last 5 minutes:\n\n%s\n\nSummarize what I was doing in one sentence:", commandList.String())

	// Create the chat completion request
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}

	request := openai.ChatCompletionRequest{
		Model:    modelConfig.ModelId,
		Messages: messages,
	}

	if modelConfig.Temperature != nil {
		request.Temperature = float32(*modelConfig.Temperature)
	}

	// Make the API call
	resp, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Clean up the summary - remove quotes if present
	summary = strings.Trim(summary, "\"'")

	g.logger.Debug("generated idle summary",
		zap.String("summary", summary),
		zap.Int("command_count", len(entries)),
	)

	return summary, nil
}
