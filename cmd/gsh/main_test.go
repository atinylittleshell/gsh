package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// newTestRunner creates a basic runner for testing gsh scripts.
func newTestRunner(t *testing.T) *interp.Runner {
	t.Helper()
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
	)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}
	return runner
}

// TestSyncRunnerEnvToOS tests that syncRunnerEnvToOS copies exported runner
// variables to the OS process environment.
func TestSyncRunnerEnvToOS(t *testing.T) {
	const envName = "GSH_TEST_RUNNER_SYNC"
	os.Unsetenv(envName)
	t.Cleanup(func() { os.Unsetenv(envName) })

	// Create a runner and export a variable via bash
	runner := newTestRunner(t)
	prog, err := syntax.NewParser().Parse(
		strings.NewReader("export "+envName+"=runner_value"), "")
	if err != nil {
		t.Fatalf("failed to parse export command: %v", err)
	}
	if err := runner.Run(context.Background(), prog); err != nil {
		t.Fatalf("failed to run export command: %v", err)
	}

	// Before sync, OS env should not have the variable
	if got := os.Getenv(envName); got == "runner_value" {
		t.Skip("variable unexpectedly already in os env")
	}

	syncRunnerEnvToOS(runner)

	got := os.Getenv(envName)
	if got != "runner_value" {
		t.Errorf("os.Getenv(%q) = %q after syncRunnerEnvToOS, want %q",
			envName, got, "runner_value")
	}
}

// TestSyncRunnerEnvToOS_NonExportedNotSynced tests that non-exported variables
// are not copied to the OS environment.
func TestSyncRunnerEnvToOS_NonExportedNotSynced(t *testing.T) {
	const envName = "GSH_TEST_RUNNER_NOEXPORT"
	os.Unsetenv(envName)
	t.Cleanup(func() { os.Unsetenv(envName) })

	runner := newTestRunner(t)
	// Set a non-exported variable
	prog, err := syntax.NewParser().Parse(
		strings.NewReader(envName+"=local_only"), "")
	if err != nil {
		t.Fatalf("failed to parse command: %v", err)
	}
	if err := runner.Run(context.Background(), prog); err != nil {
		t.Fatalf("failed to run command: %v", err)
	}

	syncRunnerEnvToOS(runner)

	got := os.Getenv(envName)
	if got != "" {
		t.Errorf("os.Getenv(%q) = %q, non-exported vars should not be synced", envName, got)
	}
}

// TestHelpText tests that the help text contains all essential information
func TestHelpText(t *testing.T) {
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		// Header and usage
		{"has title", "gsh -", "Should have descriptive title"},
		{"has usage section", "USAGE:", "Should have usage section"},
		{"has commands section", "COMMANDS:", "Should have commands section"},

		// Commands
		{"has run command", "run <script>", "Should document run command"},
		{"has telemetry command", "telemetry", "Should document telemetry command"},
		{"has login shell", "--login", "Should document login shell flag"},

		// Examples
		{"has examples section", "EXAMPLES:", "Should have examples section"},
		{"has gsh script example", "gsh run script.gsh", "Should show .gsh script execution"},
		{"has bash script example", "gsh run deploy.sh", "Should show bash script execution"},

		// Options section
		{"has options section", "OPTIONS:", "Should have options section header"},
		{"has repl-config option", "--repl-config", "Should document --repl-config flag"},
		{"has dash-c option", "-c <command>", "Should document -c flag"},
		{"has dash-c example", "gsh -c \"echo hello\"", "Should show -c example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(mainHelpText, tt.contains) {
				t.Errorf("%s: mainHelpText should contain %q", tt.desc, tt.contains)
			}
		})
	}
}

// TestParseREPLOptions_DashC tests -c flag parsing
func TestParseREPLOptions_DashC(t *testing.T) {
	t.Run("basic -c command", func(t *testing.T) {
		opts := parseREPLOptions([]string{"-c", "echo hello"})
		if opts.command != "echo hello" {
			t.Errorf("expected command %q, got %q", "echo hello", opts.command)
		}
		if opts.login {
			t.Error("login should be false")
		}
	})

	t.Run("-l -c command", func(t *testing.T) {
		opts := parseREPLOptions([]string{"-l", "-c", "echo hello"})
		if opts.command != "echo hello" {
			t.Errorf("expected command %q, got %q", "echo hello", opts.command)
		}
		if !opts.login {
			t.Error("login should be true")
		}
	})

	t.Run("-c -l order", func(t *testing.T) {
		opts := parseREPLOptions([]string{"-c", "echo hello", "-l"})
		if opts.command != "echo hello" {
			t.Errorf("expected command %q, got %q", "echo hello", opts.command)
		}
		if !opts.login {
			t.Error("login should be true even after -c")
		}
	})
}

// TestContainsHelpFlag_DashC tests that -c stops help flag scanning
func TestContainsHelpFlag_DashC(t *testing.T) {
	t.Run("--help before -c", func(t *testing.T) {
		if !containsHelpFlag([]string{"--help", "-c", "echo"}) {
			t.Error("should find --help before -c")
		}
	})

	t.Run("--help after -c is command not flag", func(t *testing.T) {
		if containsHelpFlag([]string{"-c", "--help"}) {
			t.Error("should not treat --help after -c as a flag")
		}
	})
}

// TestContainsVersionFlag_DashC tests that -c stops version flag scanning
func TestContainsVersionFlag_DashC(t *testing.T) {
	t.Run("--version before -c", func(t *testing.T) {
		if !containsVersionFlag([]string{"--version", "-c", "echo"}) {
			t.Error("should find --version before -c")
		}
	})

	t.Run("--version after -c is command not flag", func(t *testing.T) {
		if containsVersionFlag([]string{"-c", "--version"}) {
			t.Error("should not treat --version after -c as a flag")
		}
	})
}

// TestHelpTextStructure tests the overall structure and formatting of help text
func TestHelpTextStructure(t *testing.T) {
	t.Run("sections are in logical order", func(t *testing.T) {
		usageIdx := strings.Index(mainHelpText, "USAGE:")
		commandsIdx := strings.Index(mainHelpText, "COMMANDS:")
		optionsIdx := strings.Index(mainHelpText, "OPTIONS:")
		examplesIdx := strings.Index(mainHelpText, "EXAMPLES:")

		if usageIdx == -1 || commandsIdx == -1 || optionsIdx == -1 || examplesIdx == -1 {
			t.Fatal("Missing required sections")
		}

		if usageIdx > commandsIdx {
			t.Error("USAGE should come before COMMANDS")
		}
		if commandsIdx > optionsIdx {
			t.Error("COMMANDS should come before OPTIONS")
		}
		if optionsIdx > examplesIdx {
			t.Error("OPTIONS should come before EXAMPLES")
		}
	})

	t.Run("help text is not empty", func(t *testing.T) {
		if len(mainHelpText) < 100 {
			t.Error("Help text seems too short")
		}
	})

	t.Run("help text ends with newline", func(t *testing.T) {
		if !strings.HasSuffix(mainHelpText, "\n") {
			t.Error("Help text should end with newline")
		}
	})

	t.Run("no trailing whitespace issues", func(t *testing.T) {
		lines := strings.Split(mainHelpText, "\n")
		for i, line := range lines {
			if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
				t.Errorf("Line %d has trailing whitespace: %q", i+1, line)
			}
		}
	})

	t.Run("help text is concise", func(t *testing.T) {
		// Help text should be reasonably short - under 50 lines
		lines := strings.Split(mainHelpText, "\n")
		if len(lines) > 50 {
			t.Errorf("Help text should be concise, got %d lines", len(lines))
		}
	})
}

// TestRunHelpText tests the run subcommand help text
func TestRunHelpText(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{"has usage", "USAGE:"},
		{"has gsh run", "gsh run"},
		{"has script arg", "<script>"},
		{"has options", "OPTIONS:"},
		{"has help flag", "--help"},
		{"has examples", "EXAMPLES:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(runHelpText, tt.contains) {
				t.Errorf("runHelpText should contain %q", tt.contains)
			}
		})
	}
}

// TestTelemetryHelpText tests the telemetry subcommand help text
func TestTelemetryHelpText(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{"has usage", "USAGE:"},
		{"has status command", "status"},
		{"has on command", "on"},
		{"has off command", "off"},
		{"has env vars", "ENVIRONMENT VARIABLES:"},
		{"has what we collect", "WHAT WE COLLECT:"},
		{"has what we never collect", "WHAT WE NEVER COLLECT:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(telemetryHelpText, tt.contains) {
				t.Errorf("telemetryHelpText should contain %q", tt.contains)
			}
		})
	}
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
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
		err := runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
		if err == nil {
			t.Error("Expected error for syntax error, got nil")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "nonexistent.gsh")
		ctx := context.Background()
		err := runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
		err := runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

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
				err = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

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
			errorShouldContain: []string{"property", "not found", "number", "line 2", "column"},
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
				execErr = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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

// TestRunInteractiveShell tests the runInteractiveShell function
func TestRunInteractiveShell(t *testing.T) {
	t.Run("context cancellation exits cleanly", func(t *testing.T) {
		logger, err := zap.NewDevelopment()
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Create a test runner
		runner := newTestRunner(t)

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Should return immediately with context error
		// Pass zero time, nil tracker, and empty config path since we don't need telemetry for this test
		err = runInteractiveShell(ctx, logger, runner, time.Time{}, nil, "")
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})
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
	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

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
			execErr = runGshScript(ctx, scriptPath, logger, logLevel, newTestRunner(t))
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
			parseErr = runGshScript(ctx, parseErrorPath, logger, logLevel, newTestRunner(t))
		})

		// Runtime error
		var runtimeErr error
		runtimeStderr := captureStderr(func() {
			runtimeErr = runGshScript(ctx, runtimeErrorPath, logger, logLevel, newTestRunner(t))
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
		err := runGshScript(ctx, nonExistentPath, logger, logLevel, newTestRunner(t))

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
