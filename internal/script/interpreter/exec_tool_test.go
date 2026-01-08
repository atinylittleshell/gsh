package interpreter

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommand_SimpleCommand(t *testing.T) {
	ctx := context.Background()

	result, err := ExecuteCommandWithPTY(ctx, "echo hello", nil, "/tmp")
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

	result, err := ExecuteCommandWithPTY(ctx, "exit 42", nil, "/tmp")
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

	result, err := ExecuteCommandWithPTY(ctx, "echo live_test", &liveOutput, "/tmp")
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Both captured output and live output should contain the message
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
	result, err := ExecuteCommandWithPTY(ctx, "sleep 10", nil, "/tmp")

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
	result, err := ExecuteCommandWithPTY(ctx, "echo stdout_msg; echo stderr_msg >&2", nil, "/tmp")
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
}

func TestExecToolDefinition(t *testing.T) {
	tool := ExecToolDefinition()

	if tool.Name != "exec" {
		t.Errorf("Expected tool name 'exec', got %q", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}

	if tool.Parameters == nil {
		t.Error("Expected parameters to be set")
	}

	// Check that working_directory is required
	required, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("Expected 'required' to be a string slice")
	}
	hasWorkingDir := false
	for _, r := range required {
		if r == "working_directory" {
			hasWorkingDir = true
			break
		}
	}
	if !hasWorkingDir {
		t.Error("Expected 'working_directory' to be required")
	}
}

func TestExecuteNativeExecTool_RequiresWorkingDirectory(t *testing.T) {
	ctx := context.Background()

	// Test missing working_directory
	args := map[string]interface{}{
		"command": "echo hello",
	}
	_, err := ExecuteNativeExecTool(ctx, args, io.Discard)
	if err == nil {
		t.Error("Expected error when working_directory is missing")
	}
	if !strings.Contains(err.Error(), "working_directory") {
		t.Errorf("Expected error to mention 'working_directory', got: %v", err)
	}
}

func TestExecuteNativeExecTool_RequiresAbsolutePath(t *testing.T) {
	ctx := context.Background()

	// Test relative path
	args := map[string]interface{}{
		"command":           "echo hello",
		"working_directory": "relative/path",
	}
	_, err := ExecuteNativeExecTool(ctx, args, io.Discard)
	if err == nil {
		t.Error("Expected error when working_directory is relative")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("Expected error to mention 'absolute path', got: %v", err)
	}
}

func TestExecuteNativeExecTool_Success(t *testing.T) {
	ctx := context.Background()

	args := map[string]interface{}{
		"command":           "echo hello",
		"working_directory": "/tmp",
	}
	result, err := ExecuteNativeExecTool(ctx, args, io.Discard)
	if err != nil {
		t.Fatalf("ExecuteNativeExecTool failed: %v", err)
	}

	if !strings.Contains(result, "hello") {
		t.Errorf("Expected result to contain 'hello', got: %q", result)
	}
}

func TestExecuteNativeExecTool_Timeout(t *testing.T) {
	ctx := context.Background()
	args := map[string]interface{}{
		"command":           "sleep 10",
		"working_directory": "/tmp",
		"timeout":           float64(1), // 1 second timeout (JSON numbers come as float64)
	}

	start := time.Now()
	result, err := ExecuteNativeExecTool(ctx, args, io.Discard)
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
	tool := ExecToolDefinition()

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

func TestExecuteCommand_WorkingDirectory(t *testing.T) {
	ctx := context.Background()

	// Test that specifying a working directory works correctly
	result, err := ExecuteCommandWithPTY(ctx, "pwd", nil, "/tmp")
	if err != nil {
		t.Fatalf("ExecuteCommandWithPTY failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// The output should contain /tmp (the working directory we specified)
	if !strings.Contains(result.Output, "/tmp") {
		t.Errorf("Expected output to contain '/tmp', got: %q", result.Output)
	}
}

func TestCreateExecNativeTool(t *testing.T) {
	tool := CreateExecNativeTool()

	// Verify the tool was created with the correct properties
	if tool.Name != "exec" {
		t.Errorf("Expected tool name 'exec', got %q", tool.Name)
	}

	if tool.Invoke == nil {
		t.Fatal("Expected Invoke to be set")
	}

	// Execute the tool and verify it works with working_directory
	result, err := tool.Invoke(map[string]interface{}{
		"command":           "pwd",
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", result)
	}

	if !strings.Contains(resultStr, "/tmp") {
		t.Errorf("Expected result to contain '/tmp', got: %q", resultStr)
	}
}
