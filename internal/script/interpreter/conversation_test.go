package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// TestPipeStringToAgent tests piping a string to an agent
func TestPipeStringToAgent(t *testing.T) {
	input := `
model testModel {
	provider: "smart-mock",
	model: "test"
}

agent TestAgent {
	model: testModel,
	systemPrompt: "You are a helpful assistant. Keep responses brief."
}

conv = "What is 2+2?" | TestAgent
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Create interpreter with mock provider
	interp := New(nil)
	mockProvider := NewSmartMockProvider()
	interp.providerRegistry.Register(mockProvider)

	result, err := interp.Eval(program)

	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	// Check that result is a conversation
	conv, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have user and assistant messages (no system prompt in conversation)
	if len(conv.Messages) != 2 {
		t.Fatalf("Expected 2 messages (user and assistant), got %d", len(conv.Messages))
	}

	// Check message roles
	if conv.Messages[0].Role != "user" {
		t.Errorf("Expected first message role to be 'user', got '%s'", conv.Messages[0].Role)
	}
	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message role to be 'assistant', got '%s'", conv.Messages[1].Role)
	}

	// Check user message content
	if conv.Messages[0].Content != "What is 2+2?" {
		t.Errorf("Expected user message 'What is 2+2?', got '%s'", conv.Messages[0].Content)
	}

	// Check that assistant response contains "4"
	assistantMsg := conv.Messages[1].Content
	if !strings.Contains(assistantMsg, "4") {
		t.Errorf("Expected assistant response to contain '4', got: %s", assistantMsg)
	}

	// Verify system prompt was sent to the provider but not stored in conversation
	lastReq := mockProvider.GetLastRequest()
	if lastReq == nil {
		t.Fatal("Expected provider to be called")
	}
	if len(lastReq.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages sent to provider (system + user), got %d", len(lastReq.Messages))
	}
	if lastReq.Messages[0].Role != "system" {
		t.Errorf("Expected first message sent to provider to be 'system', got '%s'", lastReq.Messages[0].Role)
	}
}

// TestPipeConversationString tests piping a conversation to a string
func TestPipeConversationString(t *testing.T) {
	input := `
model testModel {
	provider: "smart-mock",
	model: "test"
}

agent TestAgent {
	model: testModel,
	systemPrompt: "You are helpful."
}

conv1 = "Hello" | TestAgent
conv2 = conv1 | "How are you?"
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	interp := New(nil)
	mockProvider := NewSmartMockProvider()
	interp.providerRegistry.Register(mockProvider)

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	convVal, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have 3 messages: user, assistant, user
	if len(convVal.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(convVal.Messages))
	}

	// Check message sequence
	if convVal.Messages[0].Role != "user" || convVal.Messages[0].Content != "Hello" {
		t.Errorf("Expected first message to be user:'Hello', got %s:'%s'", convVal.Messages[0].Role, convVal.Messages[0].Content)
	}
	if convVal.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message to be assistant, got '%s'", convVal.Messages[1].Role)
	}
	if convVal.Messages[2].Role != "user" || convVal.Messages[2].Content != "How are you?" {
		t.Errorf("Expected third message to be user:'How are you?', got %s:'%s'", convVal.Messages[2].Role, convVal.Messages[2].Content)
	}
}

// TestPipeConversationAgent tests piping a conversation to an agent
func TestPipeConversationAgent(t *testing.T) {
	input := `
model testModel {
	provider: "smart-mock",
	model: "test"
}

agent MathAgent {
	model: testModel,
	systemPrompt: "You are a math tutor. Be concise."
}

conv = "What is 5+3?" | MathAgent | "Now multiply that by 2" | MathAgent
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	interp := New(nil)
	mockProvider := NewSmartMockProvider()
	interp.providerRegistry.Register(mockProvider)

	result, err := interp.Eval(program)

	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	// Check that result is a conversation
	conv, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have multiple messages (user, assistant, user, assistant)
	if len(conv.Messages) < 4 {
		t.Fatalf("Expected at least 4 messages, got %d", len(conv.Messages))
	}

	// Verify message sequence
	if conv.Messages[0].Role != "user" {
		t.Errorf("Expected message 0 to be user, got '%s'", conv.Messages[0].Role)
	}
	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Expected message 1 to be assistant, got '%s'", conv.Messages[1].Role)
	}
	if conv.Messages[2].Role != "user" {
		t.Errorf("Expected message 2 to be user, got '%s'", conv.Messages[2].Role)
	}
	if conv.Messages[3].Role != "assistant" {
		t.Errorf("Expected message 3 to be assistant, got '%s'", conv.Messages[3].Role)
	}

	// Response should contain "16" (5+3=8, 8*2=16)
	lastMsg := conv.Messages[len(conv.Messages)-1]
	if !strings.Contains(lastMsg.Content, "16") {
		t.Logf("Assistant response: %s", lastMsg.Content)
		t.Errorf("Expected assistant response to contain '16'")
	}
}

// TestPipeWithTools tests agent with tool calling
func TestPipeWithTools(t *testing.T) {
	input := `
model testModel {
	provider: "smart-mock",
	model: "test"
}

tool get_weather(city: string): string {
	return "The weather in " + city + " is sunny and 72°F"
}

agent WeatherAgent {
	model: testModel,
	systemPrompt: "You are a weather assistant. Use the get_weather tool to answer questions.",
	tools: [get_weather]
}

conv = "What's the weather in San Francisco?" | WeatherAgent
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	interp := New(nil)
	mockProvider := NewSmartMockProvider()
	interp.providerRegistry.Register(mockProvider)

	result, err := interp.Eval(program)

	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	// Check that result is a conversation
	conv, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have messages including tool call and response
	// user -> assistant (with tool call) -> tool result -> assistant (final)
	if len(conv.Messages) < 3 {
		t.Fatalf("Expected at least 3 messages, got %d", len(conv.Messages))
	}

	// Check for tool message
	hasToolMessage := false
	for _, msg := range conv.Messages {
		if msg.Role == "tool" {
			hasToolMessage = true
			// Should contain weather info
			if !strings.Contains(msg.Content, "sunny") && !strings.Contains(msg.Content, "72") {
				t.Errorf("Expected tool response to contain weather info, got: %s", msg.Content)
			}
		}
	}

	if !hasToolMessage {
		t.Error("Expected conversation to include tool message")
	}

	// Find the assistant's final response
	var finalResponse string
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		if conv.Messages[i].Role == "assistant" {
			finalResponse = conv.Messages[i].Content
			break
		}
	}

	// Response should mention tool results
	if !strings.Contains(strings.ToLower(finalResponse), "tool") &&
		!strings.Contains(strings.ToLower(finalResponse), "result") {
		t.Logf("Final response: %s", finalResponse)
		t.Errorf("Expected response to reference tool results")
	}
}

// TestInvalidPipeCombinations tests invalid pipe operations
func TestInvalidPipeCombinations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "Number to Agent",
			input: `
model testModel {
	provider: "smart-mock",
	model: "test"
}
agent A { model: testModel }
result = 42 | A
`,
		},
		{
			name: "Agent to String",
			input: `
model testModel {
	provider: "smart-mock",
	model: "test"
}
agent A { model: testModel }
result = A | "hello"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			interp := New(nil)
			mockProvider := NewSmartMockProvider()
			interp.providerRegistry.Register(mockProvider)

			_, err := interp.Eval(program)

			if err == nil {
				t.Errorf("Expected error for invalid pipe combination, got nil")
			}
		})
	}
}

// TestConversationValueType tests the ConversationValue type
func TestConversationValueType(t *testing.T) {
	conv := &ConversationValue{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
		},
	}

	if conv.Type() != ValueTypeConversation {
		t.Errorf("Expected type Conversation, got %s", conv.Type())
	}

	if !conv.IsTruthy() {
		t.Errorf("Expected conversation with messages to be truthy")
	}

	expectedStr := "<conversation with 2 messages>"
	if conv.String() != expectedStr {
		t.Errorf("Expected string '%s', got '%s'", expectedStr, conv.String())
	}

	// Test empty conversation
	emptyConv := &ConversationValue{Messages: []ChatMessage{}}
	if emptyConv.IsTruthy() {
		t.Errorf("Expected empty conversation to be falsy")
	}
}

// TestAgentHandoff tests passing conversation between different agents
func TestAgentHandoff(t *testing.T) {
	input := `
model testModel {
	provider: "smart-mock",
	model: "test"
}

agent Analyzer {
	model: testModel,
	systemPrompt: "You analyze data and provide insights. Be brief."
}

agent Writer {
	model: testModel,
	systemPrompt: "You write summaries based on analysis. Be concise."
}

conv = "Analyze: sales up 20%" | Analyzer | "Write a one-sentence summary" | Writer
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	interp := New(nil)
	mockProvider := NewSmartMockProvider()
	interp.providerRegistry.Register(mockProvider)

	result, err := interp.Eval(program)

	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	// Check that result is a conversation
	conv, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have multiple agents' responses
	if len(conv.Messages) < 4 {
		t.Fatalf("Expected at least 4 messages, got %d", len(conv.Messages))
	}

	// Verify we have assistant messages
	assistantCount := 0
	for _, msg := range conv.Messages {
		if msg.Role == "assistant" {
			assistantCount++
		}
	}

	if assistantCount < 2 {
		t.Errorf("Expected at least 2 assistant messages (one from each agent), got %d", assistantCount)
	}

	// Verify that different system prompts were used
	if mockProvider.GetCallCount() < 2 {
		t.Errorf("Expected at least 2 calls to provider (one per agent), got %d", mockProvider.GetCallCount())
	}
}

// TestAgenticLoopMultipleIterations tests that the agentic loop continues until no tool calls
func TestAgenticLoopMultipleIterations(t *testing.T) {
	// Create a custom mock provider that returns chained tool calls
	mock := &chainedToolCallMockProvider{
		callCount: 0,
	}

	interp := New(nil)
	interp.providerRegistry.Register(mock)

	input := `
model testModel {
	provider: "chained-mock",
	model: "test"
}

tool step1(input: string): string {
	return "step1_result"
}

tool step2(input: string): string {
	return "step2_result"
}

tool step3(input: string): string {
	return "step3_result"
}

agent TestAgent {
	model: testModel,
	tools: [step1, step2, step3],
	systemPrompt: "You are a test agent."
}

conv = "Process this through all steps" | TestAgent
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("Interpreter error: %v", err)
	}

	conv, ok := result.Value().(*ConversationValue)
	if !ok {
		t.Fatalf("Expected ConversationValue, got %T", result.Value())
	}

	// Should have 4 calls: initial + 3 tool call iterations
	if mock.callCount != 4 {
		t.Errorf("Expected 4 provider calls, got %d", mock.callCount)
	}

	// Verify the conversation has the right structure
	// Expected: user, assistant+tool1, tool1_result, assistant+tool2, tool2_result, assistant+tool3, tool3_result, assistant
	toolResults := 0
	assistantMsgs := 0
	for _, msg := range conv.Messages {
		if msg.Role == "tool" {
			toolResults++
		}
		if msg.Role == "assistant" {
			assistantMsgs++
		}
	}

	if toolResults != 3 {
		t.Errorf("Expected 3 tool results, got %d", toolResults)
	}

	if assistantMsgs != 4 {
		t.Errorf("Expected 4 assistant messages (3 with tool calls + 1 final), got %d", assistantMsgs)
	}
}

// TestAgenticLoopMaxIterations tests that the loop stops at max iterations
func TestAgenticLoopMaxIterations(t *testing.T) {
	mock := &infiniteToolCallMockProvider{}

	interp := New(nil)
	interp.providerRegistry.Register(mock)

	// Use a small maxIterations value for testing (5 instead of default 100)
	input := `
model testModel {
	provider: "infinite-mock",
	model: "test"
}

tool infiniteTool(): string {
	return "result"
}

agent TestAgent {
	model: testModel,
	tools: [infiniteTool],
	maxIterations: 5
}

conv = "Do something infinite" | TestAgent
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	_, err := interp.Eval(program)

	// Should return max iterations error
	if err == nil {
		t.Fatal("Expected max iterations error, got nil")
	}

	if !strings.Contains(err.Error(), "maximum iterations") {
		t.Errorf("Expected max iterations error, got: %v", err)
	}

	// Should have exactly 5 calls (the configured maxIterations)
	expectedIterations := 5
	if mock.callCount != expectedIterations {
		t.Errorf("Expected %d provider calls, got %d", expectedIterations, mock.callCount)
	}
}

// chainedToolCallMockProvider returns tool calls in sequence: step1 -> step2 -> step3 -> done
type chainedToolCallMockProvider struct {
	callCount int
}

func (c *chainedToolCallMockProvider) Name() string { return "chained-mock" }

func (c *chainedToolCallMockProvider) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
	c.callCount++

	// Check which tools have been called based on conversation history
	toolsCalled := make(map[string]bool)
	for _, msg := range request.Messages {
		if msg.Role == "tool" {
			toolsCalled[msg.Name] = true
		}
	}

	// Sequence: step1 -> step2 -> step3 -> final response
	if !toolsCalled["step1"] {
		return &ChatResponse{
			Content: "I'll start with step1.",
			ToolCalls: []ChatToolCall{
				{ID: "call_1", Name: "step1", Arguments: map[string]interface{}{"input": "start"}},
			},
		}, nil
	}

	if !toolsCalled["step2"] {
		return &ChatResponse{
			Content: "Now step2.",
			ToolCalls: []ChatToolCall{
				{ID: "call_2", Name: "step2", Arguments: map[string]interface{}{"input": "continue"}},
			},
		}, nil
	}

	if !toolsCalled["step3"] {
		return &ChatResponse{
			Content: "Finally step3.",
			ToolCalls: []ChatToolCall{
				{ID: "call_3", Name: "step3", Arguments: map[string]interface{}{"input": "finish"}},
			},
		}, nil
	}

	// All tools called, return final response
	return &ChatResponse{
		Content:   "All steps completed successfully!",
		ToolCalls: []ChatToolCall{},
	}, nil
}

func (c *chainedToolCallMockProvider) StreamingChatCompletion(request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error) {
	response, err := c.ChatCompletion(request)
	if err != nil {
		return nil, err
	}
	if callbacks != nil && callbacks.OnContent != nil && response.Content != "" {
		callbacks.OnContent(response.Content)
	}
	return response, nil
}

// infiniteToolCallMockProvider always returns a tool call (for testing max iterations)
type infiniteToolCallMockProvider struct {
	callCount int
}

func (i *infiniteToolCallMockProvider) Name() string { return "infinite-mock" }

func (i *infiniteToolCallMockProvider) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
	i.callCount++
	return &ChatResponse{
		Content: "I need to use a tool.",
		ToolCalls: []ChatToolCall{
			{ID: "call_" + string(rune(i.callCount)), Name: "infiniteTool", Arguments: map[string]interface{}{}},
		},
	}, nil
}

func (i *infiniteToolCallMockProvider) StreamingChatCompletion(request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error) {
	response, err := i.ChatCompletion(request)
	if err != nil {
		return nil, err
	}
	if callbacks != nil && callbacks.OnContent != nil && response.Content != "" {
		callbacks.OnContent(response.Content)
	}
	return response, nil
}
