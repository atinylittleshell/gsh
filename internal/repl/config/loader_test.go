package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader(nil)
	assert.NotNil(t, loader)
}

func TestLoader_LoadFromString_EmptySource(t *testing.T) {
	loader := NewLoader(nil)
	result, err := loader.LoadFromString("")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Config)
	assert.Empty(t, result.Errors)

	// Should have default values
	assert.Equal(t, "gsh> ", result.Config.Prompt)
	assert.Equal(t, "info", result.Config.LogLevel)
}

func TestLoader_LoadFromString_BasicGSHConfig(t *testing.T) {
	source := `
GSH_CONFIG = {
	prompt: "myshell> ",
	logLevel: "debug"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "myshell> ", result.Config.Prompt)
	assert.Equal(t, "debug", result.Config.LogLevel)
}

func TestLoader_LoadFromString_ModelDeclaration(t *testing.T) {
	source := `
model myModel {
	provider: "openai",
	model: "gpt-4o",
	temperature: 0.7
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Models, 1)
	model := result.Config.GetModel("myModel")
	assert.NotNil(t, model)
	assert.Equal(t, "myModel", model.Name)
}

func TestLoader_LoadFromString_ToolDeclaration(t *testing.T) {
	source := `
tool greet(name: string): string {
	return "Hello, " + name
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Tools, 1)
	tool := result.Config.GetTool("greet")
	assert.NotNil(t, tool)
	assert.Equal(t, "greet", tool.Name)
	assert.Contains(t, tool.Parameters, "name")
}

func TestLoader_LoadFromString_GSHUpdatePromptTool(t *testing.T) {
	source := `
tool GSH_UPDATE_PROMPT(exitCode: number, durationMs: number): string {
	if (exitCode == 0) {
		return "gsh> "
	}
	return "gsh [" + exitCode + "]> "
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	updatePromptTool := result.Config.GetUpdatePromptTool()
	assert.NotNil(t, updatePromptTool)
	assert.Equal(t, "GSH_UPDATE_PROMPT", updatePromptTool.Name)
}

func TestLoader_LoadFromString_CompleteConfig(t *testing.T) {
	source := `
model claude {
	provider: "openai",
	model: "claude-sonnet-4-20250514"
}

tool helper(x: number): number {
	return x * 2
}

GSH_CONFIG = {
	prompt: "$ ",
	logLevel: "warn"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	// Check config
	assert.Equal(t, "$ ", result.Config.Prompt)
	assert.Equal(t, "warn", result.Config.LogLevel)

	// Check model
	assert.Len(t, result.Config.Models, 1)
	assert.NotNil(t, result.Config.GetModel("claude"))

	// Check tool
	assert.Len(t, result.Config.Tools, 1)
	assert.NotNil(t, result.Config.GetTool("helper"))

}

func TestLoader_LoadFromString_ParseError(t *testing.T) {
	source := `
GSH_CONFIG = {
	prompt: "test>  // Missing closing quote
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err) // Should not return error, but collect it in Errors
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)

	// Should return default config
	assert.Equal(t, "gsh> ", result.Config.Prompt)
}

func TestLoader_LoadFromString_EvalError(t *testing.T) {
	source := `
x = undefinedVariable + 1
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err) // Should not return error, but collect it in Errors
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)

	// Should return default config
	assert.Equal(t, "gsh> ", result.Config.Prompt)
}

func TestLoader_LoadFromString_InvalidGSHConfigType(t *testing.T) {
	source := `GSH_CONFIG = "not an object"`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG must be an object")

	// Should still use default config
	assert.Equal(t, "gsh> ", result.Config.Prompt)
}

func TestLoader_LoadFromString_InvalidPromptType(t *testing.T) {
	source := `
GSH_CONFIG = {
	prompt: 123
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG.prompt must be a string")
}

func TestLoader_LoadFromFile_NonExistent(t *testing.T) {
	loader := NewLoader(nil)
	result, err := loader.LoadFromFile("/nonexistent/path/.gshrc.gsh")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	// Should return default config
	assert.Equal(t, "gsh> ", result.Config.Prompt)
}

func TestLoader_LoadFromFile_ValidFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gshrc.gsh")

	content := `
GSH_CONFIG = {
	prompt: "loaded> ",
	logLevel: "error"
}
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	loader := NewLoader(nil)
	result, err := loader.LoadFromFile(configPath)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "loaded> ", result.Config.Prompt)
	assert.Equal(t, "error", result.Config.LogLevel)
}

func TestLoader_LoadFromFile_UnreadableFile(t *testing.T) {
	// Create a directory instead of a file to cause a read error
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.Mkdir(configPath, 0755)
	require.NoError(t, err)

	loader := NewLoader(nil)
	_, err = loader.LoadFromFile(configPath)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoader_LoadResult_InterpreterAvailable(t *testing.T) {
	source := `
x = 42
GSH_CONFIG = {
	prompt: "test> "
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result.Interpreter)
}

func TestLoader_LoadFromString_MultipleModels(t *testing.T) {
	source := `
model gpt4 {
	provider: "openai",
	model: "gpt-4o"
}

model claude {
	provider: "openai",
	model: "claude-sonnet-4-20250514"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Models, 2)
	assert.NotNil(t, result.Config.GetModel("gpt4"))
	assert.NotNil(t, result.Config.GetModel("claude"))
}

func TestLoader_LoadFromString_MultipleTools(t *testing.T) {
	source := `
tool add(a: number, b: number): number {
	return a + b
}

tool multiply(a: number, b: number): number {
	return a * b
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Tools, 2)
	assert.NotNil(t, result.Config.GetTool("add"))
	assert.NotNil(t, result.Config.GetTool("multiply"))
}

func TestLoader_LoadFromString_PartialConfig(t *testing.T) {
	// Only setting some config values should preserve defaults for others
	source := `
GSH_CONFIG = {
	prompt: "custom> "
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Equal(t, "custom> ", result.Config.Prompt)
	assert.Equal(t, "info", result.Config.LogLevel) // Default preserved
}

func TestLoader_LoadFromString_InvalidLogLevelType(t *testing.T) {
	source := `
GSH_CONFIG = {
	logLevel: 123
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG.logLevel must be a string")
}

func TestLoader_LoadFromString_PredictModel(t *testing.T) {
	source := `
model predictModel {
	provider: "openai",
	model: "gpt-4o-mini",
	apiKey: "test-key"
}

GSH_CONFIG = {
	predictModel: "predictModel"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "predictModel", result.Config.PredictModel)

	// Verify GetPredictModel returns the correct model
	model := result.Config.GetPredictModel()
	require.NotNil(t, model)
	assert.Equal(t, "predictModel", model.Name)
}

func TestLoader_LoadFromString_PredictModelInvalidType(t *testing.T) {
	source := `
GSH_CONFIG = {
	predictModel: 123
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG.predictModel must be a string")
}

func TestLoader_LoadFromString_PredictModelWithFullConfig(t *testing.T) {
	source := `
model fastModel {
	provider: "openai",
	model: "gpt-4o-mini",
	apiKey: "test-key",
	temperature: 0.1
}

model slowModel {
	provider: "openai",
	model: "gpt-4o",
	apiKey: "test-key"
}

GSH_CONFIG = {
	prompt: "myshell> ",
	logLevel: "debug",
	predictModel: "fastModel"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Equal(t, "myshell> ", result.Config.Prompt)
	assert.Equal(t, "debug", result.Config.LogLevel)
	assert.Equal(t, "fastModel", result.Config.PredictModel)

	// Verify the correct model is returned
	model := result.Config.GetPredictModel()
	require.NotNil(t, model)
	assert.Equal(t, "fastModel", model.Name)

	// Verify the other model is also available
	slowModel := result.Config.GetModel("slowModel")
	require.NotNil(t, slowModel)
	assert.Equal(t, "slowModel", slowModel.Name)
}

func TestLoader_LoadFromString_DefaultAgent(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		agent myAgent {
			model: testModel,
			systemPrompt: "test",
		}

		GSH_CONFIG = {
			defaultAgent: "myAgent",
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)
	assert.Equal(t, 0, len(result.Errors))

	// Verify defaultAgent is set
	assert.Equal(t, "myAgent", result.Config.DefaultAgent)

	// Verify GetDefaultAgent works
	agent := result.Config.GetDefaultAgent()
	require.NotNil(t, agent)
	assert.Equal(t, "myAgent", agent.Name)
}

func TestLoader_LoadFromString_DefaultAgentInvalidType(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		GSH_CONFIG = {
			defaultAgent: 123,
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)

	// Should have error about invalid type
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Error(), "defaultAgent must be a string")
}
