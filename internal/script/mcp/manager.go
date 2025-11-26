package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig represents the configuration for an MCP server
type ServerConfig struct {
	// For stdio transport (local process)
	Command string            // Command to execute (e.g., "npx")
	Args    []string          // Command arguments
	Env     map[string]string // Environment variables

	// For HTTP/SSE transport (remote server)
	URL     string            // Server URL for remote connections
	Headers map[string]string // HTTP headers for authentication
}

// MCPServer represents a running MCP server instance
type MCPServer struct {
	Name    string
	Config  ServerConfig
	Session *mcp.ClientSession
	Tools   map[string]*mcp.Tool // Available tools from this server
	mu      sync.RWMutex
}

// Manager manages multiple MCP servers
type Manager struct {
	servers map[string]*MCPServer
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new MCP manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		servers: make(map[string]*MCPServer),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// RegisterServer registers and starts an MCP server
func (m *Manager) RegisterServer(name string, config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server already exists
	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("MCP server '%s' already registered", name)
	}

	// Validate config
	if config.Command == "" && config.URL == "" {
		return fmt.Errorf("MCP server '%s' must specify either command or URL", name)
	}

	// Create server instance
	server := &MCPServer{
		Name:   name,
		Config: config,
		Tools:  make(map[string]*mcp.Tool),
	}

	// Start the server based on transport type
	if config.Command != "" {
		// Stdio transport
		if err := m.startStdioServer(server); err != nil {
			return fmt.Errorf("failed to start stdio server '%s': %w", name, err)
		}
	} else {
		// HTTP/SSE transport
		return fmt.Errorf("HTTP/SSE transport not yet implemented for server '%s'", name)
	}

	m.servers[name] = server
	return nil
}

// startStdioServer starts an MCP server using stdio transport
func (m *Manager) startStdioServer(server *MCPServer) error {
	// Create command with arguments
	cmd := exec.Command(server.Config.Command, server.Config.Args...)

	// Set environment variables
	if len(server.Config.Env) > 0 {
		env := make([]string, 0, len(server.Config.Env))
		for k, v := range server.Config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Create client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "gsh-mcp-client",
		Version: "1.0.0",
	}, nil)

	// Create transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Connect to the server
	session, err := client.Connect(m.ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	server.Session = session

	// List available tools
	toolsList, err := session.ListTools(m.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Store tools
	server.mu.Lock()
	for _, tool := range toolsList.Tools {
		server.Tools[tool.Name] = tool
	}
	server.mu.Unlock()

	return nil
}

// GetServer returns a server by name
func (m *Manager) GetServer(name string) (*MCPServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[name]
	if !exists {
		return nil, fmt.Errorf("MCP server '%s' not found", name)
	}

	return server, nil
}

// GetTool returns a tool from a specific server
func (m *Manager) GetTool(serverName, toolName string) (*mcp.Tool, error) {
	server, err := m.GetServer(serverName)
	if err != nil {
		return nil, err
	}

	server.mu.RLock()
	defer server.mu.RUnlock()

	tool, exists := server.Tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found in MCP server '%s'", toolName, serverName)
	}

	return tool, nil
}

// CallTool invokes an MCP tool
func (m *Manager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	server, err := m.GetServer(serverName)
	if err != nil {
		return nil, err
	}

	// Verify tool exists
	if _, err := m.GetTool(serverName, toolName); err != nil {
		return nil, err
	}

	// Call the tool
	result, err := server.Session.CallTool(m.ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call tool '%s' on server '%s': %w", toolName, serverName, err)
	}

	return result, nil
}

// ListServers returns all registered server names
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// ListTools returns all tools from a server
func (m *Manager) ListTools(serverName string) ([]string, error) {
	server, err := m.GetServer(serverName)
	if err != nil {
		return nil, err
	}

	server.mu.RLock()
	defer server.mu.RUnlock()

	tools := make([]string, 0, len(server.Tools))
	for name := range server.Tools {
		tools = append(tools, name)
	}
	return tools, nil
}

// Close shuts down all MCP servers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel context to stop all operations
	m.cancel()

	// Close all server connections
	var errs []error
	for name, server := range m.servers {
		if server.Session != nil {
			if err := server.Session.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close server '%s': %w", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing MCP servers: %v", errs)
	}

	return nil
}
