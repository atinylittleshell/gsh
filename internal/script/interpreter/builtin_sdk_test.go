package interpreter

import (
	"testing"
)

// TestGshReplNull tests that gsh.repl is null when no REPL context is set
func TestGshReplNull(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Without setting REPL context, gsh.repl should be null
	result, err := interp.EvalString(`gsh.repl == null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be true (gsh.repl is null)
	if boolVal, ok := result.FinalResult.(*BoolValue); ok {
		if !boolVal.Value {
			t.Errorf("expected true, got false")
		}
	} else {
		t.Errorf("expected bool, got %s", result.FinalResult.Type())
	}
}

// TestGshReplModels tests that gsh.repl.models is accessible and settable when REPL context is set
func TestGshReplModels(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context with empty model tiers (starts as nil)
	replCtx := &REPLContext{
		Models: &REPLModels{
			Lite:      nil,
			Workhorse: nil,
			Premium:   nil,
		},
		LastCommand: &REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test that model tiers start as null
	result, err := interp.EvalString(`gsh.repl.models.lite == null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected gsh.repl.models.lite to be null initially")
	}

	// Define a model and assign it to gsh.repl.models.lite
	_, err = interp.EvalString(`
model testLite {
	provider: "openai",
	model: "gpt-4-mini",
}
gsh.repl.models.lite = testLite
`)
	if err != nil {
		t.Fatalf("unexpected error setting lite model: %v", err)
	}

	// Test accessing gsh.repl.models.lite.name after assignment
	result, err = interp.EvalString(`gsh.repl.models.lite.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "testLite" {
			t.Errorf("expected 'testLite', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}

	// Test that workhorse and premium are still null
	result, err = interp.EvalString(`gsh.repl.models.workhorse == null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boolVal, ok := result.FinalResult.(*BoolValue); !ok || !boolVal.Value {
		t.Errorf("expected gsh.repl.models.workhorse to still be null")
	}

	// Test assigning a non-model value should fail
	_, err = interp.EvalString(`gsh.repl.models.premium = "not a model"`)
	if err == nil {
		t.Fatal("expected error when assigning non-model to gsh.repl.models.premium")
	}
}

// TestGshReplLastCommand tests that gsh.repl.lastCommand is accessible
func TestGshReplLastCommand(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context
	replCtx := &REPLContext{
		Models: &REPLModels{},
		LastCommand: &REPLLastCommand{
			ExitCode:   0,
			DurationMs: 0,
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test initial values
	result, err := interp.EvalString(`gsh.repl.lastCommand.exitCode`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 0 {
			t.Errorf("expected 0, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Update lastCommand through SDKConfig
	interp.SDKConfig().UpdateLastCommand(42, 1500)

	// Test updated values
	result, err = interp.EvalString(`gsh.repl.lastCommand.exitCode`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 42 {
			t.Errorf("expected 42, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Test durationMs
	result, err = interp.EvalString(`gsh.repl.lastCommand.durationMs`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 1500 {
			t.Errorf("expected 1500, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}
}

// TestGshEventHandlers tests that event handlers can be registered and retrieved
func TestGshEventHandlers(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Register an event handler using gsh.on
	result, err := interp.EvalString("tool myHandler() { return \"handler called\" }")
	if err != nil {
		t.Fatalf("unexpected error registering tool: %v", err)
	}

	result, err = interp.EvalString("gsh.on(\"test.event\", myHandler)")
	if err != nil {
		t.Fatalf("unexpected error calling gsh.on: %v", err)
	}

	// Result should be a string (handler ID)
	if _, ok := result.FinalResult.(*StringValue); !ok {
		t.Errorf("expected string (handler ID), got %s", result.FinalResult.Type())
	}

	// Verify the handler was registered
	handlers := interp.GetEventHandlers("test.event")
	if len(handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(handlers))
	}
}

// TestGshOnWithoutHandler tests gsh.on error handling
func TestGshOnWithoutHandler(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Try to register a non-tool as handler (should fail)
	_, err := interp.EvalString(`gsh.on("test.event", "not a tool")`)
	if err == nil {
		t.Fatal("expected error when passing non-tool to gsh.on")
	}
}

// TestGshOffRemovesAllHandlers tests that gsh.off without handlerID removes all handlers
func TestGshOffRemovesAllHandlers(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Register multiple handlers
	_, err := interp.EvalString(`
		tool handler1() { return "handler1" }
		tool handler2() { return "handler2" }
		gsh.on("test.event", handler1)
		gsh.on("test.event", handler2)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both handlers are registered
	handlers := interp.GetEventHandlers("test.event")
	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}

	// Remove all handlers
	_, err = interp.EvalString(`gsh.off("test.event")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all handlers are removed
	handlers = interp.GetEventHandlers("test.event")
	if len(handlers) != 0 {
		t.Errorf("expected 0 handlers after gsh.off without handlerID, got %d", len(handlers))
	}
}

// TestGshReplReadOnly tests that gsh.repl properties are read-only
func TestGshReplReadOnly(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Set up REPL context
	replCtx := &REPLContext{
		Models:      &REPLModels{},
		LastCommand: &REPLLastCommand{},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Try to modify gsh.repl (should fail)
	_, err := interp.EvalString(`gsh.repl = "something"`)
	if err == nil {
		t.Fatal("expected error when trying to assign to gsh.repl")
	}
}

// TestGshReplAgentsArray tests the gsh.repl.agents array
func TestGshReplAgentsArray(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Create a test model
	testModel := &ModelValue{
		Name: "test-model",
		Config: map[string]Value{
			"model": &StringValue{Value: "gpt-4"},
		},
	}

	// Set up REPL context with a default agent using AgentValue
	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "You are a helpful assistant."},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test agents.length
	result, err := interp.EvalString(`gsh.repl.agents.length`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 1 {
			t.Errorf("expected agents.length == 1, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Test accessing agents[0].name
	result, err = interp.EvalString(`gsh.repl.agents[0].name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "default" {
			t.Errorf("expected 'default', got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}

	// Test accessing agents[0].systemPrompt
	result, err = interp.EvalString(`gsh.repl.agents[0].systemPrompt`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "You are a helpful assistant." {
			t.Errorf("expected system prompt, got '%s'", strVal.Value)
		}
	} else {
		t.Errorf("expected string, got %s", result.FinalResult.Type())
	}
}

// TestGshReplAgentsModifyDefaultAgent tests modifying the default agent's properties
func TestGshReplAgentsModifyDefaultAgent(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	testModel := &ModelValue{
		Name:   "test-model",
		Config: map[string]Value{},
	}
	newModel := &ModelValue{
		Name:   "new-model",
		Config: map[string]Value{},
	}

	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Original prompt"},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store the new model in the interpreter so we can reference it
	interp.env.Set("newModel", newModel)

	// Test modifying systemPrompt
	_, err := interp.EvalString(`gsh.repl.agents[0].systemPrompt = "New system prompt"`)
	if err != nil {
		t.Fatalf("unexpected error modifying systemPrompt: %v", err)
	}

	// Verify the change - now stored in Config map
	if promptVal, ok := defaultAgent.Config["systemPrompt"].(*StringValue); ok {
		if promptVal.Value != "New system prompt" {
			t.Errorf("expected systemPrompt to be updated, got '%s'", promptVal.Value)
		}
	} else {
		t.Error("expected systemPrompt to be a StringValue")
	}

	// Test modifying model using the "model" property (quoted because 'model' is a keyword)
	_, err = interp.EvalString(`gsh.repl.agents[0]["model"] = newModel`)
	if err != nil {
		t.Fatalf("unexpected error modifying model: %v", err)
	}

	if modelVal, ok := defaultAgent.Config["model"].(*ModelValue); ok {
		if modelVal.Name != "new-model" {
			t.Errorf("expected model to be updated to new-model, got '%s'", modelVal.Name)
		}
	} else {
		t.Error("expected model to be a ModelValue")
	}

	// Test that agent names cannot be changed (they are used as keys in agent manager)
	_, err = interp.EvalString(`gsh.repl.agents[0].name = "renamed"`)
	if err == nil {
		t.Fatal("expected error when trying to rename an agent")
	}
}

// TestGshReplAgentsPush tests adding new agents via agents.push()
func TestGshReplAgentsPush(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	testModel := &ModelValue{
		Name:   "test-model",
		Config: map[string]Value{},
	}

	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Default prompt"},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	var addedAgent *AgentValue
	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
		OnAgentAdded: func(agent *AgentValue) {
			addedAgent = agent
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store model in interpreter for reference
	interp.env.Set("testModel", testModel)

	// Push a new agent (using quoted keys for reserved words)
	result, err := interp.EvalString(`
		gsh.repl.agents.push({
			name: "reviewer",
			"model": testModel,
			systemPrompt: "You are a code reviewer.",
			tools: []
		})
	`)
	if err != nil {
		t.Fatalf("unexpected error pushing agent: %v", err)
	}

	// Result should be the new length (2)
	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 2 {
			t.Errorf("expected push to return 2, got %v", numVal.Value)
		}
	} else {
		t.Errorf("expected number, got %s", result.FinalResult.Type())
	}

	// Verify agents.length is now 2
	result, err = interp.EvalString(`gsh.repl.agents.length`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 2 {
			t.Errorf("expected agents.length == 2, got %v", numVal.Value)
		}
	}

	// Verify the new agent's properties
	result, err = interp.EvalString(`gsh.repl.agents[1].name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "reviewer" {
			t.Errorf("expected 'reviewer', got '%s'", strVal.Value)
		}
	}

	// Verify callback was called
	if addedAgent == nil {
		t.Error("expected OnAgentAdded callback to be called")
	} else if addedAgent.Name != "reviewer" {
		t.Errorf("expected callback agent name 'reviewer', got '%s'", addedAgent.Name)
	}
}

// TestGshReplAgentsPushValidation tests validation in agents.push()
func TestGshReplAgentsPushValidation(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	testModel := &ModelValue{
		Name:   "test-model",
		Config: map[string]Value{},
	}

	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Default prompt"},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	interp.env.Set("testModel", testModel)

	// Test pushing without a name (should fail)
	_, err := interp.EvalString(`gsh.repl.agents.push({ "model": testModel })`)
	if err == nil {
		t.Fatal("expected error when pushing agent without name")
	}

	// Test pushing with "default" name (should fail)
	_, err = interp.EvalString(`gsh.repl.agents.push({ name: "default", "model": testModel })`)
	if err == nil {
		t.Fatal("expected error when pushing agent with name 'default'")
	}

	// Test pushing without model (should fail)
	_, err = interp.EvalString(`gsh.repl.agents.push({ name: "test" })`)
	if err == nil {
		t.Fatal("expected error when pushing agent without model")
	}

	// Test pushing duplicate name (should fail)
	_, err = interp.EvalString(`gsh.repl.agents.push({ name: "custom", "model": testModel })`)
	if err != nil {
		t.Fatalf("unexpected error pushing first custom agent: %v", err)
	}
	_, err = interp.EvalString(`gsh.repl.agents.push({ name: "custom", "model": testModel })`)
	if err == nil {
		t.Fatal("expected error when pushing agent with duplicate name")
	}
}

// TestGshReplCurrentAgent tests reading and writing currentAgent
func TestGshReplCurrentAgent(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	testModel := &ModelValue{
		Name:   "test-model",
		Config: map[string]Value{},
	}

	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Default prompt"},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	reviewerAgent := &AgentValue{
		Name: "reviewer",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Reviewer prompt"},
			"tools":        &ArrayValue{Elements: []Value{}},
		},
	}

	var switchedAgent *AgentValue
	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent, reviewerAgent},
		CurrentAgent: defaultAgent,
		OnAgentSwitch: func(agent *AgentValue) {
			switchedAgent = agent
		},
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Test reading currentAgent.name
	result, err := interp.EvalString(`gsh.repl.currentAgent.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "default" {
			t.Errorf("expected 'default', got '%s'", strVal.Value)
		}
	}

	// Test switching currentAgent
	_, err = interp.EvalString(`gsh.repl.currentAgent = gsh.repl.agents[1]`)
	if err != nil {
		t.Fatalf("unexpected error switching agent: %v", err)
	}

	// Verify the switch
	if replCtx.CurrentAgent != reviewerAgent {
		t.Error("expected currentAgent to be switched to reviewer")
	}

	// Verify callback was called
	if switchedAgent == nil || switchedAgent.Name != "reviewer" {
		t.Error("expected OnAgentSwitch callback to be called with reviewer agent")
	}

	// Test reading updated currentAgent.name
	result, err = interp.EvalString(`gsh.repl.currentAgent.name`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strVal, ok := result.FinalResult.(*StringValue); ok {
		if strVal.Value != "reviewer" {
			t.Errorf("expected 'reviewer', got '%s'", strVal.Value)
		}
	}
}

// TestGshReplAgentsToolsModification tests modifying agent tools
func TestGshReplAgentsToolsModification(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	testModel := &ModelValue{
		Name:   "test-model",
		Config: map[string]Value{},
	}

	execTool := CreateExecNativeTool()
	grepTool := CreateGrepNativeTool()

	defaultAgent := &AgentValue{
		Name: "default",
		Config: map[string]Value{
			"model":        testModel,
			"systemPrompt": &StringValue{Value: "Default prompt"},
			"tools":        &ArrayValue{Elements: []Value{execTool}},
		},
	}

	replCtx := &REPLContext{
		Models:       &REPLModels{},
		LastCommand:  &REPLLastCommand{},
		Agents:       []*AgentValue{defaultAgent},
		CurrentAgent: defaultAgent,
	}
	interp.SDKConfig().SetREPLContext(replCtx)

	// Store grepTool in interpreter
	interp.env.Set("grepTool", grepTool)

	// Test reading tools.length
	result, err := interp.EvalString(`gsh.repl.agents[0].tools.length`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if numVal, ok := result.FinalResult.(*NumberValue); ok {
		if numVal.Value != 1 {
			t.Errorf("expected tools.length == 1, got %v", numVal.Value)
		}
	}

	// Test replacing tools array
	_, err = interp.EvalString(`gsh.repl.agents[0].tools = [grepTool]`)
	if err != nil {
		t.Fatalf("unexpected error replacing tools: %v", err)
	}

	// Verify the change - tools are now stored in Config map
	toolsVal, ok := defaultAgent.Config["tools"].(*ArrayValue)
	if !ok {
		t.Fatal("expected tools to be an ArrayValue")
	}
	if len(toolsVal.Elements) != 1 {
		t.Errorf("expected 1 tool after replacement, got %d", len(toolsVal.Elements))
	}

	if nativeTool, ok := toolsVal.Elements[0].(*NativeToolValue); ok {
		if nativeTool.Name != "grep" {
			t.Errorf("expected tool name 'grep', got '%s'", nativeTool.Name)
		}
	} else {
		t.Error("expected NativeToolValue")
	}
}
