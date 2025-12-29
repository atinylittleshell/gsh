// Package interpreter provides native tool implementations that can be shared
// between the SDK (gsh.tools) and the REPL agent.
package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// DefaultExecTimeout is the default timeout for command execution.
const DefaultExecTimeout = 60 * time.Second

// maxViewFileOutputLen is the maximum output size (100KB) before middle truncation is applied.
const maxViewFileOutputLen = 100000

// maxGrepOutputLen is the maximum grep output size (~50KB) before truncation.
const maxGrepOutputLen = 50000

// ExecuteNativeExecTool executes a shell command with PTY support.
// This is the shared implementation used by both gsh.tools.exec and the REPL agent.
func ExecuteNativeExecTool(ctx context.Context, args map[string]interface{}, liveOutput io.Writer) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("exec tool requires 'command' argument as string")
	}

	// Parse timeout (optional, defaults to DefaultExecTimeout)
	timeout := DefaultExecTimeout
	if timeoutVal, ok := args["timeout"]; ok {
		switch v := timeoutVal.(type) {
		case float64:
			timeout = time.Duration(v) * time.Second
		case int:
			timeout = time.Duration(v) * time.Second
		case int64:
			timeout = time.Duration(v) * time.Second
		}
	}

	// Create a timeout context
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with live output
	result, err := ExecuteCommandWithPTY(execCtx, command, liveOutput)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	// Truncate very long outputs to avoid overwhelming the model
	output := result.Output
	const maxOutputLen = 50000 // ~50KB limit
	truncated := false
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen]
		truncated = true
	}

	// Return result as JSON for the agent
	if truncated {
		return fmt.Sprintf(`{"output": %q, "exitCode": %d, "truncated": true}`, output, result.ExitCode), nil
	}
	return fmt.Sprintf(`{"output": %q, "exitCode": %d}`, output, result.ExitCode), nil
}

// ExecResult contains the result of executing a shell command.
type ExecResult struct {
	Output   string // Combined stdout/stderr (PTY combines them)
	ExitCode int
}

// ExecuteCommandWithPTY runs a shell command with PTY support.
// This allows live output display while also capturing the output.
// The liveOutput writer receives output in real-time as the command runs.
// If liveOutput is nil, output is only captured (not displayed).
func ExecuteCommandWithPTY(ctx context.Context, command string, liveOutput io.Writer) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// Set environment variables to disable interactive pagers and prompts
	cmd.Env = append(os.Environ(),
		"PAGER=cat",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
	)

	// Create PTY for the command
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Set PTY size
	cols, rows := 80, 24
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 && height > 0 {
		cols, rows = width, height
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})

	// Capture output while also writing to provided writer
	var outputBuf bytes.Buffer
	var mu sync.Mutex

	var writer io.Writer
	if liveOutput != nil {
		writer = io.MultiWriter(&safeWriter{w: liveOutput, mu: &mu}, &safeWriter{w: &outputBuf, mu: &mu})
	} else {
		writer = &outputBuf
	}

	// Copy PTY output in a goroutine
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		_, _ = io.Copy(writer, ptmx)
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for copy to finish
	<-copyDone

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		} else {
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	mu.Lock()
	output := outputBuf.String()
	mu.Unlock()

	return &ExecResult{
		Output:   output,
		ExitCode: exitCode,
	}, nil
}

// safeWriter wraps an io.Writer with mutex protection for thread safety.
type safeWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (sw *safeWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// GrepBackend represents the grep implementation to use.
type GrepBackend int

const (
	GrepBackendNone GrepBackend = iota
	GrepBackendRipgrep
	GrepBackendGitGrep
	GrepBackendGrep
)

// excludeDirs contains directories to exclude when using standard grep.
var excludeDirs = []string{
	".git", ".svn", ".hg",
	"node_modules", ".npm", ".yarn", ".pnpm-store", "bower_components",
	".venv", "venv", "env", ".env", "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", "*.egg-info", ".tox", ".nox",
	"vendor", "target",
	".gradle", ".mvn", "build", "bin", "obj",
	".bundle", "_build", "deps",
	"dist", "out", ".cache", ".parcel-cache", ".next", ".nuxt", ".output", ".turbo",
	"coverage", ".nyc_output", "htmlcov",
	".terraform", ".serverless",
}

// ExecuteNativeGrepTool searches for a regex pattern in files.
func ExecuteNativeGrepTool(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("grep tool requires 'pattern' argument as string")
	}

	if pattern == "" {
		return "", fmt.Errorf("grep tool requires non-empty 'pattern' argument")
	}

	result, err := ExecuteGrep(ctx, pattern)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	// Truncate very long outputs
	output := result.Output
	truncated := false
	if len(output) > maxGrepOutputLen {
		output = output[:maxGrepOutputLen]
		truncated = true
	}

	// Determine match status
	matchStatus := "matches_found"
	if result.ExitCode == 1 {
		matchStatus = "no_matches"
	} else if result.ExitCode > 1 {
		matchStatus = "error"
	}

	if truncated {
		return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q, "truncated": true}`,
			output, result.ExitCode, result.Backend, matchStatus), nil
	}
	return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q}`,
		output, result.ExitCode, result.Backend, matchStatus), nil
}

// GrepResult contains the result of a grep search.
type GrepResult struct {
	Output   string
	ExitCode int
	Backend  string
}

// ExecuteGrep runs a grep search with the detected backend.
func ExecuteGrep(ctx context.Context, pattern string) (*GrepResult, error) {
	backend := DetectGrepBackend()
	return ExecuteGrepWithBackend(ctx, pattern, backend)
}

// ExecuteGrepWithBackend runs a grep search with a specific backend.
func ExecuteGrepWithBackend(ctx context.Context, pattern string, backend GrepBackend) (*GrepResult, error) {
	cmdName, args, err := BuildGrepCommand(backend, pattern)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Env = os.Environ()

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		} else {
			return nil, fmt.Errorf("grep execution failed: %w", err)
		}
	}

	return &GrepResult{
		Output:   outputBuf.String(),
		ExitCode: exitCode,
		Backend:  GrepBackendName(backend),
	}, nil
}

// DetectGrepBackend determines which grep backend to use based on available tools.
// Priority: rg > git grep (if in git repo) > grep > none
func DetectGrepBackend() GrepBackend {
	if _, err := exec.LookPath("rg"); err == nil {
		return GrepBackendRipgrep
	}
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		if err := cmd.Run(); err == nil {
			return GrepBackendGitGrep
		}
	}
	if _, err := exec.LookPath("grep"); err == nil {
		return GrepBackendGrep
	}
	return GrepBackendNone
}

// BuildGrepCommand builds the appropriate grep command based on the backend and pattern.
func BuildGrepCommand(backend GrepBackend, pattern string) (string, []string, error) {
	switch backend {
	case GrepBackendRipgrep:
		return "rg", []string{"-n", "--hidden", "--color=never", "-e", pattern}, nil
	case GrepBackendGitGrep:
		return "git", []string{"grep", "-n", "--color=never", "--untracked", "-E", "-e", pattern}, nil
	case GrepBackendGrep:
		args := []string{"-rn", "--color=never", "-E"}
		for _, dir := range excludeDirs {
			args = append(args, "--exclude-dir="+dir)
		}
		args = append(args, "-e", pattern, ".")
		return "grep", args, nil
	case GrepBackendNone:
		return "", nil, fmt.Errorf("no grep tool available: install rg, git, or grep")
	default:
		return "", nil, fmt.Errorf("unknown grep backend: %d", backend)
	}
}

// GrepBackendName returns a human-readable name for the backend.
func GrepBackendName(backend GrepBackend) string {
	switch backend {
	case GrepBackendRipgrep:
		return "rg"
	case GrepBackendGitGrep:
		return "git-grep"
	case GrepBackendGrep:
		return "grep"
	default:
		return "none"
	}
}

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

// ExecuteNativeEditFileTool performs a find-and-replace edit on a file.
func ExecuteNativeEditFileTool(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("edit_file tool requires 'file_path' argument as string")
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

// execToolName, execToolDescription, and execToolParameters define the exec tool metadata.
const execToolName = "exec"
const execToolDescription = "Execute a bash command and return the output."

func execToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in seconds for the command execution. Defaults to 60 seconds if not specified.",
			},
		},
		"required": []string{"command"},
	}
}

// ExecToolDefinition returns the ChatTool definition for the exec tool.
func ExecToolDefinition() ChatTool {
	return ChatTool{
		Name:        execToolName,
		Description: execToolDescription,
		Parameters:  execToolParameters(),
	}
}

// CreateExecNativeTool creates the exec native tool for use in gsh.tools.
func CreateExecNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        execToolName,
		Description: execToolDescription,
		Parameters:  execToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeExecTool(context.Background(), args, nil)
		},
	}
}

// grepToolName, grepToolDescription, and grepToolParameters define the grep tool metadata.
const grepToolName = "grep"
const grepToolDescription = "Search for a regex pattern in files. Automatically uses the best available tool (ripgrep > git grep > grep). Returns matching lines with file names and line numbers."

func grepToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
		},
		"required": []string{"pattern"},
	}
}

// GrepToolDefinition returns the ChatTool definition for the grep tool.
func GrepToolDefinition() ChatTool {
	return ChatTool{
		Name:        grepToolName,
		Description: grepToolDescription,
		Parameters:  grepToolParameters(),
	}
}

// CreateGrepNativeTool creates the grep native tool for use in gsh.tools.
func CreateGrepNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        grepToolName,
		Description: grepToolDescription,
		Parameters:  grepToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeGrepTool(context.Background(), args)
		},
	}
}

// viewFileToolName, viewFileToolDescription, and viewFileToolParameters define the view_file tool metadata.
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

// editFileToolName, editFileToolDescription, and editFileToolParameters define the edit_file tool metadata.
const editFileToolName = "edit_file"
const editFileToolDescription = "Perform a find-and-replace edit on a file. The find string must appear exactly once in the file (or within the specified line range). Use start_line and end_line to constrain the search to a specific range."

func editFileToolParameters() map[string]interface{} {
	return map[string]interface{}{
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
