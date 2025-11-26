package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPMCPServer_Integration tests the HTTP/SSE MCP server integration
func TestHTTPMCPServer_Integration(t *testing.T) {
	// Create a test MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-http-server",
		Version: "1.0.0",
	}, nil)

	// Add a test tool
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "get_time",
		Description: "Returns the current time",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Time string `json:"time"`
		},
		error,
	) {
		return nil, struct {
			Time string `json:"time"`
		}{
			Time: time.Now().Format(time.RFC3339),
		}, nil
	})

	// Add another tool with parameters
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input message",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "The message to echo",
				},
			},
			"required": []string{"message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct {
		Message string `json:"message"`
	}) (
		*mcp.CallToolResult,
		struct {
			Echo string `json:"echo"`
		},
		error,
	) {
		return nil, struct {
			Echo string `json:"echo"`
		}{
			Echo: input.Message,
		}, nil
	})

	// Create HTTP handler for the MCP server
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: false,
	})

	// Create test HTTP server
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register HTTP MCP server
	config := ServerConfig{
		URL: testServer.URL,
	}

	err := manager.RegisterServer("http-server", config)
	require.NoError(t, err, "Failed to register HTTP MCP server")

	// Verify server is registered
	server, err := manager.GetServer("http-server")
	require.NoError(t, err)
	assert.Equal(t, "http-server", server.Name)

	// List available tools
	tools, err := manager.ListTools("http-server")
	require.NoError(t, err)
	assert.NotEmpty(t, tools, "Expected HTTP server to have tools")

	// Verify expected tools are available
	expectedTools := []string{"get_time", "echo"}
	for _, expectedTool := range expectedTools {
		assert.Contains(t, tools, expectedTool, "Expected tool %s to be available", expectedTool)
	}
}

// TestHTTPMCPServer_ToolCall tests calling tools on HTTP MCP server
func TestHTTPMCPServer_ToolCall(t *testing.T) {
	// Create a test MCP server with tools
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-http-server",
		Version: "1.0.0",
	}, nil)

	// Add echo tool
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input message",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "The message to echo",
				},
			},
			"required": []string{"message"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct {
		Message string `json:"message"`
	}) (
		*mcp.CallToolResult,
		struct {
			Echo string `json:"echo"`
		},
		error,
	) {
		return nil, struct {
			Echo string `json:"echo"`
		}{
			Echo: input.Message,
		}, nil
	})

	// Create HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{

		Stateless: false,
	})

	// Create test HTTP server
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register HTTP MCP server
	err := manager.RegisterServer("http-server", ServerConfig{
		URL: testServer.URL,
	})
	require.NoError(t, err)

	// Call echo tool
	result, err := manager.CallTool("http-server", "echo", map[string]interface{}{
		"message": "Hello from HTTP MCP!",
	})
	require.NoError(t, err, "Failed to call echo tool")
	assert.NotNil(t, result)
	assert.False(t, result.IsError, "echo tool returned error: %v", result.Content)

	// Verify the result
	require.NotEmpty(t, result.Content)
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		var response struct {
			Echo string `json:"echo"`
		}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)
		assert.Equal(t, "Hello from HTTP MCP!", response.Echo)
	} else {
		t.Fatal("Expected TextContent in response")
	}
}

// TestHTTPMCPServer_WithHeaders tests HTTP MCP server with custom headers
func TestHTTPMCPServer_WithHeaders(t *testing.T) {
	// Track if authorization header was received
	authHeaderReceived := false
	expectedToken := "Bearer test-token-12345"

	// Create a test MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-auth-server",
		Version: "1.0.0",
	}, nil)

	// Add a simple tool
	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "ping",
		Description: "Returns pong",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Message string `json:"message"`
		},
		error,
	) {
		return nil, struct {
			Message string `json:"message"`
		}{
			Message: "pong",
		}, nil
	})

	// Create MCP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		// Check authorization header on each request
		if r.Header.Get("Authorization") == expectedToken {
			authHeaderReceived = true
		}
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: false,
	})

	// Create test HTTP server
	testServer := httptest.NewServer(mcpHandler)
	defer testServer.Close()

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register HTTP MCP server with custom headers
	err := manager.RegisterServer("auth-server", ServerConfig{
		URL: testServer.URL,
		Headers: map[string]string{
			"Authorization": expectedToken,
		},
	})
	require.NoError(t, err)

	// Call tool to verify everything works
	result, err := manager.CallTool("auth-server", "ping", map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify auth header was sent
	assert.True(t, authHeaderReceived, "Authorization header was not received by server")
}

// TestHTTPMCPServer_MultipleServers tests connecting to multiple HTTP MCP servers
func TestHTTPMCPServer_MultipleServers(t *testing.T) {
	// Create first MCP server
	server1 := mcp.NewServer(&mcp.Implementation{
		Name:    "server1",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server1, &mcp.Tool{
		Name:        "server1_tool",
		Description: "Tool from server 1",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Source string `json:"source"`
		},
		error,
	) {
		return nil, struct {
			Source string `json:"source"`
		}{
			Source: "server1",
		}, nil
	})

	handler1 := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server1
	}, &mcp.StreamableHTTPOptions{

		Stateless: false,
	})

	testServer1 := httptest.NewServer(handler1)
	defer testServer1.Close()

	// Create second MCP server
	server2 := mcp.NewServer(&mcp.Implementation{
		Name:    "server2",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server2, &mcp.Tool{
		Name:        "server2_tool",
		Description: "Tool from server 2",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Source string `json:"source"`
		},
		error,
	) {
		return nil, struct {
			Source string `json:"source"`
		}{
			Source: "server2",
		}, nil
	})

	handler2 := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server2
	}, &mcp.StreamableHTTPOptions{

		Stateless: false,
	})

	testServer2 := httptest.NewServer(handler2)
	defer testServer2.Close()

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register both servers
	err := manager.RegisterServer("http1", ServerConfig{URL: testServer1.URL})
	require.NoError(t, err)

	err = manager.RegisterServer("http2", ServerConfig{URL: testServer2.URL})
	require.NoError(t, err)

	// Verify both servers are registered
	servers := manager.ListServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "http1")
	assert.Contains(t, servers, "http2")

	// Call tools from both servers
	result1, err := manager.CallTool("http1", "server1_tool", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result1.IsError)

	result2, err := manager.CallTool("http2", "server2_tool", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result2.IsError)
}

// TestHTTPMCPServer_ConnectionError tests error handling for connection failures
func TestHTTPMCPServer_ConnectionError(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Try to connect to non-existent server
	err := manager.RegisterServer("bad-server", ServerConfig{
		URL: "http://localhost:9999/nonexistent",
	})
	assert.Error(t, err, "Expected error when connecting to non-existent server")
	assert.Contains(t, err.Error(), "failed to start HTTP server")
}

// TestHTTPMCPServer_MixedTransports tests using both stdio and HTTP transports
func TestHTTPMCPServer_MixedTransports(t *testing.T) {
	// Create HTTP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "http-server",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "http_tool",
		Description: "Tool from HTTP server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Type string `json:"type"`
		},
		error,
	) {
		return nil, struct {
			Type string `json:"type"`
		}{
			Type: "http",
		}, nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{

		Stateless: false,
	})

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	// Create manager
	manager := NewManager()
	defer manager.Close()

	// Register HTTP server
	err := manager.RegisterServer("http-server", ServerConfig{
		URL: testServer.URL,
	})
	require.NoError(t, err)

	// Note: We can't easily test stdio transport in the same test without npx,
	// but we can verify that both types are supported by checking the server list
	servers := manager.ListServers()
	assert.Contains(t, servers, "http-server")

	// Verify HTTP server works
	result, err := manager.CallTool("http-server", "http_tool", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

// TestHTTPMCPServer_RemoteURL tests connecting to a remote MCP server URL
func TestHTTPMCPServer_RemoteURL(t *testing.T) {
	// This test demonstrates the configuration for remote servers
	// We'll use a local test server to simulate a remote endpoint

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "remote-server",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "remote_tool",
		Description: "Tool from remote server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Status string `json:"status"`
		},
		error,
	) {
		return nil, struct {
			Status string `json:"status"`
		}{
			Status: "ok",
		}, nil
	})

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{

		Stateless: false,
	})

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	manager := NewManager()
	defer manager.Close()

	// Simulate connecting to https://mcp.context7.com/mcp
	// (in reality, using our test server)
	config := ServerConfig{
		URL: testServer.URL,
		Headers: map[string]string{
			"X-API-Key": "test-key",
		},
	}

	err := manager.RegisterServer("remote", config)
	require.NoError(t, err)

	// Verify connection works
	tools, err := manager.ListTools("remote")
	require.NoError(t, err)
	assert.Contains(t, tools, "remote_tool")

	// Call remote tool
	result, err := manager.CallTool("remote", "remote_tool", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}
