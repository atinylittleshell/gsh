package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteEdit(t *testing.T) {
	ctx := context.Background()

	t.Run("basic replacement", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		initialContent := "hello world\nfoo bar\nbaz qux"
		err := os.WriteFile(filePath, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Perform edit
		result, err := ExecuteEdit(ctx, filePath, "foo bar", "replaced text", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got: %s", result.Message)
		}

		// Verify content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		expected := "hello world\nreplaced text\nbaz qux"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("find string not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("hello world"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteEdit(ctx, filePath, "nonexistent", "replacement", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure when find string not found")
		}
		if result.Message != "find string not found in file" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("multiple occurrences error", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("foo bar foo bar foo bar"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteEdit(ctx, filePath, "foo bar", "replacement", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure when multiple occurrences found")
		}
		if result.Message != "find string appears 3 times in file (must appear exactly once)" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("replacement within line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		initialContent := "line 1 foo\nline 2 foo\nline 3 foo\nline 4 foo\nline 5 foo"
		err := os.WriteFile(filePath, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Replace only within lines 2-3
		result, err := ExecuteEdit(ctx, filePath, "line 2 foo", "line 2 bar", 2, 3)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got: %s", result.Message)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		expected := "line 1 foo\nline 2 bar\nline 3 foo\nline 4 foo\nline 5 foo"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("find string not found in line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		initialContent := "line 1 foo\nline 2 bar\nline 3 baz"
		err := os.WriteFile(filePath, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Try to find "foo" in lines 2-3 (it's only on line 1)
		result, err := ExecuteEdit(ctx, filePath, "foo", "replacement", 2, 3)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure when find string not in line range")
		}
	})

	t.Run("invalid line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("line 1\nline 2\nline 3"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// start_line > end_line
		result, err := ExecuteEdit(ctx, filePath, "line", "replacement", 3, 1)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure for invalid line range")
		}
	})

	t.Run("start_line exceeds file length", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("line 1\nline 2"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteEdit(ctx, filePath, "line", "replacement", 10, 20)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if result.Success {
			t.Fatal("expected failure when start_line exceeds file length")
		}
	})

	t.Run("relative path resolution", func(t *testing.T) {
		// Save current dir and change to temp dir
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

		// Create file with relative path
		err = os.WriteFile("test.txt", []byte("hello world"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteEdit(ctx, "test.txt", "hello", "goodbye", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got: %s", result.Message)
		}

		content, err := os.ReadFile("test.txt")
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "goodbye world" {
			t.Errorf("expected 'goodbye world', got %q", string(content))
		}
	})

	t.Run("CRLF line endings preserved", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		// File with Windows-style CRLF line endings
		initialContent := "line 1\r\nline 2\r\nline 3"
		err := os.WriteFile(filePath, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := ExecuteEdit(ctx, filePath, "line 2", "modified", 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got: %s", result.Message)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		// Should preserve CRLF line endings
		expected := "line 1\r\nmodified\r\nline 3"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("multiline find and replace", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		initialContent := "func foo() {\n    return 1\n}\n\nfunc bar() {\n    return 2\n}"
		err := os.WriteFile(filePath, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		find := "func foo() {\n    return 1\n}"
		replace := "func foo() {\n    return 42\n}"

		result, err := ExecuteEdit(ctx, filePath, find, replace, 0, 0)
		if err != nil {
			t.Fatalf("ExecuteEdit failed: %v", err)
		}
		if !result.Success {
			t.Fatalf("expected success, got: %s", result.Message)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		expected := "func foo() {\n    return 42\n}\n\nfunc bar() {\n    return 2\n}"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})
}

func TestExecuteEditTool(t *testing.T) {
	ctx := context.Background()

	t.Run("successful edit", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("hello world"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		args := map[string]interface{}{
			"file_path": filePath,
			"find":      "hello",
			"replace":   "goodbye",
		}

		result, err := ExecuteEditTool(ctx, args)
		if err != nil {
			t.Fatalf("ExecuteEditTool failed: %v", err)
		}
		if result != `{"success": true, "message": "edit applied successfully"}` {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("with line range", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(filePath, []byte("line 1\nline 2\nline 3"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		args := map[string]interface{}{
			"file_path":  filePath,
			"find":       "line 2",
			"replace":    "modified",
			"start_line": float64(2), // JSON numbers come as float64
			"end_line":   float64(2),
		}

		result, err := ExecuteEditTool(ctx, args)
		if err != nil {
			t.Fatalf("ExecuteEditTool failed: %v", err)
		}
		if result != `{"success": true, "message": "edit applied successfully"}` {
			t.Errorf("unexpected result: %s", result)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "line 1\nmodified\nline 3" {
			t.Errorf("unexpected content: %s", string(content))
		}
	})

	t.Run("missing required argument", func(t *testing.T) {
		args := map[string]interface{}{
			"file_path": "/tmp/test.txt",
			"find":      "hello",
			// missing "replace"
		}

		_, err := ExecuteEditTool(ctx, args)
		if err == nil {
			t.Fatal("expected error for missing argument")
		}
	})
}

func TestEditToolDefinition(t *testing.T) {
	def := EditToolDefinition()

	if def.Name != "edit_file" {
		t.Errorf("expected name 'edit_file', got %q", def.Name)
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
	requiredProps := []string{"file_path", "find", "replace", "start_line", "end_line"}
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
	expectedRequired := map[string]bool{"file_path": true, "find": true, "replace": true}
	for _, r := range required {
		if !expectedRequired[r] {
			t.Errorf("unexpected required field: %s", r)
		}
		delete(expectedRequired, r)
	}
	if len(expectedRequired) > 0 {
		t.Errorf("missing required fields: %v", expectedRequired)
	}
}
