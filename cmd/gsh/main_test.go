package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// captureStdout captures stdout during the execution of fn and returns the captured output
func captureStdout(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestIsGshScript(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"gsh script", "test.gsh", true},
		{"gsh script with path", "/path/to/script.gsh", true},
		{"shell script", "test.sh", false},
		{"bash script", "script.bash", false},
		{"no extension", "script", false},
		{"other extension", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGshScript(tt.filePath)
			if result != tt.expected {
				t.Errorf("isGshScript(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestRunGshScript(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gsh-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple logger for testing
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	t.Run("simple script", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "simple.gsh")
		scriptContent := `x = 5
y = 10
z = x + y
print("Result: " + z)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript failed: %v", err)
		}

		// Assert output
		if !strings.Contains(output, "Result: 15") {
			t.Errorf("Expected output to contain 'Result: 15', got: %s", output)
		}
	})

	t.Run("script with variables and operations", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "variables.gsh")
		scriptContent := `name = "Alice"
age = 30
greeting = "Hello, " + name + "!"
print(greeting)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript failed: %v", err)
		}

		// Assert output
		if !strings.Contains(output, "Hello, Alice!") {
			t.Errorf("Expected output to contain 'Hello, Alice!', got: %s", output)
		}
	})

	t.Run("script with control flow", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "control_flow.gsh")
		scriptContent := `count = 0
for (i of [1, 2, 3]) {
    count = count + i
}
print(count)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript failed: %v", err)
		}

		// Assert output
		output = strings.TrimSpace(output)
		if output != "6" {
			t.Errorf("Expected output to be '6', got: %s", output)
		}
	})

	t.Run("script with tool declaration", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "tool.gsh")
		scriptContent := `tool add(a, b) {
    return a + b
}

result = add(5, 3)
print(result)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript failed: %v", err)
		}

		// Assert output
		output = strings.TrimSpace(output)
		if output != "8" {
			t.Errorf("Expected output to be '8', got: %s", output)
		}
	})

	t.Run("script with shebang", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "shebang.gsh")
		scriptContent := `#!/usr/bin/env gsh
x = 42
print(x)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript with shebang failed: %v", err)
		}

		// Assert output
		output = strings.TrimSpace(output)
		if output != "42" {
			t.Errorf("Expected output to be '42', got: %s", output)
		}
	})

	t.Run("script with syntax error", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "syntax_error.gsh")
		scriptContent := `x = 
y = 10
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		err := runGshScript(ctx, scriptPath, logger)
		if err == nil {
			t.Error("Expected error for syntax error, got nil")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "nonexistent.gsh")
		ctx := context.Background()
		err := runGshScript(ctx, scriptPath, logger)
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("script with runtime error", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "runtime_error.gsh")
		scriptContent := `x = undefinedVariable
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		err := runGshScript(ctx, scriptPath, logger)
		if err == nil {
			t.Error("Expected runtime error, got nil")
		}
	})

	t.Run("script with try-catch", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "try_catch.gsh")
		scriptContent := `result = "success"
try {
    x = 10
    y = 0
} catch (e) {
    result = "caught"
}
print(result)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript with try-catch failed: %v", err)
		}

		// Assert output (no error, so result should be "success")
		output = strings.TrimSpace(output)
		if output != "success" {
			t.Errorf("Expected output to be 'success', got: %s", output)
		}
	})

	t.Run("script with arrays and objects", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "collections.gsh")
		scriptContent := `arr = [1, 2, 3]
obj = {name: "Bob", age: 25}
print(arr[0])
print(obj.name)
`
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript with collections failed: %v", err)
		}

		// Assert output
		if !strings.Contains(output, "1") {
			t.Errorf("Expected output to contain '1', got: %s", output)
		}
		if !strings.Contains(output, "Bob") {
			t.Errorf("Expected output to contain 'Bob', got: %s", output)
		}
	})

	t.Run("script with template literals", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "template.gsh")
		scriptContent := "name = \"World\"\ngreeting = `Hello, ${name}!`\nprint(greeting)\n"
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var err error
		output := captureStdout(func() {
			err = runGshScript(ctx, scriptPath, logger)
		})

		if err != nil {
			t.Errorf("runGshScript with template literals failed: %v", err)
		}

		// Assert output
		output = strings.TrimSpace(output)
		if output != "Hello, World!" {
			t.Errorf("Expected output to be 'Hello, World!', got: %s", output)
		}
	})
}

func TestShebangSupport(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gsh-shebang-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple logger for testing
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name           string
		shebang        string
		script         string
		wantErr        bool
		expectedOutput string
	}{
		{
			name:           "standard shebang",
			shebang:        "#!/usr/bin/env gsh",
			script:         "x = 1\nprint(x)",
			wantErr:        false,
			expectedOutput: "1",
		},
		{
			name:           "direct path shebang",
			shebang:        "#!/usr/local/bin/gsh",
			script:         "y = 2\nprint(y)",
			wantErr:        false,
			expectedOutput: "2",
		},
		{
			name:           "no shebang",
			shebang:        "",
			script:         "z = 3\nprint(z)",
			wantErr:        false,
			expectedOutput: "3",
		},
		{
			name:           "shebang with comment after",
			shebang:        "#!/usr/bin/env gsh\n# This is a comment",
			script:         "a = 4\nprint(a)",
			wantErr:        false,
			expectedOutput: "4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, tt.name+".gsh")
			content := tt.shebang
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += tt.script

			if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
				t.Fatalf("Failed to write test script: %v", err)
			}

			ctx := context.Background()
			var err error
			output := captureStdout(func() {
				err = runGshScript(ctx, scriptPath, logger)
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("runGshScript() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Assert output
			output = strings.TrimSpace(output)
			if !tt.wantErr && output != tt.expectedOutput {
				t.Errorf("Expected output to be '%s', got: %s", tt.expectedOutput, output)
			}
		})
	}
}
