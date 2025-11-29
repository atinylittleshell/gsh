// Package config provides configuration management for the gsh REPL.
// It handles loading and parsing of .gshrc.gsh configuration files,
// mapping configuration values to the Config struct, and maintaining
// backward compatibility with bash-style .gshrc files.
package config

import (
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
)

// Config holds all REPL configuration extracted from GSH_CONFIG and declarations.
// Configuration can come from .gshrc.gsh files (gsh format) or environment variables
// (for backward compatibility with bash-style .gshrc files).
type Config struct {
	// Prompt configuration (from GSH_CONFIG.prompt)
	Prompt string

	// LogLevel controls logging verbosity (from GSH_CONFIG.logLevel)
	LogLevel string

	// Agent configuration (from GSH_CONFIG.agent)
	// ApprovedBashCommands contains regex patterns for commands that don't require confirmation
	ApprovedBashCommands []string

	// Macros maps macro names to their expansion text
	Macros map[string]string

	// All declarations from .gshrc.gsh (using gsh language syntax)
	// These are available for use in scripts and agent mode

	// MCPServers holds MCP server configurations from `mcp` declarations
	MCPServers map[string]*mcp.MCPServer

	// Models holds model configurations from `model` declarations
	Models map[string]*interpreter.ModelValue

	// Agents holds agent configurations from `agent` declarations
	Agents map[string]*interpreter.AgentValue

	// Tools holds tool definitions from `tool` declarations
	// Reserved tool names:
	//   - "GSH_UPDATE_PROMPT" - called before each prompt, signature: (exitCode: number, durationMs: number): string
	Tools map[string]*interpreter.ToolValue
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Prompt:               "gsh> ",
		LogLevel:             "info",
		ApprovedBashCommands: []string{},
		Macros:               make(map[string]string),
		MCPServers:           make(map[string]*mcp.MCPServer),
		Models:               make(map[string]*interpreter.ModelValue),
		Agents:               make(map[string]*interpreter.AgentValue),
		Tools:                make(map[string]*interpreter.ToolValue),
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

// GetUpdatePromptTool returns the GSH_UPDATE_PROMPT tool if configured.
// This tool is called before each prompt to generate a dynamic prompt string.
func (c *Config) GetUpdatePromptTool() *interpreter.ToolValue {
	return c.GetTool("GSH_UPDATE_PROMPT")
}

// GetMCPServer returns an MCP server by name, or nil if not found.
func (c *Config) GetMCPServer(name string) *mcp.MCPServer {
	if c.MCPServers == nil {
		return nil
	}
	return c.MCPServers[name]
}

// HasMacro returns true if a macro with the given name exists.
func (c *Config) HasMacro(name string) bool {
	if c.Macros == nil {
		return false
	}
	_, exists := c.Macros[name]
	return exists
}

// GetMacro returns the expansion text for a macro, or empty string if not found.
func (c *Config) GetMacro(name string) string {
	if c.Macros == nil {
		return ""
	}
	return c.Macros[name]
}
