package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestModelDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result *EvalResult, err error)
	}{
		{
			name: "Model declaration with Anthropic",
			input: `model claude {
				provider: "openai",
				apiKey: env.ANTHROPIC_API_KEY,
				model: "claude-3-5-sonnet-20241022",
				temperature: 0.7,
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check that the model is registered in the environment
				modelVal, ok := result.Env.Get("claude")
				if !ok {
					t.Fatalf("model 'claude' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				if model.Name != "claude" {
					t.Errorf("expected model name 'claude', got %q", model.Name)
				}

				// Check provider
				provider, ok := model.Config["provider"]
				if !ok {
					t.Fatalf("model config missing 'provider'")
				}
				providerStr, ok := provider.(*StringValue)
				if !ok {
					t.Fatalf("expected provider to be *StringValue, got %T", provider)
				}
				if providerStr.Value != "openai" {
					t.Errorf("expected provider 'openai', got %q", providerStr.Value)
				}

				// Check apiKey exists (can be any type including null)
				_, ok = model.Config["apiKey"]
				if !ok {
					t.Fatalf("model config missing 'apiKey'")
				}

				// Check model
				modelName, ok := model.Config["model"]
				if !ok {
					t.Fatalf("model config missing 'model'")
				}
				modelNameStr, ok := modelName.(*StringValue)
				if !ok {
					t.Fatalf("expected model to be *StringValue, got %T", modelName)
				}
				if modelNameStr.Value != "claude-3-5-sonnet-20241022" {
					t.Errorf("expected model 'claude-3-5-sonnet-20241022', got %q", modelNameStr.Value)
				}

				// Check temperature
				temp, ok := model.Config["temperature"]
				if !ok {
					t.Fatalf("model config missing 'temperature'")
				}
				tempNum, ok := temp.(*NumberValue)
				if !ok {
					t.Fatalf("expected temperature to be *NumberValue, got %T", temp)
				}
				if tempNum.Value != 0.7 {
					t.Errorf("expected temperature 0.7, got %f", tempNum.Value)
				}
			},
		},
		{
			name: "Model declaration with OpenAI",
			input: `model gpt4 {
				provider: "openai",
				apiKey: env.OPENAI_API_KEY,
				model: "gpt-4",
				temperature: 0.5,
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("gpt4")
				if !ok {
					t.Fatalf("model 'gpt4' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				if model.Name != "gpt4" {
					t.Errorf("expected model name 'gpt4', got %q", model.Name)
				}

				// Check provider
				provider, ok := model.Config["provider"]
				if !ok {
					t.Fatalf("model config missing 'provider'")
				}
				providerStr, ok := provider.(*StringValue)
				if !ok {
					t.Fatalf("expected provider to be *StringValue, got %T", provider)
				}
				if providerStr.Value != "openai" {
					t.Errorf("expected provider 'openai', got %q", providerStr.Value)
				}
			},
		},
		{
			name: "Model declaration with Ollama (local)",
			input: `model llama {
				provider: "openai",
				baseURL: "http://localhost:11434/v1",
				model: "llama3.2:3b",
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("llama")
				if !ok {
					t.Fatalf("model 'llama' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check baseURL
				baseURL, ok := model.Config["baseURL"]
				if !ok {
					t.Fatalf("model config missing 'baseURL'")
				}
				baseURLStr, ok := baseURL.(*StringValue)
				if !ok {
					t.Fatalf("expected baseURL to be *StringValue, got %T", baseURL)
				}
				if baseURLStr.Value != "http://localhost:11434/v1" {
					t.Errorf("expected baseURL 'http://localhost:11434/v1', got %q", baseURLStr.Value)
				}
			},
		},
		{
			name: "Model declaration with minimal config",
			input: `model minimal {
				provider: "openai",
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("minimal")
				if !ok {
					t.Fatalf("model 'minimal' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				if len(model.Config) != 1 {
					t.Errorf("expected 1 config field, got %d", len(model.Config))
				}
			},
		},
		{
			name: "Model declaration with template literal",
			input: `model dynamic {
				provider: "openai",
				apiKey: "Bearer ${env.TOKEN}",
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("dynamic")
				if !ok {
					t.Fatalf("model 'dynamic' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check apiKey
				apiKey, ok := model.Config["apiKey"]
				if !ok {
					t.Fatalf("model config missing 'apiKey'")
				}
				apiKeyStr, ok := apiKey.(*StringValue)
				if !ok {
					t.Fatalf("expected apiKey to be *StringValue, got %T", apiKey)
				}
				// Template literals are currently parsed as regular strings
				if !strings.Contains(apiKeyStr.Value, "Bearer") {
					t.Errorf("expected apiKey to contain 'Bearer', got %q", apiKeyStr.Value)
				}
			},
		},
		{
			name: "Multiple model declarations",
			input: `model claude {
				provider: "openai",
				model: "claude-3-5-sonnet-20241022",
			}
			model gpt4 {
				provider: "openai",
				model: "gpt-4",
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check both models are registered
				claudeVal, ok := result.Env.Get("claude")
				if !ok {
					t.Fatalf("model 'claude' not found in environment")
				}
				if _, ok := claudeVal.(*ModelValue); !ok {
					t.Fatalf("expected claude to be *ModelValue, got %T", claudeVal)
				}

				gpt4Val, ok := result.Env.Get("gpt4")
				if !ok {
					t.Fatalf("model 'gpt4' not found in environment")
				}
				if _, ok := gpt4Val.(*ModelValue); !ok {
					t.Fatalf("expected gpt4 to be *ModelValue, got %T", gpt4Val)
				}
			},
		},
		{
			name: "Model declaration with computed values",
			input: `baseUrl = "http://localhost"
			port = 11434
			model mymodel {
				provider: "openai",
				baseURL: baseUrl,
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("mymodel")
				if !ok {
					t.Fatalf("model 'mymodel' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check baseURL
				baseURL, ok := model.Config["baseURL"]
				if !ok {
					t.Fatalf("model config missing 'baseURL'")
				}
				baseURLStr, ok := baseURL.(*StringValue)
				if !ok {
					t.Fatalf("expected baseURL to be *StringValue, got %T", baseURL)
				}
				if baseURLStr.Value != "http://localhost" {
					t.Errorf("expected baseURL 'http://localhost', got %q", baseURLStr.Value)
				}
			},
		},
		{
			name: "Model with maxTokens",
			input: `model limited {
				provider: "openai",
				model: "gpt-4",
				maxTokens: 1000,
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("limited")
				if !ok {
					t.Fatalf("model 'limited' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check maxTokens
				maxTokens, ok := model.Config["maxTokens"]
				if !ok {
					t.Fatalf("model config missing 'maxTokens'")
				}
				maxTokensNum, ok := maxTokens.(*NumberValue)
				if !ok {
					t.Fatalf("expected maxTokens to be *NumberValue, got %T", maxTokens)
				}
				if maxTokensNum.Value != 1000 {
					t.Errorf("expected maxTokens 1000, got %f", maxTokensNum.Value)
				}
			},
		},
		{
			name: "Model with custom headers",
			input: `model withHeaders {
				provider: "openai",
				model: "gpt-4",
				headers: {
					"X-Custom-Header": "custom-value",
					"X-Another-Header": "another-value",
				},
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("withHeaders")
				if !ok {
					t.Fatalf("model 'withHeaders' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check headers
				headers, ok := model.Config["headers"]
				if !ok {
					t.Fatalf("model config missing 'headers'")
				}
				headersObj, ok := headers.(*ObjectValue)
				if !ok {
					t.Fatalf("expected headers to be *ObjectValue, got %T", headers)
				}

				// Check specific header values
				customHeader := headersObj.GetPropertyValue("X-Custom-Header")
				if customHeader.Type() != ValueTypeString {
					t.Fatalf("expected X-Custom-Header to be string, got %s", customHeader.Type())
				}
				if customHeader.String() != "custom-value" {
					t.Errorf("expected X-Custom-Header 'custom-value', got %q", customHeader.String())
				}

				anotherHeader := headersObj.GetPropertyValue("X-Another-Header")
				if anotherHeader.Type() != ValueTypeString {
					t.Fatalf("expected X-Another-Header to be string, got %s", anotherHeader.Type())
				}
				if anotherHeader.String() != "another-value" {
					t.Errorf("expected X-Another-Header 'another-value', got %q", anotherHeader.String())
				}
			},
		},
		{
			name: "Model with headers using env vars",
			input: `model withEnvHeaders {
				provider: "openai",
				model: "gpt-4",
				headers: {
					"Authorization": "Bearer ${env.CUSTOM_TOKEN}",
				},
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("withEnvHeaders")
				if !ok {
					t.Fatalf("model 'withEnvHeaders' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check headers exist
				headers, ok := model.Config["headers"]
				if !ok {
					t.Fatalf("model config missing 'headers'")
				}
				_, ok = headers.(*ObjectValue)
				if !ok {
					t.Fatalf("expected headers to be *ObjectValue, got %T", headers)
				}
			},
		},
		{
			name: "Model with extraBody",
			input: `model withExtraBody {
				provider: "openai",
				model: "gpt-4",
				extraBody: {
					"custom_param": "custom-value",
					"nested": {
						"key": "value",
					},
					"numeric": 42,
					"flag": true,
				},
			}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				modelVal, ok := result.Env.Get("withExtraBody")
				if !ok {
					t.Fatalf("model 'withExtraBody' not found in environment")
				}

				model, ok := modelVal.(*ModelValue)
				if !ok {
					t.Fatalf("expected *ModelValue, got %T", modelVal)
				}

				// Check extraBody exists
				extraBody, ok := model.Config["extraBody"]
				if !ok {
					t.Fatalf("model config missing 'extraBody'")
				}
				extraBodyObj, ok := extraBody.(*ObjectValue)
				if !ok {
					t.Fatalf("expected extraBody to be *ObjectValue, got %T", extraBody)
				}

				// Check specific values
				customParam := extraBodyObj.GetPropertyValue("custom_param")
				if customParam.Type() != ValueTypeString {
					t.Fatalf("expected custom_param to be string, got %s", customParam.Type())
				}
				if customParam.String() != "custom-value" {
					t.Errorf("expected custom_param 'custom-value', got %q", customParam.String())
				}

				// Check nested object
				nested := extraBodyObj.GetPropertyValue("nested")
				if nested.Type() != ValueTypeObject {
					t.Fatalf("expected nested to be object, got %s", nested.Type())
				}

				// Check numeric value
				numeric := extraBodyObj.GetPropertyValue("numeric")
				if numeric.Type() != ValueTypeNumber {
					t.Fatalf("expected numeric to be number, got %s", numeric.Type())
				}

				// Check boolean value
				flag := extraBodyObj.GetPropertyValue("flag")
				if flag.Type() != ValueTypeBool {
					t.Fatalf("expected flag to be boolean, got %s", flag.Type())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			defer interp.Close()

			result, err := interp.Eval(program)
			tt.checkFunc(t, result, err)
		})
	}
}

func TestModelDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "Model with invalid provider type",
			input: `model bad {
				provider: 123,
			}`,
			expectedError: "provider' must be a string",
		},
		{
			name: "Model with invalid model type",
			input: `model bad {
				provider: "openai",
				model: true,
			}`,
			expectedError: "model' must be a string",
		},
		{
			name: "Model with invalid baseURL type",
			input: `model bad {
				provider: "openai",
				baseURL: 123,
			}`,
			expectedError: "baseURL' must be a string",
		},
		{
			name: "Model with invalid temperature type",
			input: `model bad {
				provider: "openai",
				temperature: "hot",
			}`,
			expectedError: "temperature' must be a number",
		},
		{
			name: "Model with invalid maxTokens type",
			input: `model bad {
				provider: "openai",
				maxTokens: "many",
			}`,
			expectedError: "maxTokens' must be a number",
		},
		{
			name: "Model with invalid headers type (not an object)",
			input: `model bad {
				provider: "openai",
				headers: "not-an-object",
			}`,
			expectedError: "headers' must be an object",
		},
		{
			name: "Model with invalid header value type (not a string)",
			input: `model bad {
				provider: "openai",
				headers: {
					"X-Custom-Header": 123,
				},
			}`,
			expectedError: "headers.X-Custom-Header' must be a string",
		},
		{
			name: "Model with invalid extraBody type (not an object)",
			input: `model bad {
				provider: "openai",
				extraBody: "not-an-object",
			}`,
			expectedError: "extraBody' must be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			defer interp.Close()

			_, err := interp.Eval(program)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedError)
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestModelValueMethods(t *testing.T) {
	model := &ModelValue{
		Name: "testmodel",
		Config: map[string]Value{
			"provider": &StringValue{Value: "openai"},
		},
	}

	// Test Type()
	if model.Type() != ValueTypeModel {
		t.Errorf("expected Type() to return ValueTypeModel, got %v", model.Type())
	}

	// Test String()
	expected := "<model testmodel>"
	if model.String() != expected {
		t.Errorf("expected String() to return %q, got %q", expected, model.String())
	}

	// Test IsTruthy()
	if !model.IsTruthy() {
		t.Error("expected IsTruthy() to return true")
	}

	// Test Equals()
	sameModel := &ModelValue{Name: "testmodel"}
	if !model.Equals(sameModel) {
		t.Error("expected Equals() to return true for same model name")
	}

	differentModel := &ModelValue{Name: "othermodel"}
	if model.Equals(differentModel) {
		t.Error("expected Equals() to return false for different model name")
	}

	notAModel := &StringValue{Value: "testmodel"}
	if model.Equals(notAModel) {
		t.Error("expected Equals() to return false for non-model value")
	}
}
