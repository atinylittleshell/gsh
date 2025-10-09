package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestCreateFileToolDefinition(t *testing.T) {
	assert.Equal(t, openai.ToolType("function"), CreateFileToolDefinition.Type)
	assert.Equal(t, "create_file", CreateFileToolDefinition.Function.Name)
	assert.Equal(
		t,
		"Create a file with the specified content.",
		CreateFileToolDefinition.Function.Description,
	)
	parameters, ok := CreateFileToolDefinition.Function.Parameters.(*jsonschema.Definition)
	assert.True(t, ok, "Parameters should be of type *jsonschema.Definition")
	assert.Equal(t, jsonschema.DataType("object"), parameters.Type)
	assert.Equal(t, "Absolute path to the file", parameters.Properties["path"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["path"].Type)
	assert.Equal(t, "The content to write to the file", parameters.Properties["content"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["content"].Type)
	assert.Equal(t, []string{"path", "content"}, parameters.Required)
}

func TestCreateFileToolParams(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	tests := []struct {
		name          string
		params        map[string]any
		expectedError bool
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"path":    "/test/path",
				"content": "test content",
			},
			expectedError: false,
		},
		{
			name: "missing path",
			params: map[string]any{
				"content": "test content",
			},
			expectedError: true,
		},
		{
			name: "missing content",
			params: map[string]any{
				"path": "/test/path",
			},
			expectedError: true,
		},
		{
			name: "invalid path type",
			params: map[string]any{
				"path":    123,
				"content": "test content",
			},
			expectedError: true,
		},
		{
			name: "invalid content type",
			params: map[string]any{
				"path":    "/test/path",
				"content": 123,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateFileTool(runner, logger, tt.params)
			if tt.expectedError {
				assert.Contains(t, result, "failed")
			} else {
				// Since we can't actually create files in this test, we expect it to fail at file creation
				assert.Contains(t, result, "Error creating")
			}
		})
	}
}

func TestCreateFile(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "gsh_create_file_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name          string
		path          string
		content       string
		expectedError bool
	}{
		{
			name:          "successful create",
			path:          tmpFile.Name(),
			content:       "test content",
			expectedError: false,
		},
		{
			name:          "invalid path",
			path:          "/nonexistent/directory/file.txt",
			content:       "test content",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{
				"path":    tt.path,
				"content": tt.content,
			}
			result := CreateFileTool(runner, logger, params)
			if tt.expectedError {
				assert.Contains(t, result, "Error")
			} else {
				assert.Contains(t, result, "successfully")
			}
		})
	}
}

func TestGenerateFileOperationRegex(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		operation string
		expected  string
	}{
		{
			name:      "file with extension in directory",
			filePath:  "/home/user/project/src/main.go",
			operation: "create_file",
			expected:  "create_file:/home/user/project/src/.*\\\\.go$",
		},
		{
			name:      "file with extension in root",
			filePath:  "/tmp/test.txt",
			operation: "create_file",
			expected:  "create_file:/tmp/.*\\\\.txt$",
		},
		{
			name:      "file without extension",
			filePath:  "/home/user/README",
			operation: "create_file",
			expected:  "create_file:/home/user/README$",
		},
		{
			name:      "file with multiple dots",
			filePath:  "/var/log/app.log.backup",
			operation: "create_file",
			expected:  "create_file:/var/log/.*\\\\.backup$",
		},
		{
			name:      "relative path with extension",
			filePath:  "./src/utils.js",
			operation: "create_file",
			expected:  "create_file:src/.*\\\\.js$",
		},
		{
			name:      "edit operation",
			filePath:  "/home/user/config.yaml",
			operation: "edit_file",
			expected:  "edit_file:/home/user/.*\\\\.yaml$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFileOperationRegex(tt.filePath, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateFileToolWithRelativePath(t *testing.T) {
	logger := zap.NewNop()

	// Create test environment
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	// Set PWD to temp directory
	tempDir, err := os.MkdirTemp("", "gsh_createfile_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}
	runner.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: tempDir}

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Test with relative path
	params := map[string]any{
		"path":    "test_file.txt",
		"content": "test content",
	}

	result := CreateFileTool(runner, logger, params)
	assert.Contains(t, result, "successfully")

	// Verify file was created in the right location
	expectedPath := filepath.Join(tempDir, "test_file.txt")
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCreateFileToolUserDeclines(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "n"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	params := map[string]any{
		"path":    "/test/path.txt",
		"content": "test content",
	}

	result := CreateFileTool(runner, logger, params)
	assert.Contains(t, result, "User declined this request")
}

func TestCreateFileToolManagePermissions(t *testing.T) {
	// Test that "manage" response is treated as invalid (declined) for createfile
	// The manage menu should only be available for bash commands, not file operations
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "manage"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	params := map[string]any{
		"path":    "/test/path.txt",
		"content": "test content for manage",
	}

	result := CreateFileTool(runner, logger, params)
	// "manage" should be treated as an invalid response and declined
	assert.Contains(t, result, "User declined this request: manage")
}

func TestCreateFileToolLegacyAlways(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_createfile_always")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the environment variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	defer func() {
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()
	}()

	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "always"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create temp file for testing
	tempFile, err := os.CreateTemp("", "gsh_test_createfile_always")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	params := map[string]any{
		"path":    tempFile.Name(),
		"content": "test content for always",
	}

	err = os.MkdirAll(tempConfigDir, 0700)
	require.NoError(t, err)

	result := CreateFileTool(runner, logger, params)
	assert.Contains(t, result, "successfully")

	// Verify pattern was added to authorized commands
	patterns, err := environment.LoadAuthorizedCommandsFromFile()
	assert.NoError(t, err)
	assert.Len(t, patterns, 1)

	expectedPattern := GenerateFileOperationRegex(tempFile.Name(), "create_file")
	assert.Contains(t, patterns, expectedPattern)
}

func TestCreateFileToolFreeformResponse(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "custom freeform response"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	params := map[string]any{
		"path":    "/test/path.txt",
		"content": "test content",
	}

	result := CreateFileTool(runner, logger, params)
	assert.Contains(t, result, "User declined this request: custom freeform response")
}

func TestCreateFileToolFileOperationErrors(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	tests := []struct {
		name        string
		path        string
		content     string
		expectError string
	}{
		{
			name:        "directory doesn't exist",
			path:        "/nonexistent/directory/file.txt",
			content:     "test content",
			expectError: "Error creating file",
		},
		{
			name:        "permission denied",
			path:        "/root/file.txt", // Typically no write permission
			content:     "test content",
			expectError: "Error creating file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if os.Geteuid() == 0 {
				t.Skip("Skipping permission test when running as root")
			}

			params := map[string]any{
				"path":    tt.path,
				"content": tt.content,
			}

			result := CreateFileTool(runner, logger, params)
			assert.Contains(t, result, tt.expectError)
		})
	}
}

func TestGenerateFileOperationRegexEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		operation string
		expected  string
	}{
		{
			name:      "file with special regex characters",
			filePath:  "/home/user/test[file].go",
			operation: "create_file",
			expected:  "create_file:/home/user/.*\\\\.go$",
		},
		{
			name:      "file with parentheses",
			filePath:  "/tmp/test(1).txt",
			operation: "create_file",
			expected:  "create_file:/tmp/.*\\\\.txt$",
		},
		{
			name:      "file with plus signs",
			filePath:  "/var/log/app++.log",
			operation: "create_file",
			expected:  "create_file:/var/log/.*\\\\.log$",
		},
		{
			name:      "file with question marks",
			filePath:  "/home/user/file?.txt",
			operation: "create_file",
			expected:  "create_file:/home/user/.*\\\\.txt$",
		},
		{
			name:      "file with asterisks",
			filePath:  "/tmp/file*.backup",
			operation: "create_file",
			expected:  "create_file:/tmp/.*\\\\.backup$",
		},
		{
			name:      "file with curly braces",
			filePath:  "/config/{env}.json",
			operation: "create_file",
			expected:  "create_file:/config/.*\\\\.json$",
		},
		{
			name:      "file without extension with special chars",
			filePath:  "/home/user/README[backup]",
			operation: "create_file",
			expected:  "create_file:/home/user/README\\[backup\\]$",
		},
		{
			name:      "dot file with extension",
			filePath:  "/home/user/.env.local",
			operation: "create_file",
			expected:  "create_file:/home/user/.*\\\\.local$",
		},
		{
			name:      "hidden file without extension",
			filePath:  "/home/user/.gitignore",
			operation: "create_file",
			expected:  "create_file:/home/user/.*\\\\.gitignore$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFileOperationRegex(tt.filePath, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateFileToolWithExistingFile(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create a temporary file with existing content
	tempFile, err := os.CreateTemp("", "gsh_test_existing")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some initial content
	initialContent := "initial content"
	err = os.WriteFile(tempFile.Name(), []byte(initialContent), 0644)
	require.NoError(t, err)

	// Test overwriting the file
	params := map[string]any{
		"path":    tempFile.Name(),
		"content": "new content",
	}

	result := CreateFileTool(runner, logger, params)
	assert.Contains(t, result, "successfully")

	// Verify the file was overwritten
	content, err := os.ReadFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestCreateFileToolContentVariations(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "multiline content",
			content: "line 1\nline 2\nline 3",
		},
		{
			name:    "content with special characters",
			content: "content with symbols: !@#$%^&*()[]{}|\\:;\"'<>?,.`~",
		},
		{
			name:    "unicode content",
			content: "unicode: 你好世界 🌍 café naïve résumé",
		},
		{
			name:    "json content",
			content: `{"key": "value", "number": 123, "array": [1, 2, 3]}`,
		},
		{
			name:    "code content",
			content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "gsh_test_content")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			params := map[string]any{
				"path":    tempFile.Name(),
				"content": tt.content,
			}

			result := CreateFileTool(runner, logger, params)
			assert.Contains(t, result, "successfully")

			// Verify content was written correctly
			writtenContent, err := os.ReadFile(tempFile.Name())
			assert.NoError(t, err)
			assert.Equal(t, tt.content, string(writtenContent))
		})
	}
}

func TestGenerateFileOperationRegexPathCleaning(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		operation string
		expected  string
	}{
		{
			name:      "path with redundant slashes",
			filePath:  "/home//user///project//file.go",
			operation: "create_file",
			expected:  "create_file:/home/user/project/.*\\\\.go$",
		},
		{
			name:      "path with dot segments",
			filePath:  "/home/user/./project/../project/file.go",
			operation: "create_file",
			expected:  "create_file:/home/user/project/.*\\\\.go$",
		},
		{
			name:      "relative path with dot segments",
			filePath:  "./src/../lib/utils.js",
			operation: "create_file",
			expected:  "create_file:lib/.*\\\\.js$",
		},
		{
			name:      "path ending with slash",
			filePath:  "/home/user/project/",
			operation: "create_file",
			expected:  "create_file:/home/user/project$", // No extension, so exact match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFileOperationRegex(tt.filePath, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}
