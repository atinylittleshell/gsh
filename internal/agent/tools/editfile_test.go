package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestValidateAndExtractParams(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	// Helper to normalize paths for the current OS
	p := func(path string) string {
		return filepath.FromSlash(path)
	}

	tests := []struct {
		name           string
		params         map[string]any
		expectedParams *editFileParams
		expectedError  string
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"path":    p("/test/path"),
				"old_str": "old content",
				"new_str": "new content",
			},
			expectedParams: &editFileParams{
				path:   p("/test/path"),
				oldStr: "old content",
				newStr: "new content",
			},
			expectedError: "",
		},
		{
			name: "missing path",
			params: map[string]any{
				"old_str": "old content",
				"new_str": "new content",
			},
			expectedParams: nil,
			expectedError:  "The edit_file tool failed to parse parameter 'path'",
		},
		{
			name: "missing old_str",
			params: map[string]any{
				"path":    p("/test/path"),
				"new_str": "new content",
			},
			expectedParams: nil,
			expectedError:  "The edit_file tool failed to parse parameter 'old_str'",
		},
		{
			name: "missing new_str",
			params: map[string]any{
				"path":    p("/test/path"),
				"old_str": "old content",
			},
			expectedParams: nil,
			expectedError:  "The edit_file tool failed to parse parameter 'new_str'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, errMsg := validateAndExtractParams(runner, logger, tt.params)
			assert.Equal(t, tt.expectedParams, params)
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}

func TestValidateAndReplaceContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		oldStr        string
		newStr        string
		expectedOut   string
		expectedError string
	}{
		{
			name:          "successful replacement",
			content:       "Hello world!",
			oldStr:        "world",
			newStr:        "there",
			expectedOut:   "Hello there!",
			expectedError: "",
		},
		{
			name:          "no matches",
			content:       "Hello world!",
			oldStr:        "foo",
			newStr:        "bar",
			expectedOut:   "",
			expectedError: "The old string must be unique in the file",
		},
		{
			name:          "multiple matches",
			content:       "Hello world world!",
			oldStr:        "world",
			newStr:        "there",
			expectedOut:   "",
			expectedError: "The old string must be unique in the file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newContent, errMsg := validateAndReplaceContent(tt.content, tt.oldStr, tt.newStr)
			assert.Equal(t, tt.expectedOut, newContent)
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}

type mockFileSystem struct {
	filesystem.FileSystem
	readFileError  error
	writeFileError error
	fileContent    string
}

func (m *mockFileSystem) ReadFile(path string) (string, error) {
	if m.readFileError != nil {
		return "", m.readFileError
	}
	return m.fileContent, nil
}

func (m *mockFileSystem) WriteFile(path string, content string) error {
	return m.writeFileError
}

func TestReadFileContents(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name          string
		fs            *mockFileSystem
		expectedOut   string
		expectedError string
	}{
		{
			name: "successful read",
			fs: &mockFileSystem{
				fileContent: "test content",
			},
			expectedOut:   "test content",
			expectedError: "",
		},
		{
			name: "read error",
			fs: &mockFileSystem{
				readFileError: assert.AnError,
			},
			expectedOut:   "",
			expectedError: "Error reading file: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, errMsg := readFileContents(logger, tt.fs, "/test/path")
			assert.Equal(t, tt.expectedOut, content)
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}

func TestWriteFile(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name          string
		fs            *mockFileSystem
		expectedError string
	}{
		{
			name: "successful write",
			fs: &mockFileSystem{
				writeFileError: nil,
			},
			expectedError: "",
		},
		{
			name: "write error",
			fs: &mockFileSystem{
				writeFileError: assert.AnError,
			},
			expectedError: "Error writing to file: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := writeFile(logger, tt.fs, "/test/path", "test content")
			assert.Equal(t, tt.expectedError, errMsg)
		})
	}
}

func TestPreviewAndConfirmUserDeclines(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	// Mock userConfirmation to return "n"
	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		return "n"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create a temporary file to simulate the existing file
	tempFile, err := os.CreateTemp("", "test_preview")
	assert.NoError(t, err)
	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempFile.Name()) }()

	// Write some content to it
	err = os.WriteFile(tempFile.Name(), []byte("original content"), 0644)
	assert.NoError(t, err)

	errMsg := previewAndConfirm(runner, logger, tempFile.Name(), "new content")
	assert.Equal(t, "User declined this request", errMsg)
}

func TestPreviewAndConfirmManageResponse(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	// Mock userConfirmation to return "manage"
	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		return "manage"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create a temporary file to simulate the existing file
	tempFile, err := os.CreateTemp("", "test_preview_manage")
	assert.NoError(t, err)
	_ = tempFile.Close()

	// Write some content to it
	err = os.WriteFile(tempFile.Name(), []byte("original content"), 0644)
	assert.NoError(t, err)

	errMsg := previewAndConfirm(runner, logger, tempFile.Name(), "new content")
	// "manage" is not a valid response for editfile operations, should return error
	assert.Equal(t, "User declined this request: manage", errMsg)
}

func TestPreviewAndConfirmLegacyAlways(t *testing.T) {
	// This test is no longer relevant since we removed the "always" feature
	t.Skip("Test skipped: 'always' feature has been removed")
}

func TestPreviewAndConfirmFreeformResponse(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	// Mock userConfirmation to return custom response
	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		return "custom response"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Create a temporary file to simulate the existing file
	tempFile, err := os.CreateTemp("", "test_preview_freeform")
	assert.NoError(t, err)
	_ = tempFile.Close()

	// Write some content to it
	err = os.WriteFile(tempFile.Name(), []byte("original content"), 0644)
	assert.NoError(t, err)

	errMsg := previewAndConfirm(runner, logger, tempFile.Name(), "new content")
	assert.Equal(t, "User declined this request: custom response", errMsg)
}

func TestEditFileToolIntegration(t *testing.T) {
	logger := zap.NewNop()
	runner, _ := interp.New()

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_edit_integration")
	assert.NoError(t, err)
	_ = tempFile.Close()

	originalContent := "Hello world!\nThis is a test file.\nEnd of file."
	err = os.WriteFile(tempFile.Name(), []byte(originalContent), 0644)
	assert.NoError(t, err)

	// Mock userConfirmation to return "y" (accept)
	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Test successful edit
	params := map[string]any{
		"path":    tempFile.Name(),
		"old_str": "world",
		"new_str": "universe",
	}

	result := EditFileTool(runner, logger, params)
	assert.Contains(t, result, "File successfully edited")

	// Verify the file was actually edited
	newContent, err := os.ReadFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(newContent), "Hello universe!")
	assert.NotContains(t, string(newContent), "Hello world!")
}

func TestEditFileToolWithRelativePath(t *testing.T) {
	logger := zap.NewNop()
	env := expand.ListEnviron(os.Environ()...)
	runner, _ := interp.New(interp.Env(env))
	runner.Vars = make(map[string]expand.Variable)

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_edit_relative")
	assert.NoError(t, err)
	_ = tempFile.Close() // Close the file handle

	originalContent := "Test content for relative path"
	err = os.WriteFile(tempFile.Name(), []byte(originalContent), 0644)
	assert.NoError(t, err)

	// Mock userConfirmation to return "y" (accept)
	origUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		return "y"
	}
	defer func() { userConfirmation = origUserConfirmation }()

	// Use relative path by using just the filename
	relativePath := filepath.Base(tempFile.Name())

	// Change to the temp directory
	tempDir := filepath.Dir(tempFile.Name())
	originalPwd := environment.GetPwd(runner)
	defer func() {
		// Restore original directory
		runner.Dir = originalPwd
		if originalPwd != "" {
			runner.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: originalPwd}
		}
	}()
	runner.Dir = tempDir
	runner.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: tempDir}

	params := map[string]any{
		"path":    relativePath,
		"old_str": "relative",
		"new_str": "absolute",
	}

	result := EditFileTool(runner, logger, params)
	assert.Contains(t, result, "File successfully edited")
}
