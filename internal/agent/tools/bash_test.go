package tools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestBashToolDefinition(t *testing.T) {
	assert.Equal(t, openai.ToolType("function"), BashToolDefinition.Type)
	assert.Equal(t, "bash", BashToolDefinition.Function.Name)
	assert.Equal(
		t,
		`Run a single-line command in a bash shell.
* When invoking this tool, the contents of the "command" parameter does NOT need to be XML-escaped.
* Avoid combining multiple bash commands into one using "&&", ";" or multiple lines. Instead, run each command separately.
* State is persistent across command calls and discussions with the user.`,
		BashToolDefinition.Function.Description,
	)
	parameters, ok := BashToolDefinition.Function.Parameters.(*jsonschema.Definition)
	assert.True(t, ok, "Parameters should be of type *jsonschema.Definition")
	assert.Equal(t, jsonschema.DataType("object"), parameters.Type)
	assert.Equal(t, "A concise reason for why you need to run this command", parameters.Properties["reason"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["reason"].Type)
	assert.Equal(t, "The bash command to run", parameters.Properties["command"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["command"].Type)
	assert.Equal(t, []string{"reason", "command"}, parameters.Required)
}
func TestGenerateCommandRegex(t *testing.T) {
	// Test regular commands
	assert.Equal(t, "^ls.*", GenerateCommandRegex("ls -la /tmp"))
	assert.Equal(t, "^pwd.*", GenerateCommandRegex("pwd"))
	assert.Equal(t, "^cat.*", GenerateCommandRegex("cat file.txt"))

	// Test special commands with subcommands
	assert.Equal(t, "^git status.*", GenerateCommandRegex("git status"))
	assert.Equal(t, "^git commit.*", GenerateCommandRegex("git commit -m \"message\""))
	assert.Equal(t, "^npm install.*", GenerateCommandRegex("npm install package"))
	assert.Equal(t, "^npm run.*", GenerateCommandRegex("npm run test"))
	assert.Equal(t, "^yarn add.*", GenerateCommandRegex("yarn add package"))
	assert.Equal(t, "^docker run.*", GenerateCommandRegex("docker run image"))
	assert.Equal(t, "^kubectl get.*", GenerateCommandRegex("kubectl get pods"))

	// Test edge cases
	assert.Equal(t, "^$", GenerateCommandRegex(""))
}

func TestGenerateSpecificCommandRegex(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "single command",
			command:  "ls",
			expected: "^ls$",
		},
		{
			name:     "command with args",
			command:  "ls -la",
			expected: "^ls -la.*",
		},
		{
			name:     "command with complex args",
			command:  "awk -F'|' '{print $1}'",
			expected: "^awk -F'\\|' '\\{print \\$1\\}'.*",
		},
		{
			name:     "empty command",
			command:  "",
			expected: "^$",
		},
		{
			name:     "command with special chars",
			command:  "grep '[0-9]' file.txt",
			expected: "^grep '\\[0-9\\]' file\\.txt.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSpecificCommandRegex(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePreselectionPattern(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:     "single command",
			prefix:   "ls",
			expected: "^ls.*",
		},
		{
			name:     "git subcommand",
			prefix:   "git status",
			expected: "^git status.*",
		},
		{
			name:     "git with args",
			prefix:   "git commit -m message",
			expected: "^git commit.*", // Only includes subcommand for special commands
		},
		{
			name:     "npm subcommand",
			prefix:   "npm install",
			expected: "^npm install.*",
		},
		{
			name:     "regular command with args",
			prefix:   "awk -F'|' '{print $1}'",
			expected: "^awk -F'\\|' '\\{print \\$1\\}'.*",
		},
		{
			name:     "docker subcommand",
			prefix:   "docker run",
			expected: "^docker run.*",
		},
		{
			name:     "kubectl subcommand",
			prefix:   "kubectl get",
			expected: "^kubectl get.*",
		},
		{
			name:     "yarn subcommand",
			prefix:   "yarn add",
			expected: "^yarn add.*",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			expected: "^$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePreselectionPattern(tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBashToolParameterParsing(t *testing.T) {
	// Create test environment
	logger := zap.NewNop()
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	historyManager := &history.HistoryManager{}

	// Test missing reason parameter
	params := map[string]any{
		"command": "echo test",
	}
	result := BashTool(runner, historyManager, logger, params)
	assert.Contains(t, result, "failed to parse parameter 'reason'")

	// Test missing command parameter
	params = map[string]any{
		"reason": "test reason",
	}
	result = BashTool(runner, historyManager, logger, params)
	assert.Contains(t, result, "failed to parse parameter 'command'")

	// Test invalid command parameter type
	params = map[string]any{
		"reason":  "test reason",
		"command": 123, // Wrong type
	}
	result = BashTool(runner, historyManager, logger, params)
	assert.Contains(t, result, "failed to parse parameter 'command'")

	// Test invalid reason parameter type
	params = map[string]any{
		"reason":  123, // Wrong type
		"command": "echo test",
	}
	result = BashTool(runner, historyManager, logger, params)
	assert.Contains(t, result, "failed to parse parameter 'reason'")
}

func TestBashToolInvalidCommand(t *testing.T) {
	// Create test environment
	logger := zap.NewNop()
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	historyManager := &history.HistoryManager{}

	// Test syntactically invalid bash command
	params := map[string]any{
		"reason":  "test reason",
		"command": "if without fi", // Invalid bash syntax
	}
	result := BashTool(runner, historyManager, logger, params)
	assert.Contains(t, result, "is not a valid bash command")
}

func TestBashToolWithPreApprovedCommand(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_bash_preapproved")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tempConfigDir))
		environment.ResetCacheForTesting()
	})

	// Create authorized command
	err := os.MkdirAll(tempConfigDir, 0700)
	require.NoError(t, err)

	err = environment.AppendToAuthorizedCommands("^echo.*")
	require.NoError(t, err)

	// Create test environment
	logger := zap.NewNop()
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	// Create temporary database for testing
	tempDB, err := os.CreateTemp("", "test_history.db")
	require.NoError(t, err)
	tempDBPath := tempDB.Name()
	// Close the file handle immediately so the HistoryManager can open it exclusively
	require.NoError(t, tempDB.Close())

	historyManager, err := history.NewHistoryManager(tempDBPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Close the database connection before removing the file (required on Windows)
		_ = historyManager.Close()
		_ = os.Remove(tempDBPath)
	})

	// Test with pre-approved command - should execute without user confirmation
	params := map[string]any{
		"reason":  "testing echo",
		"command": "echo 'hello world'",
	}

	// Capture stdout to verify command execution
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	result := BashTool(runner, historyManager, logger, params)

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read captured output
	outBuf := &bytes.Buffer{}
	_, _ = outBuf.ReadFrom(r)
	require.NoError(t, r.Close())

	// Verify successful execution
	var response map[string]any
	err = json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(response["exitCode"].(float64)))
	assert.Contains(t, response["stdout"], "hello world")
}

func TestBashToolUserConfirmationFlow(t *testing.T) {
	// This test would require mocking the user confirmation dialog,
	// which is complex due to the interactive nature. We'll test the
	// parameter validation and pre-approval logic instead.

	// Create test environment
	logger := zap.NewNop()
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	historyManager := &history.HistoryManager{}

	// Valid parameters should not fail due to parameter parsing
	params := map[string]any{
		"reason":  "testing",
		"command": "echo test",
	}

	// This will likely fail at user confirmation since we can't mock it easily,
	// but it should not fail at parameter parsing
	result := BashTool(runner, historyManager, logger, params)

	// Should not contain parameter parsing errors
	assert.NotContains(t, result, "failed to parse parameter")
	assert.NotContains(t, result, "is not a valid bash command")
}

func TestGenerateCommandRegexWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "command with regex special chars",
			command:  "grep '[0-9]+' file.txt",
			expected: "^grep.*",
		},
		{
			name:     "command with parentheses",
			command:  "find . -name '*.txt' -exec echo {} \\;",
			expected: "^find.*",
		},
		{
			name:     "git with special chars in message",
			command:  "git commit -m 'fix: (urgent) resolve [bug] $variable'",
			expected: "^git commit.*",
		},
		{
			name:     "docker with complex args",
			command:  "docker run -e VAR=$HOME --rm image:tag",
			expected: "^docker run.*",
		},
		{
			name:     "whitespace handling",
			command:  "   ls   -la   /tmp   ",
			expected: "^ls.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCommandRegex(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateCommandRegexWithNewCommands(t *testing.T) {
	// Test that the heuristic-based approach works for commands
	// that weren't in the original hardcoded list
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "cargo build",
			command:  "cargo build --release",
			expected: "^cargo build.*",
		},
		{
			name:     "brew install",
			command:  "brew install package",
			expected: "^brew install.*",
		},
		{
			name:     "apt-get update",
			command:  "apt-get update",
			expected: "^apt-get update.*",
		},
		{
			name:     "systemctl start",
			command:  "systemctl start nginx",
			expected: "^systemctl start.*",
		},
		{
			name:     "go test",
			command:  "go test ./...",
			expected: "^go test.*",
		},
		{
			name:     "terraform apply",
			command:  "terraform apply -auto-approve",
			expected: "^terraform apply.*",
		},
		{
			name:     "helm install",
			command:  "helm install myapp ./chart",
			expected: "^helm install.*",
		},
		{
			name:     "podman run",
			command:  "podman run -d nginx",
			expected: "^podman run.*",
		},
		{
			name:     "command with flag first",
			command:  "ls -la /tmp",
			expected: "^ls.*", // Flag comes first, so no subcommand
		},
		{
			name:     "command with path argument",
			command:  "cd /usr/local/bin",
			expected: "^cd.*", // Path is not a subcommand
		},
		{
			name:     "command with numeric argument",
			command:  "sleep 5",
			expected: "^sleep.*", // Number is not a subcommand
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCommandRegex(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateCommandRegexEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "only whitespace",
			command:  "   \t\n   ",
			expected: "^$",
		},
		{
			name:     "special command without subcommand",
			command:  "git",
			expected: "^git.*",
		},
		{
			name:     "command with quotes",
			command:  "echo \"hello world\"",
			expected: "^echo.*",
		},
		{
			name:     "command starting with number",
			command:  "7z extract file.7z",
			expected: "^7z extract.*", // "extract" is correctly identified as a subcommand
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCommandRegex(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}
