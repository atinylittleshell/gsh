package mcp

import (
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	require.NotNil(t, manager)
	assert.NotNil(t, manager.servers)
	assert.NotNil(t, manager.ctx)
	assert.NotNil(t, manager.cancel)
	assert.Empty(t, manager.servers)
}

func TestManagerRegisterServer_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		config      ServerConfig
		expectedErr string
	}{
		{
			name:        "missing command and URL",
			serverName:  "test",
			config:      ServerConfig{},
			expectedErr: "must specify either command or URL",
		},
		{
			name:       "duplicate server registration",
			serverName: "duplicate",
			config: ServerConfig{
				Command: "echo",
				Args:    []string{"test"},
			},
			expectedErr: "", // Will be tested separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			defer manager.Close()

			err := manager.RegisterServer(tt.serverName, tt.config)
			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestManagerRegisterServer_Duplicate(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := ServerConfig{
		Command: "echo",
		Args:    []string{"test"},
	}

	// First registration should fail because we can't actually connect to echo
	// But we're testing the duplicate check, so we manually add it
	manager.mu.Lock()
	manager.servers["test"] = &MCPServer{
		Name:   "test",
		Config: config,
		Tools:  make(map[string]*sdkmcp.Tool),
	}
	manager.mu.Unlock()

	// Second registration should fail with duplicate error
	err := manager.RegisterServer("test", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestManagerGetServer(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Test getting non-existent server
	_, err := manager.GetServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Add a mock server
	mockServer := &MCPServer{
		Name:  "test",
		Tools: make(map[string]*sdkmcp.Tool),
	}
	manager.mu.Lock()
	manager.servers["test"] = mockServer
	manager.mu.Unlock()

	// Test getting existing server
	server, err := manager.GetServer("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", server.Name)
}

func TestManagerGetTool(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Add a mock server with tools
	mockTool := &sdkmcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}
	mockServer := &MCPServer{
		Name: "test",
		Tools: map[string]*sdkmcp.Tool{
			"test_tool": mockTool,
		},
	}
	manager.mu.Lock()
	manager.servers["test"] = mockServer
	manager.mu.Unlock()

	// Test getting tool from non-existent server
	_, err := manager.GetTool("nonexistent", "test_tool")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting non-existent tool from existing server
	_, err = manager.GetTool("test", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test getting existing tool
	tool, err := manager.GetTool("test", "test_tool")
	assert.NoError(t, err)
	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "A test tool", tool.Description)
}

func TestManagerListServers(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Initially empty
	servers := manager.ListServers()
	assert.Empty(t, servers)

	// Add mock servers
	manager.mu.Lock()
	manager.servers["server1"] = &MCPServer{Name: "server1", Tools: make(map[string]*sdkmcp.Tool)}
	manager.servers["server2"] = &MCPServer{Name: "server2", Tools: make(map[string]*sdkmcp.Tool)}
	manager.mu.Unlock()

	// Should return all server names
	servers = manager.ListServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "server1")
	assert.Contains(t, servers, "server2")
}

func TestManagerListTools(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Test listing tools from non-existent server
	_, err := manager.ListTools("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Add a mock server with tools
	mockServer := &MCPServer{
		Name: "test",
		Tools: map[string]*sdkmcp.Tool{
			"tool1": {Name: "tool1"},
			"tool2": {Name: "tool2"},
			"tool3": {Name: "tool3"},
		},
	}
	manager.mu.Lock()
	manager.servers["test"] = mockServer
	manager.mu.Unlock()

	// Should return all tool names
	tools, err := manager.ListTools("test")
	assert.NoError(t, err)
	assert.Len(t, tools, 3)
	assert.Contains(t, tools, "tool1")
	assert.Contains(t, tools, "tool2")
	assert.Contains(t, tools, "tool3")
}

func TestManagerClose(t *testing.T) {
	manager := NewManager()

	// Add a mock server (without actual session to avoid connection issues)
	manager.mu.Lock()
	manager.servers["test"] = &MCPServer{
		Name:    "test",
		Session: nil, // No actual session
		Tools:   make(map[string]*sdkmcp.Tool),
	}
	manager.mu.Unlock()

	// Close should not error even with nil client
	err := manager.Close()
	assert.NoError(t, err)

	// Context should be cancelled
	select {
	case <-manager.ctx.Done():
		// Context is cancelled, as expected
	default:
		t.Error("Expected context to be cancelled after Close()")
	}
}

func TestServerConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
		valid  bool
	}{
		{
			name: "valid stdio config",
			config: ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
				Env:     map[string]string{"HOME": "/home/user"},
			},
			valid: true,
		},
		{
			name: "valid HTTP config",
			config: ServerConfig{
				URL:     "http://localhost:3000/mcp",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
			valid: true,
		},
		{
			name:   "invalid empty config",
			config: ServerConfig{},
			valid:  false,
		},
		{
			name: "valid command without args",
			config: ServerConfig{
				Command: "mcp-server",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			defer manager.Close()

			err := manager.RegisterServer("test", tt.config)
			if tt.valid {
				// We expect errors for valid configs because we can't actually connect
				// But the error should not be about validation
				if err != nil {
					assert.NotContains(t, err.Error(), "must specify either command or URL")
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must specify either command or URL")
			}
		})
	}
}

func TestManagerConcurrency(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Add initial servers
	manager.mu.Lock()
	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		manager.servers[name] = &MCPServer{
			Name:  name,
			Tools: make(map[string]*sdkmcp.Tool),
		}
	}
	manager.mu.Unlock()

	// Concurrent reads should work
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			servers := manager.ListServers()
			assert.Len(t, servers, 5)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMCPServer_ThreadSafety(t *testing.T) {
	server := &MCPServer{
		Name:  "test",
		Tools: make(map[string]*sdkmcp.Tool),
	}

	// Concurrent writes to tools
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			server.mu.Lock()
			toolName := string(rune('a' + idx))
			server.Tools[toolName] = &sdkmcp.Tool{Name: toolName}
			server.mu.Unlock()
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all tools were added
	server.mu.RLock()
	assert.Len(t, server.Tools, 10)
	server.mu.RUnlock()
}

func TestManagerHTTPTransport_NotImplemented(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := ServerConfig{
		URL:     "http://localhost:3000/mcp",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}

	err := manager.RegisterServer("http-server", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP/SSE transport not yet implemented")
}
