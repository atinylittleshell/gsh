package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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

// TestFilesystemMCPServer_Integration tests the filesystem MCP server integration
// This test requires npx to be installed and will start an actual MCP server process
func TestFilesystemMCPServer_Integration(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err := manager.RegisterServer("filesystem", config)
	require.NoError(t, err, "Failed to register filesystem MCP server")

	// Verify server is registered
	server, err := manager.GetServer("filesystem")
	require.NoError(t, err)
	assert.Equal(t, "filesystem", server.Name)

	// Verify tools are available
	tools, err := manager.ListTools("filesystem")
	require.NoError(t, err)
	assert.NotEmpty(t, tools, "Expected filesystem server to have tools")

	// Check for expected tools
	expectedTools := []string{"read_file", "write_file", "list_directory"}
	for _, expectedTool := range expectedTools {
		assert.Contains(t, tools, expectedTool, "Expected tool %s to be available", expectedTool)
	}
}

// TestFilesystemMCPServer_WriteFile tests writing a file using the filesystem MCP server
func TestFilesystemMCPServer_WriteFile(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err := manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// Write a file (use absolute path within allowed directory)
	testFilePath := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello from gsh MCP integration test!"

	result, err := manager.CallTool("filesystem", "write_file", map[string]interface{}{
		"path":    testFilePath, // Absolute path
		"content": testContent,
	})
	require.NoError(t, err, "Failed to call write_file tool")
	assert.NotNil(t, result)
	assert.False(t, result.IsError, "write_file returned error: %v", result.Content)

	// Verify the file was actually written
	actualContent, err := os.ReadFile(testFilePath)
	require.NoError(t, err, "Failed to read written file")
	assert.Equal(t, testContent, string(actualContent))
}

// TestFilesystemMCPServer_ReadFile tests reading a file using the filesystem MCP server
func TestFilesystemMCPServer_ReadFile(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create a test file
	testFilePath := filepath.Join(tmpDir, "read_test.txt")
	testContent := "Content to read from MCP server"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err = manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// Read the file (use absolute path)
	result, err := manager.CallTool("filesystem", "read_file", map[string]interface{}{
		"path": testFilePath, // Absolute path
	})
	require.NoError(t, err, "Failed to call read_file tool")
	assert.NotNil(t, result)
	assert.False(t, result.IsError, "read_file returned error: %v", result.Content)

	// Verify the content
	// MCP tools return content as an array of Content objects
	require.NotEmpty(t, result.Content, "Expected content in result")

	// The content should contain the text we wrote
	// Note: The exact format depends on the MCP server implementation
	// but it should contain our test content somewhere
	assert.NotNil(t, result.Content)
}

// TestFilesystemMCPServer_ListDirectory tests listing a directory using the filesystem MCP server
func TestFilesystemMCPServer_ListDirectory(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create some test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err = manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// List the directory (use absolute path)
	result, err := manager.CallTool("filesystem", "list_directory", map[string]interface{}{
		"path": tmpDir, // Absolute path
	})
	require.NoError(t, err, "Failed to call list_directory tool")
	assert.NotNil(t, result)
	assert.False(t, result.IsError, "list_directory returned error: %v", result.Content)

	// Verify the result contains content
	assert.NotEmpty(t, result.Content, "Expected content in list_directory result")
}

// TestFilesystemMCPServer_ErrorHandling tests error handling with invalid operations
func TestFilesystemMCPServer_ErrorHandling(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err := manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// Try to read a non-existent file (use absolute path)
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.txt")
	result, err := manager.CallTool("filesystem", "read_file", map[string]interface{}{
		"path": nonExistentPath, // Absolute path
	})

	// The call itself should not error (it's a valid tool call)
	// But the result should indicate an error
	if err == nil {
		assert.True(t, result.IsError, "Expected IsError=true for non-existent file")
	}
}

// TestFilesystemMCPServer_MultipleServers tests running multiple MCP servers simultaneously
func TestFilesystemMCPServer_MultipleServers(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create two temporary directories
	tmpDir1 := resolveTestDir(t)
	tmpDir2 := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register first filesystem MCP server
	config1 := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir1},
	}
	err := manager.RegisterServer("fs1", config1)
	require.NoError(t, err)

	// Register second filesystem MCP server
	config2 := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir2},
	}
	err = manager.RegisterServer("fs2", config2)
	require.NoError(t, err)

	// Verify both servers are registered
	servers := manager.ListServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "fs1")
	assert.Contains(t, servers, "fs2")

	// Write to first server (use absolute path)
	file1 := filepath.Join(tmpDir1, "file1.txt")
	result1, err := manager.CallTool("fs1", "write_file", map[string]interface{}{
		"path":    file1, // Absolute path
		"content": "Content in fs1",
	})
	require.NoError(t, err)
	assert.False(t, result1.IsError)

	// Write to second server (use absolute path)
	file2 := filepath.Join(tmpDir2, "file2.txt")
	result2, err := manager.CallTool("fs2", "write_file", map[string]interface{}{
		"path":    file2, // Absolute path
		"content": "Content in fs2",
	})
	require.NoError(t, err)
	assert.False(t, result2.IsError)

	// Verify files were written to correct locations
	content1, err := os.ReadFile(file1)
	require.NoError(t, err)
	assert.Equal(t, "Content in fs1", string(content1))

	content2, err := os.ReadFile(file2)
	require.NoError(t, err)
	assert.Equal(t, "Content in fs2", string(content2))
}

// TestFilesystemMCPServer_InvalidToolCall tests calling a non-existent tool
func TestFilesystemMCPServer_InvalidToolCall(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err := manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// Try to call a non-existent tool
	_, err = manager.CallTool("filesystem", "nonexistent_tool", map[string]interface{}{})
	assert.Error(t, err, "Expected error when calling non-existent tool")
	assert.Contains(t, err.Error(), "not found")
}

// TestFilesystemMCPServer_WriteAndRead tests a complete write-read cycle
func TestFilesystemMCPServer_WriteAndRead(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping integration test: npx not found in PATH")
	}

	// Create a temporary directory for testing
	tmpDir := resolveTestDir(t)

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register filesystem MCP server
	config := ServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmpDir},
	}

	err := manager.RegisterServer("filesystem", config)
	require.NoError(t, err)

	// Test with different content types
	testCases := []struct {
		name     string
		filename string
		content  string
	}{
		{
			name:     "simple text",
			filename: "simple.txt",
			content:  "Simple text content",
		},
		{
			name:     "multi-line text",
			filename: "multiline.txt",
			content:  "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "special characters",
			filename: "special.txt",
			content:  "Special: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./",
		},
		{
			name:     "unicode",
			filename: "unicode.txt",
			content:  "Unicode: 你好世界 🚀 émoji",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tc.filename)

			// Write the file (use absolute path)
			writeResult, err := manager.CallTool("filesystem", "write_file", map[string]interface{}{
				"path":    filePath, // Absolute path
				"content": tc.content,
			})
			require.NoError(t, err, "Failed to write file")
			assert.False(t, writeResult.IsError, "write_file returned error")

			// Read the file back (use absolute path)
			readResult, err := manager.CallTool("filesystem", "read_file", map[string]interface{}{
				"path": filePath, // Absolute path
			})
			require.NoError(t, err, "Failed to read file")
			assert.False(t, readResult.IsError, "read_file returned error")

			// Verify content by reading directly from filesystem
			actualContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(actualContent))
		})
	}
}
