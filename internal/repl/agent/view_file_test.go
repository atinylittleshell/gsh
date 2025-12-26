package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteViewFile(t *testing.T) {
	ctx := context.Background()

	t.Run("view entire file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line one\nline two\nline three"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 0, 0)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		expected := "    1:line one\n    2:line two\n    3:line three"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("view line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line 1\nline 2\nline 3\nline 4\nline 5"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 2, 4)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		expected := "    2:line 2\n    3:line 3\n    4:line 4"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("single line", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line 1\nline 2\nline 3"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 2, 2)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		expected := "    2:line 2"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("line numbers with many digits", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		// Create file with 100+ lines
		var lines []string
		for i := 1; i <= 150; i++ {
			lines = append(lines, "content")
		}
		err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 98, 102)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		expected := "   98:content\n   99:content\n  100:content\n  101:content\n  102:content"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("end_line exceeds file length", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line 1\nline 2\nline 3"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// end_line 100 should be clamped to 3
		result, err := ExecuteViewFile(ctx, filePath, 1, 100)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		// Should contain all 3 lines
		expected := "    1:line 1\n    2:line 2\n    3:line 3"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("start_line exceeds file length", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line 1\nline 2"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err = ExecuteViewFile(ctx, filePath, 10, 20)
		if err == nil {
			t.Fatal("expected error when start_line exceeds file length")
		}
	})

	t.Run("invalid line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "line 1\nline 2\nline 3"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// start_line > end_line
		_, err = ExecuteViewFile(ctx, filePath, 3, 1)
		if err == nil {
			t.Fatal("expected error for invalid line range")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ExecuteViewFile(ctx, "/nonexistent/file.txt", 0, 0)
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("relative path resolution", func(t *testing.T) {
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		if err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}
		defer os.Chdir(origDir)

		err = os.WriteFile("test.txt", []byte("hello world"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, "test.txt", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		expected := "    1:hello world"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("CRLF line endings", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		// File with Windows-style CRLF line endings
		content := "line one\r\nline two\r\nline three"
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 0, 0)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		// Should normalize CRLF to LF in output
		expected := "    1:line one\n    2:line two\n    3:line three"
		if result != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(filePath, []byte(""), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteViewFile(ctx, filePath, 0, 0)
		if err != nil {
			t.Fatalf("ExecuteViewFile failed: %v", err)
		}

		// Empty file has one empty line
		expected := "    1:"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestExecuteViewFileTool(t *testing.T) {
	ctx := context.Background()

	t.Run("successful view", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("line 1\nline 2"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		args := map[string]interface{}{
			"file_path": filePath,
		}

		result, err := ExecuteViewFileTool(ctx, args)
		if err != nil {
			t.Fatalf("ExecuteViewFileTool failed: %v", err)
		}

		// Result should be plain text with line numbers
		expected := "    1:line 1\n    2:line 2"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("with line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("line 1\nline 2\nline 3\nline 4"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		args := map[string]interface{}{
			"file_path":  filePath,
			"start_line": float64(2), // JSON numbers come as float64
			"end_line":   float64(3),
		}

		result, err := ExecuteViewFileTool(ctx, args)
		if err != nil {
			t.Fatalf("ExecuteViewFileTool failed: %v", err)
		}

		expected := "    2:line 2\n    3:line 3"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
	})

	t.Run("missing required argument", func(t *testing.T) {
		args := map[string]interface{}{
			// missing "file_path"
		}

		_, err := ExecuteViewFileTool(ctx, args)
		if err == nil {
			t.Fatal("expected error for missing argument")
		}
	})

	t.Run("file not found returns error", func(t *testing.T) {
		args := map[string]interface{}{
			"file_path": "/nonexistent/file.txt",
		}

		_, err := ExecuteViewFileTool(ctx, args)
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "failed to read file") {
			t.Errorf("expected 'failed to read file' in error: %v", err)
		}
	})
}

func TestTruncateFromMiddle(t *testing.T) {
	t.Run("no truncation needed", func(t *testing.T) {
		lines := []string{"    1:short", "    2:line"}
		result := truncateFromMiddle(lines, 1000)
		expected := "    1:short\n    2:line"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("truncation with marker", func(t *testing.T) {
		// Create lines that exceed the limit
		lines := make([]string, 100)
		for i := 0; i < 100; i++ {
			lines[i] = fmt.Sprintf("%5d:this is line content number %d", i+1, i+1)
		}

		// Use a small limit to force truncation
		result := truncateFromMiddle(lines, 500)

		// Should contain (truncated) marker
		if !strings.Contains(result, "(truncated)") {
			t.Errorf("expected result to contain '(truncated)': %s", result)
		}

		// Should start with first line
		if !strings.HasPrefix(result, "    1:") {
			t.Errorf("expected result to start with first line: %s", result)
		}

		// Should end with last line
		if !strings.HasSuffix(result, "100") {
			t.Errorf("expected result to end with last line: %s", result)
		}

		// Should be under limit
		if len(result) > 500 {
			t.Errorf("expected result length <= 500, got %d", len(result))
		}
	})

	t.Run("empty lines", func(t *testing.T) {
		result := truncateFromMiddle([]string{}, 1000)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestViewFileToolDefinition(t *testing.T) {
	def := ViewFileToolDefinition()

	if def.Name != "view_file" {
		t.Errorf("expected name 'view_file', got %q", def.Name)
	}

	if def.Description == "" {
		t.Error("expected non-empty description")
	}

	params := def.Parameters

	if params["type"] != "object" {
		t.Errorf("expected type 'object', got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	// Check required properties exist
	requiredProps := []string{"file_path", "start_line", "end_line"}
	for _, prop := range requiredProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("expected property %q to exist", prop)
		}
	}

	// Check required array
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required to be []string")
	}
	if len(required) != 1 || required[0] != "file_path" {
		t.Errorf("expected required to be ['file_path'], got %v", required)
	}
}
