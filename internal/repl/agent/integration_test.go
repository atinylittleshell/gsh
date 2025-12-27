package agent

import (
	"testing"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// Integration tests to verify agent state is properly initialized.
// These tests catch bugs where required fields are missing during initialization.

func TestAgentState_RequiredFields(t *testing.T) {
	// This test verifies that all required fields are set when using SetupAgentWithDefaultTools
	provider := newMockProvider()
	interp := interpreter.New()

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

	// SetupAgentWithDefaultTools should populate Tools and ToolExecutor
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

	if len(state.Tools) == 0 {
		t.Error("Tools should be populated by SetupAgentWithDefaultTools")
	}

	if state.ToolExecutor == nil {
		t.Error("ToolExecutor should be populated by SetupAgentWithDefaultTools")
	}
}

func TestAgentState_ToolsAndExecutorConsistency(t *testing.T) {
	// This test verifies that tools and executor are set together
	state := &State{}
	SetupAgentWithDefaultTools(state)

	// Both should be set
	if len(state.Tools) == 0 {
		t.Error("Tools should be set")
	}

	if state.ToolExecutor == nil {
		t.Error("ToolExecutor should be set")
	}

	// Verify default tools are present
	toolNames := make(map[string]bool)
	for _, tool := range state.Tools {
		toolNames[tool.Name] = true
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
	logger := newTestLogger()
	manager := NewManager(logger)

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

	err := manager.SendMessage(nil, "hello", nil)

	if err == nil {
		t.Fatal("Expected error when Interpreter is nil")
	}

	expectedMsg := "BUG: interpreter not configured"
	if !containsString(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestDefaultTools_AllHaveExecutors(t *testing.T) {
	// Verify that all default tools can be executed
	tools := DefaultTools()
	executor := DefaultToolExecutor(nil) // nil writer is fine for this test

	for _, tool := range tools {
		// The executor should recognize all default tools (even if execution fails due to missing args)
		_, err := executor(nil, tool.Name, map[string]interface{}{})

		// We expect errors due to missing arguments, but NOT "unknown tool" errors
		if err != nil && containsString(err.Error(), "unknown tool") {
			t.Errorf("Tool '%s' is defined but not handled by executor", tool.Name)
		}
	}
}

func TestDefaultTools_Definitions(t *testing.T) {
	tools := DefaultTools()

	// Verify each tool has required fields
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("Tool has empty name")
		}

		if tool.Description == "" {
			t.Errorf("Tool '%s' has empty description", tool.Name)
		}

		if tool.Parameters == nil {
			t.Errorf("Tool '%s' has nil parameters", tool.Name)
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
