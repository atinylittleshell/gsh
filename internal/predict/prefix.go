package predict

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type LLMPrefixPredictor struct {
	runner            *interp.Runner
	historyManager    *history.HistoryManager
	llmClient         *openai.Client
	contextText       string
	logger            *zap.Logger
	modelId           string
	temperature       *float64
	numHistoryContext int
}

func NewLLMPrefixPredictor(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	logger *zap.Logger,
) *LLMPrefixPredictor {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)
	return &LLMPrefixPredictor{
		runner:         runner,
		historyManager: historyManager,
		llmClient:      llmClient,
		contextText:    "",
		logger:         logger,
		modelId:        modelConfig.ModelId,
		temperature:    modelConfig.Temperature,
	}
}

func (p *LLMPrefixPredictor) UpdateContext(context *map[string]string) {
	contextTypes := environment.GetContextTypesForPredictionWithPrefix(p.runner, p.logger)
	p.contextText = utils.ComposeContextText(context, contextTypes, p.logger)
	p.numHistoryContext = environment.GetContextNumHistoryConcise(p.runner, p.logger)
}

func (p *LLMPrefixPredictor) Predict(input string) (string, string, error) {
	if strings.HasPrefix(input, "#") {
		// Don't do prediction for agent chat messages
		p.logger.Debug("skipping prediction for agent chat message")
		return "", "", nil
	}

	schema, err := PREDICTED_COMMAND_SCHEMA.MarshalJSON()
	if err != nil {
		p.logger.Error("failed to marshal schema", zap.Error(err))
		return "", "", err
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
You are asked to predict what the complete bash command is.

# Instructions
* Based on the prefix and other context, analyze the my potential intent
* Your prediction must start with the partial command as a prefix
* Your prediction must be a valid, single-line, complete bash command

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
		"predicting using LLM",
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

	chatCompletion, err := p.llmClient.CreateChatCompletion(context.TODO(), request)

	if err != nil {
		p.logger.Error("LLM API call failed", zap.Error(err))
		return "", "", err
	}

	prediction := PredictedCommand{}
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &prediction)
	if err != nil {
		p.logger.Error("failed to unmarshal prediction", zap.Error(err), zap.String("content", chatCompletion.Choices[0].Message.Content))
	}

	return prediction.PredictedCommand, userMessage, nil
}
