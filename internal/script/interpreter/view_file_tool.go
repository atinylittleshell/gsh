package interpreter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteNativeViewFileTool reads a file and returns its content with line numbers.
func ExecuteNativeViewFileTool(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("view_file tool requires 'file_path' argument as string")
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

	return ExecuteViewFile(ctx, filePath, startLine, endLine)
}

// ExecuteViewFile reads a file and returns its content with line numbers.
// Line numbers are 1-indexed and formatted as 5-digit prefixes (e.g., "    1:content").
// If startLine and endLine are provided (1-indexed, inclusive), only that range is returned.
// If the output exceeds 100KB, lines from the middle are truncated and replaced with "(truncated)".
func ExecuteViewFile(ctx context.Context, filePath string, startLine, endLine int) (string, error) {
	absPath, err := resolveFilePath(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	fileContent = strings.ReplaceAll(fileContent, "\r", "\n")
	lines := strings.Split(fileContent, "\n")
	totalLines := len(lines)

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

	outputLines := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		lineNum := fmt.Sprintf("%5d", i)
		lineContent := lines[i-1]
		outputLines = append(outputLines, fmt.Sprintf("%s:%s", lineNum, lineContent))
	}

	result := strings.Join(outputLines, "\n")

	if len(result) > maxViewFileOutputLen {
		result = TruncateFromMiddle(outputLines, maxViewFileOutputLen)
	}

	return result, nil
}

// TruncateFromMiddle removes lines from the middle of the output to fit within maxLen,
// replacing them with a "(truncated)" marker.
func TruncateFromMiddle(lines []string, maxLen int) string {
	if len(lines) == 0 {
		return ""
	}

	totalLen := 0
	for _, line := range lines {
		totalLen += len(line) + 1
	}
	totalLen--

	if totalLen <= maxLen {
		return strings.Join(lines, "\n")
	}

	truncationMarker := "(truncated)"
	markerLen := len(truncationMarker) + 2
	targetContentLen := maxLen - markerLen
	halfTarget := targetContentLen / 2

	startLines := 0
	startLen := 0
	for i := 0; i < len(lines); i++ {
		lineLen := len(lines[i])
		if i > 0 {
			lineLen++
		}
		if startLen+lineLen > halfTarget {
			break
		}
		startLen += lineLen
		startLines++
	}

	endLines := 0
	endLen := 0
	for i := len(lines) - 1; i >= 0; i-- {
		lineLen := len(lines[i])
		if endLines > 0 {
			lineLen++
		}
		if endLen+lineLen > halfTarget {
			break
		}
		endLen += lineLen
		endLines++
	}

	if startLines+endLines >= len(lines) {
		startLines = len(lines) / 2
		endLines = len(lines) - startLines - 1
		if endLines < 0 {
			endLines = 0
		}
	}

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

// resolveFilePath converts a file path to an absolute path.
func resolveFilePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Clean(filepath.Join(cwd, path)), nil
}

const viewFileToolName = "view_file"
const viewFileToolDescription = "View the contents of a file with line numbers. " +
	"Each line is prefixed with a 5-digit 1-indexed line number (e.g., '    1:content'). " +
	"Prefer reading the whole file without specifying start_line and end_line, " +
	"until you saw the file's too big and got truncated."

func viewFileToolParameters() map[string]interface{} {
	return map[string]interface{}{
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
	}
}

// ViewFileToolDefinition returns the ChatTool definition for the view_file tool.
func ViewFileToolDefinition() ChatTool {
	return ChatTool{
		Name:        viewFileToolName,
		Description: viewFileToolDescription,
		Parameters:  viewFileToolParameters(),
	}
}

// CreateViewFileNativeTool creates the view_file native tool for use in gsh.tools.
func CreateViewFileNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        viewFileToolName,
		Description: viewFileToolDescription,
		Parameters:  viewFileToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeViewFileTool(context.Background(), args)
		},
	}
}
