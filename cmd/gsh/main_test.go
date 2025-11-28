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

// TestHelpText tests that the help text contains all essential information
func TestHelpText(t *testing.T) {
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		// Header and usage
		{"has title", "gsh - An AI-powered shell", "Should have descriptive title"},
		{"has usage section", "USAGE:", "Should have usage section"},
		{"has modes section", "MODES:", "Should have modes section"},

		// Modes
		{"has interactive mode", "interactive POSIX-compatible shell", "Should document interactive mode"},
		{"has gsh script example", "gsh script.gsh", "Should show .gsh script execution"},
		{"has bash script example", "gsh script.sh", "Should show bash script execution"},
		{"has command mode", "-c \"command\"", "Should document -c flag"},
		{"has login shell", "-l", "Should document login shell flag"},

		// Scripting section
		{"has scripting section", "SCRIPTING:", "Should have scripting section"},
		{"has gsh extension info", ".gsh extension", "Should mention .gsh extension"},
		{"has agentic workflows", "agentic", "Should mention agentic workflows"},
		{"has MCP mention", "MCP", "Should mention MCP servers"},
		{"has AI models mention", "AI models", "Should mention AI models"},
		{"has agents mention", "agents", "Should mention agents"},

		// Documentation link
		{"has docs link", "https://github.com/atinylittleshell/gsh", "Should have documentation link"},

		// Options section
		{"has options section", "OPTIONS:", "Should have options section header"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(helpText, tt.contains) {
				t.Errorf("%s: helpText should contain %q", tt.desc, tt.contains)
			}
		})
	}
}

// TestHelpTextStructure tests the overall structure and formatting of help text
func TestHelpTextStructure(t *testing.T) {
	t.Run("sections are in logical order", func(t *testing.T) {
		usageIdx := strings.Index(helpText, "USAGE:")
		modesIdx := strings.Index(helpText, "MODES:")
		scriptingIdx := strings.Index(helpText, "SCRIPTING:")
		optionsIdx := strings.Index(helpText, "OPTIONS:")

		if usageIdx == -1 || modesIdx == -1 || scriptingIdx == -1 || optionsIdx == -1 {
			t.Fatal("Missing required sections")
		}

		if usageIdx > modesIdx {
			t.Error("USAGE should come before MODES")
		}
		if modesIdx > scriptingIdx {
			t.Error("MODES should come before SCRIPTING")
		}
		if scriptingIdx > optionsIdx {
			t.Error("SCRIPTING should come before OPTIONS")
		}
	})

	t.Run("help text is not empty", func(t *testing.T) {
		if len(helpText) < 100 {
			t.Error("Help text seems too short")
		}
	})

	t.Run("help text ends properly", func(t *testing.T) {
		// Should end with OPTIONS: followed by newline (flag.PrintDefaults adds the options)
		if !strings.HasSuffix(helpText, "OPTIONS:\n") {
			t.Error("Help text should end with OPTIONS: section header")
		}
	})

	t.Run("no trailing whitespace issues", func(t *testing.T) {
		lines := strings.Split(helpText, "\n")
		for i, line := range lines {
			if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
				t.Errorf("Line %d has trailing whitespace: %q", i+1, line)
			}
		}
	})

	t.Run("help text is concise", func(t *testing.T) {
		// Help text should be reasonably short - under 50 lines
		lines := strings.Split(helpText, "\n")
		if len(lines) > 50 {
			t.Errorf("Help text should be concise, got %d lines", len(lines))
		}
	})
}

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

// captureStderr captures stderr during the execution of fn and returns the captured output
func captureStderr(fn func()) string {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// TestErrorMessagesE2E tests that error messages are clear and helpful when executing .gsh scripts
func TestErrorMessagesE2E(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gsh-error-test-*")
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
		name               string
		script             string
		expectError        bool
		errorShouldContain []string // Multiple strings that should appear in error output
		description        string
	}{
		{
			name: "undefined variable",
			script: `x = 5
result = undefinedVar
print(result)`,
			expectError:        true,
			errorShouldContain: []string{"undefined variable", "undefinedVar", "line 2", "column"},
			description:        "Should report undefined variable with clear message and location",
		},
		{
			name: "syntax error - missing expression",
			script: `x = 
y = 10`,
			expectError:        true,
			errorShouldContain: []string{"Parse error", "expected"},
			description:        "Should report parse error with line information",
		},
		{
			name: "syntax error - unexpected token",
			script: `x = 5
y = @ 10`,
			expectError:        true,
			errorShouldContain: []string{"Parse error"},
			description:        "Should report unexpected token error",
		},
		{
			name: "type error - calling non-function",
			script: `x = 5
result = x()`,
			expectError:        true,
			errorShouldContain: []string{"cannot call", "non-tool", "number", "line 2", "column"},
			description:        "Should report type error when calling non-function with location",
		},
		{
			name: "type error - indexing non-array",
			script: `x = 5
result = x[0]`,
			expectError:        true,
			errorShouldContain: []string{"cannot index", "number", "line 2", "column"},
			description:        "Should report error when indexing non-indexable value with location",
		},
		{
			name: "type error - member access on non-object",
			script: `x = 5
result = x.property`,
			expectError:        true,
			errorShouldContain: []string{"cannot access property", "property", "number", "line 2", "column"},
			description:        "Should report error when accessing member on non-object with location",
		},
		{
			name: "division by zero",
			script: `x = 10
y = 0
result = x / y`,
			expectError:        true,
			errorShouldContain: []string{"division by zero"},
			description:        "Should report division by zero error",
		},
		{
			name: "tool call - wrong argument count",
			script: `tool add(a, b) {
    return a + b
}
result = add(5)`,
			expectError:        true,
			errorShouldContain: []string{"expects 2 arguments", "got 1", "line 4", "column"},
			description:        "Should report argument count mismatch with location",
		},
		{
			name: "array index out of bounds",
			script: `arr = [1, 2, 3]
result = arr[10]`,
			expectError:        true,
			errorShouldContain: []string{"index out of bounds", "10", "length", "line 2", "column"},
			description:        "Should report index out of bounds error with location",
		},
		{
			name: "object property not found",
			script: `obj = {name: "Alice", age: 30}
result = obj.nonexistent`,
			expectError:        true,
			errorShouldContain: []string{"property", "nonexistent", "not found", "line 2", "column"},
			description:        "Should report property not found error with location",
		},
		{
			name:               "invalid JSON parse",
			script:             `data = JSON.parse("{invalid json}")`,
			expectError:        true,
			errorShouldContain: []string{"JSON", "parse"},
			description:        "Should report JSON parse error",
		},
		{
			name: "break outside loop",
			script: `x = 5
break`,
			expectError:        true,
			errorShouldContain: []string{"break", "outside", "loop", "line 2", "column"},
			description:        "Should report break outside loop error with location",
		},
		{
			name: "continue outside loop",
			script: `x = 5
continue`,
			expectError:        true,
			errorShouldContain: []string{"continue", "outside", "loop", "line 2", "column"},
			description:        "Should report continue outside loop error with location",
		},
		{
			name: "invalid for-of - non-iterable",
			script: `for (item of 42) {
    print(item)
}`,
			expectError:        true,
			errorShouldContain: []string{"requires an iterable", "number", "line 1", "column"},
			description:        "Should report error when iterating over non-iterable with location",
		},
		{
			name: "return outside tool",
			script: `x = 5
return x`,
			expectError:        true,
			errorShouldContain: []string{"return", "outside", "line 2", "column"},
			description:        "Should report return outside function error with location",
		},
		{
			name: "syntax error - unclosed string",
			script: `x = "unclosed string
y = 10`,
			expectError:        true,
			errorShouldContain: []string{"Parse error"},
			description:        "Should report unclosed string error",
		},
		{
			name: "syntax error - unclosed brace",
			script: `if (true) {
    x = 5
# Missing closing brace`,
			expectError:        true,
			errorShouldContain: []string{"Parse error"},
			description:        "Should report unclosed brace error",
		},
		{
			name: "invalid operator",
			script: `x = 5
y = 10
result = x % y`,
			expectError:        false, // Modulo is actually supported, changed to expect success
			errorShouldContain: []string{},
			description:        "Modulo operator should work",
		},
		{
			name: "type error in binary operation",
			script: `x = "hello"
y = 5
result = x - y`,
			expectError:        true,
			errorShouldContain: []string{"operator", "-"},
			description:        "Should report type error in binary operation",
		},
		{
			name: "nested error with stack trace",
			script: `tool inner() {
    return undefinedVar
}
tool outer() {
    return inner()
}
result = outer()`,
			expectError:        true,
			errorShouldContain: []string{"undefinedVar", "undefined variable", "line 2", "column"},
			description:        "Should show error with proper context and location for nested calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tmpDir, tt.name+".gsh")
			if err := os.WriteFile(scriptPath, []byte(tt.script), 0644); err != nil {
				t.Fatalf("Failed to write test script: %v", err)
			}

			ctx := context.Background()
			var execErr error
			stderr := captureStderr(func() {
				execErr = runGshScript(ctx, scriptPath, logger)
			})

			if tt.expectError && execErr == nil {
				t.Errorf("%s: Expected error but got none", tt.description)
				return
			}

			if !tt.expectError && execErr != nil {
				t.Errorf("%s: Unexpected error: %v", tt.description, execErr)
				return
			}

			if tt.expectError {
				// Combine error message and stderr for checking
				errorOutput := execErr.Error() + "\n" + stderr

				// Check that all expected strings appear in the error output
				for _, expected := range tt.errorShouldContain {
					if !strings.Contains(strings.ToLower(errorOutput), strings.ToLower(expected)) {
						t.Errorf("%s: Error message should contain '%s'\nGot error: %v\nStderr: %s",
							tt.description, expected, execErr, stderr)
					}
				}
			}
		})
	}
}

// TestErrorMessageQuality tests the overall quality and formatting of error messages
func TestErrorMessageQuality(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-error-quality-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	t.Run("parse error shows line number", func(t *testing.T) {
		script := `x = 5
y = 10
z =
w = 20`
		scriptPath := filepath.Join(tmpDir, "line_number.gsh")
		if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()
		var execErr error
		stderr := captureStderr(func() {
			execErr = runGshScript(ctx, scriptPath, logger)
		})

		if execErr == nil {
			t.Error("Expected parse error")
			return
		}

		errorOutput := execErr.Error() + "\n" + stderr
		// Should mention line 3 where the error occurs
		if !strings.Contains(errorOutput, "3") && !strings.Contains(errorOutput, "line") {
			t.Errorf("Error should reference line number. Got: %v\nStderr: %s", execErr, stderr)
		}
	})

	t.Run("runtime error is distinguishable from parse error", func(t *testing.T) {
		parseErrorScript := `x = `
		parseErrorPath := filepath.Join(tmpDir, "parse_err.gsh")
		if err := os.WriteFile(parseErrorPath, []byte(parseErrorScript), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		runtimeErrorScript := `x = undefinedVariable`
		runtimeErrorPath := filepath.Join(tmpDir, "runtime_err.gsh")
		if err := os.WriteFile(runtimeErrorPath, []byte(runtimeErrorScript), 0644); err != nil {
			t.Fatalf("Failed to write test script: %v", err)
		}

		ctx := context.Background()

		// Parse error
		var parseErr error
		parseStderr := captureStderr(func() {
			parseErr = runGshScript(ctx, parseErrorPath, logger)
		})

		// Runtime error
		var runtimeErr error
		runtimeStderr := captureStderr(func() {
			runtimeErr = runGshScript(ctx, runtimeErrorPath, logger)
		})

		if parseErr == nil || runtimeErr == nil {
			t.Fatal("Expected both errors to occur")
		}

		parseOutput := parseErr.Error() + "\n" + parseStderr
		runtimeOutput := runtimeErr.Error() + "\n" + runtimeStderr

		// Parse error should mention "Parse error"
		if !strings.Contains(parseOutput, "Parse error") && !strings.Contains(parseOutput, "parse") {
			t.Errorf("Parse error should be clearly labeled. Got: %v\nStderr: %s", parseErr, parseStderr)
		}

		// Runtime error should mention "runtime"
		if !strings.Contains(runtimeOutput, "runtime") && !strings.Contains(runtimeOutput, "Runtime") {
			t.Errorf("Runtime error should be clearly labeled. Got: %v\nStderr: %s", runtimeErr, runtimeStderr)
		}
	})

	t.Run("file read error is clear", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "does_not_exist.gsh")

		ctx := context.Background()
		err := runGshScript(ctx, nonExistentPath, logger)

		if err == nil {
			t.Error("Expected file read error")
			return
		}

		errorMsg := err.Error()
		// Should mention file and that it failed to read
		if !strings.Contains(errorMsg, "read") && !strings.Contains(errorMsg, "file") {
			t.Errorf("File read error should be clear. Got: %v", err)
		}
	})
}
