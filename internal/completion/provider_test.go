package completion

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Mock getFileCompletions for testing
var mockGetFileCompletions fileCompleter = func(prefix, currentDirectory string) []shellinput.CompletionCandidate {
	switch prefix {
	case "some/pa":
		return []shellinput.CompletionCandidate{
			{Value: "some/path.txt"},
			{Value: "some/path2.txt"},
		}
	case "/usr/local/b":
		return []shellinput.CompletionCandidate{
			{Value: "/usr/local/bin", Suffix: "/"},
			{Value: "/usr/local/bin/"},
		}
	case "'my documents/som":
		return []shellinput.CompletionCandidate{
			{Value: "my documents/something.txt"},
			{Value: "my documents/somefile.txt"},
		}
	case "":
		// Empty prefix means list everything in current directory
		// Note: On Windows, os.PathSeparator is '\', on Unix it's '/'
		return []shellinput.CompletionCandidate{
			{Value: "folder1", Suffix: string(os.PathSeparator)},
			{Value: "folder2", Suffix: string(os.PathSeparator)},
			{Value: "file1.txt"},
			{Value: "file2.txt"},
		}
	case "foo/bar/b":
		return []shellinput.CompletionCandidate{
			{Value: "foo/bar/baz"},
			{Value: "foo/bar/bin"},
		}
	case "other/path/te":
		return []shellinput.CompletionCandidate{
			{Value: "other/path/test.txt"},
			{Value: "other/path/temp.txt"},
		}
	case "/bin/":
		// Mock some common executables for testing, independent of actual system
		return []shellinput.CompletionCandidate{
			{Value: "/bin/bash"},
			{Value: "/bin/cat"},
			{Value: "/bin/ls"},
			{Value: "/bin/sh"},
		}
	default:
		// No match found
		return []shellinput.CompletionCandidate{}
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

func (m *mockCompletionManager) ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string, line string, pos int) ([]shellinput.CompletionCandidate, error) {
	callArgs := m.Called(ctx, runner, spec, args)
	return callArgs.Get(0).([]shellinput.CompletionCandidate), callArgs.Error(1)
}

// Mock osReadDir for testing
var mockOsReadDir = func(name string) ([]os.DirEntry, error) {
	// On Windows, /bin paths don't exist natively, so we mock them specifically
	// On Unix, we also mock them to ensure test stability
	if name == "/bin" || name == "/bin/" || name == "\\bin" || name == "\\bin\\" {
		// Return mock directory entries for /bin
		return []os.DirEntry{
			&mockDirEntry{name: "bash", isDir: false, mode: 0755},
			&mockDirEntry{name: "cat", isDir: false, mode: 0755},
			&mockDirEntry{name: "ls", isDir: false, mode: 0755},
			&mockDirEntry{name: "sh", isDir: false, mode: 0755},
		}, nil
	}

	// For other paths, return empty to avoid system dependencies
	return []os.DirEntry{}, nil
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
	t.Setenv("GSH_AGENT_MACROS", `{"macro1": {}, "macro2": {}, "macro3": {}}`)

	// Create a proper runner with the macros variable
	runner, _ := interp.New(interp.StdIO(nil, nil, nil))
	runner.Vars = map[string]expand.Variable{
		"GSH_AGENT_MACROS": {Kind: expand.String, Str: `{"macro1": {}, "macro2": {}, "macro3": {}}`},
	}

	manager := &mockCompletionManager{}
	provider := NewShellCompletionProvider(manager, runner)

	// Helper to determine expected /bin/ completions based on OS
	var binCompletions []string
	if runtime.GOOS == "windows" {
		// On Windows, paths will be normalized with backslashes
		// Using filepath.Join to construct expected Windows paths
		binCompletions = []string{
			filepath.Join("\\bin", "bash"),
			filepath.Join("\\bin", "cat"),
			filepath.Join("\\bin", "ls"),
			filepath.Join("\\bin", "sh"),
		}
	} else {
		binCompletions = []string{"/bin/bash", "/bin/cat", "/bin/ls", "/bin/sh"}
	}

	tests := []struct {
		name     string
		line     string
		pos      int
		setup    func()
		expected []shellinput.CompletionCandidate
	}{
		{
			name: "empty line returns no completions",
			line: "",
			pos:  0,
			setup: func() {
				// no setup needed
			},
			expected: []shellinput.CompletionCandidate{},
		},
		{
			name: "command with no completion spec returns no completions",
			line: "unknown-command arg1",
			pos:  20,
			setup: func() {
				manager.On("GetSpec", "unknown-command").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{},
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
					Return([]shellinput.CompletionCandidate{
						{Value: "checkout"},
						{Value: "cherry-pick"},
					}, nil)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "checkout"},
				{Value: "cherry-pick"},
			},
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
					Return([]shellinput.CompletionCandidate{
						{Value: "checkout"},
						{Value: "cherry-pick"},
					}, nil)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "checkout"},
				{Value: "cherry-pick"},
			},
		},
		{
			name: "file completion preserves command and path prefix",
			line: "cat some/pa",
			pos:  11,
			setup: func() {
				manager.On("GetSpec", "cat").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "some/path.txt"},
				{Value: "some/path2.txt"},
			},
		},
		{
			name: "file completion with multiple path segments",
			line: "vim /usr/local/bi",
			pos:  16,
			setup: func() {
				manager.On("GetSpec", "vim").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "/usr/local/bin", Suffix: "/"},
				{Value: "/usr/local/bin/"},
			},
		},
		{
			name: "file completion with spaces in path",
			line: "less 'my documents/some",
			pos:  22,
			setup: func() {
				manager.On("GetSpec", "less").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "\"my documents/something.txt\""},
				{Value: "\"my documents/somefile.txt\""},
			},
		},
		{
			name: "file completion after command with space",
			line: "cd ",
			pos:  3,
			setup: func() {
				manager.On("GetSpec", "cd").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "folder1", Suffix: string(os.PathSeparator), Description: "Directory"},
				{Value: "folder2", Suffix: string(os.PathSeparator), Description: "Directory"},
			},
		},
		{
			name: "file completion after command with multiple spaces",
			line: "cd   ",
			pos:  5,
			setup: func() {
				manager.On("GetSpec", "cd").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "folder1", Suffix: string(os.PathSeparator), Description: "Directory"},
				{Value: "folder2", Suffix: string(os.PathSeparator), Description: "Directory"},
			},
		},
		{
			name: "file completion with multiple path segments should only replace last segment",
			line: "ls foo/bar/b",
			pos:  12,
			setup: func() {
				manager.On("GetSpec", "ls").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "foo/bar/baz"},
				{Value: "foo/bar/bin"},
			},
		},
		{
			name: "file completion with multiple arguments should preserve earlier arguments",
			line: "ls some/path other/path/te",
			pos:  26,
			setup: func() {
				manager.On("GetSpec", "ls").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "other/path/test.txt"},
				{Value: "other/path/temp.txt"},
			},
		},
		{
			name: "macro completion with @/ prefix",
			line: "@/mac",
			pos:  5,
			setup: func() {
				// No setup needed - macro completion doesn't depend on manager
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@/macro1"},
				{Value: "@/macro2"},
				{Value: "@/macro3"},
			},
		},
		{
			name: "builtin command completion with @! prefix",
			line: "@!n",
			pos:  3,
			setup: func() {
				// No setup needed - builtin completion doesn't depend on manager
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@!new"},
			},
		},
		{
			name: "partial macro match should complete to macro, not fall back",
			line: "@/m",
			pos:  3,
			setup: func() {
				// No setup needed - should match macros
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@/macro1"},
				{Value: "@/macro2"},
				{Value: "@/macro3"},
			},
		},
		{
			name: "partial builtin match should complete to builtin, not fall back",
			line: "@!t",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtins
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@!tokens"},
			},
		},
		{
			name: "subagent commands completion with 's' prefix",
			line: "@!s",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtin subagent commands
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@!subagents"},
			},
		},
		{
			name: "reload-subagents completion with 'r' prefix",
			line: "@!r",
			pos:  3,
			setup: func() {
				// No setup needed - should match builtin reload command
			},
			expected: []shellinput.CompletionCandidate{
				{Value: "@!reload-subagents"},
			},
		},
		{
			name: "path-based command completion with ./",
			line: "./",
			pos:  2,
			setup: func() {
				// Mock GetSpec to return no completion spec for path-based commands
				manager.On("GetSpec", "./").Return(CompletionSpec{}, false)
			},
			expected: []shellinput.CompletionCandidate{}, // Will depend on actual executable files in current directory
		},
		{
			name: "path-based command completion with /bin/",
			line: "/bin/",
			pos:  5,
			setup: func() {
				// Mock GetSpec to return no completion spec for path-based commands
				manager.On("GetSpec", "/bin/").Return(CompletionSpec{}, false)
			},
			expected: func() []shellinput.CompletionCandidate {
				candidates := make([]shellinput.CompletionCandidate, len(binCompletions))
				for i, path := range binCompletions {
					candidates[i] = shellinput.CompletionCandidate{Value: path}
				}
				return candidates
			}(),
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
			expected: []shellinput.CompletionCandidate{
				{Value: "test123"},
				{Value: "testfoo"},
			},
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
			expected: []shellinput.CompletionCandidate{
				{Value: "test123"},
			},
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
			expected: []shellinput.CompletionCandidate{},
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
		_ = runner.Run(context.Background(), prog)
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
			name:     "help for @! empty",
			line:     "@!",
			pos:      2,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **@!config** - Open the configuration menu\n• **@!new** - Start a new chat session\n• **@!tokens** - Show token usage statistics\n• **@!subagents [name]** - List subagents or show details\n• **@!reload-subagents** - Reload subagent configurations\n• **@!coach [subcommand]** - Productivity coach (stats, achievements, challenges, tips, reset-tips)",
		},
		{
			name:     "help for @!new",
			line:     "@!new",
			pos:      5,
			expected: "**@!new** - Start a new chat session with the agent\n\nThis command resets the conversation history and starts fresh.",
		},
		{
			name:     "help for @!tokens",
			line:     "@!tokens",
			pos:      8,
			expected: "**@!tokens** - Display token usage statistics\n\nShows information about token consumption for the current chat session.",
		},
		{
			name:     "help for @/ empty (no macros)",
			line:     "@/",
			pos:      2,
			expected: "**Chat Macros** - Quick shortcuts for common agent messages\n\nNo macros are currently configured.",
		},
		{
			name:     "help for partial @!n (matches new)",
			line:     "@!n",
			pos:      3,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **@!config** - Open the configuration menu\n• **@!new** - Start a new chat session\n• **@!tokens** - Show token usage statistics\n• **@!subagents [name]** - List subagents or show details\n• **@!reload-subagents** - Reload subagent configurations\n• **@!coach [subcommand]** - Productivity coach (stats, achievements, challenges, tips, reset-tips)",
		},
		{
			name:     "help for partial @!t (matches tokens)",
			line:     "@!t",
			pos:      3,
			expected: "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:\n• **@!config** - Open the configuration menu\n• **@!new** - Start a new chat session\n• **@!tokens** - Show token usage statistics\n• **@!subagents [name]** - List subagents or show details\n• **@!reload-subagents** - Reload subagent configurations\n• **@!coach [subcommand]** - Productivity coach (stats, achievements, challenges, tips, reset-tips)",
		},
		{
			name:     "help for @!subagents",
			line:     "@!subagents",
			pos:      11,
			expected: "**@!subagents [name]** - List subagents or show details about a specific one\n\nWithout arguments, displays all configured Claude-style subagents and Roo Code-style modes. With a subagent name, shows detailed information including tools, file restrictions, and configuration.",
		},
		{
			name:     "help for @!reload-subagents",
			line:     "@!reload-subagents",
			pos:      18,
			expected: "**@!reload-subagents** - Reload subagent configurations from disk\n\nRefreshes the subagent configurations by rescanning the .claude/agents/ and .roo/modes/ directories.",
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
	t.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro", "help": "Show help information"}`)

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
			name:     "help for @/ with macros",
			line:     "@/",
			pos:      2,
			expected: "**Chat Macros** - Quick shortcuts for common agent messages\n\nAvailable macros:\n• **@/help**\n• **@/test**",
		},
		{
			name:     "help for specific macro",
			line:     "@/test",
			pos:      6,
			expected: "**@/test** - Chat macro\n\n**Expands to:**\nThis is a test macro",
		},
		{
			name:     "help for partial macro match",
			line:     "@/t",
			pos:      3,
			expected: "**Chat Macros** - Matching macros:\n\n• **@/test** - This is a test macro",
		},
		{
			name:     "help for partial macro match with multiple results",
			line:     "@/he",
			pos:      4,
			expected: "**Chat Macros** - Matching macros:\n\n• **@/help** - Show help information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.GetHelpInfo(tt.line, tt.pos)
			assert.Equal(t, tt.expected, result)
		})
	}
}
