package repl

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/repl/agent"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
)

// TestSDKAgentAddSync tests that agents added via gsh.repl.agents.push() are synced to the agent manager
func TestSDKAgentAddSync(t *testing.T) {
	logger := zap.NewNop()

	// Create interpreter and agent manager
	interp := interpreter.New(&interpreter.Options{Logger: logger})
	defer interp.Close()

	agentManager := agent.NewManager()

	// Create a test model with a mock provider
	testModel := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: &mockProvider{},
	}

	// Create default agent
	defaultAgent := &interpreter.AgentValue{
		Name: "default",
		Config: map[string]interpreter.Value{
			"model": testModel,
		},
	}
	defaultState := &agent.State{
		Agent:        defaultAgent,
		Provider:     testModel.Provider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp,
	}
	agent.SetupAgentWithDefaultTools(defaultState)
	agentManager.AddAgent("default", defaultState)

	// Set up REPL context with OnAgentAdded callback
	replCtx := &interpreter.REPLContext{
		Models:       &interpreter.REPLModels{},
		LastCommand:  &interpreter.REPLLastCommand{},
		Agents:       []*interpreter.AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
		OnAgentAdded: func(newAgent *interpreter.AgentValue) {
			// Simulate what handleAgentAddedFromSDK does
			modelVal, ok := newAgent.Config["model"]
			if !ok {
				return
			}
			model, ok := modelVal.(*interpreter.ModelValue)
			if !ok || model.Provider == nil {
				return
			}

			state := &agent.State{
				Agent:        newAgent,
				Provider:     model.Provider,
				Conversation: []interpreter.ChatMessage{},
				Interpreter:  interp,
			}
			agent.SetupAgentWithDefaultTools(state)
			agentManager.AddAgent(newAgent.Name, state)
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store model in interpreter for the script
	interp.SetVariable("testModel", testModel)

	// Push a new agent via SDK
	_, err := interp.EvalString(`
		gsh.repl.agents.push({
			name: "custom",
			"model": testModel,
			systemPrompt: "Custom agent prompt"
		})
	`)
	if err != nil {
		t.Fatalf("unexpected error pushing agent: %v", err)
	}

	// Verify agent was added to the manager
	customState := agentManager.GetAgent("custom")
	if customState == nil {
		t.Fatal("expected custom agent to be added to agent manager")
	}

	// Verify the agent has the correct provider
	if customState.Provider == nil {
		t.Error("expected custom agent to have a provider")
	}

	// Verify the agent has default tools in its config
	toolsVal, ok := customState.Agent.Config["tools"]
	if !ok {
		t.Error("expected custom agent to have tools in config")
	} else if arr, ok := toolsVal.(*interpreter.ArrayValue); ok {
		if len(arr.Elements) == 0 {
			t.Error("expected custom agent to have default tools")
		}
	}
}

// TestSDKAgentModifySync tests that agent modifications via SDK are synced to the agent manager
func TestSDKAgentModifySync(t *testing.T) {
	logger := zap.NewNop()

	// Create interpreter and agent manager
	interp := interpreter.New(&interpreter.Options{Logger: logger})
	defer interp.Close()

	agentManager := agent.NewManager()

	// Create test models
	originalModel := &interpreter.ModelValue{
		Name:     "original-model",
		Config:   map[string]interpreter.Value{},
		Provider: &mockProvider{providerName: "original"},
	}
	newModel := &interpreter.ModelValue{
		Name:     "new-model",
		Config:   map[string]interpreter.Value{},
		Provider: &mockProvider{providerName: "new"},
	}

	// Create default agent
	defaultAgent := &interpreter.AgentValue{
		Name: "default",
		Config: map[string]interpreter.Value{
			"model":        originalModel,
			"systemPrompt": &interpreter.StringValue{Value: "Original prompt"},
		},
	}
	defaultState := &agent.State{
		Agent:        defaultAgent,
		Provider:     originalModel.Provider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp,
	}
	agent.SetupAgentWithDefaultTools(defaultState)
	agentManager.AddAgent("default", defaultState)

	// Set up REPL context with OnAgentModified callback
	replCtx := &interpreter.REPLContext{
		Models:       &interpreter.REPLModels{},
		LastCommand:  &interpreter.REPLLastCommand{},
		Agents:       []*interpreter.AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
		OnAgentModified: func(modifiedAgent *interpreter.AgentValue) {
			// Simulate what handleAgentModifiedFromSDK does
			state := agentManager.GetAgent(modifiedAgent.Name)
			if state == nil {
				return
			}

			// Sync model/provider if changed
			if modelVal, ok := modifiedAgent.Config["model"]; ok {
				if model, ok := modelVal.(*interpreter.ModelValue); ok && model.Provider != nil {
					state.Provider = model.Provider
				}
			}

			// Tools are now stored in agent.Config["tools"], no syncing needed
			// Agent config is the source of truth
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store new model in interpreter
	interp.SetVariable("newModel", newModel)

	// Verify original provider
	if defaultState.Provider.(*mockProvider).providerName != "original" {
		t.Errorf("expected original provider, got %s", defaultState.Provider.(*mockProvider).providerName)
	}

	// Modify the agent's model via SDK
	_, err := interp.EvalString(`gsh.repl.agents[0]["model"] = newModel`)
	if err != nil {
		t.Fatalf("unexpected error modifying agent: %v", err)
	}

	// Verify the provider was synced to the state
	if defaultState.Provider.(*mockProvider).providerName != "new" {
		t.Errorf("expected new provider after modification, got %s", defaultState.Provider.(*mockProvider).providerName)
	}
}

// TestSDKAgentToolsSync tests that tool modifications are synced to the agent manager
func TestSDKAgentToolsSync(t *testing.T) {
	logger := zap.NewNop()

	// Create interpreter and agent manager
	interp := interpreter.New(&interpreter.Options{Logger: logger})
	defer interp.Close()

	agentManager := agent.NewManager()

	// Create test model
	testModel := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: &mockProvider{},
	}

	// Create a custom tool
	customTool := &interpreter.NativeToolValue{
		Name:        "custom_tool",
		Description: "A custom tool for testing",
		Parameters:  map[string]interface{}{},
	}

	// Create default agent
	defaultAgent := &interpreter.AgentValue{
		Name: "default",
		Config: map[string]interpreter.Value{
			"model": testModel,
		},
	}
	defaultState := &agent.State{
		Agent:        defaultAgent,
		Provider:     testModel.Provider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  interp,
	}
	agent.SetupAgentWithDefaultTools(defaultState)
	agentManager.AddAgent("default", defaultState)

	// Get original tool count from agent config
	originalToolCount := 0
	if toolsVal, ok := defaultState.Agent.Config["tools"]; ok {
		if toolsArr, ok := toolsVal.(*interpreter.ArrayValue); ok {
			originalToolCount = len(toolsArr.Elements)
		}
	}

	// Set up REPL context with OnAgentModified callback
	replCtx := &interpreter.REPLContext{
		Models:       &interpreter.REPLModels{},
		LastCommand:  &interpreter.REPLLastCommand{},
		Agents:       []*interpreter.AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
		OnAgentModified: func(modifiedAgent *interpreter.AgentValue) {
			// Tools are now stored in agent.Config["tools"], no syncing needed
			// Agent config is the source of truth
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store custom tool in interpreter
	interp.SetVariable("customTool", customTool)

	// Replace tools with just the custom tool
	_, err := interp.EvalString(`gsh.repl.agents[0].tools = [customTool]`)
	if err != nil {
		t.Fatalf("unexpected error modifying tools: %v", err)
	}

	// Verify tools were updated in agent config
	toolsVal, ok := defaultState.Agent.Config["tools"]
	if !ok {
		t.Fatalf("expected tools to be set in agent config after modification")
	}
	toolsArr, ok := toolsVal.(*interpreter.ArrayValue)
	if !ok {
		t.Fatalf("expected tools to be an array")
	}

	// Should now have just the custom tool
	if len(toolsArr.Elements) != 1 {
		t.Errorf("expected 1 tool after modification, got %d (was %d)", len(toolsArr.Elements), originalToolCount)
	}

	if len(toolsArr.Elements) > 0 {
		if nativeTool, ok := toolsArr.Elements[0].(*interpreter.NativeToolValue); ok {
			if nativeTool.Name != "custom_tool" {
				t.Errorf("expected custom_tool, got %s", nativeTool.Name)
			}
		}
	}
}

// mockProvider implements interpreter.ModelProvider for testing
type mockProvider struct {
	providerName string
}

func (m *mockProvider) Name() string {
	return m.providerName
}

func (m *mockProvider) ChatCompletion(request interpreter.ChatRequest) (*interpreter.ChatResponse, error) {
	return &interpreter.ChatResponse{
		Content: "mock response",
	}, nil
}

func (m *mockProvider) StreamingChatCompletion(request interpreter.ChatRequest, callbacks *interpreter.StreamCallbacks) (*interpreter.ChatResponse, error) {
	return &interpreter.ChatResponse{
		Content: "mock response",
	}, nil
}
