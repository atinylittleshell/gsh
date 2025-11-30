package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func TestViewDirectoryToolDefinition(t *testing.T) {
	assert.Equal(t, openai.ToolType("function"), ViewDirectoryToolDefinition.Type)
	assert.Equal(t, "view_directory", ViewDirectoryToolDefinition.Function.Name)
	assert.Equal(
		t,
		"View the content in a directory up to 2 levels deep.",
		ViewDirectoryToolDefinition.Function.Description,
	)
	parameters, ok := ViewDirectoryToolDefinition.Function.Parameters.(*jsonschema.Definition)
	assert.True(t, ok, "Parameters should be of type *jsonschema.Definition")
	assert.Equal(t, jsonschema.DataType("object"), parameters.Type)
	assert.Equal(t, "Absolute path to the directory", parameters.Properties["path"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["path"].Type)
	assert.Equal(t, []string{"path"}, parameters.Required)
}

func TestViewDirectoryTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testdir*")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	})

	// Create nested directories and files
	nestedDir := filepath.Join(tempDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	assert.NoError(t, err)

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(nestedDir, "file2.txt")
	f1, err := os.Create(file1)
	assert.NoError(t, err)
	_ = f1.Close()
	f2, err := os.Create(file2)
	assert.NoError(t, err)
	_ = f2.Close()

	runner, _ := interp.New()
	logger := zap.NewNop()

	t.Run("Valid directory path", func(t *testing.T) {
		params := map[string]any{"path": tempDir}
		result := ViewDirectoryTool(runner, logger, params)
		assert.Contains(t, result, "file1.txt")

		// Check for nested directory with path separator
		// On Unix it will be nested/, on Windows it might be nested\
		assert.True(t,
			strings.Contains(result, "nested/") || strings.Contains(result, "nested\\"),
			"Result should contain nested directory")
	})

	t.Run("Directory path with nested directories", func(t *testing.T) {
		params := map[string]any{"path": tempDir}
		result := ViewDirectoryTool(runner, logger, params)
		assert.Contains(t, result, "file2.txt")
	})

	t.Run("Invalid directory path", func(t *testing.T) {
		params := map[string]any{"path": "nonexistent_dir"}
		result := ViewDirectoryTool(runner, logger, params)
		assert.Contains(t, result, "Error reading directory")
	})

	t.Run("Directory with no content", func(t *testing.T) {
		emptyDir, err := os.MkdirTemp("", "emptydir*")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, os.RemoveAll(emptyDir))
		})

		params := map[string]any{"path": emptyDir}
		result := ViewDirectoryTool(runner, logger, params)
		assert.NotContains(t, result, "file")
	})
}
