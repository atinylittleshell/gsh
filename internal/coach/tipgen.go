package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// LLMTipGenerator generates personalized tips using LLM
type LLMTipGenerator struct {
	runner         *interp.Runner
	historyManager *history.HistoryManager
	coachManager   *CoachManager
	logger         *zap.Logger
	cache          *TipCache
}

// NewLLMTipGenerator creates a new LLM tip generator
func NewLLMTipGenerator(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	coachManager *CoachManager,
	logger *zap.Logger,
) *LLMTipGenerator {
	return &LLMTipGenerator{
		runner:         runner,
		historyManager: historyManager,
		coachManager:   coachManager,
		logger:         logger,
		cache:          NewTipCache(50, 24*time.Hour),
	}
}

// TipContext contains all data needed for personalized tip generation
type TipContext struct {
	Username        string
	Level           int
	Title           string
	CurrentStreak   int
	TotalCommands   int
	TodayCommands   int
	TodayErrors     int
	TodayAccuracy   float64
	TopCommands     []commandFreq
	ErrorCommands   []commandFreq
	LongCommands    []string
	RecentErrors    []string
	Directories     []string
	GitUsage        int
	PipelineUsage   int
	RecentTipIDs    []string
}

type commandFreq struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

// GenerateTip generates a single personalized tip
func (g *LLMTipGenerator) GenerateTip(ctx context.Context) (*GeneratedTip, error) {
	// Check cache first
	if cached := g.cache.GetRandom(); cached != nil {
		g.cache.MarkShown(cached.ID)
		return cached, nil
	}

	// Build context
	tipContext, err := g.buildTipContext(ctx)
	if err != nil {
		g.logger.Warn("Failed to build tip context", zap.Error(err))
		return nil, err
	}

	// Generate with LLM
	tip, err := g.generateWithLLM(ctx, tipContext)
	if err != nil {
		return nil, err
	}

	// Cache and return
	g.cache.Add(tip)
	g.cache.MarkShown(tip.ID)
	return tip, nil
}

// GenerateBatchTips generates multiple tips at once
func (g *LLMTipGenerator) GenerateBatchTips(ctx context.Context, count int) ([]*GeneratedTip, error) {
	tipContext, err := g.buildTipContext(ctx)
	if err != nil {
		return nil, err
	}

	tips, err := g.generateBatchWithLLM(ctx, tipContext, count)
	if err != nil {
		return nil, err
	}

	// Cache all
	for _, tip := range tips {
		g.cache.Add(tip)
	}

	return tips, nil
}

// GetCachedTip returns a cached tip if available
func (g *LLMTipGenerator) GetCachedTip() *GeneratedTip {
	tip := g.cache.GetRandom()
	if tip != nil {
		g.cache.MarkShown(tip.ID)
	}
	return tip
}

// buildTipContext builds context from user history
func (g *LLMTipGenerator) buildTipContext(ctx context.Context) (*TipContext, error) {
	profile := g.coachManager.GetProfile()
	todayStats := g.coachManager.GetTodayStats()

	tipContext := &TipContext{
		Username:      profile.Username,
		Level:         profile.Level,
		Title:         profile.Title,
		CurrentStreak: profile.CurrentStreak,
	}

	// Get recent history
	entries, err := g.historyManager.GetRecentEntries("", 500)
	if err != nil {
		g.logger.Warn("Failed to get history", zap.Error(err))
	} else {
		tipContext.TotalCommands = len(entries)
		tipContext.TopCommands = g.analyzeCommandFrequency(entries, 10)
		tipContext.ErrorCommands = g.analyzeErrorCommands(entries, 5)
		tipContext.LongCommands = g.findLongCommands(entries, 5)
		tipContext.RecentErrors = g.getRecentErrors(entries, 5)
		tipContext.Directories = g.getUniqueDirectories(entries, 5)
		tipContext.GitUsage = g.countGitCommands(entries)
		tipContext.PipelineUsage = g.countPipelines(entries)
	}

	if todayStats != nil {
		tipContext.TodayCommands = todayStats.CommandsExecuted
		tipContext.TodayErrors = todayStats.CommandsFailed
		if todayStats.CommandsExecuted > 0 {
			tipContext.TodayAccuracy = float64(todayStats.CommandsSuccessful) / float64(todayStats.CommandsExecuted)
		}
	}

	tipContext.RecentTipIDs = g.cache.GetRecentIDs(20)

	return tipContext, nil
}

// analyzeCommandFrequency finds most frequently used commands
func (g *LLMTipGenerator) analyzeCommandFrequency(entries []history.HistoryEntry, limit int) []commandFreq {
	freq := make(map[string]int)

	for _, entry := range entries {
		cmd := normalizeCommand(entry.Command)
		freq[cmd]++
	}

	var result []commandFreq
	for cmd, count := range freq {
		if count >= 3 { // Only include commands used 3+ times
			result = append(result, commandFreq{Command: cmd, Count: count})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// analyzeErrorCommands finds commands with high error rates
func (g *LLMTipGenerator) analyzeErrorCommands(entries []history.HistoryEntry, limit int) []commandFreq {
	errorCounts := make(map[string]int)

	for _, entry := range entries {
		if entry.ExitCode.Valid && entry.ExitCode.Int32 != 0 {
			cmd := normalizeCommand(entry.Command)
			errorCounts[cmd]++
		}
	}

	var result []commandFreq
	for cmd, count := range errorCounts {
		if count >= 2 { // Only include if failed 2+ times
			result = append(result, commandFreq{Command: cmd, Count: count})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// findLongCommands finds frequently used long commands
func (g *LLMTipGenerator) findLongCommands(entries []history.HistoryEntry, limit int) []string {
	freq := make(map[string]int)

	for _, entry := range entries {
		cmd := entry.Command
		if len(cmd) > 20 { // Commands longer than 20 chars
			freq[cmd]++
		}
	}

	var result []string
	for cmd, count := range freq {
		if count >= 5 { // Used 5+ times
			result = append(result, cmd)
		}
	}

	// Sort by frequency
	sort.Slice(result, func(i, j int) bool {
		return freq[result[i]] > freq[result[j]]
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// getRecentErrors gets recent error commands
func (g *LLMTipGenerator) getRecentErrors(entries []history.HistoryEntry, limit int) []string {
	var errors []string

	for i := len(entries) - 1; i >= 0 && len(errors) < limit; i-- {
		entry := entries[i]
		if entry.ExitCode.Valid && entry.ExitCode.Int32 != 0 {
			errors = append(errors, entry.Command)
		}
	}

	return errors
}

// getUniqueDirectories gets unique working directories
func (g *LLMTipGenerator) getUniqueDirectories(entries []history.HistoryEntry, limit int) []string {
	seen := make(map[string]bool)
	var result []string

	for _, entry := range entries {
		if !seen[entry.Directory] && entry.Directory != "" {
			seen[entry.Directory] = true
			result = append(result, entry.Directory)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// countGitCommands counts git-related commands
func (g *LLMTipGenerator) countGitCommands(entries []history.HistoryEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Command, "git ") {
			count++
		}
	}
	return count
}

// countPipelines counts commands with pipes
func (g *LLMTipGenerator) countPipelines(entries []history.HistoryEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.Contains(entry.Command, "|") {
			count++
		}
	}
	return count
}

// normalizeCommand normalizes a command for comparison
func normalizeCommand(cmd string) string {
	// Get first word(s) up to arguments
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}

	// For common commands, include first argument
	first := parts[0]
	switch first {
	case "git", "docker", "kubectl", "npm", "yarn", "cargo", "go":
		if len(parts) > 1 {
			return first + " " + parts[1]
		}
	}

	return first
}

// generateWithLLM generates a tip using LLM
func (g *LLMTipGenerator) generateWithLLM(ctx context.Context, tipContext *TipContext) (*GeneratedTip, error) {
	llmClient, modelConfig := utils.GetLLMClient(g.runner, utils.FastModel)

	prompt := g.buildPrompt(tipContext)

	request := openai.ChatCompletionRequest{
		Model: modelConfig.ModelId,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: tipGeneratorSystemPrompt},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	if modelConfig.Temperature != nil {
		request.Temperature = float32(*modelConfig.Temperature)
	}

	response, err := llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	var tip GeneratedTip
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &tip); err != nil {
		return nil, fmt.Errorf("failed to parse tip: %w", err)
	}

	tip.ID = GenerateTipID()
	tip.GeneratedAt = time.Now()
	tip.ExpiresAt = time.Now().Add(24 * time.Hour)

	return &tip, nil
}

// generateBatchWithLLM generates multiple tips
func (g *LLMTipGenerator) generateBatchWithLLM(ctx context.Context, tipContext *TipContext, count int) ([]*GeneratedTip, error) {
	llmClient, modelConfig := utils.GetLLMClient(g.runner, utils.FastModel)

	prompt := g.buildBatchPrompt(tipContext, count)

	request := openai.ChatCompletionRequest{
		Model: modelConfig.ModelId,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: tipGeneratorSystemPrompt},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	if modelConfig.Temperature != nil {
		request.Temperature = float32(*modelConfig.Temperature)
	}

	response, err := llmClient.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	var batchResponse struct {
		Tips []*GeneratedTip `json:"tips"`
	}

	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &batchResponse); err != nil {
		return nil, err
	}

	for _, tip := range batchResponse.Tips {
		tip.ID = GenerateTipID()
		tip.GeneratedAt = time.Now()
		tip.ExpiresAt = time.Now().Add(24 * time.Hour)
	}

	return batchResponse.Tips, nil
}

// buildPrompt builds the prompt for tip generation
func (g *LLMTipGenerator) buildPrompt(ctx *TipContext) string {
	var sb strings.Builder

	sb.WriteString("Generate a personalized productivity tip for this gsh user.\n\n")

	sb.WriteString("## User Profile\n")
	sb.WriteString(fmt.Sprintf("- Username: %s\n", ctx.Username))
	sb.WriteString(fmt.Sprintf("- Level: %d (%s)\n", ctx.Level, ctx.Title))
	sb.WriteString(fmt.Sprintf("- Current Streak: %d days\n", ctx.CurrentStreak))
	sb.WriteString(fmt.Sprintf("- Total Commands: %d\n\n", ctx.TotalCommands))

	sb.WriteString("## Today's Activity\n")
	sb.WriteString(fmt.Sprintf("- Commands: %d\n", ctx.TodayCommands))
	sb.WriteString(fmt.Sprintf("- Errors: %d\n", ctx.TodayErrors))
	sb.WriteString(fmt.Sprintf("- Accuracy: %.1f%%\n\n", ctx.TodayAccuracy*100))

	if len(ctx.TopCommands) > 0 {
		sb.WriteString("## Top Commands (Most Used)\n")
		for _, cmd := range ctx.TopCommands {
			sb.WriteString(fmt.Sprintf("- %s (%d times)\n", cmd.Command, cmd.Count))
		}
		sb.WriteString("\n")
	}

	if len(ctx.ErrorCommands) > 0 {
		sb.WriteString("## Commands With Errors\n")
		for _, cmd := range ctx.ErrorCommands {
			sb.WriteString(fmt.Sprintf("- %s (%d errors)\n", cmd.Command, cmd.Count))
		}
		sb.WriteString("\n")
	}

	if len(ctx.LongCommands) > 0 {
		sb.WriteString("## Long Commands (Potential Alias Opportunities)\n")
		for _, cmd := range ctx.LongCommands {
			sb.WriteString(fmt.Sprintf("- %s\n", cmd))
		}
		sb.WriteString("\n")
	}

	if ctx.GitUsage > 0 {
		sb.WriteString(fmt.Sprintf("## Git Usage: %d commands\n\n", ctx.GitUsage))
	}

	if ctx.PipelineUsage > 0 {
		sb.WriteString(fmt.Sprintf("## Pipeline Usage: %d commands with pipes\n\n", ctx.PipelineUsage))
	}

	if len(ctx.RecentTipIDs) > 0 {
		sb.WriteString("## Recent Tips (Avoid Repeating)\n")
		sb.WriteString(strings.Join(ctx.RecentTipIDs, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("Based on this data, generate ONE personalized tip. The tip should:\n")
	sb.WriteString("1. Reference specific commands from their history\n")
	sb.WriteString("2. Provide actionable advice\n")
	sb.WriteString("3. Estimate the potential impact\n\n")
	sb.WriteString("Respond with JSON matching this schema:\n")
	sb.WriteString(tipResponseSchema)

	return sb.String()
}

// buildBatchPrompt builds prompt for multiple tips
func (g *LLMTipGenerator) buildBatchPrompt(ctx *TipContext, count int) string {
	base := g.buildPrompt(ctx)

	return base + fmt.Sprintf(`

---

Generate %d DIFFERENT tips, each addressing a unique aspect of this user's workflow.
Ensure variety across tip types (alias, workflow, learning, error prevention, etc.).

Respond with JSON:
{
  "tips": [
    { ... tip 1 ... },
    { ... tip 2 ... },
    ...
  ]
}`, count)
}

const tipGeneratorSystemPrompt = `You are an expert shell productivity coach for gsh, an intelligent shell. Your role is to analyze user command history and generate highly personalized, actionable tips.

## Guidelines
1. Tips MUST be based on the user's actual command history
2. Reference specific commands and counts from their data
3. Provide clear, actionable suggestions (aliases, tools, workflows)
4. Estimate realistic impact (keystrokes saved, time saved)
5. Be encouraging but not condescending
6. Be technically precise

## Tip Types
- alias: Suggest aliases for frequently typed commands
- workflow: Suggest combining command sequences
- tool_discovery: Recommend better tools for their common tasks
- error_fix: Address recurring error patterns
- efficiency: Keyboard shortcuts, command options they're missing
- git: Git-specific optimizations
- encouragement: Celebrate improvements

## Response Format
Always respond with valid JSON. Include reasoning to explain relevance.`

const tipResponseSchema = `{
  "type": "string (productivity|efficiency|learning|error_fix|workflow|alias|tool_discovery|security|git|time_management|fun_fact|encouragement)",
  "category": "string (specific subcategory)",
  "title": "string (short, catchy, max 50 chars)",
  "content": "string (1-3 sentences)",
  "reasoning": "string (why this is relevant to THIS user)",
  "command": "string (related command if applicable)",
  "suggestion": "string (actionable suggestion, e.g., alias definition)",
  "impact": "string (estimated impact)",
  "confidence": "number (0-1)",
  "priority": "number (1-10)",
  "actionable": "boolean",
  "action_type": "string (alias|function|tool|config|learning|none)",
  "based_on": ["array of commands/patterns this tip is based on"]
}`
