package interpreter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resolveTestDir creates a temp directory and resolves symlinks
// This is important on macOS where /tmp is a symlink to /private/tmp
func resolveTestDir(t *testing.T) string {
	tmpDir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	return resolved
}

// TestFilesystemMCPIntegration_WriteFile tests writing files through the interpreter
func TestFilesystemMCPIntegration_WriteFile(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)
	testFile := filepath.Join(tmpDir, "test.txt")

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

filesystem.write_file({
	path: "` + testFile + `",
	content: "Hello from gsh!"
})
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, result)

	// Verify the file was written
	content, err := os.ReadFile(testFile)
	require.NoError(t, err, "Failed to read written file")
	assert.Equal(t, "Hello from gsh!", string(content))
}

// TestFilesystemMCPIntegration_ReadFile tests reading files through the interpreter
func TestFilesystemMCPIntegration_ReadFile(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory and file
	tmpDir := resolveTestDir(t)
	testFile := filepath.Join(tmpDir, "read_test.txt")
	testContent := "Content to read"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

result = filesystem.read_file({
	path: "` + testFile + `"
})
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	evalResult, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, evalResult)

	// Check that result variable exists
	resultVal, ok := interp.env.Get("result")
	require.True(t, ok, "result variable not found")
	assert.NotNil(t, resultVal)
}

// TestFilesystemMCPIntegration_WriteAndReadWithVariables tests a complete write-read cycle with variables
func TestFilesystemMCPIntegration_WriteAndReadWithVariables(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory
	tmpDir := resolveTestDir(t)
	testFile := filepath.Join(tmpDir, "variable_test.txt")

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

message = "Hello from gsh with variables!"
filepath = "` + testFile + `"

filesystem.write_file({
	path: filepath,
	content: message
})

readResult = filesystem.read_file({
	path: filepath
})
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, result)

	// Verify the file was written correctly
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Hello from gsh with variables!", string(content))

	// Verify variables exist
	message, ok := interp.env.Get("message")
	require.True(t, ok)
	assert.Equal(t, "Hello from gsh with variables!", message.(*StringValue).Value)

	readResult, ok := interp.env.Get("readResult")
	require.True(t, ok)
	assert.NotNil(t, readResult)
}

// TestFilesystemMCPIntegration_ListDirectory tests listing directories through the interpreter
func TestFilesystemMCPIntegration_ListDirectory(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory with files
	tmpDir := resolveTestDir(t)
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		err := os.WriteFile(filename, []byte("test"), 0644)
		require.NoError(t, err)
	}

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

listing = filesystem.list_directory({
	path: "` + tmpDir + `"
})
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, result)

	// Verify listing variable exists
	listing, ok := interp.env.Get("listing")
	require.True(t, ok, "listing variable not found")
	assert.NotNil(t, listing)
}

// TestFilesystemMCPIntegration_ErrorHandling tests error handling with MCP tools
func TestFilesystemMCPIntegration_ErrorHandling(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory
	tmpDir := resolveTestDir(t)
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

success = true
errorMsg = ""

try {
	result = filesystem.read_file({
		path: "` + nonExistentFile + `"
	})
} catch (error) {
	success = false
	errorMsg = error.message
}
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation should not error (error should be caught)")
	assert.NotNil(t, result)

	// Verify the error was caught
	success, hasSuccess := result.Env.Get("success")
	require.True(t, hasSuccess, "success variable should be set")

	successBool, ok := success.(*BoolValue)
	require.True(t, ok, "success should be a boolean")
	assert.False(t, successBool.Value, "Expected success=false due to error")

	// Verify error message was captured
	errorMsg, hasError := result.Env.Get("errorMsg")
	require.True(t, hasError, "errorMsg should be set")

	errorMsgStr, ok := errorMsg.(*StringValue)
	require.True(t, ok, "errorMsg should be a string")
	assert.Contains(t, errorMsgStr.Value, "ENOENT", "Error message should mention file not found")
}

// TestFilesystemMCPIntegration_ToolInLoop tests calling MCP tools in a loop
func TestFilesystemMCPIntegration_ToolInLoop(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory
	tmpDir := resolveTestDir(t)

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

files = ["file1.txt", "file2.txt", "file3.txt"]
count = 0

for (filename of files) {
	filesystem.write_file({
		path: "` + tmpDir + `/" + filename,
		content: "Content of " + filename
	})
	count = count + 1
}
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, result)

	// Verify count
	count, ok := interp.env.Get("count")
	require.True(t, ok)
	assert.Equal(t, float64(3), count.(*NumberValue).Value)

	// Verify files were created
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		content, err := os.ReadFile(filename)
		require.NoError(t, err, "File should exist: %s", filename)
		expected := "Content of file" + string(rune('0'+i)) + ".txt"
		assert.Equal(t, expected, string(content))
	}
}

// TestFilesystemMCPIntegration_ToolDeclaration tests declaring a tool that uses MCP
func TestFilesystemMCPIntegration_ToolDeclaration(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory
	tmpDir := resolveTestDir(t)
	testFile := filepath.Join(tmpDir, "tool_test.txt")

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

tool saveMessage(msg: string, path: string) {
	filesystem.write_file({
		path: path,
		content: msg
	})
	return "Saved: " + msg
}

result = saveMessage("Hello from tool!", "` + testFile + `")
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	evalResult, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, evalResult)

	// Verify the file was written
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Hello from tool!", string(content))

	// Verify the return value
	result, ok := interp.env.Get("result")
	require.True(t, ok)
	assert.Equal(t, "Saved: Hello from tool!", result.(*StringValue).Value)
}

// TestFilesystemMCPIntegration_MultipleServers tests using multiple MCP servers
func TestFilesystemMCPIntegration_MultipleServers(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create two temporary directories
	tmpDir1 := resolveTestDir(t)
	tmpDir2 := resolveTestDir(t)
	file1 := filepath.Join(tmpDir1, "file1.txt")
	file2 := filepath.Join(tmpDir2, "file2.txt")

	input := `
mcp fs1 {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir1 + `"],
}

mcp fs2 {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir2 + `"],
}

fs1.write_file({
	path: "` + file1 + `",
	content: "Content in fs1"
})

fs2.write_file({
	path: "` + file2 + `",
	content: "Content in fs2"
})
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, result)

	// Verify both files were written
	content1, err := os.ReadFile(file1)
	require.NoError(t, err)
	assert.Equal(t, "Content in fs1", string(content1))

	content2, err := os.ReadFile(file2)
	require.NoError(t, err)
	assert.Equal(t, "Content in fs2", string(content2))
}

// TestFilesystemMCPIntegration_ComplexScript tests a more complex script with MCP
func TestFilesystemMCPIntegration_ComplexScript(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory
	tmpDir := resolveTestDir(t)

	input := `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "` + tmpDir + `"],
}

tool writeLog(message: string, level: string): string {
	timestamp = "2024-01-01"
	logMessage = "[" + timestamp + "] " + level + ": " + message
	
	logFile = "` + tmpDir + `/app.log"
	
	try {
		filesystem.write_file({
			path: logFile,
			content: logMessage
		})
		return "Logged: " + message
	} catch (error) {
		return "Failed to log: " + error.message
	}
}

result1 = writeLog("Application started", "INFO")
result2 = writeLog("Processing data", "DEBUG")
result3 = writeLog("Task completed", "INFO")
`

	// Parse and evaluate
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "Parse errors: %v", p.Errors())

	interp := New()
	defer interp.Close()

	evalResult, err := interp.Eval(program)
	require.NoError(t, err, "Evaluation error")
	assert.NotNil(t, evalResult)

	// Verify results
	for i := 1; i <= 3; i++ {
		varName := "result" + string(rune('0'+i))
		result, ok := interp.env.Get(varName)
		require.True(t, ok, "%s should exist", varName)
		assert.True(t, strings.HasPrefix(result.(*StringValue).Value, "Logged:"), "Result should start with 'Logged:'")
	}

	// Verify the log file exists (will have the last write)
	logFile := filepath.Join(tmpDir, "app.log")
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "INFO: Task completed")
}
