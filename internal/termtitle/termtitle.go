// Package termtitle provides dynamic terminal window title updates based on
// user command history using an LLM to generate contextual titles.
package termtitle

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/atinylittleshell/gsh/internal/termfeatures"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

const (
	// MaxWindowSize is the maximum number of commands to track in the sliding window.
	MaxWindowSize = 100

	// MaxBackoffInterval is the maximum number of commands between title updates.
	MaxBackoffInterval = 40

	// InitialBackoffInterval is the initial number of commands after which to update.
	InitialBackoffInterval = 1
)

// Manager handles terminal title updates with exponential backoff.
type Manager struct {
	runner   *interp.Runner
	terminal *termfeatures.Terminal
	logger   *zap.Logger

	mu                  sync.Mutex
	commandWindow       []string // Sliding window of recent commands
	commandsSinceUpdate int      // Commands since last title update
	nextUpdateInterval  int      // Commands until next update (exponential backoff)
	currentTitle        string   // Current window title
}

// NewManager creates a new terminal title manager.
func NewManager(runner *interp.Runner, logger *zap.Logger) *Manager {
	return &Manager{
		runner:             runner,
		terminal:           termfeatures.New(),
		logger:             logger,
		commandWindow:      make([]string, 0, MaxWindowSize),
		nextUpdateInterval: InitialBackoffInterval,
	}
}

// RecordCommand records a command and potentially triggers a title update.
// This should be called after each command is executed.
func (m *Manager) RecordCommand(command string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Skip empty commands
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Add to sliding window
	m.commandWindow = append(m.commandWindow, command)
	if len(m.commandWindow) > MaxWindowSize {
		// Remove oldest command
		m.commandWindow = m.commandWindow[1:]
	}

	// Increment counter
	m.commandsSinceUpdate++

	// Check if we should update the title
	if m.commandsSinceUpdate >= m.nextUpdateInterval {
		m.commandsSinceUpdate = 0

		// Calculate next interval (exponential backoff, capped at max)
		m.nextUpdateInterval *= 2
		if m.nextUpdateInterval > MaxBackoffInterval {
			m.nextUpdateInterval = MaxBackoffInterval
		}

		// Trigger async title update
		go m.updateTitle()
	}
}

// updateTitle generates and sets a new window title based on recent commands.
func (m *Manager) updateTitle() {
	if !m.terminal.SupportsWindowTitle() {
		m.logger.Debug("terminal does not support window title")
		return
	}

	m.mu.Lock()
	commands := make([]string, len(m.commandWindow))
	copy(commands, m.commandWindow)
	m.mu.Unlock()

	if len(commands) == 0 {
		return
	}

	title, err := m.generateTitle(commands)
	if err != nil {
		m.logger.Debug("failed to generate title", zap.Error(err))
		return
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return
	}

	m.mu.Lock()
	m.currentTitle = title
	m.mu.Unlock()

	result := m.terminal.SetWindowTitle(title)
	if !result.Success && result.Error != nil {
		m.logger.Debug("failed to set window title",
			zap.Error(result.Error),
			zap.String("method", result.Method))
	} else {
		m.logger.Debug("window title updated",
			zap.String("title", title),
			zap.String("method", result.Method))
	}
}

// generateTitle uses the fast LLM to generate a window title.
func (m *Manager) generateTitle(commands []string) (string, error) {
	client, modelConfig := utils.GetLLMClient(m.runner, utils.FastModel)

	// Format commands for the prompt (most recent last)
	var commandList strings.Builder
	for i, cmd := range commands {
		commandList.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
	}

	systemPrompt := `You are a helpful assistant that generates concise terminal window titles.
Given a list of recent shell commands, generate a short, descriptive title that captures the user's current activity.

Rules:
- Keep the title under 40 characters
- Be specific about the task, not generic
- Focus on the most recent activity pattern
- Do not include shell prompt symbols or command syntax
- Use title case
- Output ONLY the title, nothing else

Examples of good titles:
- "Git: Debugging Auth Tests"
- "Docker: Building API Image"
- "React: Component Styling"
- "Python: Data Analysis"
- "DevOps: K8s Deployment"`

	userPrompt := fmt.Sprintf("Recent commands (oldest to newest):\n\n%s\nGenerate a window title:", commandList.String())

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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	title := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Clean up the title - remove quotes if present
	title = strings.Trim(title, "\"'`")

	// Ensure title is not too long
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	return title, nil
}

// GetCurrentTitle returns the current window title.
func (m *Manager) GetCurrentTitle() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTitle
}

// Reset resets the manager state, clearing the command window and resetting backoff.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandWindow = make([]string, 0, MaxWindowSize)
	m.commandsSinceUpdate = 0
	m.nextUpdateInterval = InitialBackoffInterval
	m.currentTitle = ""
}

// ResetTitle clears the terminal window title.
func (m *Manager) ResetTitle() {
	m.terminal.ResetWindowTitle()
}
