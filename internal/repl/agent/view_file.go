// Package agent provides the view_file tool for viewing file contents with line numbers.
package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// maxOutputLen is the maximum output size (100KB) before middle truncation is applied.
const maxOutputLen = 100000

// ExecuteViewFile reads a file and returns its content with line numbers.
// Line numbers are 1-indexed and formatted as 5-digit prefixes (e.g., "    1:content").
// If startLine and endLine are provided (1-indexed, inclusive), only that range is returned.
// If the output exceeds 100KB, lines from the middle are truncated and replaced with "(truncated)".
func ExecuteViewFile(ctx context.Context, filePath string, startLine, endLine int) (string, error) {
	// Resolve file path (handle relative paths)
	absPath, err := resolveFilePath(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Normalize line endings (handle both \n and \r\n)
	fileContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	fileContent = strings.ReplaceAll(fileContent, "\r", "\n") // Handle standalone \r (old Mac style)
	lines := strings.Split(fileContent, "\n")
	totalLines := len(lines)

	// Validate and adjust line range
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 || endLine > totalLines {
		endLine = totalLines
	}
	if startLine > totalLines {
		return "", fmt.Errorf("start_line (%d) exceeds file length (%d lines)", startLine, totalLines)
	}
	if startLine > endLine {
		return "", fmt.Errorf("invalid line range: start_line (%d) > end_line (%d)", startLine, endLine)
	}

	// Build output with line numbers
	outputLines := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		// Format line number as 5-digit right-aligned prefix
		lineNum := fmt.Sprintf("%5d", i)
		lineContent := lines[i-1] // Convert to 0-indexed
		outputLines = append(outputLines, fmt.Sprintf("%s:%s", lineNum, lineContent))
	}

	result := strings.Join(outputLines, "\n")

	// If output exceeds maxOutputLen, truncate from the middle
	if len(result) > maxOutputLen {
		result = truncateFromMiddle(outputLines, maxOutputLen)
	}

	return result, nil
}

// truncateFromMiddle removes lines from the middle of the output to fit within maxLen,
// replacing them with a "(truncated)" marker.
func truncateFromMiddle(lines []string, maxLen int) string {
	if len(lines) == 0 {
		return ""
	}

	// Calculate total size
	totalLen := 0
	for _, line := range lines {
		totalLen += len(line) + 1 // +1 for newline
	}
	totalLen-- // No newline after last line

	if totalLen <= maxLen {
		return strings.Join(lines, "\n")
	}

	// Binary search to find how many lines we can keep from start and end
	// We want roughly equal parts from start and end
	truncationMarker := "(truncated)"
	markerLen := len(truncationMarker) + 2 // +2 for newlines around it

	// Target size for content (excluding marker)
	targetContentLen := maxLen - markerLen
	halfTarget := targetContentLen / 2

	// Find lines to keep from start
	startLines := 0
	startLen := 0
	for i := 0; i < len(lines); i++ {
		lineLen := len(lines[i])
		if i > 0 {
			lineLen++ // Account for newline before this line
		}
		if startLen+lineLen > halfTarget {
			break
		}
		startLen += lineLen
		startLines++
	}

	// Find lines to keep from end
	endLines := 0
	endLen := 0
	for i := len(lines) - 1; i >= 0; i-- {
		lineLen := len(lines[i])
		if endLines > 0 {
			lineLen++ // Account for newline before this line
		}
		if endLen+lineLen > halfTarget {
			break
		}
		endLen += lineLen
		endLines++
	}

	// Ensure we don't overlap
	if startLines+endLines >= len(lines) {
		// Not much to truncate, just take what we can
		startLines = len(lines) / 2
		endLines = len(lines) - startLines - 1
		if endLines < 0 {
			endLines = 0
		}
	}

	// Build result
	var builder strings.Builder
	for i := 0; i < startLines; i++ {
		builder.WriteString(lines[i])
		builder.WriteString("\n")
	}
	builder.WriteString(truncationMarker)
	if endLines > 0 {
		builder.WriteString("\n")
		endStart := len(lines) - endLines
		for i := endStart; i < len(lines); i++ {
			builder.WriteString(lines[i])
			if i < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String()
}

// ViewFileToolDefinition returns the tool definition for the view_file tool.
func ViewFileToolDefinition() interpreter.ChatTool {
	return interpreter.ChatTool{
		Name:        "view_file",
		Description: "View the contents of a file with line numbers. Each line is prefixed with a 5-digit 1-indexed line number (e.g., '    1:content'). Use start_line and end_line to view a specific range.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "The path to the file to view (can be relative or absolute)",
				},
				"start_line": map[string]interface{}{
					"type":        "integer",
					"description": "Optional 1-indexed start line to begin viewing (inclusive). Defaults to 1.",
				},
				"end_line": map[string]interface{}{
					"type":        "integer",
					"description": "Optional 1-indexed end line to stop viewing (inclusive). Defaults to end of file.",
				},
			},
			"required": []string{"file_path"},
		},
	}
}

// ExecuteViewFileTool handles execution of the view_file tool.
func ExecuteViewFileTool(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("view_file tool requires 'file_path' argument as string")
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

	// ExecuteViewFile handles truncation internally
	return ExecuteViewFile(ctx, filePath, startLine, endLine)
}
