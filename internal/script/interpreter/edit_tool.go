package interpreter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteNativeEditFileTool performs a find-and-replace edit on a file.
func ExecuteNativeEditFileTool(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("edit_file tool requires 'file_path' argument as string")
	}
	if !filepath.IsAbs(filePath) {
		return "", fmt.Errorf("edit_file tool requires 'file_path' to be an absolute path, got: %s", filePath)
	}

	find, ok := args["find"].(string)
	if !ok {
		return "", fmt.Errorf("edit_file tool requires 'find' argument as string")
	}

	replace, ok := args["replace"].(string)
	if !ok {
		return "", fmt.Errorf("edit_file tool requires 'replace' argument as string")
	}

	startLine := 0
	endLine := 0

	if startLineVal, ok := args["start_line"]; ok {
		switch v := startLineVal.(type) {
		case float64:
			startLine = int(v)
		case int:
			startLine = v
		case int64:
			startLine = int(v)
		}
	}

	if endLineVal, ok := args["end_line"]; ok {
		switch v := endLineVal.(type) {
		case float64:
			endLine = int(v)
		case int:
			endLine = v
		case int64:
			endLine = int(v)
		}
	}

	result, err := ExecuteEdit(ctx, filePath, find, replace, startLine, endLine)
	if err != nil {
		return fmt.Sprintf(`{"success": false, "error": %q}`, err.Error()), nil
	}

	if !result.Success {
		return fmt.Sprintf(`{"success": false, "error": %q}`, result.Message), nil
	}

	return fmt.Sprintf(`{"success": true, "message": %q}`, result.Message), nil
}

// EditResult contains the result of an edit operation.
type EditResult struct {
	Success bool
	Message string
}

// ExecuteEdit performs a find-and-replace edit on a file.
// The find string must appear exactly once in the file (or within the specified line range).
// If startLine and endLine are provided (1-indexed, inclusive), the search is constrained to that range.
func ExecuteEdit(ctx context.Context, filePath, find, replace string, startLine, endLine int) (*EditResult, error) {
	absPath, err := resolveFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	lineEnding := "\n"
	if strings.Contains(fileContent, "\r\n") {
		lineEnding = "\r\n"
	} else if strings.Contains(fileContent, "\r") {
		lineEnding = "\r"
	}

	normalizedContent := strings.ReplaceAll(fileContent, "\r\n", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\r", "\n")
	lines := strings.Split(normalizedContent, "\n")
	totalLines := len(lines)

	if startLine > 0 || endLine > 0 {
		if startLine <= 0 {
			startLine = 1
		}
		if endLine <= 0 || endLine > totalLines {
			endLine = totalLines
		}
		if startLine > endLine {
			return &EditResult{
				Success: false,
				Message: fmt.Sprintf("invalid line range: start_line (%d) > end_line (%d)", startLine, endLine),
			}, nil
		}
		if startLine > totalLines {
			return &EditResult{
				Success: false,
				Message: fmt.Sprintf("start_line (%d) exceeds file length (%d lines)", startLine, totalLines),
			}, nil
		}
	}

	var newContent string
	var matchCount int

	if startLine > 0 && endLine > 0 {
		beforeRange := ""
		if startLine > 1 {
			beforeRange = strings.Join(lines[:startLine-1], "\n") + "\n"
		}

		rangeContent := strings.Join(lines[startLine-1:endLine], "\n")

		afterRange := ""
		if endLine < totalLines {
			afterRange = "\n" + strings.Join(lines[endLine:], "\n")
		}

		matchCount = strings.Count(rangeContent, find)

		if matchCount == 0 {
			return &EditResult{
				Success: false,
				Message: fmt.Sprintf("find string not found within lines %d-%d", startLine, endLine),
			}, nil
		}
		if matchCount > 1 {
			return &EditResult{
				Success: false,
				Message: fmt.Sprintf("find string appears %d times within lines %d-%d (must appear exactly once)", matchCount, startLine, endLine),
			}, nil
		}

		newRangeContent := strings.Replace(rangeContent, find, replace, 1)
		newContent = beforeRange + newRangeContent + afterRange
	} else {
		matchCount = strings.Count(normalizedContent, find)

		if matchCount == 0 {
			return &EditResult{
				Success: false,
				Message: "find string not found in file",
			}, nil
		}
		if matchCount > 1 {
			return &EditResult{
				Success: false,
				Message: fmt.Sprintf("find string appears %d times in file (must appear exactly once)", matchCount),
			}, nil
		}

		newContent = strings.Replace(normalizedContent, find, replace, 1)
	}

	if lineEnding != "\n" {
		newContent = strings.ReplaceAll(newContent, "\n", lineEnding)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	err = os.WriteFile(absPath, []byte(newContent), info.Mode())
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &EditResult{
		Success: true,
		Message: "edit applied successfully",
	}, nil
}

const editFileToolName = "edit_file"
const editFileToolDescription = "Perform a find-and-replace edit on a file. The find string must appear exactly once in the file (or within the specified line range). Use start_line and end_line to constrain the search to a specific range."

func editFileToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to edit",
			},
			"find": map[string]interface{}{
				"type":        "string",
				"description": "The exact string to find in the file. Must appear exactly once (or once within the line range if specified).",
			},
			"replace": map[string]interface{}{
				"type":        "string",
				"description": "The string to replace the find string with",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional 1-indexed start line to constrain the search (inclusive)",
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional 1-indexed end line to constrain the search (inclusive)",
			},
		},
		"required": []string{"file_path", "find", "replace"},
	}
}

// EditFileToolDefinition returns the ChatTool definition for the edit_file tool.
func EditFileToolDefinition() ChatTool {
	return ChatTool{
		Name:        editFileToolName,
		Description: editFileToolDescription,
		Parameters:  editFileToolParameters(),
	}
}

// CreateEditFileNativeTool creates the edit_file native tool for use in gsh.tools.
func CreateEditFileNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        editFileToolName,
		Description: editFileToolDescription,
		Parameters:  editFileToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeEditFileTool(context.Background(), args)
		},
	}
}
