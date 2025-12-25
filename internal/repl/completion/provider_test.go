package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/interp"
)

// mockRunnerProvider implements RunnerProvider for testing.
type mockRunnerProvider struct {
	runner *interp.Runner
	pwd    string
}

func (m *mockRunnerProvider) Runner() *interp.Runner {
	return m.runner
}

func (m *mockRunnerProvider) GetPwd() string {
	return m.pwd
}

func TestNewProvider(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	require.NotNil(t, p)
	assert.NotNil(t, p.specRegistry)
	assert.Equal(t, rp, p.runnerProvider)
}

func TestProviderGetCompletionsEmpty(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	completions := p.GetCompletions("", 0)
	assert.Empty(t, completions)
}

func TestProviderGetCompletionsWithWordList(t *testing.T) {
	// Create a real runner for the test
	runner, err := interp.New()
	require.NoError(t, err)

	rp := &mockRunnerProvider{pwd: "/tmp", runner: runner}
	p := NewProvider(rp)

	p.RegisterSpec(CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit push pull",
	})

	// Test full word list when completing after command
	completions := p.GetCompletions("git ", 4)
	assert.ElementsMatch(t, []string{"add", "commit", "push", "pull"}, completions)

	// Test filtered completions
	completions = p.GetCompletions("git p", 5)
	assert.ElementsMatch(t, []string{"push", "pull"}, completions)

	// Test more specific filter
	completions = p.GetCompletions("git pu", 6)
	assert.ElementsMatch(t, []string{"push", "pull"}, completions)
}

func TestProviderGetCompletionsFileCompletion(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "provider_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))

	rp := &mockRunnerProvider{pwd: tmpDir}
	p := NewProvider(rp)

	// Test file completion when no command spec exists
	completions := p.GetCompletions("cat ", 4)
	assert.Contains(t, completions, "file1.txt")
	assert.Contains(t, completions, "file2.txt")
	assert.Contains(t, completions, "subdir/")

	// Test with file prefix
	completions = p.GetCompletions("cat file", 8)
	assert.ElementsMatch(t, []string{"file1.txt", "file2.txt"}, completions)
}

func TestProviderGetCompletionsFileCompletionMultipleArgs(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "provider_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))

	rp := &mockRunnerProvider{pwd: tmpDir}
	p := NewProvider(rp)

	// Test file completion after multiple arguments with trailing space
	// This was a bug where "git ls-files " wouldn't complete because
	// the trailing space check was in an else-if that was never reached
	// when len(words) > 1
	completions := p.GetCompletions("git ls-files ", 13)
	assert.Contains(t, completions, "file1.txt")
	assert.Contains(t, completions, "file2.txt")
	assert.Contains(t, completions, "subdir/")

	// Test with even more arguments
	completions = p.GetCompletions("command arg1 arg2 ", 18)
	assert.Contains(t, completions, "file1.txt")
	assert.Contains(t, completions, "file2.txt")
	assert.Contains(t, completions, "subdir/")

	// Test with partial file prefix after multiple args
	completions = p.GetCompletions("git ls-files file", 17)
	assert.ElementsMatch(t, []string{"file1.txt", "file2.txt"}, completions)
}

func TestProviderGetCompletionsBuiltinCommands(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	// Test #! completions
	completions := p.GetCompletions("#!", 2)
	assert.Contains(t, completions, "#!new")
	assert.Contains(t, completions, "#!tokens")

	// Test #!n completions
	completions = p.GetCompletions("#!n", 3)
	assert.Contains(t, completions, "#!new")
	assert.NotContains(t, completions, "#!tokens")

	// Test #!t completions
	completions = p.GetCompletions("#!t", 3)
	assert.Contains(t, completions, "#!tokens")
	assert.NotContains(t, completions, "#!new")
}

func TestProviderGetCompletionsMacros(t *testing.T) {
	// Set up test macros via environment variable
	os.Setenv("GSH_AGENT_MACROS", `{"test": "test message", "hello": "hello world"}`)
	defer os.Unsetenv("GSH_AGENT_MACROS")

	rp := &mockRunnerProvider{pwd: "/tmp", runner: nil}
	p := NewProvider(rp)

	// Test #/ completions
	completions := p.GetCompletions("#/", 2)
	assert.Contains(t, completions, "#/test")
	assert.Contains(t, completions, "#/hello")

	// Test #/t completions
	completions = p.GetCompletions("#/t", 3)
	assert.Contains(t, completions, "#/test")
	assert.NotContains(t, completions, "#/hello")
}

func TestProviderGetHelpInfoBuiltinCommands(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	// Test help for #!new
	help := p.GetHelpInfo("#!new", 5)
	assert.Contains(t, help, "#!new")
	assert.Contains(t, help, "new chat session")

	// Test help for #!tokens
	help = p.GetHelpInfo("#!tokens", 8)
	assert.Contains(t, help, "#!tokens")
	assert.Contains(t, help, "token usage")

	// Test help for partial #!
	help = p.GetHelpInfo("#!", 2)
	assert.Contains(t, help, "Agent Controls")
}

func TestProviderGetHelpInfoMacros(t *testing.T) {
	// Set up test macros via environment variable
	os.Setenv("GSH_AGENT_MACROS", `{"test": "test message", "hello": "hello world"}`)
	defer os.Unsetenv("GSH_AGENT_MACROS")

	rp := &mockRunnerProvider{pwd: "/tmp", runner: nil}
	p := NewProvider(rp)

	// Test help for #/test
	help := p.GetHelpInfo("#/test", 6)
	assert.Contains(t, help, "#/test")
	assert.Contains(t, help, "test message")

	// Test help for #/ (general)
	help = p.GetHelpInfo("#/", 2)
	assert.Contains(t, help, "Chat Macros")
}

func TestProviderGetHelpInfoNoMacros(t *testing.T) {
	// Ensure no macros are set
	os.Unsetenv("GSH_AGENT_MACROS")

	rp := &mockRunnerProvider{pwd: "/tmp", runner: nil}
	p := NewProvider(rp)

	// Test help for #/ with no macros
	help := p.GetHelpInfo("#/", 2)
	assert.Contains(t, help, "No macros are currently configured")
}

func TestProviderIsPathBasedCommand(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	tests := []struct {
		command  string
		expected bool
	}{
		{"/bin/ls", true},
		{"./script.sh", true},
		{"../other/script", true},
		{"~/bin/script", true},
		{"path/to/script", true},
		{"ls", false},
		{"git", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := p.commandCompleter.IsPathBasedCommand(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderGetCurrentWordBoundary(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	tests := []struct {
		name          string
		line          string
		pos           int
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "empty line",
			line:          "",
			pos:           0,
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "single word at start",
			line:          "hello",
			pos:           0,
			expectedStart: 0,
			expectedEnd:   5,
		},
		{
			name:          "single word in middle",
			line:          "hello",
			pos:           2,
			expectedStart: 0,
			expectedEnd:   5,
		},
		{
			name:          "two words, cursor on first",
			line:          "hello world",
			pos:           3,
			expectedStart: 0,
			expectedEnd:   5,
		},
		{
			name:          "two words, cursor on second",
			line:          "hello world",
			pos:           8,
			expectedStart: 6,
			expectedEnd:   11,
		},
		{
			name:          "cursor at space",
			line:          "hello world",
			pos:           5,
			expectedStart: 0,
			expectedEnd:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := p.getCurrentWordBoundary(tt.line, tt.pos)
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}

// TestProviderImplementsInterface verifies that Provider implements the input.CompletionProvider interface.
func TestProviderImplementsInterface(t *testing.T) {
	rp := &mockRunnerProvider{pwd: "/tmp"}
	p := NewProvider(rp)

	// These method calls verify the interface is implemented correctly
	_ = p.GetCompletions("test", 4)
	_ = p.GetHelpInfo("test", 4)
}
