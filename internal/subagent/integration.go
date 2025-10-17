package subagent

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// SubagentIntegration handles the integration of subagents with gsh's shell system
type SubagentIntegration struct {
	manager   *SubagentManager
	executors map[string]*SubagentExecutor // Cache of active executors
	selector  *SubagentSelector            // Intelligent subagent selector
	runner    *interp.Runner
	history   *history.HistoryManager
	logger    *zap.Logger
}

// NewSubagentIntegration creates a new subagent integration instance
func NewSubagentIntegration(runner *interp.Runner, history *history.HistoryManager, logger *zap.Logger) *SubagentIntegration {
	manager := NewSubagentManager(runner, logger)

	// Load subagents on initialization
	if err := manager.LoadSubagents(logger); err != nil {
		logger.Warn("Failed to load subagents during initialization", zap.Error(err))
	}

	return &SubagentIntegration{
		manager:   manager,
		executors: make(map[string]*SubagentExecutor),
		selector:  NewSubagentSelector(runner, logger),
		runner:    runner,
		history:   history,
		logger:    logger,
	}
}

// HandleCommand processes potential subagent commands and returns true if handled
func (si *SubagentIntegration) HandleCommand(chatMessage string) (bool, <-chan string, *Subagent, error) {
	// Ensure subagents are up-to-date (reload if directory changed)
	si.ensureSubagentsUpToDate()

	// Check for subagent invocation patterns
	subagentID, prompt := si.parseSubagentCommand(chatMessage)
	if subagentID == "" {
		return false, nil, nil, nil // Not a subagent command
	}

	si.logger.Debug("Subagent command detected",
		zap.String("subagentID", subagentID),
		zap.String("prompt", prompt))

	// Find the subagent
	subagent, exists := si.manager.GetSubagent(subagentID)
	if !exists {
		// Try fuzzy matching by name
		subagent, exists = si.manager.FindSubagentByName(subagentID)
		if !exists {
			return true, nil, nil, fmt.Errorf("subagent '%s' not found", subagentID)
		}
	}

	// Get or create executor for this subagent
	executor := si.getExecutor(subagent)

	// Execute the command
	responseChannel, err := executor.Chat(prompt)
	if err != nil {
		return true, nil, subagent, fmt.Errorf("failed to chat with subagent '%s': %w", subagent.Name, err)
	}

	return true, responseChannel, subagent, nil
}

// parseSubagentCommand parses various subagent invocation patterns
func (si *SubagentIntegration) parseSubagentCommand(chatMessage string) (string, string) {
	chatMessage = strings.TrimSpace(chatMessage)

	// Pattern 1: @subagent-name prompt (Claude style)
	if strings.HasPrefix(chatMessage, "@") {
		parts := strings.SplitN(chatMessage[1:], " ", 2)
		if len(parts) >= 1 {
			subagentID := parts[0]
			prompt := ""
			if len(parts) > 1 {
				prompt = parts[1]
			}
			return subagentID, prompt
		}
	}

	// Pattern 2: @:mode-slug prompt (Roo Code style)
	if strings.HasPrefix(chatMessage, "@:") {
		parts := strings.SplitN(chatMessage[2:], " ", 2)
		if len(parts) >= 1 {
			subagentID := parts[0]
			prompt := ""
			if len(parts) > 1 {
				prompt = parts[1]
			}
			return subagentID, prompt
		}
	}

	// Pattern 3: Intelligent auto-detection using LLM
	// Use the intelligent selector to find the best subagent for the entire message
	availableSubagents := si.manager.GetAllSubagents()
	if len(availableSubagents) > 0 {
		selectedSubagent, err := si.selector.SelectBestSubagent(chatMessage, availableSubagents)
		if err == nil && selectedSubagent != nil {
			return selectedSubagent.ID, chatMessage
		}
		// Log the error but continue with fallback
		si.logger.Debug("Intelligent subagent selection failed, trying fallback", zap.Error(err))

		// Fallback: Try the old string matching approach
		words := strings.Fields(chatMessage)
		if len(words) > 0 {
			firstWord := words[0]
			if subagent, exists := si.manager.FindSubagentByName(firstWord); exists {
				prompt := strings.Join(words[1:], " ")
				si.logger.Debug("Used fallback string matching for subagent selection",
					zap.String("subagent", subagent.ID))
				return subagent.ID, prompt
			}
		}
	}

	return "", "" // Not a subagent command
}

// getExecutor gets or creates an executor for a subagent
func (si *SubagentIntegration) getExecutor(subagent *Subagent) *SubagentExecutor {
	if executor, exists := si.executors[subagent.ID]; exists {
		return executor
	}

	// Create new executor
	executor := NewSubagentExecutor(si.runner, si.history, si.logger, subagent)
	si.executors[subagent.ID] = executor

	si.logger.Debug("Created new subagent executor", zap.String("subagent", subagent.ID))
	return executor
}

// HandleAgentControl processes subagent-related agent controls
func (si *SubagentIntegration) HandleAgentControl(control string) bool {
	// Ensure subagents are up-to-date before handling controls
	si.ensureSubagentsUpToDate()

	switch control {
	case "subagents":
		fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: "+si.manager.GetSubagentsSummary()) + gline.RESET_CURSOR_COLUMN)
		return true

	case "reload-subagents":
		if err := si.manager.Reload(si.logger); err != nil {
			fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR(fmt.Sprintf("Failed to reload subagents: %s", err)) + "\n")
		} else {
			// Clear executor cache to pick up changes
			si.executors = make(map[string]*SubagentExecutor)
			fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Subagents reloaded successfully.\n") + gline.RESET_CURSOR_COLUMN)
		}
		return true

	default:
		// Check if it's a subagent-info command
		if strings.HasPrefix(control, "subagent-info ") {
			subagentID := strings.TrimSpace(strings.TrimPrefix(control, "subagent-info "))
			si.showSubagentInfo(subagentID)
			return true
		}

		// Check if it's a subagent reset command
		if strings.HasPrefix(control, "reset-") {
			subagentID := strings.TrimSpace(strings.TrimPrefix(control, "reset-"))
			si.resetSubagent(subagentID)
			return true
		}

		return false
	}
}

// showSubagentInfo displays detailed information about a specific subagent
func (si *SubagentIntegration) showSubagentInfo(subagentID string) {
	subagent, exists := si.manager.GetSubagent(subagentID)
	if !exists {
		subagent, exists = si.manager.FindSubagentByName(subagentID)
		if !exists {
			fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR(fmt.Sprintf("Subagent '%s' not found", subagentID)) + "\n")
			return
		}
	}

	var info strings.Builder
	info.WriteString(fmt.Sprintf("Subagent: %s (%s)\n", subagent.Name, subagent.ID))
	info.WriteString(fmt.Sprintf("Type: %s\n", subagent.Type))
	info.WriteString(fmt.Sprintf("Description: %s\n", subagent.Description))
	info.WriteString(fmt.Sprintf("Available Tools: %v\n", subagent.AllowedTools))
	if subagent.FileRegex != "" {
		info.WriteString(fmt.Sprintf("File Access Pattern: %s\n", subagent.FileRegex))
	}
	if subagent.Model != "" {
		info.WriteString(fmt.Sprintf("Model: %s\n", subagent.Model))
	}
	info.WriteString(fmt.Sprintf("Configuration File: %s\n", subagent.FilePath))

	fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: "+info.String()) + gline.RESET_CURSOR_COLUMN)
}

// resetSubagent resets the chat session for a specific subagent
func (si *SubagentIntegration) resetSubagent(subagentID string) {
	if executor, exists := si.executors[subagentID]; exists {
		executor.ResetChat()
		fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(fmt.Sprintf("gsh: Reset chat session for subagent '%s'.\n", subagentID)) + gline.RESET_CURSOR_COLUMN)
	} else {
		fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(fmt.Sprintf("gsh: No active session for subagent '%s'.\n", subagentID)) + gline.RESET_CURSOR_COLUMN)
	}
}

// GetManager returns the subagent manager for external access
func (si *SubagentIntegration) GetManager() *SubagentManager {
	return si.manager
}

// GetManagerInterface returns the subagent manager as an interface for external access
func (si *SubagentIntegration) GetManagerInterface() SubagentManagerInterface {
	return si.manager
}

// GetCompletionProvider returns a completion provider for the subagent system
func (si *SubagentIntegration) GetCompletionProvider() completion.SubagentProvider {
	return NewCompletionAdapter(si.manager, si.ensureSubagentsUpToDate)
}

// ensureSubagentsUpToDate checks if subagents should be reloaded and reloads them if necessary
func (si *SubagentIntegration) ensureSubagentsUpToDate() {
	if si.manager.ShouldReload() {
		si.logger.Debug("Subagents need to be reloaded")
		if err := si.manager.LoadSubagents(si.logger); err != nil {
			si.logger.Warn("Failed to reload subagents", zap.Error(err))
		} else {
			// Clear executor cache when subagents are reloaded to pick up new configurations
			si.executors = make(map[string]*SubagentExecutor)
			si.logger.Debug("Subagents reloaded successfully")
		}
	}
}
