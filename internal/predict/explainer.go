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

type LLMExplainer struct {
	runner      *interp.Runner
	llmClient   *openai.Client
	contextText string
	logger      *zap.Logger
	modelId     string
	temperature *float64
}

func NewLLMExplainer(
	runner *interp.Runner,
	logger *zap.Logger,
) *LLMExplainer {
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.FastModel)
	return &LLMExplainer{
		runner:      runner,
		llmClient:   llmClient,
		contextText: "",
		logger:      logger,
		modelId:     modelConfig.ModelId,
		temperature: modelConfig.Temperature,
	}
}

func (p *LLMExplainer) UpdateContext(context *map[string]string) {
	contextTypes := environment.GetContextTypesForExplanation(p.runner, p.logger)
	p.contextText = utils.ComposeContextText(context, contextTypes, p.logger)
}

func (e *LLMExplainer) Explain(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	schema, err := EXPLAINED_COMMAND_SCHEMA.MarshalJSON()
	if err != nil {
		return "", err
	}

	systemMessage := fmt.Sprintf(`You are gsh, an intelligent shell program.
You will be given a bash command entered by me, enclosed in <command> tags.

# Instructions
* Give a concise explanation of what the command will do for me
* If any uncommon arguments are present in the command, 
  format your explanation in markdown and explain arguments in a bullet point list

# Latest Context
%s

# Response JSON Schema
%s`,
		e.contextText,
		string(schema),
	)

	userMessage := fmt.Sprintf(
		`<command>%s</command>`,
		input,
	)

	e.logger.Debug(
		"explaining prediction using LLM",
		zap.String("system", systemMessage),
		zap.String("user", userMessage),
	)

	request := openai.ChatCompletionRequest{
		Model: e.modelId,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}
	if e.temperature != nil {
		request.Temperature = float32(*e.temperature)
	}

	chatCompletion, err := e.llmClient.CreateChatCompletion(context.TODO(), request)

	if err != nil {
		return "", err
	}

	explanation := explainedCommand{}
	_ = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &explanation)

	e.logger.Debug(
		"LLM explanation response",
		zap.Any("response", explanation),
	)

	return explanation.Explanation, nil
}
