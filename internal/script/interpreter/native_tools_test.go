package interpreter

import (
	"strings"
	"testing"
)

func TestNativeToolsRegistered(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	// Get the gsh object
	gshVal, exists := interp.env.Get("gsh")
	if !exists {
		t.Fatal("gsh object not found")
	}

	gshObj, ok := gshVal.(*GshObjectValue)
	if !ok {
		t.Fatalf("gsh is not a GshObjectValue, got %T", gshVal)
	}

	// Get the tools object
	toolsVal := gshObj.GetProperty("tools")
	if toolsVal == nil || toolsVal.Type() == ValueTypeNull {
		t.Fatal("gsh.tools not found")
	}

	toolsObj, ok := toolsVal.(*ObjectValue)
	if !ok {
		t.Fatalf("gsh.tools is not an ObjectValue, got %T", toolsVal)
	}

	// Check that all native tools are registered
	expectedTools := []string{"exec", "grep", "view_file", "edit_file"}
	for _, toolName := range expectedTools {
		toolVal := toolsObj.GetPropertyValue(toolName)
		if toolVal == nil || toolVal.Type() == ValueTypeNull {
			t.Errorf("gsh.tools.%s not found", toolName)
			continue
		}

		nativeTool, ok := toolVal.(*NativeToolValue)
		if !ok {
			t.Errorf("gsh.tools.%s is not a NativeToolValue, got %T", toolName, toolVal)
			continue
		}

		if nativeTool.Name != toolName {
			t.Errorf("gsh.tools.%s has wrong name: %s", toolName, nativeTool.Name)
		}

		if nativeTool.Description == "" {
			t.Errorf("gsh.tools.%s has empty description", toolName)
		}

		if nativeTool.Parameters == nil {
			t.Errorf("gsh.tools.%s has nil parameters", toolName)
		}

		if nativeTool.Invoke == nil {
			t.Errorf("gsh.tools.%s has nil Invoke function", toolName)
		}
	}
}

func TestNativeToolType(t *testing.T) {
	tool := &NativeToolValue{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
		},
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return "result", nil
		},
	}

	if tool.Type() != ValueTypeTool {
		t.Errorf("NativeToolValue.Type() = %v, want %v", tool.Type(), ValueTypeTool)
	}

	if !tool.IsTruthy() {
		t.Error("NativeToolValue.IsTruthy() should return true")
	}

	expected := "<native_tool test_tool>"
	if tool.String() != expected {
		t.Errorf("NativeToolValue.String() = %q, want %q", tool.String(), expected)
	}

	// Test equality
	tool2 := &NativeToolValue{Name: "test_tool"}
	tool3 := &NativeToolValue{Name: "other_tool"}

	if !tool.Equals(tool2) {
		t.Error("NativeToolValue.Equals() should return true for same name")
	}

	if tool.Equals(tool3) {
		t.Error("NativeToolValue.Equals() should return false for different name")
	}
}

func TestNativeToolCallFromScript(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	// Test calling gsh.tools.grep with a pattern
	result, err := interp.EvalString(`
		result = gsh.tools.grep({pattern: "TestNativeToolCallFromScript"})
		result
	`, nil)

	if err != nil {
		t.Fatalf("Failed to call gsh.tools.grep: %v", err)
	}

	// The result should be a string (JSON format)
	strVal, ok := result.Value().(*StringValue)
	if !ok {
		t.Fatalf("Expected string result, got %T", result.Value())
	}

	// The output should contain this test file
	if !strings.Contains(strVal.Value, "native_tools_test.go") {
		t.Logf("Result: %s", strVal.Value)
		t.Error("grep result should contain native_tools_test.go")
	}
}

func TestNativeToolInAgentConfig(t *testing.T) {
	// Test that native tools can be used in agent tool arrays
	interp := New(nil)
	defer interp.Close()

	// Create an agent with native tools
	_, err := interp.EvalString(`
		model testModel {
			provider: "openai",
			apiKey: "test",
			baseURL: "http://localhost:11434/v1",
			model: "test",
		}

		agent testAgent {
			model: testModel,
			systemPrompt: "You are a test agent",
			tools: [gsh.tools.exec, gsh.tools.grep, gsh.tools.view_file, gsh.tools.edit_file],
		}
	`, nil)

	if err != nil {
		t.Fatalf("Failed to create agent with native tools: %v", err)
	}

	// Get the agent
	agentVal, exists := interp.env.Get("testAgent")
	if !exists {
		t.Fatal("testAgent not found")
	}

	agent, ok := agentVal.(*AgentValue)
	if !ok {
		t.Fatalf("testAgent is not an AgentValue, got %T", agentVal)
	}

	// Check the tools array
	toolsVal, ok := agent.Config["tools"]
	if !ok {
		t.Fatal("agent has no tools config")
	}

	toolsArr, ok := toolsVal.(*ArrayValue)
	if !ok {
		t.Fatalf("agent tools is not an ArrayValue, got %T", toolsVal)
	}

	if len(toolsArr.Elements) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(toolsArr.Elements))
	}

	// Verify each tool is a NativeToolValue
	expectedNames := []string{"exec", "grep", "view_file", "edit_file"}
	for i, elem := range toolsArr.Elements {
		nativeTool, ok := elem.(*NativeToolValue)
		if !ok {
			t.Errorf("Tool %d is not a NativeToolValue, got %T", i, elem)
			continue
		}
		if nativeTool.Name != expectedNames[i] {
			t.Errorf("Tool %d has wrong name: got %s, want %s", i, nativeTool.Name, expectedNames[i])
		}
	}
}

func TestConvertNativeToolToChatTool(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	tool := &NativeToolValue{
		Name:        "test_tool",
		Description: "A test tool description",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{
					"type":        "string",
					"description": "First argument",
				},
			},
			"required": []string{"arg1"},
		},
	}

	chatTool := interp.convertNativeToolToChatTool(tool)

	if chatTool.Name != "test_tool" {
		t.Errorf("ChatTool.Name = %q, want %q", chatTool.Name, "test_tool")
	}

	if chatTool.Description != "A test tool description" {
		t.Errorf("ChatTool.Description = %q, want %q", chatTool.Description, "A test tool description")
	}

	if chatTool.Parameters == nil {
		t.Error("ChatTool.Parameters should not be nil")
	}
}
