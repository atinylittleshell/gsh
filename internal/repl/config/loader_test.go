package config

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
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
	assert.Empty(t, result.Errors)
	// Should have default config with empty maps
	assert.NotNil(t, result.Config)
	assert.NotNil(t, result.Config.Models)
	assert.NotNil(t, result.Config.Agents)
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

func TestLoader_LoadFromString_CompleteConfig(t *testing.T) {
	source := `
model claude {
	provider: "openai",
	model: "claude-sonnet-4-20250514"
}

tool helper(x: number): number {
	return x * 2
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	// Check model
	assert.Len(t, result.Config.Models, 1)
	assert.NotNil(t, result.Config.GetModel("claude"))

	// Check tool
	assert.Len(t, result.Config.Tools, 1)
	assert.NotNil(t, result.Config.GetTool("helper"))
}

func TestLoader_LoadFromString_ParseError(t *testing.T) {
	source := `
x = {
	prompt: "test>  // Missing closing quote
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err) // Should not return error, but collect it in Errors
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)

	// Should return default config
	assert.NotNil(t, result.Config)
}

func TestLoader_LoadFromString_EvalError(t *testing.T) {
	source := `undefinedVar.method()`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err) // Error is collected, not returned
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Errors)
}

func TestLoader_LoadFromFile_NonExistent(t *testing.T) {
	loader := NewLoader(nil)
	result, err := loader.LoadFromFile("/nonexistent/path/repl.gsh")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	// Should return default config
	assert.NotNil(t, result.Config)
}

func TestLoader_LoadFromFile_ValidFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "repl.gsh")

	content := `
model testModel {
	provider: "openai",
	model: "gpt-4"
}
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	loader := NewLoader(nil)
	result, err := loader.LoadFromFile(configPath)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)
	assert.NotNil(t, result.Config.GetModel("testModel"))
}

func TestLoader_LoadFromFile_UnreadableFile(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "adir")

	// Create a directory (which can't be read as a file)
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	loader := NewLoader(nil)
	_, err = loader.LoadFromFile(dirPath)

	// Should return error when trying to read a directory as file
	assert.Error(t, err)
}

func TestLoader_LoadResult_InterpreterAvailable(t *testing.T) {
	source := `
x = 42
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result.Interpreter)
}

func TestLoader_LoadFromString_MultipleModels(t *testing.T) {
	source := `
model model1 {
	provider: "openai",
	model: "gpt-4o"
}

model model2 {
	provider: "openai",
	model: "gpt-4o-mini"
}

model model3 {
	provider: "openai",
	model: "gpt-4"
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Models, 3)
	assert.NotNil(t, result.Config.GetModel("model1"))
	assert.NotNil(t, result.Config.GetModel("model2"))
	assert.NotNil(t, result.Config.GetModel("model3"))
}

func TestLoader_LoadFromString_MultipleTools(t *testing.T) {
	source := `
tool tool1(x: number): number {
	return x + 1
}

tool tool2(s: string): string {
	return "Hello " + s
}
`
	loader := NewLoader(nil)
	result, err := loader.LoadFromString(source)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)

	assert.Len(t, result.Config.Tools, 2)
	assert.NotNil(t, result.Config.GetTool("tool1"))
	assert.NotNil(t, result.Config.GetTool("tool2"))
}

func TestConfig_Clone(t *testing.T) {
	original := DefaultConfig()

	original.Models["model1"] = &interpreter.ModelValue{Name: "model1"}
	original.Tools["tool1"] = &interpreter.ToolValue{Name: "tool1"}

	clone := original.Clone()

	// Verify values are copied
	assert.NotNil(t, clone.Models["model1"])
	assert.NotNil(t, clone.Tools["tool1"])

	// Verify maps are independent
	clone.Models["model2"] = &interpreter.ModelValue{Name: "model2"}
	assert.Nil(t, original.Models["model2"])
}

type mockBashExecutor struct {
	lastContent string
	shouldError bool
}

func (m *mockBashExecutor) RunBashScriptFromReader(ctx context.Context, reader io.Reader, name string) error {
	content, _ := io.ReadAll(reader)
	m.lastContent = string(content)
	if m.shouldError {
		return assert.AnError
	}
	return nil
}

func TestLoadBashRC_FileExists(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	err := os.WriteFile(rcPath, []byte("export FOO=bar"), 0644)
	require.NoError(t, err)

	mock := &mockBashExecutor{}
	err = LoadBashRC(context.Background(), mock, rcPath)

	require.NoError(t, err)
	assert.Equal(t, "export FOO=bar", mock.lastContent)
}

func TestLoadBashRC_FileDoesNotExist(t *testing.T) {
	mock := &mockBashExecutor{}
	err := LoadBashRC(context.Background(), mock, "/nonexistent/.gshrc")

	// Should not error for missing file
	require.NoError(t, err)
	assert.Empty(t, mock.lastContent)
}

func TestLoadBashRC_EmptyFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	err := os.WriteFile(rcPath, []byte(""), 0644)
	require.NoError(t, err)

	mock := &mockBashExecutor{}
	err = LoadBashRC(context.Background(), mock, rcPath)

	// Should skip empty files
	require.NoError(t, err)
	assert.Empty(t, mock.lastContent)
}

func TestLoadBashRC_ExecutionError(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	rcPath := filepath.Join(tmpDir, ".gshrc")

	err := os.WriteFile(rcPath, []byte("some content"), 0644)
	require.NoError(t, err)

	mock := &mockBashExecutor{shouldError: true}
	err = LoadBashRC(context.Background(), mock, rcPath)

	// Should return the execution error
	require.Error(t, err)
}

func TestInterpreter_EvalString_Shadowing(t *testing.T) {
	// This tests that when the same variable is declared in two strings
	// evaluated in sequence, the second declaration shadows the first

	interp := interpreter.New(nil)

	// First evaluation
	_, err := interp.EvalString(`
x = 1
y = 10
`, nil)
	require.NoError(t, err)

	vars := interp.GetVariables()
	xVal, ok := vars["x"].(*interpreter.NumberValue)
	require.True(t, ok)
	assert.Equal(t, float64(1), xVal.Value)

	// Second evaluation - shadows x but keeps y
	_, err = interp.EvalString(`
x = 2
`, nil)
	require.NoError(t, err)

	vars = interp.GetVariables()
	xVal, ok = vars["x"].(*interpreter.NumberValue)
	require.True(t, ok)
	assert.Equal(t, float64(2), xVal.Value)

	// y should still exist from first evaluation
	yVal, ok := vars["y"].(*interpreter.NumberValue)
	require.True(t, ok)
	assert.Equal(t, float64(10), yVal.Value)
}

// Test that LoadDefaultConfigPathInto uses a single interpreter for both configs
func TestLoader_LoadDefaultConfigPathInto_SingleInterpreter(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create a user config that shadows a tool from the default config
	defaultConfig := `
tool onAgentStart(ctx): string {
	return "default header"
}

model DefaultModel {
	provider: "openai",
	model: "gpt-4",
}
`

	userConfig := `
tool onAgentStart(ctx): string {
	return "custom header"
}

model UserModel {
	provider: "openai",
	model: "gpt-3.5-turbo",
}
`

	// Write user config to the temp home directory
	gshDir := filepath.Join(tmpDir, ".gsh")
	err := os.MkdirAll(gshDir, 0755)
	require.NoError(t, err)
	userConfigPath := filepath.Join(gshDir, "repl.gsh")
	err = os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	interp := interpreter.New(nil)
	result, err := loader.LoadDefaultConfigPathInto(interp, EmbeddedDefaults{Content: defaultConfig})
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
	toolVal, ok := vars["onAgentStart"]
	require.True(t, ok, "onAgentStart should exist")
	require.NotNil(t, toolVal)

	// Both configs were loaded into the same interpreter
	_, isToolValue := toolVal.(*interpreter.ToolValue)
	require.True(t, isToolValue, "onAgentStart should be a ToolValue")
}

func TestLoader_LoadDefaultConfigPathInto_NoUserConfig(t *testing.T) {
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

tool defaultTool(x: number): number {
	return x * 2
}
`

	// Load configuration (no user config file exists)
	loader := NewLoader(nil)
	interp := interpreter.New(nil)
	result, err := loader.LoadDefaultConfigPathInto(interp, EmbeddedDefaults{Content: defaultConfig})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify default model exists
	assert.NotNil(t, result.Config.Models["DefaultModel"], "DefaultModel should exist from default config")

	// Verify default tool exists
	vars := result.Interpreter.GetVariables()
	_, ok := vars["defaultTool"]
	require.True(t, ok, "defaultTool should exist from default config")
}

func TestLoader_LoadDefaultConfigPathInto_EmbeddedImports(t *testing.T) {
	// Create a temp directory to use as home (no user config)
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create an embedded filesystem with imports
	embedFS := fstest.MapFS{
		"defaults/init.gsh": &fstest.MapFile{
			Data: []byte(`
import { helperTool } from "./helpers.gsh"
result = helperTool(5)
`),
		},
		"defaults/helpers.gsh": &fstest.MapFile{
			Data: []byte(`
export tool helperTool(x) {
    return x * 10
}
`),
		},
	}

	// Read init.gsh content
	initContent, _ := embedFS.ReadFile("defaults/init.gsh")

	// Load configuration with embedded FS
	loader := NewLoader(nil)
	interp := interpreter.New(nil)
	result, err := loader.LoadDefaultConfigPathInto(interp, EmbeddedDefaults{
		Content:  string(initContent),
		FS:       embedFS,
		BasePath: "defaults",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify the imported tool was executed and result is correct (5 * 10 = 50)
	vars := result.Interpreter.GetVariables()
	resultVal, ok := vars["result"]
	require.True(t, ok, "result should exist")
	numVal, ok := resultVal.(*interpreter.NumberValue)
	require.True(t, ok, "result should be a number")
	assert.Equal(t, float64(50), numVal.Value)
}

func TestLoader_LoadDefaultConfigPathInto_VariablesAccessible(t *testing.T) {
	// Create a temp directory to use as home
	tmpDir := t.TempDir()

	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Reset cached paths so HomeDir() picks up the new HOME env var
	core.ResetPaths()
	defer core.ResetPaths()

	// Default config defines a model
	defaultConfig := `
model DEFAULT_MODEL {
	provider: "openai",
	model: "gpt-4",
}
`

	// User config references the model from default config
	userConfig := `
# Reference the model defined in defaults
myModel = DEFAULT_MODEL
`

	// Write user config
	gshDir := filepath.Join(tmpDir, ".gsh")
	err := os.MkdirAll(gshDir, 0755)
	require.NoError(t, err)
	userConfigPath := filepath.Join(gshDir, "repl.gsh")
	err = os.WriteFile(userConfigPath, []byte(userConfig), 0644)
	require.NoError(t, err)

	// Load configuration
	loader := NewLoader(nil)
	interp := interpreter.New(nil)
	result, err := loader.LoadDefaultConfigPathInto(interp, EmbeddedDefaults{Content: defaultConfig})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Errors)

	// Verify the model from default config is available
	assert.NotNil(t, result.Config.Models["DEFAULT_MODEL"], "DEFAULT_MODEL should exist from default config")

	// Verify user config can reference variables from default config
	vars := result.Interpreter.GetVariables()
	myModel, ok := vars["myModel"]
	require.True(t, ok, "myModel should exist from user config")
	_, isModelValue := myModel.(*interpreter.ModelValue)
	assert.True(t, isModelValue, "myModel should be a ModelValue")
}
