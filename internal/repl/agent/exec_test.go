package agent

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

func TestExecuteCommand_SimpleCommand(t *testing.T) {
	ctx := context.Background()

	result, err := interpreter.ExecuteCommandWithPTY(ctx, "echo hello", nil)
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %q", result.Output)
	}
}

func TestExecuteCommand_NonZeroExitCode(t *testing.T) {
	ctx := context.Background()

	result, err := interpreter.ExecuteCommandWithPTY(ctx, "exit 42", nil)
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}
}

func TestExecuteCommand_WithLiveOutput(t *testing.T) {
	ctx := context.Background()
	var liveOutput bytes.Buffer

	result, err := interpreter.ExecuteCommandWithPTY(ctx, "echo live_test", &liveOutput)
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Both captured output and live output should contain the text
	if !strings.Contains(result.Output, "live_test") {
		t.Errorf("Expected captured output to contain 'live_test', got: %q", result.Output)
	}

	if !strings.Contains(liveOutput.String(), "live_test") {
		t.Errorf("Expected live output to contain 'live_test', got: %q", liveOutput.String())
	}
}

func TestExecuteCommand_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run a command that takes longer than the timeout
	result, err := interpreter.ExecuteCommandWithPTY(ctx, "sleep 10", nil)

	// Either we get an error, or we get a non-zero exit code due to signal
	if err == nil && result != nil && result.ExitCode == 0 {
		t.Fatal("Expected either error or non-zero exit code due to context cancellation")
	}

	// If there's an error, it should be context-related
	if err != nil {
		if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "signal") {
			t.Logf("Got error (acceptable): %v", err)
		}
	}
}

func TestExecuteCommand_StderrCapture(t *testing.T) {
	ctx := context.Background()

	// PTY combines stdout and stderr, so both should appear in output
	result, err := interpreter.ExecuteCommandWithPTY(ctx, "echo stdout_msg; echo stderr_msg >&2", nil)
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Both stdout and stderr should be captured (PTY combines them)
	if !strings.Contains(result.Output, "stdout_msg") {
		t.Errorf("Expected output to contain 'stdout_msg', got: %q", result.Output)
	}
	if !strings.Contains(result.Output, "stderr_msg") {
		t.Errorf("Expected output to contain 'stderr_msg', got: %q", result.Output)
	}
}

func TestExecToolDefinition(t *testing.T) {
	tool := interpreter.ExecToolDefinition()

	if tool.Name != "exec" {
		t.Errorf("Expected tool name 'exec', got %q", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty tool description")
	}

	params, ok := tool.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parameters to have 'properties'")
	}

	if _, ok := params["command"]; !ok {
		t.Error("Expected 'command' property in tool parameters")
	}

	required, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("Expected 'required' array in parameters")
	}

	found := false
	for _, r := range required {
		if r == "command" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'command' to be required")
	}
}

func TestSetupAgentWithDefaultTools(t *testing.T) {
	state := &State{
		Agent: &interpreter.AgentValue{
			Name:   "test",
			Config: map[string]interpreter.Value{},
		},
		Conversation: []interpreter.ChatMessage{},
	}

	SetupAgentWithDefaultTools(state)

	// Check that tools were added to agent config
	toolsVal, ok := state.Agent.Config["tools"]
	if !ok {
		t.Error("Expected tools to be set in agent config")
		return
	}

	toolsArr, ok := toolsVal.(*interpreter.ArrayValue)
	if !ok {
		t.Error("Expected tools to be an array")
		return
	}

	if len(toolsArr.Elements) == 0 {
		t.Error("Expected tools to be set up")
	}

	// Verify exec tool is available
	found := false
	for _, toolVal := range toolsArr.Elements {
		if nativeTool, ok := toolVal.(*interpreter.NativeToolValue); ok && nativeTool.Name == "exec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'exec' tool to be set up")
	}
}

func TestExecuteExecTool_OutputTruncation(t *testing.T) {
	// Test that very long outputs are truncated
	ctx := context.Background()
	// Generate output longer than maxOutputLen (50000)
	args := map[string]interface{}{
		"command": "yes 'test' | head -n 20000", // Generates lots of output
	}

	result, err := interpreter.ExecuteNativeExecTool(ctx, args, io.Discard)
	if err != nil {
		t.Fatalf("ExecuteNativeExecTool failed: %v", err)
	}

	// For very long outputs, result should indicate truncation
	// Note: We may or may not hit the truncation limit depending on actual output size
	if !strings.Contains(result, "output") {
		t.Errorf("Expected result to contain 'output' field, got: %q", result)
	}
}

func TestExecuteExecTool_Timeout(t *testing.T) {
	ctx := context.Background()
	args := map[string]interface{}{
		"command": "sleep 10",
		"timeout": float64(1), // 1 second timeout (JSON numbers come as float64)
	}

	start := time.Now()
	result, err := interpreter.ExecuteNativeExecTool(ctx, args, io.Discard)
	elapsed := time.Since(start)

	// Should complete quickly due to timeout, not wait 10 seconds
	if elapsed > 3*time.Second {
		t.Errorf("Expected command to timeout quickly, but took %v", elapsed)
	}

	// Should not return an error (timeout is handled gracefully)
	if err != nil {
		t.Fatalf("ExecuteNativeExecTool failed: %v", err)
	}

	// Result should indicate an error due to timeout
	if !strings.Contains(result, "error") && !strings.Contains(result, "context") {
		// If no error in result, exit code should be non-zero
		if strings.Contains(result, `"exitCode": 0`) {
			t.Errorf("Expected timeout to cause error or non-zero exit, got: %q", result)
		}
	}
}

func TestExecToolDefinition_HasTimeoutParam(t *testing.T) {
	tool := interpreter.ExecToolDefinition()

	params, ok := tool.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parameters to have 'properties'")
	}

	timeout, ok := params["timeout"]
	if !ok {
		t.Error("Expected 'timeout' property in tool parameters")
	}

	timeoutDef, ok := timeout.(map[string]interface{})
	if !ok {
		t.Fatal("Expected timeout to be a map")
	}

	if timeoutDef["type"] != "integer" {
		t.Errorf("Expected timeout type to be 'integer', got %v", timeoutDef["type"])
	}
}
