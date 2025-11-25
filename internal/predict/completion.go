package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type LLMCompletionPredictor struct {
	runner            *interp.Runner
	historyManager    *history.HistoryManager
	llmClient         *openai.Client
	contextText       string
	logger            *zap.Logger
	modelId           string
	temperature       *float64
	numHistoryContext int
}

func NewLLMCompletionPredictor(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	logger *zap.Logger,
) *LLMCompletionPredictor {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)
	return &LLMCompletionPredictor{
		runner:         runner,
		historyManager: historyManager,
		llmClient:      llmClient,
		contextText:    "",
		logger:         logger,
		modelId:        modelConfig.ModelId,
		temperature:    modelConfig.Temperature,
	}
}

func (p *LLMCompletionPredictor) UpdateContext(context *map[string]string) {
	contextTypes := environment.GetContextTypesForPredictionWithPrefix(p.runner, p.logger)
	p.contextText = utils.ComposeContextText(context, contextTypes, p.logger)
	p.numHistoryContext = environment.GetContextNumHistoryConcise(p.runner, p.logger)
}

func (p *LLMCompletionPredictor) Predict(input string) ([]string, error) {
	if strings.HasPrefix(input, "#") {
		// Don't do prediction for agent chat messages
		return nil, nil
	}

	schema, err := COMPLETION_CANDIDATES_SCHEMA.MarshalJSON()
	if err != nil {
		return nil, err
	}

	matchingHistoryEntries, err := p.historyManager.GetRecentEntriesByPrefix(
		input,
		p.numHistoryContext,
	)
	matchingHistoryContext := strings.Builder{}
	if err == nil {
		for _, entry := range matchingHistoryEntries {
			matchingHistoryContext.WriteString(fmt.Sprintf(
				"%s\n",
				entry.Command,
			))
		}
	}

	userMessage := fmt.Sprintf(`You are gsh, an intelligent shell program.
You will be given a partial bash command prefix entered by me, enclosed in <prefix> tags.
You are asked to provide a list of valid completions for the command.

# Instructions
* Based on the prefix and other context, analyze the my potential intent
* Provide valid completions that replace the LAST word/token in the prefix
* Do NOT return the full command prefix, only the text that should replace the current partial word
* Return a list of strings

# Example
Prefix: "git ch"
Candidates: ["checkout", "cherry-pick"] (NOT "git checkout")

Prefix: "ls -l"
Candidates: ["-la", "-lh"] (NOT "ls -la")

# Best Practices
%s

# Latest Context
%s

# Previous Commands with Similar Prefix
%s

# Response JSON Schema
%s

<prefix>%s</prefix>`,
		BEST_PRACTICES,
		p.contextText,
		matchingHistoryContext.String(),
		string(schema),
		input,
	)

	p.logger.Debug(
		"completing using LLM",
		zap.String("user", userMessage),
	)

	request := openai.ChatCompletionRequest{
		Model: p.modelId,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}
	if p.temperature != nil {
		request.Temperature = float32(*p.temperature)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	chatCompletion, err := p.llmClient.CreateChatCompletion(ctx, request)

	if err != nil {
		return nil, err
	}

	prediction := CompletionCandidates{}
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &prediction)
	if err != nil {
		return nil, err
	}

	p.logger.Debug(
		"LLM completion response",
		zap.Any("response", prediction),
	)

	return prediction.Candidates, nil
}
