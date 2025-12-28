package interpreter

import (
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
