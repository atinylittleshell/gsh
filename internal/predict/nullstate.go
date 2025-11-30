package predict

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

type LLMNullStatePredictor struct {
	runner      *interp.Runner
	llmClient   *openai.Client
	contextText string
	logger      *zap.Logger
	modelId     string
	temperature *float64
}

func NewLLMNullStatePredictor(
	runner *interp.Runner,
	logger *zap.Logger,
) *LLMNullStatePredictor {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)
	return &LLMNullStatePredictor{
		runner:      runner,
		llmClient:   llmClient,
		contextText: "",
		logger:      logger,
		modelId:     modelConfig.ModelId,
		temperature: modelConfig.Temperature,
	}
}

func (p *LLMNullStatePredictor) UpdateContext(context *map[string]string) {
	contextTypes := environment.GetContextTypesForPredictionWithoutPrefix(p.runner, p.logger)
	p.contextText = utils.ComposeContextText(context, contextTypes, p.logger)
}

func (p *LLMNullStatePredictor) Predict(input string) (string, string, error) {
	if input != "" {
		// this predictor is only for null state
		p.logger.Debug("skipping null-state prediction for non-empty input")
		return "", "", nil
	}

	schema, err := PREDICTED_COMMAND_SCHEMA.MarshalJSON()
	if err != nil {
		p.logger.Error("failed to marshal schema", zap.Error(err))
		return "", "", err
	}

	userMessage := fmt.Sprintf(`You are gsh, an intelligent shell program.
You are asked to predict the next command I'm likely to want to run.

# Instructions
* Based on the context, analyze the my potential intent
* Your prediction must be a valid, single-line, complete bash command

# Best Practices
%s

# Latest Context
%s

# Response JSON Schema
%s

Now predict what my next command should be.`,
		BEST_PRACTICES,
		p.contextText,
		string(schema),
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
