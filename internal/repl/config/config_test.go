package config

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "gsh> ", cfg.Prompt)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.NotNil(t, cfg.ApprovedBashCommands)
	assert.Empty(t, cfg.ApprovedBashCommands)
	assert.NotNil(t, cfg.Macros)
	assert.Empty(t, cfg.Macros)
	assert.NotNil(t, cfg.MCPServers)
	assert.Empty(t, cfg.MCPServers)
	assert.NotNil(t, cfg.Models)
	assert.Empty(t, cfg.Models)
	assert.NotNil(t, cfg.Agents)
	assert.Empty(t, cfg.Agents)
	assert.NotNil(t, cfg.Tools)
	assert.Empty(t, cfg.Tools)
}

func TestConfig_GetModel(t *testing.T) {
	t.Run("returns nil for nil Models map", func(t *testing.T) {
		cfg := &Config{Models: nil}
		assert.Nil(t, cfg.GetModel("test"))
	})

	t.Run("returns nil for non-existent model", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Nil(t, cfg.GetModel("non-existent"))
	})

	t.Run("returns model when exists", func(t *testing.T) {
		model := &interpreter.ModelValue{
			Name:   "test-model",
			Config: map[string]interpreter.Value{},
		}
		cfg := DefaultConfig()
		cfg.Models["test-model"] = model

		result := cfg.GetModel("test-model")
		assert.Equal(t, model, result)
	})
}

func TestConfig_GetAgent(t *testing.T) {
	t.Run("returns nil for nil Agents map", func(t *testing.T) {
		cfg := &Config{Agents: nil}
		assert.Nil(t, cfg.GetAgent("test"))
	})

	t.Run("returns nil for non-existent agent", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Nil(t, cfg.GetAgent("non-existent"))
	})

	t.Run("returns agent when exists", func(t *testing.T) {
		agent := &interpreter.AgentValue{
			Name:   "coder",
			Config: map[string]interpreter.Value{},
		}
		cfg := DefaultConfig()
		cfg.Agents["coder"] = agent

		result := cfg.GetAgent("coder")
		assert.Equal(t, agent, result)
	})
}

func TestConfig_GetTool(t *testing.T) {
	t.Run("returns nil for nil Tools map", func(t *testing.T) {
		cfg := &Config{Tools: nil}
		assert.Nil(t, cfg.GetTool("test"))
	})

	t.Run("returns nil for non-existent tool", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Nil(t, cfg.GetTool("non-existent"))
	})

	t.Run("returns tool when exists", func(t *testing.T) {
		tool := &interpreter.ToolValue{
			Name:       "myTool",
			Parameters: []string{"arg1", "arg2"},
		}
		cfg := DefaultConfig()
		cfg.Tools["myTool"] = tool

		result := cfg.GetTool("myTool")
		assert.Equal(t, tool, result)
	})
}

func TestConfig_GetUpdatePromptTool(t *testing.T) {
	t.Run("returns nil when GSH_UPDATE_PROMPT not configured", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Nil(t, cfg.GetUpdatePromptTool())
	})

	t.Run("returns tool when GSH_UPDATE_PROMPT is configured", func(t *testing.T) {
		tool := &interpreter.ToolValue{
			Name:       "GSH_UPDATE_PROMPT",
			Parameters: []string{"exitCode", "durationMs"},
			ParamTypes: map[string]string{
				"exitCode":   "number",
				"durationMs": "number",
			},
			ReturnType: "string",
		}
		cfg := DefaultConfig()
		cfg.Tools["GSH_UPDATE_PROMPT"] = tool

		result := cfg.GetUpdatePromptTool()
		require.NotNil(t, result)
		assert.Equal(t, "GSH_UPDATE_PROMPT", result.Name)
		assert.Equal(t, []string{"exitCode", "durationMs"}, result.Parameters)
		assert.Equal(t, "string", result.ReturnType)
	})
}

func TestConfig_GetMCPServer(t *testing.T) {
	t.Run("returns nil for nil MCPServers map", func(t *testing.T) {
		cfg := &Config{MCPServers: nil}
		assert.Nil(t, cfg.GetMCPServer("test"))
	})

	t.Run("returns nil for non-existent server", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Nil(t, cfg.GetMCPServer("non-existent"))
	})

	t.Run("returns server when exists", func(t *testing.T) {
		server := &mcp.MCPServer{
			Name: "filesystem",
			Config: mcp.ServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
			},
		}
		cfg := DefaultConfig()
		cfg.MCPServers["filesystem"] = server

		result := cfg.GetMCPServer("filesystem")
		assert.Equal(t, server, result)
	})
}

func TestConfig_HasMacro(t *testing.T) {
	t.Run("returns false for nil Macros map", func(t *testing.T) {
		cfg := &Config{Macros: nil}
		assert.False(t, cfg.HasMacro("test"))
	})

	t.Run("returns false for non-existent macro", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.False(t, cfg.HasMacro("non-existent"))
	})

	t.Run("returns true when macro exists", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Macros["gitdiff"] = "Review all staged and unstaged changes"

		assert.True(t, cfg.HasMacro("gitdiff"))
	})

	t.Run("returns true for empty macro value", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Macros["empty"] = ""

		assert.True(t, cfg.HasMacro("empty"))
	})
}

func TestConfig_GetMacro(t *testing.T) {
	t.Run("returns empty string for nil Macros map", func(t *testing.T) {
		cfg := &Config{Macros: nil}
		assert.Equal(t, "", cfg.GetMacro("test"))
	})

	t.Run("returns empty string for non-existent macro", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Equal(t, "", cfg.GetMacro("non-existent"))
	})

	t.Run("returns macro expansion when exists", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Macros["gitdiff"] = "Review all staged and unstaged changes"

		assert.Equal(t, "Review all staged and unstaged changes", cfg.GetMacro("gitdiff"))
	})
}

func TestConfig_FullConfiguration(t *testing.T) {
	// Test a fully configured Config object
	cfg := DefaultConfig()

	// Set prompt and log level
	cfg.Prompt = "custom> "
	cfg.LogLevel = "debug"

	// Add approved bash commands
	cfg.ApprovedBashCommands = []string{
		"^ls.*$",
		"^pwd$",
		"^git\\s+status.*$",
	}

	// Add macros
	cfg.Macros["gitdiff"] = "Review all staged and unstaged changes"
	cfg.Macros["gitpush"] = "Review changes, create a commit, and push"

	// Add models
	cfg.Models["myModel"] = &interpreter.ModelValue{
		Name: "myModel",
		Config: map[string]interpreter.Value{
			"provider":    &interpreter.StringValue{Value: "openai"},
			"model":       &interpreter.StringValue{Value: "gpt-4o"},
			"temperature": &interpreter.NumberValue{Value: 0.1},
		},
	}

	// Add agent
	cfg.Agents["coder"] = &interpreter.AgentValue{
		Name: "coder",
		Config: map[string]interpreter.Value{
			"systemPrompt": &interpreter.StringValue{Value: "You are a coding assistant."},
		},
	}

	// Add tool
	cfg.Tools["GSH_UPDATE_PROMPT"] = &interpreter.ToolValue{
		Name:       "GSH_UPDATE_PROMPT",
		Parameters: []string{"exitCode", "durationMs"},
		ReturnType: "string",
	}

	// Add MCP server
	cfg.MCPServers["filesystem"] = &mcp.MCPServer{
		Name: "filesystem",
		Config: mcp.ServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
		},
	}

	// Verify all values
	assert.Equal(t, "custom> ", cfg.Prompt)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Len(t, cfg.ApprovedBashCommands, 3)
	assert.True(t, cfg.HasMacro("gitdiff"))
	assert.True(t, cfg.HasMacro("gitpush"))
	assert.NotNil(t, cfg.GetModel("myModel"))
	assert.NotNil(t, cfg.GetAgent("coder"))
	assert.NotNil(t, cfg.GetUpdatePromptTool())
	assert.NotNil(t, cfg.GetMCPServer("filesystem"))
}

func TestConfig_ZeroValueBehavior(t *testing.T) {
	// Test that a zero-value Config doesn't panic
	cfg := &Config{}

	assert.Nil(t, cfg.GetModel("test"))
	assert.Nil(t, cfg.GetAgent("test"))
	assert.Nil(t, cfg.GetTool("test"))
	assert.Nil(t, cfg.GetUpdatePromptTool())
	assert.Nil(t, cfg.GetMCPServer("test"))
	assert.False(t, cfg.HasMacro("test"))
	assert.Equal(t, "", cfg.GetMacro("test"))
}
