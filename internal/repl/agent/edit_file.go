// Package agent provides the edit tool for performing text edits on files.
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// EditResult contains the result of an edit operation.
type EditResult struct {
	Success bool
	Message string
}

// ExecuteEdit performs a find-and-replace edit on a file.
// The find string must appear exactly once in the file (or within the specified line range).
// If startLine and endLine are provided (1-indexed, inclusive), the search is constrained to that range.
func ExecuteEdit(ctx context.Context, filePath, find, replace string, startLine, endLine int) (*EditResult, error) {
	// Resolve file path (handle relative paths)
	absPath, err := resolveFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	// Detect original line ending style for preservation
	lineEnding := "\n"
	if strings.Contains(fileContent, "\r\n") {
		lineEnding = "\r\n"
	} else if strings.Contains(fileContent, "\r") {
		lineEnding = "\r" // Old Mac style
	}

	// Normalize to \n for processing
	normalizedContent := strings.ReplaceAll(fileContent, "\r\n", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\r", "\n")
	lines := strings.Split(normalizedContent, "\n")
	totalLines := len(lines)

	// Validate line range if provided
	if startLine > 0 || endLine > 0 {
		// Convert to 0-indexed for internal use
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
		// Search within specified line range
		// Build the content sections: before range, within range, after range
		beforeRange := ""
		if startLine > 1 {
			beforeRange = strings.Join(lines[:startLine-1], "\n") + "\n"
		}

		rangeContent := strings.Join(lines[startLine-1:endLine], "\n")

		afterRange := ""
		if endLine < totalLines {
			afterRange = "\n" + strings.Join(lines[endLine:], "\n")
		}

		// Count occurrences in the range
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

		// Perform replacement within the range
		newRangeContent := strings.Replace(rangeContent, find, replace, 1)
		newContent = beforeRange + newRangeContent + afterRange
	} else {
		// Search entire file (use normalized content for consistent matching)
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

		// Perform replacement
		newContent = strings.Replace(normalizedContent, find, replace, 1)
	}

	// Restore original line endings if needed
	if lineEnding != "\n" {
		newContent = strings.ReplaceAll(newContent, "\n", lineEnding)
	}

	// Write the modified content back to the file
	// Preserve the original file permissions
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

// resolveFilePath resolves a potentially relative file path to an absolute path.
func resolveFilePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Clean(filepath.Join(cwd, path)), nil
}

// EditToolDefinition returns the tool definition for the edit_file tool.
func EditToolDefinition() interpreter.ChatTool {
	return interpreter.ChatTool{
		Name:        "edit_file",
		Description: "Perform a find-and-replace edit on a file. The find string must appear exactly once in the file (or within the specified line range). Use start_line and end_line to constrain the search to a specific range.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "The path to the file to edit (can be relative or absolute)",
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
		},
	}
}

// ExecuteEditTool handles execution of the edit tool.
func ExecuteEditTool(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("edit tool requires 'file_path' argument as string")
	}

	find, ok := args["find"].(string)
	if !ok {
		return "", fmt.Errorf("edit tool requires 'find' argument as string")
	}

	replace, ok := args["replace"].(string)
	if !ok {
		return "", fmt.Errorf("edit tool requires 'replace' argument as string")
	}

	// Parse optional line range parameters
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
