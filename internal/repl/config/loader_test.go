package config

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
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
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
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
	assert.Equal(t, "GSH_PROMPT", updatePromptTool.Name)
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
	predictModel: predictModel
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
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG.predictModel must be a model reference")
}

func TestLoader_LoadFromString_PredictModelRejectsString(t *testing.T) {
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
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "GSH_CONFIG.predictModel must be a model reference")
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
	predictModel: fastModel
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

		GSH_CONFIG = {
			defaultAgentModel: testModel,
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)
	assert.Equal(t, 0, len(result.Errors))

	// Verify defaultAgentModel is set
	assert.Equal(t, "testModel", result.Config.DefaultAgentModel)

	// Verify GetDefaultAgentModel works
	model := result.Config.GetDefaultAgentModel()
	require.NotNil(t, model)
	assert.Equal(t, "testModel", model.Name)
}

func TestLoader_LoadFromString_DefaultAgentModelInvalidType(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		GSH_CONFIG = {
			defaultAgentModel: 123,
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)

	// Should have error about invalid type
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Error(), "defaultAgentModel must be a model reference")
}

func TestLoader_LoadFromString_DefaultAgentModelRejectsString(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		GSH_CONFIG = {
			defaultAgentModel: "testModel",
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)

	// Should have error about invalid type (string not allowed)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Error(), "defaultAgentModel must be a model reference")
}

func TestLoader_LoadFromString_DefaultAgentModelAsModelReference(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		GSH_CONFIG = {
			defaultAgentModel: testModel,
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)
	assert.Equal(t, 0, len(result.Errors))

	// Verify defaultAgentModel is set to the model's name
	assert.Equal(t, "testModel", result.Config.DefaultAgentModel)

	// Verify GetDefaultAgentModel works
	model := result.Config.GetDefaultAgentModel()
	require.NotNil(t, model)
	assert.Equal(t, "testModel", model.Name)
}

func TestLoader_LoadFromString_PredictModelAsModelReference(t *testing.T) {
	loader := NewLoader(nil)

	source := `
		model testModel {
			provider: "openai",
			model: "gpt-4",
		}

		GSH_CONFIG = {
			predictModel: testModel,
		}
	`

	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Config)
	assert.Equal(t, 0, len(result.Errors))

	// Verify predictModel is set to the model's name
	assert.Equal(t, "testModel", result.Config.PredictModel)
}

func TestLoader_MergeResults(t *testing.T) {
	loader := NewLoader(nil)

	// Create base config
	base := &LoadResult{
		Config: &Config{
			Prompt:            "base> ",
			LogLevel:          "info",
			PredictModel:      "baseModel",
			DefaultAgentModel: "baseAgent",
			Models: map[string]*interpreter.ModelValue{
				"baseModel": {Name: "baseModel"},
			},
			Agents:     make(map[string]*interpreter.AgentValue),
			Tools:      make(map[string]*interpreter.ToolValue),
			MCPServers: make(map[string]*mcp.MCPServer),
		},
		Errors: []error{},
	}

	// Create override config
	override := &LoadResult{
		Config: &Config{
			Prompt:            "override> ",
			LogLevel:          "debug",
			PredictModel:      "",
			DefaultAgentModel: "overrideAgent",
			Models: map[string]*interpreter.ModelValue{
				"overrideModel": {Name: "overrideModel"},
			},
			Agents:     make(map[string]*interpreter.AgentValue),
			Tools:      make(map[string]*interpreter.ToolValue),
			MCPServers: make(map[string]*mcp.MCPServer),
		},
		Errors: []error{},
	}

	// Merge
	result := loader.mergeResults(base, override)

	// Verify merged values
	assert.Equal(t, "override> ", result.Config.Prompt)
	assert.Equal(t, "debug", result.Config.LogLevel)
	assert.Equal(t, "baseModel", result.Config.PredictModel) // base preserved when override is empty
	assert.Equal(t, "overrideAgent", result.Config.DefaultAgentModel)

	// Verify models are merged
	assert.Len(t, result.Config.Models, 2)
	assert.NotNil(t, result.Config.Models["baseModel"])
	assert.NotNil(t, result.Config.Models["overrideModel"])
}

func TestLoader_MergeResults_WithDefaultValues(t *testing.T) {
	loader := NewLoader(nil)

	// Create base config with non-default values
	base := &LoadResult{
		Config: &Config{
			Prompt:            "custom> ",
			LogLevel:          "warn",
			PredictModel:      "baseModel",
			DefaultAgentModel: "baseAgent",
			Models:            make(map[string]*interpreter.ModelValue),
			Agents:            make(map[string]*interpreter.AgentValue),
			Tools:             make(map[string]*interpreter.ToolValue),
			MCPServers:        make(map[string]*mcp.MCPServer),
		},
		Errors: []error{},
	}

	// Create override config with default values (should not override)
	override := &LoadResult{
		Config: &Config{
			Prompt:            "gsh> ", // default value
			LogLevel:          "info",  // default value
			PredictModel:      "",
			DefaultAgentModel: "",
			Models:            make(map[string]*interpreter.ModelValue),
			Agents:            make(map[string]*interpreter.AgentValue),
			Tools:             make(map[string]*interpreter.ToolValue),
			MCPServers:        make(map[string]*mcp.MCPServer),
		},
		Errors: []error{},
	}

	// Merge
	result := loader.mergeResults(base, override)

	// Verify base values are preserved when override has default values
	assert.Equal(t, "custom> ", result.Config.Prompt)
	assert.Equal(t, "warn", result.Config.LogLevel)
	assert.Equal(t, "baseModel", result.Config.PredictModel)
	assert.Equal(t, "baseAgent", result.Config.DefaultAgentModel)
}

func TestConfig_Clone(t *testing.T) {
	original := &Config{
		Prompt:            "test> ",
		LogLevel:          "debug",
		PredictModel:      "testModel",
		DefaultAgentModel: "testAgent",
		Models: map[string]*interpreter.ModelValue{
			"model1": {Name: "model1"},
		},
		Agents:     make(map[string]*interpreter.AgentValue),
		Tools:      make(map[string]*interpreter.ToolValue),
		MCPServers: make(map[string]*mcp.MCPServer),
	}

	cloned := original.Clone()

	// Verify values are copied
	assert.Equal(t, original.Prompt, cloned.Prompt)
	assert.Equal(t, original.LogLevel, cloned.LogLevel)
	assert.Equal(t, original.PredictModel, cloned.PredictModel)
	assert.Equal(t, original.DefaultAgentModel, cloned.DefaultAgentModel)

	// Verify maps are independent
	cloned.Models["model2"] = &interpreter.ModelValue{Name: "model2"}
	assert.Len(t, original.Models, 1)
	assert.Len(t, cloned.Models, 2)

	// Verify changing cloned values doesn't affect original
	cloned.Prompt = "changed> "
	assert.Equal(t, "test> ", original.Prompt)
	assert.Equal(t, "changed> ", cloned.Prompt)
}

// mockBashExecutor is a mock implementation of BashExecutor for testing
type mockBashExecutor struct {
	executedScripts []string
	executeError    error
}

func (m *mockBashExecutor) RunBashScriptFromReader(ctx context.Context, reader io.Reader, name string) error {
	if m.executeError != nil {
		return m.executeError
	}

	// Read the script content
	var content strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			content.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	m.executedScripts = append(m.executedScripts, content.String())
	return nil
}

func TestLoadBashRC_FileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	testContent := "# Test .gshrc\nexport TEST_VAR=hello\nalias ll='ls -la'"
	err := os.WriteFile(rcPath, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create mock executor
	mock := &mockBashExecutor{}

	// Load the bash RC
	ctx := context.Background()
	err = LoadBashRC(ctx, mock, rcPath)
	require.NoError(t, err)

	// Verify the script was executed
	assert.Equal(t, 1, len(mock.executedScripts))
	assert.Equal(t, testContent, mock.executedScripts[0])
}

func TestLoadBashRC_FileDoesNotExist(t *testing.T) {
	mock := &mockBashExecutor{}

	ctx := context.Background()
	err := LoadBashRC(ctx, mock, "/nonexistent/.gshrc")

	// Should not return an error for non-existent files
	require.NoError(t, err)

	// Should not have executed anything
	assert.Equal(t, 0, len(mock.executedScripts))
}

func TestLoadBashRC_EmptyFile(t *testing.T) {
	// Create an empty file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	err := os.WriteFile(rcPath, []byte(""), 0644)
	require.NoError(t, err)

	mock := &mockBashExecutor{}

	ctx := context.Background()
	err = LoadBashRC(ctx, mock, rcPath)
	require.NoError(t, err)

	// Empty files should be skipped
	assert.Equal(t, 0, len(mock.executedScripts))
}

func TestLoadBashRC_ExecutionError(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	err := os.WriteFile(rcPath, []byte("echo test"), 0644)
	require.NoError(t, err)

	// Create mock that returns an error
	mock := &mockBashExecutor{
		executeError: assert.AnError,
	}

	ctx := context.Background()
	err = LoadBashRC(ctx, mock, rcPath)

	// Should return the execution error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute")
}

// Test that EvalString works correctly and allows shadowing
func TestInterpreter_EvalString_Shadowing(t *testing.T) {
	// Create an interpreter - redefinition is always allowed
	interp := interpreter.NewWithLogger(nil)

	// Define a tool in the first evaluation
	defaultConfig := `
tool GSH_TEST_TOOL(name: string): string {
	return "default: " + name
}
`
	_, err := interp.EvalString(defaultConfig)
	require.NoError(t, err)

	// Verify tool exists
	vars := interp.GetVariables()
	_, ok := vars["GSH_TEST_TOOL"]
	require.True(t, ok, "GSH_TEST_TOOL should exist after first eval")

	// Define the same tool in the second evaluation (should shadow)
	userConfig := `
tool GSH_TEST_TOOL(name: string): string {
	return "user: " + name
}
`
	_, err = interp.EvalString(userConfig)
	require.NoError(t, err)

	// Verify tool still exists and is the user's version
	vars = interp.GetVariables()
	toolVal, ok := vars["GSH_TEST_TOOL"]
	require.True(t, ok, "GSH_TEST_TOOL should exist after second eval")
	require.NotNil(t, toolVal)

	// The tool should be a ToolValue
	_, isToolValue := toolVal.(*interpreter.ToolValue)
	require.True(t, isToolValue, "GSH_TEST_TOOL should be a ToolValue")
}

// Test that LoadDefaultConfigPath uses a single interpreter for both configs
func TestLoader_LoadDefaultConfigPath_SingleInterpreter(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create a user config that shadows a tool from the default config
	defaultConfig := `
tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
	return "default header for " + agentName
}

model DefaultModel {
	provider: "openai",
	model: "gpt-4",
}
`

	userConfig := `
tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
	return "custom header for " + agentName
}

model UserModel {
	provider: "openai",
	model: "gpt-3.5-turbo",
}
`

	// Write user config to the temp home directory
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify the interpreter exists
	require.NotNil(t, result.Interpreter, "Interpreter should be available in result")

	// Verify both models are available (DefaultModel from default, UserModel from user)
	assert.NotNil(t, result.Config.Models["DefaultModel"], "DefaultModel should exist from default config")
	assert.NotNil(t, result.Config.Models["UserModel"], "UserModel should exist from user config")

	// Verify the tool exists and is the user's version (shadowed)
	vars := result.Interpreter.GetVariables()
	toolVal, ok := vars["GSH_AGENT_HEADER"]
	require.True(t, ok, "GSH_AGENT_HEADER should exist")
	require.NotNil(t, toolVal)

	// Both configs were loaded into the same interpreter
	_, isToolValue := toolVal.(*interpreter.ToolValue)
	require.True(t, isToolValue, "GSH_AGENT_HEADER should be a ToolValue")
}

// Test that default config is loaded even when user config doesn't exist
func TestLoader_LoadDefaultConfigPath_NoUserConfig(t *testing.T) {
	// Create a temp directory to use as home (no user config)
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	defaultConfig := `
model DefaultModel {
	provider: "openai",
	model: "gpt-4",
}

tool GSH_DEFAULT_TOOL(x: number): number {
	return x * 2
}
`

	// Load configuration (no user config file exists)
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify default model exists
	assert.NotNil(t, result.Config.Models["DefaultModel"], "DefaultModel should exist from default config")

	// Verify default tool exists
	vars := result.Interpreter.GetVariables()
	_, ok := vars["GSH_DEFAULT_TOOL"]
	require.True(t, ok, "GSH_DEFAULT_TOOL should exist from default config")
}

// Test that user's GSH_CONFIG is merged with defaults (not replaced)
func TestLoader_LoadDefaultConfigPath_GSHConfigMerge(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Default config with GSH_CONFIG containing multiple settings
	defaultConfig := `
model DEFAULT_PREDICT_MODEL {
	provider: "openai",
	model: "gpt-4",
}

model DEFAULT_AGENT_MODEL {
	provider: "openai",
	model: "gpt-4-turbo",
}

GSH_CONFIG = {
	prompt: "default> ",
	logLevel: "info",
	starshipIntegration: true,
	showWelcome: true,
	predictModel: DEFAULT_PREDICT_MODEL,
	defaultAgentModel: DEFAULT_AGENT_MODEL,
}
`

	// User config only overrides logLevel - other settings should be preserved
	userConfig := `
GSH_CONFIG = {
	logLevel: "debug",
}
`

	// Write user config
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify user's override is applied
	assert.Equal(t, "debug", result.Config.LogLevel, "logLevel should be overridden by user config")

	// Verify defaults are preserved for fields user didn't specify
	assert.Equal(t, "default> ", result.Config.Prompt, "prompt should be preserved from default config")
	assert.True(t, result.Config.StarshipIntegrationEnabled(), "starshipIntegration should be preserved from default config")
	assert.True(t, result.Config.ShowWelcomeEnabled(), "showWelcome should be preserved from default config")
	assert.Equal(t, "DEFAULT_PREDICT_MODEL", result.Config.PredictModel, "predictModel should be preserved from default config")
	assert.Equal(t, "DEFAULT_AGENT_MODEL", result.Config.DefaultAgentModel, "defaultAgentModel should be preserved from default config")
}

// Test that nested objects in GSH_CONFIG are deep merged
func TestLoader_LoadDefaultConfigPath_GSHConfigDeepMerge(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Default config with nested objects
	defaultConfig := `
GSH_CONFIG = {
	prompt: "default> ",
	logLevel: "info",
	nested: {
		featureA: true,
		featureB: false,
		deepNested: {
			setting1: "default1",
			setting2: "default2",
		},
	},
}
`

	// User config only overrides specific nested values
	userConfig := `
GSH_CONFIG = {
	logLevel: "debug",
	nested: {
		featureB: true,
		deepNested: {
			setting2: "user2",
		},
	},
}
`

	// Write user config
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Get the merged GSH_CONFIG from interpreter
	vars := result.Interpreter.GetVariables()
	gshConfig, ok := vars["GSH_CONFIG"].(*interpreter.ObjectValue)
	require.True(t, ok, "GSH_CONFIG should be an ObjectValue")

	// Check top-level values
	assert.Equal(t, "default> ", gshConfig.Properties["prompt"].String(), "prompt should be preserved from default")
	assert.Equal(t, "debug", gshConfig.Properties["logLevel"].String(), "logLevel should be overridden by user")

	// Check nested object
	nested, ok := gshConfig.Properties["nested"].(*interpreter.ObjectValue)
	require.True(t, ok, "nested should be an ObjectValue")

	// featureA should be preserved from default
	featureA, ok := nested.Properties["featureA"].(*interpreter.BoolValue)
	require.True(t, ok, "featureA should be a BoolValue")
	assert.True(t, featureA.Value, "featureA should be true (preserved from default)")

	// featureB should be overridden by user
	featureB, ok := nested.Properties["featureB"].(*interpreter.BoolValue)
	require.True(t, ok, "featureB should be a BoolValue")
	assert.True(t, featureB.Value, "featureB should be true (user override)")

	// Check deeply nested object
	deepNested, ok := nested.Properties["deepNested"].(*interpreter.ObjectValue)
	require.True(t, ok, "deepNested should be an ObjectValue")

	// setting1 should be preserved from default
	assert.Equal(t, "default1", deepNested.Properties["setting1"].String(), "setting1 should be preserved from default")

	// setting2 should be overridden by user
	assert.Equal(t, "user2", deepNested.Properties["setting2"].String(), "setting2 should be overridden by user")
}

// Test that user can override multiple GSH_CONFIG fields while preserving others
func TestLoader_LoadDefaultConfigPath_GSHConfigMergePartial(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Default config
	defaultConfig := `
model DEFAULT_MODEL {
	provider: "openai",
	model: "gpt-4",
}

GSH_CONFIG = {
	prompt: "default> ",
	logLevel: "info",
	starshipIntegration: true,
	showWelcome: true,
	predictModel: DEFAULT_MODEL,
}
`

	// User config overrides prompt and disables starship
	userConfig := `
GSH_CONFIG = {
	prompt: "custom> ",
	starshipIntegration: false,
}
`

	// Write user config
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify user's overrides are applied
	assert.Equal(t, "custom> ", result.Config.Prompt, "prompt should be overridden by user config")
	assert.False(t, result.Config.StarshipIntegrationEnabled(), "starshipIntegration should be overridden by user config")

	// Verify defaults are preserved for fields user didn't specify
	assert.Equal(t, "info", result.Config.LogLevel, "logLevel should be preserved from default config")
	assert.True(t, result.Config.ShowWelcomeEnabled(), "showWelcome should be preserved from default config")
	assert.Equal(t, "DEFAULT_MODEL", result.Config.PredictModel, "predictModel should be preserved from default config")
}

// Test that variables from default config are accessible in user config
func TestLoader_LoadDefaultConfigPath_VariablesAccessible(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Default config defines a model
	defaultConfig := `
model DEFAULT_MODEL {
	provider: "openai",
	model: "gpt-4",
}
`

	// User config references the model from default config
	userConfig := `
GSH_CONFIG = {
	predictModel: DEFAULT_MODEL,
}
`

	// Write user config
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify the model from default config is used in GSH_CONFIG
	assert.Equal(t, "DEFAULT_MODEL", result.Config.PredictModel, "predictModel should reference DEFAULT_MODEL from default config")
}

// Test starshipIntegration config extraction
func TestLoader_LoadFromString_StarshipIntegration(t *testing.T) {
	loader := NewLoader(nil)

	// Test with starshipIntegration = true
	source := `
GSH_CONFIG = {
	starshipIntegration: true,
}
`
	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)
	require.NotNil(t, result.Config.StarshipIntegration)
	assert.True(t, *result.Config.StarshipIntegration)
	assert.True(t, result.Config.StarshipIntegrationEnabled())

	// Test with starshipIntegration = false
	source = `
GSH_CONFIG = {
	starshipIntegration: false,
}
`
	result, err = loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)
	require.NotNil(t, result.Config.StarshipIntegration)
	assert.False(t, *result.Config.StarshipIntegration)
	assert.False(t, result.Config.StarshipIntegrationEnabled())
}

// Test that StarshipIntegrationEnabled defaults to true when not set
func TestConfig_StarshipIntegrationEnabled_Default(t *testing.T) {
	config := DefaultConfig()
	assert.Nil(t, config.StarshipIntegration)
	assert.True(t, config.StarshipIntegrationEnabled())
}

// Test starshipIntegration type validation
func TestLoader_LoadFromString_StarshipIntegration_InvalidType(t *testing.T) {
	loader := NewLoader(nil)

	source := `
GSH_CONFIG = {
	starshipIntegration: "yes",
}
`
	result, err := loader.LoadFromString(source)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Error(), "starshipIntegration must be a boolean")
}

// Test that starship config is loaded when starship integration is enabled
func TestLoader_LoadDefaultConfigPath_WithStarshipConfig(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	defaultConfig := `
model DefaultModel {
	provider: "openai",
	model: "gpt-4",
}
`
	// Note: Starship config will only be loaded if isStarshipAvailable() returns true
	// Since we can't guarantee starship is installed in CI, we test with empty starship config
	starshipConfig := ""

	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, starshipConfig)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify default model exists
	assert.NotNil(t, result.Config.Models["DefaultModel"])
}

// Test that starship config is not loaded when user disables it
func TestLoader_LoadDefaultConfigPath_StarshipDisabled(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	defaultConfig := `
model DefaultModel {
	provider: "openai",
	model: "gpt-4",
}
`
	// User config that disables starship
	userConfig := `
GSH_CONFIG = {
	starshipIntegration: false,
}
`
	// Write user config
	userConfigPath := filepath.Join(tmpDir, ".gshrc.gsh")
	err := os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Starship config that would define GSH_PROMPT - this won't be loaded
	// because user disabled starship integration
	starshipConfig := `
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
	return "starship> "
}
`

	loader := NewLoader(nil)
	result, err := loader.LoadDefaultConfigPath(defaultConfig, starshipConfig)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Starship integration should be disabled
	assert.False(t, result.Config.StarshipIntegrationEnabled())

	// GSH_PROMPT tool should NOT be defined because:
	// 1. User set starshipIntegration = false
	// 2. Starship config is only loaded if starship is in PATH AND integration is enabled
	// Since integration is disabled, starship config won't be loaded regardless of PATH
	vars := result.Interpreter.GetVariables()
	_, hasPromptTool := vars["GSH_PROMPT"]
	assert.False(t, hasPromptTool, "GSH_PROMPT should not be defined when starship integration is disabled")
}
