// Package config provides configuration management for the gsh REPL.
// It handles loading and parsing of ~/.gsh/repl.gsh configuration files,
// extracting declarations (models, agents, tools, MCP servers), and maintaining
// backward compatibility with bash-style .gshrc files.
package config

import (
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
)

// Config holds all REPL configuration extracted from declarations in ~/.gsh/repl.gsh.
// Configuration values like logging level, prompt, model tiers, and event handlers are now
// managed via the SDK (gsh.* properties and gsh.on() event handlers).
type Config struct {
	// All declarations from ~/.gsh/repl.gsh (using gsh language syntax)
	// These are available for use in scripts and agent mode

	// MCPServers holds MCP server configurations from `mcp` declarations
	MCPServers map[string]*mcp.MCPServer

	// Models holds model configurations from `model` declarations
	Models map[string]*interpreter.ModelValue

	// Agents holds agent configurations from `agent` declarations
	Agents map[string]*interpreter.AgentValue

	// Tools holds tool definitions from `tool` declarations
	Tools map[string]*interpreter.ToolValue
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		MCPServers: make(map[string]*mcp.MCPServer),
		Models:     make(map[string]*interpreter.ModelValue),
		Agents:     make(map[string]*interpreter.AgentValue),
		Tools:      make(map[string]*interpreter.ToolValue),
	}
}

// GetModel returns a model by name, or nil if not found.
func (c *Config) GetModel(name string) *interpreter.ModelValue {
	if c.Models == nil {
		return nil
	}
	return c.Models[name]
}

// GetAgent returns an agent by name, or nil if not found.
func (c *Config) GetAgent(name string) *interpreter.AgentValue {
	if c.Agents == nil {
		return nil
	}
	return c.Agents[name]
}

// GetTool returns a tool by name, or nil if not found.
func (c *Config) GetTool(name string) *interpreter.ToolValue {
	if c.Tools == nil {
		return nil
	}
	return c.Tools[name]
}

// GetMCPServer returns an MCP server by name, or nil if not found.
func (c *Config) GetMCPServer(name string) *mcp.MCPServer {
	if c.MCPServers == nil {
		return nil
	}
	return c.MCPServers[name]
}

// Clone creates a deep copy of the Config.
func (c *Config) Clone() *Config {
	clone := &Config{
		MCPServers: make(map[string]*mcp.MCPServer, len(c.MCPServers)),
		Models:     make(map[string]*interpreter.ModelValue, len(c.Models)),
		Agents:     make(map[string]*interpreter.AgentValue, len(c.Agents)),
		Tools:      make(map[string]*interpreter.ToolValue, len(c.Tools)),
	}

	// Copy maps (shallow copy of values, which are pointers)
	for k, v := range c.MCPServers {
		clone.MCPServers[k] = v
	}
	for k, v := range c.Models {
		clone.Models[k] = v
	}
	for k, v := range c.Agents {
		clone.Agents[k] = v
	}
	for k, v := range c.Tools {
		clone.Tools[k] = v
	}

	return clone
}
