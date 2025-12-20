package shellinput

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realCompletionProvider implements CompletionProvider for integration testing
type realCompletionProvider struct {
	completions map[string][]string
	helpInfo    map[string]string
}

func newRealCompletionProvider() *realCompletionProvider {
	return &realCompletionProvider{
		completions: map[string][]string{
			"git":    {"git add", "git commit", "git push", "git pull", "git status"},
			"git a":  {"git add"},
			"git c":  {"git commit"},
			"docker": {"docker run", "docker build", "docker ps", "docker images"},
			"ls":     {"ls -la", "ls -l", "ls -a"},
			"cd":     {}, // Will be populated dynamically with directories
			"@/":     {"@/deploy", "@/test", "@/build"},
			"@/d":    {"@/deploy"},
			"@/t":    {"@/test"},
			"@!":     {"@!new", "@!tokens"},
			"@!n":    {"@!new"},
			"@!t":    {"@!tokens"},
		},
		helpInfo: map[string]string{
			"@!":       "**Agent Controls** - Built-in commands\n\nAvailable:\n• @!new\n• @!tokens",
			"@!new":    "**@!new** - Start new session",
			"@!tokens": "**@!tokens** - Show token usage",
			"@/":       "**Chat Macros** - Quick shortcuts\n\nAvailable:\n• @/deploy\n• @/test\n• @/build",
			"@/deploy": "**@/deploy** - Deploy application",
			"@/test":   "**@/test** - Run tests",
		},
	}
}

func (r *realCompletionProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	// Extract the part of the line up to the cursor
	inputLine := line[:pos]

	var result []string

	// Try exact match first
	if completions, ok := r.completions[inputLine]; ok {
		result = completions
	} else {
		// Try prefix matching
		for prefix, completions := range r.completions {
			if len(inputLine) >= len(prefix) && inputLine[:len(prefix)] == prefix {
				result = completions
				break
			}
		}
	}

	// For file/directory completions (cd command)
	if len(result) == 0 && len(inputLine) >= 3 && inputLine[:2] == "cd" {
		// Create some mock directories for testing
		result = []string{"dir1/", "dir2/", "documents/", "downloads/"}
	}

	if len(result) > 0 {
		candidates := make([]CompletionCandidate, len(result))
		for i, s := range result {
			candidates[i] = CompletionCandidate{Value: s}
		}
		return candidates
	}

	return []CompletionCandidate{}
}

func (r *realCompletionProvider) GetHelpInfo(line string, pos int) string {
	inputLine := line[:pos]

	if helpInfo, ok := r.helpInfo[inputLine]; ok {
		return helpInfo
	}

	return ""
}

// fileSystemCompletionProvider wraps realCompletionProvider to add file system operations
type fileSystemCompletionProvider struct {
	baseProvider *realCompletionProvider
	tmpDir       string
}

func (f *fileSystemCompletionProvider) GetHelpInfo(line string, pos int) string {
	return f.baseProvider.GetHelpInfo(line, pos)
}

func (f *fileSystemCompletionProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	inputLine := line[:pos]

	// Handle cd commands with real directory listing
	if len(inputLine) >= 2 && inputLine[:2] == "cd" {
		var prefix string
		if len(inputLine) > 3 {
			prefix = inputLine[3:] // Get the part after "cd "
		}

		entries, err := os.ReadDir(f.tmpDir)
		if err != nil {
			return []CompletionCandidate{}
		}

		var completions []string
		for _, entry := range entries {
			if entry.IsDir() && (prefix == "" || len(entry.Name()) >= len(prefix) && entry.Name()[:len(prefix)] == prefix) {
				completions = append(completions, entry.Name()+"/")
			}
		}

		candidates := make([]CompletionCandidate, len(completions))
		for i, s := range completions {
			candidates[i] = CompletionCandidate{Value: s}
		}
		return candidates
	}

	// Handle file completions for other commands
	if (len(inputLine) >= 4 && inputLine[:4] == "cat ") ||
		(len(inputLine) >= 4 && inputLine[:4] == "vim ") ||
		(len(inputLine) >= 5 && inputLine[:5] == "less ") {

		var prefix string
		if inputLine[len(inputLine)-1] != ' ' {
			// Extract file prefix
			parts := []rune(inputLine)
			start := len(parts) - 1
			for start >= 0 && parts[start] != ' ' {
				start--
			}
			if start >= 0 {
				prefix = string(parts[start+1:])
			}
		}

		entries, err := os.ReadDir(f.tmpDir)
		if err != nil {
			return []CompletionCandidate{}
		}

		var completions []string
		for _, entry := range entries {
			name := entry.Name()
			if prefix == "" || (len(name) >= len(prefix) && name[:len(prefix)] == prefix) {
				if entry.IsDir() {
					completions = append(completions, name+"/")
				} else {
					completions = append(completions, name)
				}
			}
		}

		candidates := make([]CompletionCandidate, len(completions))
		for i, s := range completions {
			candidates[i] = CompletionCandidate{Value: s}
		}
		return candidates
	}

	// Fall back to base provider for non-file completions
	return f.baseProvider.GetCompletions(line, pos)
}

func TestModel_RealCompletion_Integration(t *testing.T) {
	model := New()
	model.Focus()
	model.CompletionProvider = newRealCompletionProvider()

	tests := []struct {
		name           string
		initialValue   string
		cursorPos      int
		keySequence    []tea.KeyMsg
		expectedValue  string
		expectedCursor int
		expectedActive bool
	}{
		{
			name:         "basic git completion",
			initialValue: "git",
			cursorPos:    3,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},
			},
			expectedValue:  "git ",
			expectedCursor: 4,
			expectedActive: true,
		},
		{
			name:         "cycle through git completions",
			initialValue: "git",
			cursorPos:    3,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab}, // shared prefix
				{Type: tea.KeyTab}, // git add
				{Type: tea.KeyTab}, // git commit
				{Type: tea.KeyTab}, // git push
			},
			expectedValue:  "git push",
			expectedCursor: 8,
			expectedActive: true,
		},
		{
			name:         "completion with partial match",
			initialValue: "git c",
			cursorPos:    5,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},
			},
			expectedValue:  "git commit",
			expectedCursor: 10,
			expectedActive: true,
		},
		{
			name:         "backward completion with shift+tab",
			initialValue: "git",
			cursorPos:    3,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},      // shared prefix
				{Type: tea.KeyTab},      // git add
				{Type: tea.KeyShiftTab}, // back to git status (previous in cycle)
			},
			expectedValue:  "git status",
			expectedCursor: 10,
			expectedActive: true,
		},
		{
			name:         "completion reset on typing",
			initialValue: "git",
			cursorPos:    3,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},                       // shared prefix
				{Type: tea.KeyRunes, Runes: []rune{' '}}, // add space
			},
			expectedValue:  "git  ",
			expectedCursor: 5,
			expectedActive: false,
		},
		{
			name:         "macro completion",
			initialValue: "@/",
			cursorPos:    2,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},
				{Type: tea.KeyTab},
			},
			expectedValue:  "@/deploy",
			expectedCursor: 8,
			expectedActive: true,
		},
		{
			name:         "builtin completion",
			initialValue: "@!",
			cursorPos:    2,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},
				{Type: tea.KeyTab},
			},
			expectedValue:  "@!new",
			expectedCursor: 5,
			expectedActive: true,
		},
		{
			name:         "no completion available",
			initialValue: "unknown",
			cursorPos:    7,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyTab},
			},
			expectedValue:  "unknown",
			expectedCursor: 7,
			expectedActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model state
			model = New()
			model.Focus()
			model.CompletionProvider = newRealCompletionProvider()
			model.SetValue(tt.initialValue)
			model.SetCursor(tt.cursorPos)

			// Apply key sequence
			for _, key := range tt.keySequence {
				updatedModel, _ := model.Update(key)
				model = updatedModel
			}

			assert.Equal(t, tt.expectedValue, model.Value(),
				"Expected value %q, got %q", tt.expectedValue, model.Value())
			assert.Equal(t, tt.expectedCursor, model.Position(),
				"Expected cursor at %d, got %d", tt.expectedCursor, model.Position())
			assert.Equal(t, tt.expectedActive, model.completion.active,
				"Expected completion active: %v, got %v", tt.expectedActive, model.completion.active)
		})
	}
}

func TestModel_FileCompletion_Integration(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create test files and directories
	testFiles := []string{"file1.txt", "file2.log", "script.sh"}
	testDirs := []string{"documents", "downloads", "projects"}

	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)
	}

	for _, dir := range testDirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	// Create a file system completion provider
	fileSystemProvider := &fileSystemCompletionProvider{
		baseProvider: newRealCompletionProvider(),
		tmpDir:       tmpDir,
	}

	model := New()
	model.Focus()
	model.CompletionProvider = fileSystemProvider

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tmpDir)

	tests := []struct {
		name           string
		initialValue   string
		cursorPos      int
		keyPresses     int
		expectedSubstr []string // Substrings that should be present in completions
	}{
		{
			name:           "directory completion with cd",
			initialValue:   "cd ",
			cursorPos:      3,
			keyPresses:     2,
			expectedSubstr: []string{"documents/", "downloads/", "projects/"},
		},
		{
			name:           "file completion with cat",
			initialValue:   "cat ",
			cursorPos:      4,
			keyPresses:     2,
			expectedSubstr: []string{"documents/", "downloads/", "projects/", "file1.txt", "file2.log", "script.sh"}, // Accept any completion
		},
		{
			name:           "partial file completion",
			initialValue:   "vim file",
			cursorPos:      8,
			keyPresses:     2,
			expectedSubstr: []string{"file1.txt"},
		},
		{
			name:           "directory completion with prefix",
			initialValue:   "cd d",
			cursorPos:      4,
			keyPresses:     2,
			expectedSubstr: []string{"documents/", "downloads/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model
			model = New()
			model.Focus()
			model.CompletionProvider = fileSystemProvider
			model.SetValue(tt.initialValue)
			model.SetCursor(tt.cursorPos)

			// Press TAB to get completions
			for i := 0; i < tt.keyPresses; i++ {
				updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
				model = updatedModel
			}

			// Verify that the completion contains expected substrings
			finalValue := model.Value()
			found := false
			for _, expected := range tt.expectedSubstr {
				if len(finalValue) >= len(expected) {
					// Check if any part of the final value contains the expected substring
					for i := 0; i <= len(finalValue)-len(expected); i++ {
						if finalValue[i:i+len(expected)] == expected {
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}

			assert.True(t, found,
				"Expected final value %q to contain one of %v", finalValue, tt.expectedSubstr)
		})
	}
}

func TestModel_HistoryNavigation_Integration(t *testing.T) {
	model := New()
	model.Focus()

	// Set up history
	historyValues := []string{
		"git status",
		"git add .",
		"git commit -m 'test'",
		"git push origin main",
	}
	model.SetHistoryValues(historyValues)

	tests := []struct {
		name           string
		keySequence    []tea.KeyMsg
		expectedValue  string
		expectedCursor int
	}{
		{
			name: "navigate to first history item",
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyUp},
			},
			expectedValue:  "git status",
			expectedCursor: 10,
		},
		{
			name: "navigate through history",
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyUp}, // git status
				{Type: tea.KeyUp}, // git add .
				{Type: tea.KeyUp}, // git commit -m 'test'
			},
			expectedValue:  "git commit -m 'test'",
			expectedCursor: 20, // Length of string, cursor at end
		},
		{
			name: "navigate to end of history",
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyUp}, // git status
				{Type: tea.KeyUp}, // git add .
				{Type: tea.KeyUp}, // git commit -m 'test'
				{Type: tea.KeyUp}, // git push origin main
				{Type: tea.KeyUp}, // should stay at last item
			},
			expectedValue:  "git push origin main",
			expectedCursor: 20, // Length of string, cursor at end
		},
		{
			name: "navigate forward in history",
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyUp},   // git status
				{Type: tea.KeyUp},   // git add .
				{Type: tea.KeyDown}, // back to git status
			},
			expectedValue:  "git status",
			expectedCursor: 10,
		},
		{
			name: "navigate back to current input",
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune("test input")}, // Set current input
				{Type: tea.KeyUp},   // git status
				{Type: tea.KeyDown}, // back to current input
			},
			expectedValue:  "test input",
			expectedCursor: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model
			model = New()
			model.Focus()
			model.SetHistoryValues(historyValues)

			// Apply key sequence
			for _, key := range tt.keySequence {
				updatedModel, _ := model.Update(key)
				model = updatedModel
			}

			assert.Equal(t, tt.expectedValue, model.Value(),
				"Expected value %q, got %q", tt.expectedValue, model.Value())
			assert.Equal(t, tt.expectedCursor, model.Position(),
				"Expected cursor at %d, got %d", tt.expectedCursor, model.Position())
		})
	}
}

func TestModel_EditOperations_Integration(t *testing.T) {
	tests := []struct {
		name           string
		initialValue   string
		cursorPos      int
		keySequence    []tea.KeyMsg
		expectedValue  string
		expectedCursor int
	}{
		{
			name:         "delete word backward with Ctrl+W",
			initialValue: "git commit -m 'test message'",
			cursorPos:    28,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlW},
			},
			expectedValue:  "git commit -m 'test ",
			expectedCursor: 20,
		},
		{
			name:         "delete word forward with Alt+D",
			initialValue: "git commit -m 'message here'",
			cursorPos:    23, // Between 'message' and 'here'
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true},
			},
			expectedValue:  "git commit -m 'message ",
			expectedCursor: 23,
		},
		{
			name:         "move to beginning with Ctrl+A",
			initialValue: "long command line here",
			cursorPos:    22,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlA},
			},
			expectedValue:  "long command line here",
			expectedCursor: 0,
		},
		{
			name:         "move to end with Ctrl+E",
			initialValue: "some command",
			cursorPos:    5,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlE},
			},
			expectedValue:  "some command",
			expectedCursor: 12,
		},
		{
			name:         "delete to end of line with Ctrl+K",
			initialValue: "git commit -m 'delete this part'",
			cursorPos:    15, // After -m '
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlK},
			},
			expectedValue:  "git commit -m '",
			expectedCursor: 15,
		},
		{
			name:         "delete to beginning of line with Ctrl+U",
			initialValue: "delete this part and keep this",
			cursorPos:    20, // Before "and keep this"
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlU},
			},
			expectedValue:  " keep this",
			expectedCursor: 0,
		},
		{
			name:         "complex editing sequence",
			initialValue: "git commit -m 'wrong message here'",
			cursorPos:    34,
			keySequence: []tea.KeyMsg{
				{Type: tea.KeyCtrlW},                            // delete "here'"
				{Type: tea.KeyRunes, Runes: []rune("correct'")}, // add "correct'"
			},
			expectedValue:  "git commit -m 'wrong message correct'",
			expectedCursor: 37,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := New()
			model.Focus()
			model.SetValue(tt.initialValue)
			model.SetCursor(tt.cursorPos)

			// Apply key sequence
			for _, key := range tt.keySequence {
				updatedModel, _ := model.Update(key)
				model = updatedModel
			}

			assert.Equal(t, tt.expectedValue, model.Value(),
				"Expected value %q, got %q", tt.expectedValue, model.Value())
			assert.Equal(t, tt.expectedCursor, model.Position(),
				"Expected cursor at %d, got %d", tt.expectedCursor, model.Position())
		})
	}
}

func TestModel_HelpInfo_Integration(t *testing.T) {
	provider := newRealCompletionProvider()

	model := New()
	model.Focus()
	model.CompletionProvider = provider

	tests := []struct {
		name         string
		initialValue string
		cursorPos    int
		expectedHelp string
	}{
		{
			name:         "help for @! prefix",
			initialValue: "@!",
			cursorPos:    2,
			expectedHelp: "**Agent Controls** - Built-in commands\n\nAvailable:\n• @!new\n• @!tokens",
		},
		{
			name:         "help for specific builtin command",
			initialValue: "@!new",
			cursorPos:    5,
			expectedHelp: "**@!new** - Start new session",
		},
		{
			name:         "help for macro prefix",
			initialValue: "@/",
			cursorPos:    2,
			expectedHelp: "**Chat Macros** - Quick shortcuts\n\nAvailable:\n• @/deploy\n• @/test\n• @/build",
		},
		{
			name:         "help for specific macro",
			initialValue: "@/deploy",
			cursorPos:    8,
			expectedHelp: "**@/deploy** - Deploy application",
		},
		{
			name:         "no help for regular commands",
			initialValue: "git status",
			cursorPos:    10,
			expectedHelp: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.SetValue(tt.initialValue)
			model.SetCursor(tt.cursorPos)

			// Update help info
			model.updateHelpInfo()

			actualHelp := model.HelpBoxView()
			if tt.expectedHelp == "" {
				assert.Empty(t, actualHelp, "Expected no help info, but got: %s", actualHelp)
			} else {
				assert.Contains(t, actualHelp, tt.expectedHelp,
					"Expected help to contain %q, but got %q", tt.expectedHelp, actualHelp)
			}
		})
	}
}
