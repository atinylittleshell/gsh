package interpreter

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// skipIfOllamaNotAvailable skips the test if ollama is not running
func skipIfOllamaNotAvailable(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		t.Skip("Ollama not available at localhost:11434, skipping E2E test")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skip("Ollama not responding correctly, skipping E2E test")
	}
}

// runScript is a helper to execute a gsh script and return the result
func runScript(script string) (*EvalResult, error) {
	// Parse the script
	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors: %v", p.Errors())
	}

	// Execute the script
	interp := New()
	defer interp.Close()

	return interp.Eval(program)
}

// TestE2E_ModelDeclarationWithOllamaEndpoint tests declaring a model using OpenAI provider
// but pointing to a local ollama endpoint
func TestE2E_ModelDeclarationWithOllamaEndpoint(t *testing.T) {
	skipIfOllamaNotAvailable(t)

	script := `
model testModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
		model: "devstral-small-2:latest",
    temperature: 0.5
}

agent Assistant {
    model: testModel,
    systemPrompt: "You are a helpful assistant"
}
`

	result, err := runScript(script)
	if err != nil {
		t.Fatalf("Script execution error: %v", err)
	}

	// Verify model was created with correct config
	modelVal, ok := result.Env.Get("testModel")
	if !ok {
		t.Fatal("Model 'testModel' not found in environment")
	}

	model, ok := modelVal.(*ModelValue)
	if !ok {
		t.Fatalf("Expected ModelValue, got %T", modelVal)
	}

	// Check model config fields
	if provider, ok := model.Config["provider"].(*StringValue); !ok || provider.Value != "openai" {
		t.Errorf("Expected provider 'openai'")
	}

	if apiKey, ok := model.Config["apiKey"].(*StringValue); !ok || apiKey.Value != "ollama" {
		t.Errorf("Expected apiKey 'ollama'")
	}

	if baseURL, ok := model.Config["baseURL"].(*StringValue); !ok || baseURL.Value != "http://localhost:11434/v1" {
		t.Errorf("Expected baseURL 'http://localhost:11434/v1'")
	}

	if modelName, ok := model.Config["model"].(*StringValue); !ok || modelName.Value != "devstral-small-2:latest" {
		t.Errorf("Expected model 'devstral-small-2:latest'")
	}

	if temp, ok := model.Config["temperature"].(*NumberValue); !ok || temp.Value != 0.5 {
		t.Errorf("Expected temperature 0.5")
	}

	// Verify agent was created and references the model
	agentVal, ok := result.Env.Get("Assistant")
	if !ok {
		t.Fatal("Agent 'Assistant' not found in environment")
	}

	agent, ok := agentVal.(*AgentValue)
	if !ok {
		t.Fatalf("Expected AgentValue, got %T", agentVal)
	}

	agentModel, ok := agent.Config["model"].(*ModelValue)
	if !ok {
		t.Fatal("Agent model not found or wrong type")
	}

	if agentModel.Name != "testModel" {
		t.Errorf("Expected agent model name 'testModel', got %q", agentModel.Name)
	}
}

// TestE2E_BasicPipePromptToAgent tests the basic pipe: "prompt" | Agent
func TestE2E_BasicPipePromptToAgent(t *testing.T) {
	skipIfOllamaNotAvailable(t)

	script := `
model testModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2:latest",
    temperature: 0.3
}

agent MathHelper {
    model: testModel,
    systemPrompt: "You are a helpful math assistant. Answer math questions concisely."
}

result = "What is 7 plus 5?" | MathHelper
`

	result, err := runScript(script)
	if err != nil {
		t.Fatalf("Script execution error: %v", err)
	}

	// Verify result is a conversation
	resultVal, ok := result.Env.Get("result")
	if !ok {
		t.Fatal("Variable 'result' not found")
	}

	conv, ok := resultVal.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", resultVal)
	}

	// Should have user message and assistant response
	if len(conv.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages, got %d", len(conv.Messages))
	}

	if conv.Messages[0].Role != "user" {
		t.Errorf("Expected first message to be 'user', got %q", conv.Messages[0].Role)
	}

	if conv.Messages[0].Content != "What is 7 plus 5?" {
		t.Errorf("Expected user message 'What is 7 plus 5?', got %q", conv.Messages[0].Content)
	}

	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message to be 'assistant', got %q", conv.Messages[1].Role)
	}

	// Verify the assistant response contains the answer "12"
	assistantResponse := conv.Messages[1].Content
	if !strings.Contains(assistantResponse, "12") {
		t.Errorf("Expected assistant response to contain '12', got: %s", assistantResponse)
	}

	t.Logf("Assistant response: %s", assistantResponse)
}

// TestE2E_MultiTurnConversation tests: conv | "message" | Agent
func TestE2E_MultiTurnConversation(t *testing.T) {
	skipIfOllamaNotAvailable(t)

	script := `
model testModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2:latest",
    temperature: 0.3
}

agent MathHelper {
    model: testModel,
    systemPrompt: "You are a helpful math assistant. Answer math questions concisely with just the number."
}

conv = "What is 2+2?" | MathHelper
     | "What is 5+3?" | MathHelper
     | "What is 10-7?" | MathHelper
`

	result, err := runScript(script)
	if err != nil {
		t.Fatalf("Script execution error: %v", err)
	}

	// Verify conversation has multiple turns
	convVal, ok := result.Env.Get("conv")
	if !ok {
		t.Fatal("Variable 'conv' not found")
	}

	conv, ok := convVal.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", convVal)
	}

	// Should have 6 messages: user1, assistant1, user2, assistant2, user3, assistant3
	if len(conv.Messages) != 6 {
		t.Fatalf("Expected 6 messages, got %d", len(conv.Messages))
	}

	// Verify message sequence
	expectedMessages := []struct {
		role    string
		content string
		answer  string
	}{
		{"user", "What is 2+2?", ""},
		{"assistant", "", "4"},
		{"user", "What is 5+3?", ""},
		{"assistant", "", "8"},
		{"user", "What is 10-7?", ""},
		{"assistant", "", "3"},
	}

	for i, expected := range expectedMessages {
		if conv.Messages[i].Role != expected.role {
			t.Errorf("Message %d: expected role %q, got %q", i, expected.role, conv.Messages[i].Role)
		}
		if expected.content != "" && conv.Messages[i].Content != expected.content {
			t.Errorf("Message %d: expected content %q, got %q", i, expected.content, conv.Messages[i].Content)
		}
		if expected.answer != "" && !strings.Contains(conv.Messages[i].Content, expected.answer) {
			t.Errorf("Message %d: expected response to contain %q, got: %s", i, expected.answer, conv.Messages[i].Content)
		}
	}

	t.Logf("Conversation messages:")
	for i, msg := range conv.Messages {
		t.Logf("  [%d] %s: %s", i, msg.Role, msg.Content)
	}
}

// TestE2E_AgentHandoff tests: conv | Agent1 | "message" | Agent2
func TestE2E_AgentHandoff(t *testing.T) {
	skipIfOllamaNotAvailable(t)

	script := `
model testModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2:latest",
    temperature: 0.3
}

agent Mathematician {
    model: testModel,
    systemPrompt: "You are a mathematician. Solve math problems and explain your reasoning briefly."
}

agent Simplifier {
    model: testModel,
    systemPrompt: "You simplify explanations. Rewrite explanations in very simple terms."
}

conv = "What is 15 multiplied by 4?" | Mathematician
     | "Simplify that explanation for a 5 year old" | Simplifier
`

	result, err := runScript(script)
	if err != nil {
		t.Fatalf("Script execution error: %v", err)
	}

	// Verify conversation has messages from both agents
	convVal, ok := result.Env.Get("conv")
	if !ok {
		t.Fatal("Variable 'conv' not found")
	}

	conv, ok := convVal.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", convVal)
	}

	// Should have 4 messages: user1, mathematician1, user2, simplifier1
	if len(conv.Messages) != 4 {
		t.Fatalf("Expected 4 messages, got %d", len(conv.Messages))
	}

	// Verify message sequence
	if conv.Messages[0].Role != "user" {
		t.Errorf("Message 0: expected role 'user', got %q", conv.Messages[0].Role)
	}
	if !strings.Contains(conv.Messages[0].Content, "15 multiplied by 4") {
		t.Errorf("Message 0: expected content about multiplication, got %q", conv.Messages[0].Content)
	}

	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Message 1: expected role 'assistant', got %q", conv.Messages[1].Role)
	}
	if !strings.Contains(conv.Messages[1].Content, "60") {
		t.Errorf("Message 1: expected answer to contain '60', got: %s", conv.Messages[1].Content)
	}

	if conv.Messages[2].Role != "user" {
		t.Errorf("Message 2: expected role 'user', got %q", conv.Messages[2].Role)
	}
	if conv.Messages[2].Content != "Simplify that explanation for a 5 year old" {
		t.Errorf("Message 2: expected handoff message, got %q", conv.Messages[2].Content)
	}

	if conv.Messages[3].Role != "assistant" {
		t.Errorf("Message 3: expected role 'assistant', got %q", conv.Messages[3].Role)
	}

	t.Logf("Conversation flow:")
	for i, msg := range conv.Messages {
		t.Logf("  [%d] %s: %s", i, msg.Role, msg.Content)
	}
}

// TestE2E_AgentWithUserDefinedTools tests agents calling user-defined tools
func TestE2E_AgentWithUserDefinedTools(t *testing.T) {
	skipIfOllamaNotAvailable(t)

	script := `
tool multiply(a: number, b: number): number {
    return a * b
}

tool greet(name: string): string {
    return "Hello, " + name + "! Welcome to gsh."
}

model testModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2:latest",
    temperature: 0.3
}

agent MathHelper {
    model: testModel,
    systemPrompt: "You are a math helper. Use the multiply tool when users ask for multiplication.",
    tools: [multiply]
}

agent Greeter {
    model: testModel,
    systemPrompt: "You are a friendly greeter. Use the greet tool to greet people by name.",
    tools: [greet]
}

mathResult = "What is 6 times 7?" | MathHelper
greetResult = "Please greet Bob" | Greeter
`

	result, err := runScript(script)
	if err != nil {
		t.Fatalf("Script execution error: %v", err)
	}

	// Verify both agents were created with their tools
	mathAgent, ok := result.Env.Get("MathHelper")
	if !ok {
		t.Fatal("MathHelper not found")
	}

	mathAgentVal, ok := mathAgent.(*AgentValue)
	if !ok {
		t.Fatalf("Expected AgentValue, got %T", mathAgent)
	}

	// Check tools array
	toolsVal, ok := mathAgentVal.Config["tools"]
	if !ok {
		t.Fatal("MathHelper missing 'tools' config")
	}

	toolsArray, ok := toolsVal.(*ArrayValue)
	if !ok {
		t.Fatalf("Expected ArrayValue for tools, got %T", toolsVal)
	}

	if len(toolsArray.Elements) != 1 {
		t.Errorf("Expected 1 tool for MathHelper, got %d", len(toolsArray.Elements))
	}

	// Verify mathResult conversation
	mathResultVal, ok := result.Env.Get("mathResult")
	if !ok {
		t.Fatal("Variable 'mathResult' not found")
	}

	mathConv, ok := mathResultVal.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue for mathResult, got %T", mathResultVal)
	}

	if len(mathConv.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages in mathResult, got %d", len(mathConv.Messages))
	}

	// Check if the answer "42" appears in the conversation
	foundAnswer := false
	for _, msg := range mathConv.Messages {
		if strings.Contains(msg.Content, "42") {
			foundAnswer = true
			break
		}
	}
	if !foundAnswer {
		t.Errorf("Expected to find '42' in math conversation")
	}

	// Verify greetResult conversation
	greetResultVal, ok := result.Env.Get("greetResult")
	if !ok {
		t.Fatal("Variable 'greetResult' not found")
	}

	greetConv, ok := greetResultVal.(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue for greetResult, got %T", greetResultVal)
	}

	if len(greetConv.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages in greetResult, got %d", len(greetConv.Messages))
	}

	// Check if "Bob" appears in the greeting conversation
	foundBob := false
	for _, msg := range greetConv.Messages {
		if strings.Contains(msg.Content, "Bob") {
			foundBob = true
			break
		}
	}
	if !foundBob {
		t.Errorf("Expected to find 'Bob' in greeting conversation")
	}

	t.Logf("Math conversation:")
	for i, msg := range mathConv.Messages {
		t.Logf("  [%d] %s: %s", i, msg.Role, msg.Content)
	}

	t.Logf("Greeting conversation:")
	for i, msg := range greetConv.Messages {
		t.Logf("  [%d] %s: %s", i, msg.Role, msg.Content)
	}
}
