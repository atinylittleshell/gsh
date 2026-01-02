package interpreter

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectGrepBackend(t *testing.T) {
	backend := DetectGrepBackend()

	// Should detect at least one backend on most systems
	// We don't fail if none is available, just log it
	switch backend {
	case GrepBackendRipgrep:
		t.Log("Detected ripgrep (rg)")
	case GrepBackendGitGrep:
		t.Log("Detected git grep")
	case GrepBackendGrep:
		t.Log("Detected grep")
	case GrepBackendNone:
		t.Log("No grep backend detected")
	}
}

func TestBuildGrepCommand_Ripgrep(t *testing.T) {
	cmdName, args, err := BuildGrepCommand(GrepBackendRipgrep, "test-pattern")
	if err != nil {
		t.Fatalf("BuildGrepCommand failed: %v", err)
	}

	if cmdName != "rg" {
		t.Errorf("Expected command 'rg', got %q", cmdName)
	}

	// Check that pattern is included with -e flag
	foundPattern := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == "test-pattern" {
			foundPattern = true
			break
		}
	}
	if !foundPattern {
		t.Errorf("Expected pattern 'test-pattern' with -e flag in args, got %v", args)
	}
}

func TestBuildGrepCommand_GitGrep(t *testing.T) {
	cmdName, args, err := BuildGrepCommand(GrepBackendGitGrep, "test-pattern")
	if err != nil {
		t.Fatalf("BuildGrepCommand failed: %v", err)
	}

	if cmdName != "git" {
		t.Errorf("Expected command 'git', got %q", cmdName)
	}

	// First arg should be "grep"
	if len(args) == 0 || args[0] != "grep" {
		t.Errorf("Expected first arg to be 'grep', got %v", args)
	}

	// Check that pattern is included
	foundPattern := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == "test-pattern" {
			foundPattern = true
			break
		}
	}
	if !foundPattern {
		t.Errorf("Expected pattern 'test-pattern' with -e flag in args, got %v", args)
	}
}

func TestBuildGrepCommand_Grep(t *testing.T) {
	cmdName, args, err := BuildGrepCommand(GrepBackendGrep, "test-pattern")
	if err != nil {
		t.Fatalf("BuildGrepCommand failed: %v", err)
	}

	if cmdName != "grep" {
		t.Errorf("Expected command 'grep', got %q", cmdName)
	}

	// Check that pattern is included
	foundPattern := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == "test-pattern" {
			foundPattern = true
			break
		}
	}
	if !foundPattern {
		t.Errorf("Expected pattern 'test-pattern' with -e flag in args, got %v", args)
	}
}

func TestBuildGrepCommand_None(t *testing.T) {
	_, _, err := BuildGrepCommand(GrepBackendNone, "test-pattern")
	if err == nil {
		t.Fatal("Expected error for GrepBackendNone, got nil")
	}

	if !strings.Contains(err.Error(), "no grep tool available") {
		t.Errorf("Expected 'no grep tool available' error, got: %v", err)
	}
}

func TestExecuteGrep_Integration(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello world\nfoo bar\nhello again\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	result, err := ExecuteGrep(ctx, "hello")
	if err != nil {
		t.Fatalf("ExecuteGrep failed: %v", err)
	}

	// Should find matches (exit code 0)
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Output should contain "hello"
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("Expected output to contain 'hello', got: %q", result.Output)
	}

	// Backend should be set
	if result.Backend == "" || result.Backend == "none" {
		t.Errorf("Expected valid backend name, got: %q", result.Backend)
	}
}

func TestExecuteGrep_NoMatches(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello world\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	result, err := ExecuteGrep(ctx, "nonexistent_pattern_xyz123")
	if err != nil {
		t.Fatalf("ExecuteGrep failed: %v", err)
	}

	// Should return exit code 1 for no matches
	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1 for no matches, got %d", result.ExitCode)
	}
}

func TestGrepToolDefinition(t *testing.T) {
	tool := GrepToolDefinition()

	if tool.Name != "grep" {
		t.Errorf("Expected tool name 'grep', got %q", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty tool description")
	}

	params, ok := tool.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parameters to have 'properties'")
	}

	if _, ok := params["pattern"]; !ok {
		t.Error("Expected 'pattern' property in tool parameters")
	}

	required, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("Expected 'required' array in parameters")
	}

	found := false
	for _, r := range required {
		if r == "pattern" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'pattern' to be required")
	}
}

func TestExecuteGrepTool_MissingPattern(t *testing.T) {
	ctx := context.Background()
	args := map[string]interface{}{} // No "pattern" argument

	_, err := ExecuteNativeGrepTool(ctx, args)
	if err == nil {
		t.Fatal("Expected error for missing pattern")
	}

	if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("Expected error to mention 'pattern', got: %v", err)
	}
}

func TestExecuteGrepTool_EmptyPattern(t *testing.T) {
	ctx := context.Background()
	args := map[string]interface{}{
		"pattern": "",
	}

	_, err := ExecuteNativeGrepTool(ctx, args)
	if err == nil {
		t.Fatal("Expected error for empty pattern")
	}

	if !strings.Contains(err.Error(), "non-empty") {
		t.Errorf("Expected error to mention 'non-empty', got: %v", err)
	}
}

func TestExecuteGrepTool_Integration(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "grep-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "unique_grep_tool_test_marker\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	args := map[string]interface{}{
		"pattern": "unique_grep_tool_test_marker",
	}

	result, err := ExecuteNativeGrepTool(ctx, args)
	if err != nil {
		t.Fatalf("ExecuteNativeGrepTool failed: %v", err)
	}

	if !strings.Contains(result, "unique_grep_tool_test_marker") {
		t.Errorf("Expected result to contain 'unique_grep_tool_test_marker', got: %q", result)
	}

	if !strings.Contains(result, `"status": "matches_found"`) {
		t.Errorf("Expected result to contain status matches_found, got: %q", result)
	}
}

func TestExecuteGrepTool_NoMatches(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "grep-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "some content\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	args := map[string]interface{}{
		"pattern": "nonexistent_pattern_abc789",
	}

	result, err := ExecuteNativeGrepTool(ctx, args)
	if err != nil {
		t.Fatalf("ExecuteGrepTool failed: %v", err)
	}

	if !strings.Contains(result, `"status": "no_matches"`) {
		t.Errorf("Expected result to contain status no_matches, got: %q", result)
	}

	if !strings.Contains(result, `"exitCode": 1`) {
		t.Errorf("Expected result to contain exitCode 1, got: %q", result)
	}
}

func TestExecuteGrepTool_Direct(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "executor_grep_test\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	args := map[string]interface{}{
		"pattern": "executor_grep_test",
	}

	result, err := ExecuteNativeGrepTool(ctx, args)
	if err != nil {
		t.Fatalf("ExecuteNativeGrepTool failed: %v", err)
	}

	if !strings.Contains(result, "executor_grep_test") {
		t.Errorf("Expected result to contain 'executor_grep_test', got: %q", result)
	}
}

func TestIsGrepAvailable(t *testing.T) {
	available := DetectGrepBackend() != GrepBackendNone
	backend := GrepBackendName(DetectGrepBackend())
	t.Logf("Grep available: %v, backend: %s", available, backend)
}

func TestGetGrepBackendInfo(t *testing.T) {
	detectedBackend := DetectGrepBackend()
	backend := GrepBackendName(detectedBackend)
	available := detectedBackend != GrepBackendNone

	if available {
		if backend == "" || backend == "none" {
			t.Errorf("Expected valid backend name when available, got: %q", backend)
		}
	} else {
		if backend != "none" {
			t.Errorf("Expected 'none' backend when not available, got: %q", backend)
		}
	}
}

func TestBackendName(t *testing.T) {
	tests := []struct {
		backend  GrepBackend
		expected string
	}{
		{GrepBackendRipgrep, "rg"},
		{GrepBackendGitGrep, "git-grep"},
		{GrepBackendGrep, "grep"},
		{GrepBackendNone, "none"},
		{GrepBackend(999), "none"}, // Unknown backend
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			name := GrepBackendName(tt.backend)
			if name != tt.expected {
				t.Errorf("GrepBackendName(%d) = %q, want %q", tt.backend, name, tt.expected)
			}
		})
	}
}

func TestExecuteGrep_HiddenFiles(t *testing.T) {
	// Skip if no grep backend is available
	if DetectGrepBackend() == GrepBackendNone {
		t.Skip("No grep backend available")
	}

	// Create a temporary directory with a hidden file
	tmpDir, err := os.MkdirTemp("", "grep-hidden-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a hidden file with known content
	hiddenFile := filepath.Join(tmpDir, ".hidden_config")
	hiddenContent := "hidden_file_marker_xyz\n"
	if err := os.WriteFile(hiddenFile, []byte(hiddenContent), 0644); err != nil {
		t.Fatalf("Failed to write hidden file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	ctx := context.Background()
	result, err := ExecuteGrep(ctx, "hidden_file_marker_xyz")
	if err != nil {
		t.Fatalf("ExecuteGrep failed: %v", err)
	}

	// Should find matches in hidden file (exit code 0)
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0 (found in hidden file), got %d", result.ExitCode)
	}

	// Output should contain the hidden file marker
	if !strings.Contains(result.Output, "hidden_file_marker_xyz") {
		t.Errorf("Expected output to contain 'hidden_file_marker_xyz', got: %q", result.Output)
	}
}

func TestGitGrepDetection(t *testing.T) {
	// This test verifies git grep detection logic
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "git-grep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and add a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("git grep test content\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	// Verify git directory is detected properly by testing grep in it
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to git directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Grep should work in the git directory
	ctx := context.Background()
	result, err := ExecuteGrep(ctx, "git grep test content")
	if err != nil {
		t.Fatalf("ExecuteGrep failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}
