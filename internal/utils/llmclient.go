package utils

import (
	"encoding/json"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"mvdan.cc/sh/v3/interp"
)

type LLMModelType string

type LLMModelConfig struct {
	ModelId           string
	Temperature       *float64
	ParallelToolCalls *bool
}

const (
	FastModel LLMModelType = "FAST"
	SlowModel LLMModelType = "SLOW"
)

func GetLLMClient(runner *interp.Runner, modelType LLMModelType) (*openai.Client, LLMModelConfig) {
	varPrefix := "GSH_" + string(modelType) + "_MODEL_"

	apiKey := runner.Vars[varPrefix+"API_KEY"].String()
	if apiKey == "" {
		apiKey = "ollama"
	}
	baseURL := runner.Vars[varPrefix+"BASE_URL"].String()
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1/"
	}
	modelId := runner.Vars[varPrefix+"ID"].String()
	if modelId == "" {
		modelId = "qwen2.5"
	}

	var temperature *float64
	temperatureString := runner.Vars[varPrefix+"TEMPERATURE"].String()
	if temperatureString != "" {
		temperatureValue, err := strconv.ParseFloat(temperatureString, 32)
		if err == nil {
			temperature = &temperatureValue
		}
	}

	var parallelToolCalls *bool
	parallelToolCallsString := runner.Vars[varPrefix+"PARALLEL_TOOL_CALLS"].String()
	if parallelToolCallsString != "" {
		parallelToolCallsValue, err := strconv.ParseBool(parallelToolCallsString)
		if err == nil {
			parallelToolCalls = &parallelToolCallsValue
		}
	}

	var headers map[string]string
	_ = json.Unmarshal([]byte(runner.Vars[varPrefix+"HEADERS"].String()), &headers)

	// Special headers for the openrouter.ai API
	if strings.HasPrefix(strings.ToLower(baseURL), "https://openrouter.ai/") {
		headers["HTTP-Referer"] = "https://github.com/atinylittleshell/gsh"
		headers["X-Title"] = "gsh - The Generative Shell"
	}

	llmClientConfig := openai.DefaultConfig(apiKey)
	llmClientConfig.BaseURL = baseURL
	llmClientConfig.HTTPClient = NewLLMHttpClient(headers)

	return openai.NewClientWithConfig(llmClientConfig), LLMModelConfig{
		ModelId:           modelId,
		Temperature:       temperature,
		ParallelToolCalls: parallelToolCalls,
	}
}
