package tools

import (
	"bytes"
	"os"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func TestViewFileToolDefinition(t *testing.T) {
	assert.Equal(t, openai.ToolType("function"), ViewFileToolDefinition.Type)
	assert.Equal(t, "view_file", ViewFileToolDefinition.Function.Name)
	assert.Equal(
		t,
		"View the content of a text file, at most 100 lines at a time. If the content is too large, tail will be truncated and replaced with <gsh:truncated />.",
		ViewFileToolDefinition.Function.Description,
	)
	parameters, ok := ViewFileToolDefinition.Function.Parameters.(*jsonschema.Definition)
	assert.True(t, ok, "Parameters should be of type *jsonschema.Definition")
	assert.Equal(t, jsonschema.DataType("object"), parameters.Type)
	assert.Equal(t, "Absolute path to the file", parameters.Properties["path"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["path"].Type)
	assert.Equal(
		t,
		"Optional. Line number to start viewing. The first line in the file has line number 1. If not provided, we will read from the beginning of the file.",
		parameters.Properties["start_line"].Description,
	)
	assert.Equal(t, jsonschema.DataType("integer"), parameters.Properties["start_line"].Type)
	assert.Equal(t, []string{"path"}, parameters.Required)
}

func TestViewFileTool(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testfile*.txt")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tempFile.Name()))
	})

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	_, err = tempFile.WriteString(content)
	assert.NoError(t, err)
	_ = tempFile.Close()

	runner, _ := interp.New()
	logger := zap.NewNop()

	t.Run("Valid file path with default start line", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name()}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "Line 1\nLine 2\nLine 3\nLine 4\nLine 5")
	})

	t.Run("Valid file path with specific start line", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "start_line": 3.0}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "Line 3\nLine 4\nLine 5")
	})

	t.Run("Invalid file path", func(t *testing.T) {
		params := map[string]any{"path": "nonexistent.txt"}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "Error opening file")
	})

	t.Run("Start line greater than number of lines", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "start_line": 10.0}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "start_line is greater than the number of lines in the file")
	})

	t.Run("Start line less than 1", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "start_line": 0.0}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "start_line must be greater than or equal to 1")
	})

	t.Run("File content exceeding max view size", func(t *testing.T) {
		largeContent := bytes.Repeat([]byte("A"), MAX_VIEW_SIZE+10)
		tempLargeFile, err := os.CreateTemp("", "largefile*.txt")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, os.Remove(tempLargeFile.Name()))
		})

		_, err = tempLargeFile.Write(largeContent)
		assert.NoError(t, err)
		_ = tempLargeFile.Close()

		params := map[string]any{"path": tempLargeFile.Name()}
		result := ViewFileTool(runner, logger, params)
		assert.Contains(t, result, "<gsh:truncated />")
	})
}
