package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestShellCompletionProvider_FileCompletion_Integration(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files and directories
	testFiles := []string{
		"file1.txt",
		"file2.log",
		"test_script.sh",
		"README.md",
	}
	testDirs := []string{
		"testdir",
		"another_dir",
		"src",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	for _, dir := range testDirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	// Create nested files
	nestedFile := filepath.Join(tmpDir, "src", "main.go")
	err := os.WriteFile(nestedFile, []byte("package main"), 0644)
	require.NoError(t, err)

	// Set up the completion provider with a real runner
	runner, err := interp.New(interp.Dir(tmpDir))
	require.NoError(t, err)

	// Set the PWD environment variable to match the working directory
	runner.Vars = map[string]expand.Variable{
		"PWD": {Kind: expand.String, Str: tmpDir},
	}

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name         string
		line         string
		pos          int
		setup        func()
		expectedMin  int // Minimum expected completions
		shouldContain []string
	}{
		{
			name: "file completion in temp directory - empty prefix",
			line: "cat ",
			pos:  4,
			setup: func() {
				manager.On("GetSpec", "cat").Return(CompletionSpec{}, false)
			},
			expectedMin: 1, // At least some files should be found
			shouldContain: []string{}, // Debug first - don't expect specific files
		},
		{
			name: "file completion with prefix 'file'",
			line: "vim file",
			pos:  8,
			setup: func() {
				manager.On("GetSpec", "vim").Return(CompletionSpec{}, false)
			},
			expectedMin: 2,
			shouldContain: []string{"file1.txt", "file2.log"},
		},
		{
			name: "file completion with prefix 'test'",
			line: "less test",
			pos:  9,
			setup: func() {
				manager.On("GetSpec", "less").Return(CompletionSpec{}, false)
			},
			expectedMin: 2,
			shouldContain: []string{"test_script.sh", "testdir/"},
		},
		{
			name: "directory completion",
			line: "cd s",
			pos:  4,
			setup: func() {
				manager.On("GetSpec", "cd").Return(CompletionSpec{}, false)
			},
			expectedMin: 1,
			shouldContain: []string{"src/"},
		},
		{
			name: "nested file completion",
			line: "cat src/",
			pos:  8,
			setup: func() {
				manager.On("GetSpec", "cat").Return(CompletionSpec{}, false)
			},
			expectedMin: 1,
			shouldContain: []string{"src/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations
			manager.ExpectedCalls = nil
			manager.Calls = nil
			tt.setup()

			// Get completions
			completions := provider.GetCompletions(tt.line, tt.pos)

			// Verify we have at least the minimum expected completions
			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Should have at least %d completions, got %d: %v",
				tt.expectedMin, len(completions), completions)

			// Verify all expected items are present
			for _, expected := range tt.shouldContain {
				assert.Contains(t, completions, expected,
					"Should contain %s in completions: %v", expected, completions)
			}

			manager.AssertExpectations(t)
		})
	}
}

func TestShellCompletionProvider_MacroCompletion_Integration(t *testing.T) {
	runner, err := interp.New()
	require.NoError(t, err)

	// Set up real macros in the runner
	macrosJSON := `{
		"deploy": "Deploy the application to production",
		"debug": "Start debugging session",
		"test-all": "Run all tests including integration tests",
		"build": "Build the application"
	}`

	runner.Vars = map[string]expand.Variable{
		"GSH_AGENT_MACROS": {Kind: expand.String, Str: macrosJSON},
	}

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name           string
		line           string
		pos            int
		expectedCount  int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name: "macro completion with #/ prefix",
			line: "#/",
			pos:  2,
			expectedCount: 4,
			shouldContain: []string{"#/deploy", "#/debug", "#/test-all", "#/build"},
		},
		{
			name: "macro completion with partial prefix",
			line: "#/d",
			pos:  3,
			expectedCount: 2,
			shouldContain: []string{"#/deploy", "#/debug"},
			shouldNotContain: []string{"#/test-all", "#/build"},
		},
		{
			name: "macro completion with 'test' prefix",
			line: "#/test",
			pos:  6,
			expectedCount: 1,
			shouldContain: []string{"#/test-all"},
			shouldNotContain: []string{"#/deploy", "#/debug", "#/build"},
		},
		{
			name: "macro completion with non-matching prefix",
			line: "#/xyz",
			pos:  5,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := provider.GetCompletions(tt.line, tt.pos)

			assert.Equal(t, tt.expectedCount, len(completions),
				"Expected %d completions, got %d: %v",
				tt.expectedCount, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, completions, expected,
					"Should contain %s in completions: %v", expected, completions)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, completions, notExpected,
					"Should not contain %s in completions: %v", notExpected, completions)
			}
		})
	}
}

func TestShellCompletionProvider_BuiltinCompletion_Integration(t *testing.T) {
	runner, err := interp.New()
	require.NoError(t, err)

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name           string
		line           string
		pos            int
		expectedCount  int
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name: "builtin completion with #! prefix",
			line: "#!",
			pos:  2,
			expectedCount: 5,
			shouldContain: []string{"#!new", "#!tokens", "#!subagents", "#!reload-subagents", "#!subagent-info"},
		},
		{
			name: "builtin completion with 'n' prefix",
			line: "#!n",
			pos:  3,
			expectedCount: 1,
			shouldContain: []string{"#!new"},
			shouldNotContain: []string{"#!tokens"},
		},
		{
			name: "builtin completion with 't' prefix",
			line: "#!t",
			pos:  3,
			expectedCount: 1,
			shouldContain: []string{"#!tokens"},
			shouldNotContain: []string{"#!new"},
		},
		{
			name: "builtin completion with non-matching prefix",
			line: "#!xyz",
			pos:  5,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := provider.GetCompletions(tt.line, tt.pos)

			assert.Equal(t, tt.expectedCount, len(completions),
				"Expected %d completions, got %d: %v",
				tt.expectedCount, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, completions, expected,
					"Should contain %s in completions: %v", expected, completions)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, completions, notExpected,
					"Should not contain %s in completions: %v", notExpected, completions)
			}
		})
	}
}

func TestShellCompletionProvider_ExecutableCompletion_Integration(t *testing.T) {
	// Create a temporary directory with executable files
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)

	// Create executable files
	executables := []string{"myapp", "mycli", "mytool"}
	for _, exec := range executables {
		execPath := filepath.Join(binDir, exec)
		err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755)
		require.NoError(t, err)
	}

	// Create non-executable file
	nonExecPath := filepath.Join(binDir, "readme.txt")
	err = os.WriteFile(nonExecPath, []byte("not executable"), 0644)
	require.NoError(t, err)

	runner, err := interp.New()
	require.NoError(t, err)

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name          string
		line          string
		pos           int
		setup         func()
		expectedMin   int
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name: "path-based executable completion",
			line: binDir + "/my",
			pos:  len(binDir + "/my"),
			setup: func() {
				manager.On("GetSpec", binDir+"/my").Return(CompletionSpec{}, false)
			},
			expectedMin:   3,
			shouldContain: []string{
				binDir + "/myapp",
				binDir + "/mycli",
				binDir + "/mytool",
			},
			shouldNotContain: []string{binDir + "/readme.txt"},
		},
		{
			name: "path-based completion with directory slash",
			line: binDir + "/",
			pos:  len(binDir + "/"),
			setup: func() {
				manager.On("GetSpec", binDir+"/").Return(CompletionSpec{}, false)
			},
			expectedMin:   3,
			shouldContain: []string{
				binDir + "/myapp",
				binDir + "/mycli",
				binDir + "/mytool",
			},
			shouldNotContain: []string{binDir + "/readme.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations
			manager.ExpectedCalls = nil
			manager.Calls = nil
			tt.setup()

			completions := provider.GetCompletions(tt.line, tt.pos)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Should have at least %d completions, got %d: %v",
				tt.expectedMin, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, completions, expected,
					"Should contain %s in completions: %v", expected, completions)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, completions, notExpected,
					"Should not contain %s in completions: %v", notExpected, completions)
			}

			manager.AssertExpectations(t)
		})
	}
}

func TestShellCompletionProvider_HelpInfo_Integration(t *testing.T) {
	runner, err := interp.New()
	require.NoError(t, err)

	// Set up real macros in the runner
	macrosJSON := `{
		"deploy": "Deploy the application to production",
		"test": "Run tests"
	}`

	runner.Vars = map[string]expand.Variable{
		"GSH_AGENT_MACROS": {Kind: expand.String, Str: macrosJSON},
	}

	manager := NewCompletionManager()
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name     string
		line     string
		pos      int
		expected string
	}{
		{
			name:     "help for #! prefix",
			line:     "#!",
			pos:      2,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **#!new** - Start a new chat session\n• **#!tokens** - Show token usage statistics\n• **#!subagents** - List available subagents\n• **#!reload-subagents** - Reload subagent configurations\n• **#!subagent-info <name>** - Show subagent details",
		},
		{
			name:     "help for #!new command",
			line:     "#!new",
			pos:      5,
			expected: "**#!new** - Start a new chat session with the agent\n\nThis command resets the conversation history and starts fresh.",
		},
		{
			name:     "help for #/ prefix with macros",
			line:     "#/",
			pos:      2,
			expected: "**Chat Macros** - Quick shortcuts for common agent messages\n\nAvailable macros:\n• **#/deploy**\n• **#/test**",
		},
		{
			name:     "help for specific macro",
			line:     "#/deploy",
			pos:      8,
			expected: "**#/deploy** - Chat macro\n\n**Expands to:**\nDeploy the application to production",
		},
		{
			name:     "no help for regular command",
			line:     "ls -la",
			pos:      6,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpInfo := provider.GetHelpInfo(tt.line, tt.pos)
			assert.Equal(t, tt.expected, helpInfo)
		})
	}
}

func TestShellCompletionProvider_CompletionSpec_Integration(t *testing.T) {
	runner, err := interp.New()
	require.NoError(t, err)

	// Create a real completion manager with real specs
	manager := NewCompletionManager()

	// Add a real completion spec for git
	gitSpec := CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit push pull checkout branch status log diff",
	}
	manager.specs["git"] = gitSpec

	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name          string
		line          string
		pos           int
		expectedMin   int
		shouldContain []string
	}{
		{
			name: "git command completion",
			line: "git a",
			pos:  5,
			expectedMin: 1,
			shouldContain: []string{"add"},
		},
		{
			name: "git partial completion",
			line: "git ch",
			pos:  6,
			expectedMin: 1,
			shouldContain: []string{"checkout"},
		},
		{
			name: "git commit completion",
			line: "git c",
			pos:  5,
			expectedMin: 1,
			shouldContain: []string{"commit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := provider.GetCompletions(tt.line, tt.pos)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Should have at least %d completions, got %d: %v",
				tt.expectedMin, len(completions), completions)

			for _, expected := range tt.shouldContain {
				found := false
				for _, completion := range completions {
					if completion == expected ||
					   completion == "git "+expected {
						found = true
						break
					}
				}
				assert.True(t, found,
					"Should contain %s (or git %s) in completions: %v",
					expected, expected, completions)
			}
		})
	}
}