package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// SubagentSelector uses LLM to intelligently select the best subagent for a given prompt
type SubagentSelector struct {
	llmClient      *openai.Client
	llmModelConfig utils.LLMModelConfig
	logger         *zap.Logger
}

// SelectionResult represents the LLM's subagent selection decision
type SelectionResult struct {
	SubagentID string `json:"subagent_id"`
	Confidence int    `json:"confidence"` // 0-100
	Reasoning  string `json:"reasoning"`
}

// NewSubagentSelector creates a new intelligent subagent selector
func NewSubagentSelector(runner *interp.Runner, logger *zap.Logger) *SubagentSelector {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)

	return &SubagentSelector{
		llmClient:      llmClient,
		llmModelConfig: modelConfig,
		logger:         logger,
	}
}

// SelectBestSubagent uses LLM to determine the most appropriate subagent for the given prompt
func (s *SubagentSelector) SelectBestSubagent(prompt string, availableSubagents map[string]*Subagent) (*Subagent, error) {
	if len(availableSubagents) == 0 {
		return nil, fmt.Errorf("no subagents available")
	}

	// If only one subagent available, return it
	if len(availableSubagents) == 1 {
		for _, subagent := range availableSubagents {
			return subagent, nil
		}
	}

	// Build context about available subagents
	subagentContext := s.buildSubagentContext(availableSubagents)

	// Create the selection prompt
	systemPrompt := s.buildSelectionPrompt(subagentContext)

	// Call LLM to make selection
	result, err := s.callLLMForSelection(systemPrompt, prompt)
	if err != nil {
		s.logger.Warn("LLM selection failed, falling back to string matching", zap.Error(err))
		return nil, err
	}

	// Find and return the selected subagent
	if result.SubagentID != "" {
		if subagent, exists := availableSubagents[result.SubagentID]; exists {
			s.logger.Debug("LLM selected subagent",
				zap.String("subagentID", result.SubagentID),
				zap.Int("confidence", result.Confidence),
				zap.String("reasoning", result.Reasoning))
			return subagent, nil
		}
	}

	return nil, fmt.Errorf("LLM selected unknown subagent: %s", result.SubagentID)
}

// buildSubagentContext creates a description of available subagents for the LLM
func (s *SubagentSelector) buildSubagentContext(subagents map[string]*Subagent) string {
	var context strings.Builder
	context.WriteString("Available subagents:\n")

	for id, subagent := range subagents {
		// Use fmt.Fprintf instead of manual WriteString formatting to avoid linter issues
		fmt.Fprintf(&context, "- ID: %s\n", id)
		fmt.Fprintf(&context, "  Name: %s\n", subagent.Name)
		fmt.Fprintf(&context, "  Description: %s\n", subagent.Description)
		fmt.Fprintf(&context, "  Tools: %s\n", strings.Join(subagent.AllowedTools, ", "))
		if subagent.Type == RooType {
			context.WriteString("  Type: Roo mode\n")
		} else {
			context.WriteString("  Type: Claude agent\n")
		}
		context.WriteString("\n")
	}

	return context.String()
}

// buildSelectionPrompt creates the system prompt for subagent selection
func (s *SubagentSelector) buildSelectionPrompt(subagentContext string) string {
	return fmt.Sprintf(`You are a subagent selector for a generative AI shell. Your job is to analyze a user's prompt and determine which specialized subagent would be most appropriate to handle their request.

%s

Analyze the user's prompt and select the most appropriate subagent. Consider:
1. The specific task the user is asking for
2. The tools each subagent has access to
3. The expertise area of each subagent
4. Keywords and context clues in the prompt

Respond with a JSON object containing:
- "subagent_id": the ID of the best matching subagent
- "confidence": a number from 0-100 indicating your confidence in the selection
- "reasoning": a brief explanation of why you selected this subagent

If no subagent seems appropriate, set subagent_id to "none".`, subagentContext)
}

// callLLMForSelection makes the actual LLM call to select a subagent
func (s *SubagentSelector) callLLMForSelection(systemPrompt, userPrompt string) (*SelectionResult, error) {
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

	req := openai.ChatCompletionRequest{
		Model:       s.llmModelConfig.ModelId,
		Messages:    messages,
		Temperature: 0.1, // Low temperature for consistent selection
		MaxTokens:   200,  // Short response expected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := s.llmClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse the JSON response
	var result SelectionResult
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &result, nil
}
