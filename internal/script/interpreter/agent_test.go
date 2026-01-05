package interpreter

import (
	"context"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestAgentDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result *EvalResult, err error)
	}{
		{
			name: "Agent declaration with model reference",
			input: `
				model claude {
					provider: "openai",
					apiKey: "test-key",
					model: "claude-3-5-sonnet-20241022",
				}
				agent DataAnalyst {
					model: claude,
					systemPrompt: "You are a data analyst",
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check that the agent is registered in the environment
				agentVal, ok := result.Env.Get("DataAnalyst")
				if !ok {
					t.Fatalf("agent 'DataAnalyst' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				if agent.Name != "DataAnalyst" {
					t.Errorf("expected agent name 'DataAnalyst', got %q", agent.Name)
				}

				// Check model is a ModelValue reference
				model, ok := agent.Config["model"]
				if !ok {
					t.Fatalf("agent config missing 'model'")
				}
				modelVal, ok := model.(*ModelValue)
				if !ok {
					t.Fatalf("expected model to be *ModelValue, got %T", model)
				}
				if modelVal.Name != "claude" {
					t.Errorf("expected model name 'claude', got %q", modelVal.Name)
				}

				// Check systemPrompt
				prompt, ok := agent.Config["systemPrompt"]
				if !ok {
					t.Fatalf("agent config missing 'systemPrompt'")
				}
				promptStr, ok := prompt.(*StringValue)
				if !ok {
					t.Fatalf("expected systemPrompt to be *StringValue, got %T", prompt)
				}
				if promptStr.Value != "You are a data analyst" {
					t.Errorf("expected systemPrompt 'You are a data analyst', got %q", promptStr.Value)
				}
			},
		},
		{
			name: "Agent declaration with different model reference",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent Analyst {
					model: gpt4,
					systemPrompt: "You analyze data",
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check that the agent is registered
				agentVal, ok := result.Env.Get("Analyst")
				if !ok {
					t.Fatalf("agent 'Analyst' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				// Check model is a reference to the ModelValue
				model, ok := agent.Config["model"]
				if !ok {
					t.Fatalf("agent config missing 'model'")
				}
				modelVal, ok := model.(*ModelValue)
				if !ok {
					t.Fatalf("expected model to be *ModelValue, got %T", model)
				}
				if modelVal.Name != "gpt4" {
					t.Errorf("expected model name 'gpt4', got %q", modelVal.Name)
				}
			},
		},
		{
			name: "Agent declaration with tools array",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent Helper {
					model: gpt4,
					systemPrompt: "You help users",
					tools: ["tool1", "tool2", "tool3"],
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				agentVal, ok := result.Env.Get("Helper")
				if !ok {
					t.Fatalf("agent 'Helper' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				// Check tools array
				tools, ok := agent.Config["tools"]
				if !ok {
					t.Fatalf("agent config missing 'tools'")
				}
				toolsArr, ok := tools.(*ArrayValue)
				if !ok {
					t.Fatalf("expected tools to be *ArrayValue, got %T", tools)
				}
				if len(toolsArr.Elements) != 3 {
					t.Errorf("expected tools array to have 3 elements, got %d", len(toolsArr.Elements))
				}
			},
		},
		{
			name: "Agent declaration with multiline string",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent WithMultiline {
					model: gpt4,
					systemPrompt: """
						You are a data analyst.
						Analyze the provided data and generate insights.
					""",
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				agentVal, ok := result.Env.Get("WithMultiline")
				if !ok {
					t.Fatalf("agent 'WithMultiline' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				// Check systemPrompt contains the expected text
				prompt, ok := agent.Config["systemPrompt"]
				if !ok {
					t.Fatalf("agent config missing 'systemPrompt'")
				}
				promptStr, ok := prompt.(*StringValue)
				if !ok {
					t.Fatalf("expected systemPrompt to be *StringValue, got %T", prompt)
				}
				if !strings.Contains(promptStr.Value, "You are a data analyst") {
					t.Errorf("systemPrompt doesn't contain expected text, got %q", promptStr.Value)
				}
			},
		},
		{
			name: "Agent declaration with minimal config",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent Minimal {
					model: gpt4,
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				agentVal, ok := result.Env.Get("Minimal")
				if !ok {
					t.Fatalf("agent 'Minimal' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				if len(agent.Config) != 1 {
					t.Errorf("expected agent config to have 1 field, got %d", len(agent.Config))
				}
			},
		},
		{
			name: "Multiple agent declarations",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				model claude {
					provider: "openai",
					apiKey: "test-key",
					model: "claude-3-5-sonnet-20241022",
				}
				agent First {
					model: gpt4,
				}
				agent Second {
					model: claude,
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check both agents are registered
				_, ok := result.Env.Get("First")
				if !ok {
					t.Errorf("agent 'First' not found in environment")
				}

				_, ok = result.Env.Get("Second")
				if !ok {
					t.Errorf("agent 'Second' not found in environment")
				}
			},
		},
		{
			name: "Agent declaration with custom fields",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent CustomFields {
					model: gpt4,
					systemPrompt: "Test",
					customField: "custom value",
					anotherField: 42,
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				agentVal, ok := result.Env.Get("CustomFields")
				if !ok {
					t.Fatalf("agent 'CustomFields' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				// Check custom fields are present
				customField, ok := agent.Config["customField"]
				if !ok {
					t.Fatalf("agent config missing 'customField'")
				}
				if customField.Type() != ValueTypeString {
					t.Errorf("expected customField to be string, got %s", customField.Type())
				}

				anotherField, ok := agent.Config["anotherField"]
				if !ok {
					t.Fatalf("agent config missing 'anotherField'")
				}
				if anotherField.Type() != ValueTypeNumber {
					t.Errorf("expected anotherField to be number, got %s", anotherField.Type())
				}
			},
		},
		{
			name: "Agent declaration with metadata object",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent WithMetadata {
					model: gpt4,
					systemPrompt: "Test",
					metadata: {
						category: "analysis",
						priority: 1,
						tags: ["data", "reports"],
						nested: {
							key: "value",
						},
					},
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				agentVal, ok := result.Env.Get("WithMetadata")
				if !ok {
					t.Fatalf("agent 'WithMetadata' not found in environment")
				}

				agent, ok := agentVal.(*AgentValue)
				if !ok {
					t.Fatalf("expected *AgentValue, got %T", agentVal)
				}

				// Check metadata is present and is an object
				metadata, ok := agent.Config["metadata"]
				if !ok {
					t.Fatalf("agent config missing 'metadata'")
				}
				metadataObj, ok := metadata.(*ObjectValue)
				if !ok {
					t.Fatalf("expected metadata to be *ObjectValue, got %T", metadata)
				}

				// Check nested properties
				category := metadataObj.GetPropertyValue("category")
				if category.Type() != ValueTypeString {
					t.Errorf("expected category to be string, got %s", category.Type())
				}
				if category.String() != "analysis" {
					t.Errorf("expected category 'analysis', got %q", category.String())
				}

				priority := metadataObj.GetPropertyValue("priority")
				if priority.Type() != ValueTypeNumber {
					t.Errorf("expected priority to be number, got %s", priority.Type())
				}

				tags := metadataObj.GetPropertyValue("tags")
				if tags.Type() != ValueTypeArray {
					t.Errorf("expected tags to be array, got %s", tags.Type())
				}

				nested := metadataObj.GetPropertyValue("nested")
				if nested.Type() != ValueTypeObject {
					t.Errorf("expected nested to be object, got %s", nested.Type())
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

func TestAgentDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "Agent declaration without model",
			input: `agent NoModel {
				systemPrompt: "Test",
			}`,
			expectedError: "must have a 'model' field",
		},
		{
			name: "Agent declaration with invalid model type (string)",
			input: `agent BadModel {
				model: "gpt-4",
			}`,
			expectedError: "must be a model reference",
		},
		{
			name: "Agent declaration with invalid model type (number)",
			input: `agent BadModel {
				model: 123,
			}`,
			expectedError: "must be a model reference",
		},
		{
			name: "Agent declaration with invalid systemPrompt type",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent BadPrompt {
					model: gpt4,
					systemPrompt: 123,
				}`,
			expectedError: "systemPrompt' must be a string",
		},
		{
			name: "Agent declaration with invalid tools type",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent BadTools {
					model: gpt4,
					tools: "not an array",
				}`,
			expectedError: "tools' must be an array",
		},
		{
			name: "Agent declaration with invalid metadata type (string)",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent BadMetadata {
					model: gpt4,
					metadata: "not an object",
				}`,
			expectedError: "metadata' must be an object",
		},
		{
			name: "Agent declaration with invalid metadata type (number)",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent BadMetadata {
					model: gpt4,
					metadata: 123,
				}`,
			expectedError: "metadata' must be an object",
		},
		{
			name: "Agent declaration with invalid metadata type (array)",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent BadMetadata {
					model: gpt4,
					metadata: ["not", "an", "object"],
				}`,
			expectedError: "metadata' must be an object",
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
				t.Fatalf("expected error containing %q, but got no error", tt.expectedError)
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestAgentDeclarationWithSDKModelRef(t *testing.T) {
	// Test that agent declarations with gsh.models.* create SDKModelRef for dynamic resolution
	t.Run("gsh.models.workhorse creates SDKModelRef", func(t *testing.T) {
		input := `
			agent TestAgent {
				model: gsh.models.workhorse,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Set up models so gsh.models.* is accessible
		workhorseModel := &ModelValue{Name: "workhorseModel"}
		interp.sdkConfig.GetModels().Workhorse = workhorseModel

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, ok := result.Env.Get("TestAgent")
		if !ok {
			t.Fatal("agent 'TestAgent' not found")
		}

		agent := agentVal.(*AgentValue)
		model := agent.Config["model"]

		// Should be SDKModelRef, not ModelValue
		sdkRef, ok := model.(*SDKModelRef)
		if !ok {
			t.Fatalf("expected *SDKModelRef, got %T", model)
		}
		if sdkRef.Tier != "workhorse" {
			t.Errorf("expected tier 'workhorse', got %q", sdkRef.Tier)
		}
	})

	t.Run("gsh.models.lite creates SDKModelRef", func(t *testing.T) {
		input := `
			agent TestAgent {
				model: gsh.models.lite,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Set up models so gsh.models.* is accessible
		liteModel := &ModelValue{Name: "liteModel"}
		interp.sdkConfig.GetModels().Lite = liteModel

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)
		sdkRef, ok := agent.Config["model"].(*SDKModelRef)
		if !ok {
			t.Fatalf("expected *SDKModelRef, got %T", agent.Config["model"])
		}
		if sdkRef.Tier != "lite" {
			t.Errorf("expected tier 'lite', got %q", sdkRef.Tier)
		}
	})

	t.Run("gsh.models.premium creates SDKModelRef", func(t *testing.T) {
		input := `
			agent TestAgent {
				model: gsh.models.premium,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Set up models so gsh.models.* is accessible
		premiumModel := &ModelValue{Name: "premiumModel"}
		interp.sdkConfig.GetModels().Premium = premiumModel

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)
		sdkRef, ok := agent.Config["model"].(*SDKModelRef)
		if !ok {
			t.Fatalf("expected *SDKModelRef, got %T", agent.Config["model"])
		}
		if sdkRef.Tier != "premium" {
			t.Errorf("expected tier 'premium', got %q", sdkRef.Tier)
		}
	})

	t.Run("direct model reference creates ModelValue (static)", func(t *testing.T) {
		input := `
			model myModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4o",
			}
			agent TestAgent {
				model: myModel,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)

		// Should be ModelValue, not SDKModelRef
		modelVal, ok := agent.Config["model"].(*ModelValue)
		if !ok {
			t.Fatalf("expected *ModelValue for direct model reference, got %T", agent.Config["model"])
		}
		if modelVal.Name != "myModel" {
			t.Errorf("expected model name 'myModel', got %q", modelVal.Name)
		}
	})

	t.Run("SDKModelRef resolves dynamically via ModelResolver interface", func(t *testing.T) {
		input := `
			agent TestAgent {
				model: gsh.models.workhorse,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Set up initial model in SDK context
		initialModel := &ModelValue{Name: "initialModel"}
		models := interp.sdkConfig.GetModels()
		models.Workhorse = initialModel

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)

		// Get the model resolver
		resolver, ok := agent.Config["model"].(ModelResolver)
		if !ok {
			t.Fatalf("expected model to implement ModelResolver, got %T", agent.Config["model"])
		}

		// First resolution should return the initial model
		resolved1 := resolver.GetModel()
		if resolved1 != initialModel {
			t.Errorf("first resolution: expected %v, got %v", initialModel, resolved1)
		}

		// Change the model in the tier
		newModel := &ModelValue{Name: "newModel"}
		models.Workhorse = newModel

		// Second resolution should return the new model (dynamic!)
		resolved2 := resolver.GetModel()
		if resolved2 != newModel {
			t.Errorf("second resolution: expected %v, got %v", newModel, resolved2)
		}

		// Verify we got different models
		if resolved1 == resolved2 {
			t.Error("expected different models after tier change - dynamic resolution failed")
		}
	})

	t.Run("direct ModelValue does not change when gsh.models changes", func(t *testing.T) {
		input := `
			model myModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4o",
			}
			agent TestAgent {
				model: myModel,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Models are already initialized in SDKConfig
		models := interp.sdkConfig.GetModels()

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)

		// Get the model resolver (ModelValue also implements ModelResolver)
		resolver, ok := agent.Config["model"].(ModelResolver)
		if !ok {
			t.Fatalf("expected model to implement ModelResolver, got %T", agent.Config["model"])
		}

		// First resolution
		resolved1 := resolver.GetModel()

		// Change gsh.models.workhorse (should NOT affect this agent)
		models.Workhorse = &ModelValue{Name: "differentModel"}

		// Second resolution should return the SAME model (static)
		resolved2 := resolver.GetModel()

		if resolved1 != resolved2 {
			t.Error("expected same model for direct ModelValue assignment - should be static")
		}
		if resolved1.Name != "myModel" {
			t.Errorf("expected model name 'myModel', got %q", resolved1.Name)
		}
	})
}

// mockModelProvider records calls to verify which model was used
type mockModelProvider struct {
	name            string
	calledWithModel string
}

func (m *mockModelProvider) Name() string { return m.name }

func (m *mockModelProvider) ChatCompletion(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	// Record which model was used
	m.calledWithModel = request.Model.Name
	return &ChatResponse{
		Content:      "mock response",
		FinishReason: "stop",
	}, nil
}

func (m *mockModelProvider) StreamingChatCompletion(ctx context.Context, request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error) {
	// Record which model was used
	m.calledWithModel = request.Model.Name
	return &ChatResponse{
		Content:      "mock response",
		FinishReason: "stop",
	}, nil
}

func TestAgentModelResolution_RuntimeChange(t *testing.T) {
	// Integration test: verify that changing gsh.models.workhorse at runtime
	// affects agents declared with model: gsh.models.workhorse
	t.Run("agent sees model change made via gsh script", func(t *testing.T) {
		// This script:
		// 1. Sets up initial model in gsh.models.workhorse
		// 2. Declares an agent using gsh.models.workhorse
		// 3. Changes gsh.models.workhorse to a different model
		// 4. The agent should resolve to the NEW model
		input := `
			model initialModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4o-mini",
			}
			model newModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4-turbo",
			}
			
			# Set initial model
			gsh.models.workhorse = initialModel
			
			# Declare agent - captures SDKModelRef, not the current ModelValue
			agent MyAgent {
				model: gsh.models.workhorse,
				systemPrompt: "You are helpful",
			}
			
			# Change the model AFTER agent declaration
			gsh.models.workhorse = newModel
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Models are already initialized in SDKConfig (will be filled by script)
		models := interp.sdkConfig.GetModels()

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Get the agent
		agentVal, ok := result.Env.Get("MyAgent")
		if !ok {
			t.Fatal("agent 'MyAgent' not found")
		}
		agent := agentVal.(*AgentValue)

		// Get the model resolver from the agent
		resolver, ok := agent.Config["model"].(ModelResolver)
		if !ok {
			t.Fatalf("expected model to implement ModelResolver, got %T", agent.Config["model"])
		}

		// Resolve the model - should get the NEW model (newModel), not initialModel
		resolvedModel := resolver.GetModel()
		if resolvedModel == nil {
			t.Fatal("resolved model is nil")
		}

		// Verify it's the new model, not the initial model
		if resolvedModel.Name != "newModel" {
			t.Errorf("expected agent to resolve to 'newModel' after runtime change, got %q", resolvedModel.Name)
		}

		// Verify gsh.models.workhorse is indeed newModel
		if models.Workhorse == nil {
			t.Fatal("gsh.models.workhorse is nil")
		}
		if models.Workhorse.Name != "newModel" {
			t.Errorf("expected gsh.models.workhorse to be 'newModel', got %q", models.Workhorse.Name)
		}

		// Double-check: resolvedModel should be the same instance as gsh.models.workhorse
		if resolvedModel != models.Workhorse {
			t.Error("resolved model should be the same instance as gsh.models.workhorse")
		}
	})

	t.Run("direct model assignment is not affected by gsh.models changes", func(t *testing.T) {
		// This script:
		// 1. Declares a model and assigns it directly to an agent
		// 2. Changes gsh.models.workhorse
		// 3. The agent should still use the original direct model
		input := `
			model directModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4o",
			}
			model otherModel {
				provider: "openai",
				apiKey: "test-key",
				model: "gpt-4-turbo",
			}
			
			# Declare agent with DIRECT model assignment
			agent MyAgent {
				model: directModel,
				systemPrompt: "You are helpful",
			}
			
			# Change gsh.models.workhorse - should NOT affect the agent
			gsh.models.workhorse = otherModel
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Models are already initialized in SDKConfig
		models := interp.sdkConfig.GetModels()

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("MyAgent")
		agent := agentVal.(*AgentValue)

		resolver, ok := agent.Config["model"].(ModelResolver)
		if !ok {
			t.Fatalf("expected model to implement ModelResolver, got %T", agent.Config["model"])
		}

		resolvedModel := resolver.GetModel()

		// Should still be directModel, NOT otherModel
		if resolvedModel.Name != "directModel" {
			t.Errorf("expected agent to still use 'directModel', got %q", resolvedModel.Name)
		}

		// Verify gsh.models.workhorse is otherModel (changed by script)
		if models.Workhorse.Name != "otherModel" {
			t.Errorf("expected gsh.models.workhorse to be 'otherModel', got %q", models.Workhorse.Name)
		}
	})

	t.Run("multiple agents with same SDK model ref all see the change", func(t *testing.T) {
		input := `
			model model1 {
				provider: "openai",
				apiKey: "test-key",
				model: "model-1",
			}
			model model2 {
				provider: "openai",
				apiKey: "test-key",
				model: "model-2",
			}
			
			gsh.models.workhorse = model1
			
			# Both agents use gsh.models.workhorse
			agent Agent1 {
				model: gsh.models.workhorse,
				systemPrompt: "Agent 1",
			}
			agent Agent2 {
				model: gsh.models.workhorse,
				systemPrompt: "Agent 2",
			}
			
			# Change the model
			gsh.models.workhorse = model2
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		interp := New(nil)
		defer interp.Close()

		// Models are already initialized in SDKConfig

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Both agents should resolve to model2
		agent1Val, _ := result.Env.Get("Agent1")
		agent1 := agent1Val.(*AgentValue)
		resolver1 := agent1.Config["model"].(ModelResolver)

		agent2Val, _ := result.Env.Get("Agent2")
		agent2 := agent2Val.(*AgentValue)
		resolver2 := agent2.Config["model"].(ModelResolver)

		resolved1 := resolver1.GetModel()
		resolved2 := resolver2.GetModel()

		if resolved1.Name != "model2" {
			t.Errorf("Agent1 expected to resolve to 'model2', got %q", resolved1.Name)
		}
		if resolved2.Name != "model2" {
			t.Errorf("Agent2 expected to resolve to 'model2', got %q", resolved2.Name)
		}

		// Both should resolve to the exact same instance
		if resolved1 != resolved2 {
			t.Error("both agents should resolve to the same model instance")
		}
	})

	t.Run("agent LLM call uses dynamically resolved model", func(t *testing.T) {
		// This is the true E2E test: verify that when the agent actually makes
		// an LLM call, it uses the model that was set AFTER agent declaration

		// Create mock provider that records which model was used
		mockProvider := &mockModelProvider{name: "mock"}

		// Create two models with the same mock provider
		model1 := &ModelValue{
			Name:     "model1",
			Provider: mockProvider,
			Config:   map[string]Value{},
		}
		model2 := &ModelValue{
			Name:     "model2",
			Provider: mockProvider,
			Config:   map[string]Value{},
		}

		interp := New(nil)
		defer interp.Close()

		// Set up models (available in both REPL and script mode now)
		models := interp.sdkConfig.GetModels()
		models.Workhorse = model1 // Start with model1

		// Declare agent that uses gsh.models.workhorse
		input := `
			agent TestAgent {
				model: gsh.models.workhorse,
				systemPrompt: "You are helpful",
			}
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		agentVal, _ := result.Env.Get("TestAgent")
		agent := agentVal.(*AgentValue)

		// NOW change the model AFTER agent declaration
		models.Workhorse = model2

		// Execute the agent - this should use model2, not model1
		conv := &ConversationValue{
			Messages: []ChatMessage{
				{Role: "user", Content: "hello"},
			},
		}

		ctx := context.Background()
		_, err = interp.ExecuteAgent(ctx, conv, agent, false)
		if err != nil {
			t.Fatalf("agent execution failed: %v", err)
		}

		// Verify the mock provider was called with model2
		if mockProvider.calledWithModel != "model2" {
			t.Errorf("expected LLM call to use 'model2', but was called with %q", mockProvider.calledWithModel)
		}
	})
}

func TestAgentValueMethods(t *testing.T) {
	agent := &AgentValue{
		Name: "TestAgent",
		Config: map[string]Value{
			"model":        &StringValue{Value: "gpt-4"},
			"systemPrompt": &StringValue{Value: "Test prompt"},
		},
	}

	// Test Type()
	if agent.Type() != ValueTypeAgent {
		t.Errorf("expected agent.Type() to be ValueTypeAgent, got %v", agent.Type())
	}

	// Test String()
	if agent.String() != "<agent TestAgent>" {
		t.Errorf("expected agent.String() to be '<agent TestAgent>', got %q", agent.String())
	}

	// Test IsTruthy()
	if !agent.IsTruthy() {
		t.Error("expected agent.IsTruthy() to be true")
	}

	// Test Equals()
	otherAgent := &AgentValue{Name: "TestAgent"}
	if !agent.Equals(otherAgent) {
		t.Error("expected agents with same name to be equal")
	}

	differentAgent := &AgentValue{Name: "DifferentAgent"}
	if agent.Equals(differentAgent) {
		t.Error("expected agents with different names to not be equal")
	}

	// Test Equals() with non-agent value
	notAgent := &StringValue{Value: "not an agent"}
	if agent.Equals(notAgent) {
		t.Error("expected agent to not equal non-agent value")
	}
}

func TestAgentGetProperty(t *testing.T) {
	// Create metadata object
	metadataObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"category": {Value: &StringValue{Value: "analysis"}},
			"priority": {Value: &NumberValue{Value: 1}},
			"tags":     {Value: &ArrayValue{Elements: []Value{&StringValue{Value: "data"}, &StringValue{Value: "reports"}}}},
		},
	}

	agent := &AgentValue{
		Name: "TestAgent",
		Config: map[string]Value{
			"systemPrompt": &StringValue{Value: "Test prompt"},
			"metadata":     metadataObj,
		},
	}

	// Test GetProperty for "name"
	nameVal := agent.GetProperty("name")
	if nameVal.Type() != ValueTypeString {
		t.Errorf("expected name to be string, got %s", nameVal.Type())
	}
	if nameVal.String() != "TestAgent" {
		t.Errorf("expected name 'TestAgent', got %q", nameVal.String())
	}

	// Test GetProperty for config fields
	promptVal := agent.GetProperty("systemPrompt")
	if promptVal.Type() != ValueTypeString {
		t.Errorf("expected systemPrompt to be string, got %s", promptVal.Type())
	}
	if promptVal.String() != "Test prompt" {
		t.Errorf("expected systemPrompt 'Test prompt', got %q", promptVal.String())
	}

	// Test GetProperty for metadata
	metaVal := agent.GetProperty("metadata")
	if metaVal.Type() != ValueTypeObject {
		t.Errorf("expected metadata to be object, got %s", metaVal.Type())
	}

	// Test GetProperty for non-existent field returns null
	nonExistent := agent.GetProperty("nonExistent")
	if nonExistent.Type() != ValueTypeNull {
		t.Errorf("expected non-existent property to be null, got %s", nonExistent.Type())
	}
}

func TestAgentMetadataPropertyAccess(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result *EvalResult, err error)
	}{
		{
			name: "Access agent.name property",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent MyAgent {
					model: gpt4,
					systemPrompt: "Test",
				}
				result = MyAgent.name`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				resultVal, ok := result.Env.Get("result")
				if !ok {
					t.Fatal("result variable not found")
				}
				if resultVal.String() != "MyAgent" {
					t.Errorf("expected 'MyAgent', got %q", resultVal.String())
				}
			},
		},
		{
			name: "Access agent.metadata properties",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent MyAgent {
					model: gpt4,
					metadata: {
						category: "analysis",
						priority: 42,
					},
				}
				category = MyAgent.metadata.category
				priority = MyAgent.metadata.priority`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				categoryVal, ok := result.Env.Get("category")
				if !ok {
					t.Fatal("category variable not found")
				}
				if categoryVal.String() != "analysis" {
					t.Errorf("expected category 'analysis', got %q", categoryVal.String())
				}

				priorityVal, ok := result.Env.Get("priority")
				if !ok {
					t.Fatal("priority variable not found")
				}
				numVal, ok := priorityVal.(*NumberValue)
				if !ok {
					t.Fatalf("expected *NumberValue, got %T", priorityVal)
				}
				if numVal.Value != 42 {
					t.Errorf("expected priority 42, got %v", numVal.Value)
				}
			},
		},
		{
			name: "Access nested metadata properties",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent MyAgent {
					model: gpt4,
					metadata: {
						config: {
							timeout: 5000,
							retries: 3,
						},
					},
				}
				timeout = MyAgent.metadata.config.timeout`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				timeoutVal, ok := result.Env.Get("timeout")
				if !ok {
					t.Fatal("timeout variable not found")
				}
				numVal, ok := timeoutVal.(*NumberValue)
				if !ok {
					t.Fatalf("expected *NumberValue, got %T", timeoutVal)
				}
				if numVal.Value != 5000 {
					t.Errorf("expected timeout 5000, got %v", numVal.Value)
				}
			},
		},
		{
			name: "Access metadata array property",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent MyAgent {
					model: gpt4,
					metadata: {
						tags: ["fast", "reliable"],
					},
				}
				tags = MyAgent.metadata.tags
				firstTag = tags[0]`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tagsVal, ok := result.Env.Get("tags")
				if !ok {
					t.Fatal("tags variable not found")
				}
				arrVal, ok := tagsVal.(*ArrayValue)
				if !ok {
					t.Fatalf("expected *ArrayValue, got %T", tagsVal)
				}
				if len(arrVal.Elements) != 2 {
					t.Errorf("expected 2 elements, got %d", len(arrVal.Elements))
				}

				firstTagVal, ok := result.Env.Get("firstTag")
				if !ok {
					t.Fatal("firstTag variable not found")
				}
				if firstTagVal.String() != "fast" {
					t.Errorf("expected 'fast', got %q", firstTagVal.String())
				}
			},
		},
		{
			name: "Use metadata in conditional",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent FastAgent {
					model: gpt4,
					metadata: {
						timeout: 1000,
					},
				}
				isFast = false
				if (FastAgent.metadata.timeout < 5000) {
					isFast = true
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				isFastVal, ok := result.Env.Get("isFast")
				if !ok {
					t.Fatal("isFast variable not found")
				}
				boolVal, ok := isFastVal.(*BoolValue)
				if !ok {
					t.Fatalf("expected *BoolValue, got %T", isFastVal)
				}
				if !boolVal.Value {
					t.Error("expected isFast to be true")
				}
			},
		},
		{
			name: "Access non-existent metadata property returns null",
			input: `
				model gpt4 {
					provider: "openai",
					apiKey: "test-key",
					model: "gpt-4",
				}
				agent MyAgent {
					model: gpt4,
					metadata: {
						existing: "value",
					},
				}
				nonExistent = MyAgent.metadata.nonExistent`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				nonExistentVal, ok := result.Env.Get("nonExistent")
				if !ok {
					t.Fatal("nonExistent variable not found")
				}
				if nonExistentVal.Type() != ValueTypeNull {
					t.Errorf("expected null, got %s", nonExistentVal.Type())
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
