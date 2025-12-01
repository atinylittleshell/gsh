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

	// Read provider setting (ollama, openai, openrouter)
	provider := strings.ToLower(runner.Vars[varPrefix+"PROVIDER"].String())
	if provider == "" {
		provider = "ollama" // Default to ollama
	}

	// Read API key separately from provider
	apiKey := runner.Vars[varPrefix+"API_KEY"].String()

	// Read base URL (may be overridden by user)
	baseURL := runner.Vars[varPrefix+"BASE_URL"].String()

	// Set defaults based on provider
	switch provider {
	case "openai":
		if apiKey == "" {
			apiKey = "sk-" // Placeholder, user should provide real key
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	case "openrouter":
		if apiKey == "" {
			apiKey = "sk-or-" // Placeholder, user should provide real key
		}
		if baseURL == "" {
			baseURL = "https://openrouter.ai/api/v1"
		}
	default: // "ollama" or unknown
		if apiKey == "" {
			apiKey = "ollama"
		}
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1/"
		}
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
	if provider == "openrouter" || strings.HasPrefix(strings.ToLower(baseURL), "https://openrouter.ai/") {
		if headers == nil {
			headers = make(map[string]string)
		}
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
