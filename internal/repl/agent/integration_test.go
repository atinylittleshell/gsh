package agent

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// Integration tests to verify agent state is properly initialized.
// These tests catch bugs where required fields are missing during initialization.

func TestAgentState_RequiredFields(t *testing.T) {
	// This test verifies that all required fields are set when using SetupAgentWithDefaultTools
	provider := newMockProvider()
	interp := interpreter.New(nil)

	model := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: provider,
	}

	agent := &interpreter.AgentValue{
		Name: "test",
		Config: map[string]interpreter.Value{
			"model": model,
		},
	}

	state := &State{
		Agent:       agent,
		Provider:    provider,
		Interpreter: interp,
	}

	// SetupAgentWithDefaultTools should populate tools in agent config
	SetupAgentWithDefaultTools(state)

	// Verify all required fields are set
	if state.Interpreter == nil {
		t.Error("Interpreter should be set")
	}

	if state.Provider == nil {
		t.Error("Provider should be set")
	}

	if state.Agent == nil {
		t.Error("Agent should be set")
	}

	// Verify tools were added to agent config
	toolsVal, ok := state.Agent.Config["tools"]
	if !ok {
		t.Error("Tools should be populated in agent config by SetupAgentWithDefaultTools")
	}
	if toolsVal != nil {
		if arr, ok := toolsVal.(*interpreter.ArrayValue); ok {
			if len(arr.Elements) == 0 {
				t.Error("Tools array should not be empty")
			}
		}
	}
}

func TestAgentState_ToolsAndExecutorConsistency(t *testing.T) {
	// This test verifies that tools are configured in agent config
	state := &State{
		Agent: &interpreter.AgentValue{
			Name:   "test",
			Config: map[string]interpreter.Value{},
		},
	}
	SetupAgentWithDefaultTools(state)

	// Tools should be set in agent config
	toolsVal, ok := state.Agent.Config["tools"]
	if !ok {
		t.Error("Tools should be set in agent config")
		return
	}

	toolsArr, ok := toolsVal.(*interpreter.ArrayValue)
	if !ok {
		t.Error("Tools should be an array value")
		return
	}

	if len(toolsArr.Elements) == 0 {
		t.Error("Tools array should not be empty")
		return
	}

	// Verify default tools are present by name
	toolNames := make(map[string]bool)
	for _, toolVal := range toolsArr.Elements {
		if nativeTool, ok := toolVal.(*interpreter.NativeToolValue); ok {
			toolNames[nativeTool.Name] = true
		}
	}

	expectedTools := []string{"exec", "grep", "edit_file", "view_file"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' to be present in default tools", name)
		}
	}
}

func TestAgentState_MissingInterpreter_ReturnsError(t *testing.T) {
	// This test verifies that SendMessage fails with a clear error when Interpreter is missing
	manager := NewManager()

	provider := newMockProvider()
	provider.addResponse("Hello!", nil)

	model := &interpreter.ModelValue{
		Name:     "test-model",
		Config:   map[string]interpreter.Value{},
		Provider: provider,
	}

	agent := &interpreter.AgentValue{
		Name: "test",
		Config: map[string]interpreter.Value{
			"model": model,
		},
	}

	// Intentionally NOT setting Interpreter to simulate the bug
	state := &State{
		Agent:       agent,
		Provider:    provider,
		Interpreter: nil, // BUG: missing interpreter
	}

	manager.AddAgent("test", state)
	_ = manager.SetCurrentAgent("test")

	err := manager.SendMessage(context.Background(), "hello")

	if err == nil {
		t.Fatal("Expected error when Interpreter is nil")
	}

	expectedMsg := "BUG: interpreter not configured"
	if !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestDefaultTools_AllHaveDefinitions(t *testing.T) {
	// Verify that all default tools are properly defined
	state := &State{
		Agent: &interpreter.AgentValue{
			Name:   "test",
			Config: map[string]interpreter.Value{},
		},
	}
	SetupAgentWithDefaultTools(state)

	toolsVal, ok := state.Agent.Config["tools"]
	if !ok {
		t.Fatal("Tools should be set in agent config")
	}

	toolsArr, ok := toolsVal.(*interpreter.ArrayValue)
	if !ok {
		t.Fatal("Tools should be an array value")
	}

	// Verify each tool has the required properties
	for _, toolVal := range toolsArr.Elements {
		if nativeTool, ok := toolVal.(*interpreter.NativeToolValue); ok {
			if nativeTool.Name == "" {
				t.Error("Tool has empty name")
			}

			if nativeTool.Description == "" {
				t.Errorf("Tool '%s' has empty description", nativeTool.Name)
			}

			if nativeTool.Parameters == nil {
				t.Errorf("Tool '%s' has nil parameters", nativeTool.Name)
			}

			if nativeTool.Invoke == nil {
				t.Errorf("Tool '%s' has nil Invoke function", nativeTool.Name)
			}
		}
	}
}

// Helper to create a test logger that doesn't output anything
func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

// Helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
