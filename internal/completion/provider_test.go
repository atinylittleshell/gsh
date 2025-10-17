package completion

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Mock getFileCompletions for testing
var mockGetFileCompletions fileCompleter = func(prefix, currentDirectory string) []string {
	switch prefix {
	case "some/pa":
		return []string{"some/path.txt", "some/path2.txt"}
	case "/usr/local/b":
		return []string{"/usr/local/bin", "/usr/local/bin/"}
	case "'my documents/som":
		return []string{"my documents/something.txt", "my documents/somefile.txt"}
	case "":
		// Empty prefix means list everything in current directory
		return []string{"folder1/", "folder2/", "file1.txt", "file2.txt"}
	case "foo/bar/b":
		return []string{"foo/bar/baz", "foo/bar/bin"}
	case "other/path/te":
		return []string{"other/path/test.txt", "other/path/temp.txt"}
	case "/bin/":
		// Mock some common executables for testing, independent of actual system
		return []string{"/bin/bash", "/bin/cat", "/bin/ls", "/bin/sh"}
	default:
		// No match found
		return []string{}
	}
}

// mockCompletionManager mocks the CompletionManager for testing
type mockCompletionManager struct {
	mock.Mock
}

func (m *mockCompletionManager) GetSpec(command string) (CompletionSpec, bool) {
	args := m.Called(command)
	return args.Get(0).(CompletionSpec), args.Bool(1)
}

func (m *mockCompletionManager) ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string) ([]string, error) {
	callArgs := m.Called(ctx, runner, spec, args)
	return callArgs.Get(0).([]string), callArgs.Error(1)
}

// Mock osReadDir for testing
var mockOsReadDir = func(name string) ([]os.DirEntry, error) {
	switch name {
	case "/bin", "/bin/":
		// Return mock directory entries for /bin
		return []os.DirEntry{
			&mockDirEntry{name: "bash", isDir: false, mode: 0755},
			&mockDirEntry{name: "cat", isDir: false, mode: 0755},
			&mockDirEntry{name: "ls", isDir: false, mode: 0755},
			&mockDirEntry{name: "sh", isDir: false, mode: 0755},
		}, nil
	default:
		// For PATH directories that might contain "test" commands, return empty to avoid system dependencies
		return []os.DirEntry{}, nil
	}
}

// mockDirEntry implements os.DirEntry for testing
type mockDirEntry struct {
	name  string
	isDir bool
	mode  os.FileMode
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return m.mode }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return &mockFileInfo{m.name, m.mode}, nil }

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name string
	mode os.FileMode
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestGetCompletions(t *testing.T) {
	// Replace getFileCompletions with mock for testing
	origGetFileCompletions := getFileCompletions
	getFileCompletions = mockGetFileCompletions
	defer func() {
		getFileCompletions = origGetFileCompletions
	}()

	// Replace osReadDir with mock for testing
	origOsReadDir := osReadDir
	osReadDir = mockOsReadDir
	defer func() {
		osReadDir = origOsReadDir
	}()

	// Set up environment for macro testing
	origMacrosEnv := os.Getenv("GSH_AGENT_MACROS")
	os.Setenv("GSH_AGENT_MACROS", `{"macro1": {}, "macro2": {}, "macro3": {}}`)
	defer func() {
		os.Setenv("GSH_AGENT_MACROS", origMacrosEnv)
	}()

	// Create a proper runner with the macros variable
	runner, _ := interp.New(interp.StdIO(nil, nil, nil))
	runner.Vars = map[string]expand.Variable{
		"GSH_AGENT_MACROS": {Kind: expand.String, Str: `{"macro1": {}, "macro2": {}, "macro3": {}}`},
	}

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	tests := []struct {
		name     string
		line     string
		pos      int
		setup    func()
		expected []string
	}{
		{
			name: "empty line returns no completions",
			line: "",
			pos:  0,
			setup: func() {
				// no setup needed
			},
			expected: []string{},
		},
		{
			name: "command with no completion spec returns no completions",
			line: "unknown-command arg1",
			pos:  20,
			setup: func() {
				manager.On("GetSpec", "unknown-command").Return(CompletionSpec{}, false)
			},
			expected: []string{},
		},
		{
			name: "command with word list completion returns suggestions",
			line: "git ch",
			pos:  6,
			setup: func() {
				spec := CompletionSpec{
					Command: "git",
					Type:    WordListCompletion,
					Value:   "checkout cherry-pick",
				}
				manager.On("GetSpec", "git").Return(spec, true)
				manager.On("ExecuteCompletion", mock.Anything, runner, spec, []string{"git", "ch"}).
					Return([]string{"checkout", "cherry-pick"}, nil)
			},
			expected: []string{"checkout", "cherry-pick"},
		},
		{
			name: "cursor position in middle of line only uses text up to cursor",
			line: "git checkout master",
			pos:  6, // cursor after "git ch"
			setup: func() {
				spec := CompletionSpec{
					Command: "git",
					Type:    WordListCompletion,
					Value:   "checkout cherry-pick",
				}
				manager.On("GetSpec", "git").Return(spec, true)
				manager.On("ExecuteCompletion", mock.Anything, runner, spec, []string{"git", "ch"}).
					Return([]string{"checkout", "cherry-pick"}, nil)
			},
			expected: []string{"checkout", "cherry-pick"},
		},
		{
			name: "file completion preserves command and path prefix",
			line: "cat some/pa",
			pos:  11,
			setup: func() {
				manager.On("GetSpec", "cat").Return(CompletionSpec{}, false)
			},
			expected: []string{"some/path.txt", "some/path2.txt"},
		},
		{
			name: "file completion with multiple path segments",
			line: "vim /usr/local/bi",
			pos:  16,
			setup: func() {
				manager.On("GetSpec", "vim").Return(CompletionSpec{}, false)
			},
			expected: []string{"/usr/local/bin", "/usr/local/bin/"}, // Mocked response, not dependent on actual filesystem
		},
		{
			name: "file completion with spaces in path",
			line: "less 'my documents/some",
			pos:  22,
			setup: func() {
				manager.On("GetSpec", "less").Return(CompletionSpec{}, false)
			},
			expected: []string{"\"my documents/something.txt\"", "\"my documents/somefile.txt\""},
		},
		{
			name: "file completion after command with space",
			line: "cd ",
			pos:  3,
			setup: func() {
				manager.On("GetSpec", "cd").Return(CompletionSpec{}, false)
			},
			expected: []string{"folder1/", "folder2/", "file1.txt", "file2.txt"},
		},
		{
			name: "file completion after command with multiple spaces",
			line: "cd   ",
			pos:  5,
			setup: func() {
				manager.On("GetSpec", "cd").Return(CompletionSpec{}, false)
			},
			expected: []string{"folder1/", "folder2/", "file1.txt", "file2.txt"},
		},
		{
			name: "file completion with multiple path segments should only replace last segment",
			line: "ls foo/bar/b",
			pos:  12,
			setup: func() {
				manager.On("GetSpec", "ls").Return(CompletionSpec{}, false)
			},
			expected: []string{"foo/bar/baz", "foo/bar/bin"},
		},
		{
			name: "file completion with multiple arguments should preserve earlier arguments",
			line: "ls some/path other/path/te",
			pos:  26,
			setup: func() {
				manager.On("GetSpec", "ls").Return(CompletionSpec{}, false)
			},
			expected: []string{"other/path/test.txt", "other/path/temp.txt"},
		},
		{
			name: "macro completion with #/ prefix",
			line: "#/mac",
			pos:  5,
			setup: func() {
				// No setup needed - macro completion doesn't depend on manager
			},
			expected: []string{"#/macro1", "#/macro2", "#/macro3"},
		},
		{
			name: "builtin command completion with #! prefix",
			line: "#!n",
			pos:  3,
			setup: func() {
				// No setup needed - builtin completion doesn't depend on manager
			},
			expected: []string{"#!new"},
		},
		{
			name: "partial macro match should complete to macro, not fall back",
			line: "#/m",
			pos:  3,
			setup: func() {
				// No setup needed - should match macros
			},
			expected: []string{"#/macro1", "#/macro2", "#/macro3"}, // All macros starting with 'm'
		},
		{
			name: "partial builtin match should complete to builtin, not fall back",
			line: "#!t",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtins
			},
			expected: []string{"#!tokens"}, // Only builtin starting with 't'
		},
		{
			name: "subagent commands completion with 's' prefix",
			line: "#!s",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtin subagent commands
			},
			expected: []string{"#!subagent-info", "#!subagents"}, // Both subagent commands starting with 's'
		},
		{
			name: "reload-subagents completion with 'r' prefix",
			line: "#!r",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtin reload command
			},
			expected: []string{"#!reload-subagents"}, // Only reload command starting with 'r'
		},
		{
			name: "path-based command completion with ./",
			line: "./",
			pos:  2,
			setup: func() {
				// Mock GetSpec to return no completion spec for path-based commands
				manager.On("GetSpec", "./").Return(CompletionSpec{}, false)
			},
			expected: []string{}, // Will depend on actual executable files in current directory
		},
		{
			name: "path-based command completion with /bin/",
			line: "/bin/",
			pos:  5,
			setup: func() {
				// Mock GetSpec to return no completion spec for path-based commands
				manager.On("GetSpec", "/bin/").Return(CompletionSpec{}, false)
			},
			expected: []string{"/bin/bash", "/bin/cat", "/bin/ls", "/bin/sh"}, // Mocked executables, independent of actual system
		},
		{
			name: "alias completion with matching prefix",
			line: "test",
			pos:  4,
			setup: func() {
				// Mock GetSpec to return no completion spec
				manager.On("GetSpec", "test").Return(CompletionSpec{}, false)

				// Set up aliases using reflection (simulating aliases in the runner)
				setupTestAliases(runner)
			},
			expected: []string{"test123", "testfoo"}, // Only includes test aliases, not system commands
		},
		{
			name: "alias completion with partial match",
			line: "test1",
			pos:  5,
			setup: func() {
				// Mock GetSpec to return no completion spec
				manager.On("GetSpec", "test1").Return(CompletionSpec{}, false)

				// Set up aliases using reflection
				setupTestAliases(runner)
			},
			expected: []string{"test123"},
		},
		{
			name: "alias completion with no matches falls back to system commands",
			line: "nonexistent",
			pos:  11,
			setup: func() {
				// Mock GetSpec to return no completion spec
				manager.On("GetSpec", "nonexistent").Return(CompletionSpec{}, false)

				// Set up aliases using reflection
				setupTestAliases(runner)
			},
			expected: []string{}, // No aliases or system commands start with "nonexistent"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.ExpectedCalls = nil
			manager.Calls = nil
			tt.setup()

			result := provider.GetCompletions(tt.line, tt.pos)
			assert.Equal(t, tt.expected, result)
			manager.AssertExpectations(t)
		})
	}
}

// setupTestAliases sets up test aliases in the runner by executing alias commands
func setupTestAliases(runner *interp.Runner) {
	// Since we can't directly access the unexported alias field, we'll execute alias commands
	// to set up the aliases in the runner
	aliasCommands := []string{
		"alias test123=ls",
		"alias testfoo='echo hello'",
		"alias myalias=pwd",
	}

	parser := syntax.NewParser()
	for _, cmd := range aliasCommands {
		prog, err := parser.Parse(strings.NewReader(cmd), "")
		if err != nil {
			continue // Skip invalid commands
		}

		// Execute the alias command to set up the alias in the runner
		runner.Run(context.Background(), prog)
	}
}

func TestGetHelpInfo(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		pos      int
		expected string
	}{
		{
			name:     "help for #! empty",
			line:     "#!",
			pos:      2,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **#!new** - Start a new chat session\n• **#!tokens** - Show token usage statistics\n• **#!subagents** - List available subagents\n• **#!reload-subagents** - Reload subagent configurations\n• **#!subagent-info <name>** - Show subagent details",
		},
		{
			name:     "help for #!new",
			line:     "#!new",
			pos:      5,
			expected: "**#!new** - Start a new chat session with the agent\n\nThis command resets the conversation history and starts fresh.",
		},
		{
			name:     "help for #!tokens",
			line:     "#!tokens",
			pos:      8,
			expected: "**#!tokens** - Display token usage statistics\n\nShows information about token consumption for the current chat session.",
		},
		{
			name:     "help for #/ empty (no macros)",
			line:     "#/",
			pos:      2,
			expected: "**Chat Macros** - Quick shortcuts for common agent messages\n\nNo macros are currently configured.",
		},
		{
			name:     "help for partial #!n (matches new)",
			line:     "#!n",
			pos:      3,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **#!new** - Start a new chat session\n• **#!tokens** - Show token usage statistics\n• **#!subagents** - List available subagents\n• **#!reload-subagents** - Reload subagent configurations\n• **#!subagent-info <name>** - Show subagent details",
		},
		{
			name:     "help for partial #!t (matches tokens)",
			line:     "#!t",
			pos:      3,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **#!new** - Start a new chat session\n• **#!tokens** - Show token usage statistics\n• **#!subagents** - List available subagents\n• **#!reload-subagents** - Reload subagent configurations\n• **#!subagent-info <name>** - Show subagent details",
		},
		{
			name:     "help for #!subagents",
			line:     "#!subagents",
			pos:      11,
			expected: "**#!subagents** - List all available subagents and modes\n\nDisplays all configured Claude-style subagents and Roo Code-style modes with their descriptions and capabilities.",
		},
		{
			name:     "help for #!reload-subagents",
			line:     "#!reload-subagents",
			pos:      18,
			expected: "**#!reload-subagents** - Reload subagent configurations from disk\n\nRefreshes the subagent configurations by rescanning the .claude/agents/ and .roo/modes/ directories.",
		},
		{
			name:     "help for #!subagent-info",
			line:     "#!subagent-info",
			pos:      15,
			expected: "**#!subagent-info <name>** - Show detailed information about a subagent\n\nDisplays comprehensive information about a specific subagent including tools, file restrictions, and configuration.",
		},
		{
			name:     "no help for regular command",
			line:     "ls",
			pos:      2,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, _ := interp.New()
			manager := NewCompletionManager()
			provider := NewShellCompletionProvider(manager, runner)

			result := provider.GetHelpInfo(tt.line, tt.pos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHelpInfoWithMacros(t *testing.T) {
	// Set up test macros using environment variable since runner is nil in provider
	os.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro", "help": "Show help information"}`)
	defer os.Unsetenv("GSH_AGENT_MACROS")

	// Use nil runner to force fallback to environment variable
	manager := NewCompletionManager()
	provider := NewShellCompletionProvider(manager, nil)

	tests := []struct {
		name     string
		line     string
		pos      int
		expected string
	}{
		{
			name:     "help for #/ with macros",
			line:     "#/",
			pos:      2,
			expected: "**Chat Macros** - Quick shortcuts for common agent messages\n\nAvailable macros:\n• **#/help**\n• **#/test**",
		},
		{
			name:     "help for specific macro",
			line:     "#/test",
			pos:      6,
			expected: "**#/test** - Chat macro\n\n**Expands to:**\nThis is a test macro",
		},
		{
			name:     "help for partial macro match",
			line:     "#/t",
			pos:      3,
			expected: "**Chat Macros** - Matching macros:\n\n• **#/test** - This is a test macro",
		},
		{
			name:     "help for partial macro match with multiple results",
			line:     "#/he",
			pos:      4,
			expected: "**Chat Macros** - Matching macros:\n\n• **#/help** - Show help information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.GetHelpInfo(tt.line, tt.pos)
			assert.Equal(t, tt.expected, result)
		})
	}
}
