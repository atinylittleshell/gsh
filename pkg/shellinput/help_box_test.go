package shellinput

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockHelpCompletionProvider implements CompletionProvider for testing help box functionality
type mockHelpCompletionProvider struct{}

func (m *mockHelpCompletionProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	switch line {
	case "@!":
		return []CompletionCandidate{
			{Value: "@!new"},
			{Value: "@!tokens"},
		}
	case "@/":
		return []CompletionCandidate{
			{Value: "@/test"},
		}
	default:
		return []CompletionCandidate{}
	}
}

func (m *mockHelpCompletionProvider) GetHelpInfo(line string, pos int) string {
	switch line {
	case "@!":
		return "**Agent Controls** - Built-in commands for managing the agent"
	case "@!new":
		return "**@!new** - Start a new chat session with the agent"
	case "@!tokens":
		return "**@!tokens** - Display token usage statistics"
	case "@!n":
		return "**Agent Controls** - Built-in commands for managing the agent"
	case "@/":
		return "**Chat Macros** - Quick shortcuts for common agent messages"
	case "@/test":
		return "**@/test** - Chat macro\n\n**Expands to:**\nThis is a test macro"
	case "@/t":
		return "**Chat Macros** - Quick shortcuts for common agent messages"
	default:
		return ""
	}
}

type mockSuggestionHelpProvider struct{}

func (m *mockSuggestionHelpProvider) GetCompletions(line string, pos int) []CompletionCandidate {
	if strings.HasPrefix(line, "ls") {
		return []CompletionCandidate{{Value: "ls -la"}}
	}

	return []CompletionCandidate{}
}

func (m *mockSuggestionHelpProvider) GetHelpInfo(line string, pos int) string {
	switch line {
	case "ls -la":
		return "**ls -la** - Lists all files in long format including hidden files"
	case "ls":
		return "**ls** - List directory contents"
	default:
		return ""
	}
}

func TestHelpBoxIntegration(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedHelp string
	}{
		{
			name:         "help box for @! command",
			input:        "@!",
			expectedHelp: "**Agent Controls** - Built-in commands for managing the agent",
		},
		{
			name:         "help box for @!new command",
			input:        "@!new",
			expectedHelp: "**@!new** - Start a new chat session with the agent",
		},
		{
			name:         "help box for @/ macro",
			input:        "@/",
			expectedHelp: "**Chat Macros** - Quick shortcuts for common agent messages",
		},
		{
			name:         "help box for specific macro",
			input:        "@/test",
			expectedHelp: "**@/test** - Chat macro\n\n**Expands to:**\nThis is a test macro",
		},
		{
			name:         "no help box for regular command",
			input:        "ls",
			expectedHelp: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := New()
			model.Focus()
			model.CompletionProvider = &mockHelpCompletionProvider{}

			// Set the input value
			model.SetValue(tt.input)
			model.SetCursor(len(tt.input))

			// Simulate a key press to trigger help update
			// We use a simple character input that doesn't change the text
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})

			// Check if help box shows the expected content
			helpBox := model.HelpBoxView()
			assert.Equal(t, tt.expectedHelp, helpBox, "Help box content should match expected")

			// Verify help box visibility
			if tt.expectedHelp != "" {
				assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible")
			} else {
				assert.False(t, model.completion.shouldShowHelpBox(), "Help box should not be visible")
			}
		})
	}
}

func TestHelpBoxWithMacroEnvironment(t *testing.T) {
	// Set up test environment with macros
	t.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro", "help": "Show help information"}`)

	model := New()
	model.Focus()

	// Use a provider that reads from environment (similar to real usage)
	model.CompletionProvider = &mockHelpCompletionProvider{}

	// Test that help box can be displayed
	model.SetValue("@/")
	model.SetCursor(2)

	// Trigger help update
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})

	helpBox := model.HelpBoxView()
	assert.NotEmpty(t, helpBox, "Help box should show content for macros")
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible")
}

func TestHelpBoxSpecificCommandsAndMacros(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedHelp string
		description  string
	}{
		{
			name:         "specific agent control @!new",
			input:        "@!new",
			expectedHelp: "**@!new** - Start a new chat session with the agent",
			description:  "Should show specific help for the 'new' command",
		},
		{
			name:         "specific agent control @!tokens",
			input:        "@!tokens",
			expectedHelp: "**@!tokens** - Display token usage statistics",
			description:  "Should show specific help for the 'tokens' command",
		},
		{
			name:         "partial agent control @!n",
			input:        "@!n",
			expectedHelp: "**Agent Controls** - Built-in commands for managing the agent",
			description:  "Should show general help for partial matches",
		},
		{
			name:         "specific macro @/test",
			input:        "@/test",
			expectedHelp: "**@/test** - Chat macro\n\n**Expands to:**\nThis is a test macro",
			description:  "Should show specific macro expansion",
		},
		{
			name:         "partial macro @/t",
			input:        "@/t",
			expectedHelp: "**Chat Macros** - Quick shortcuts for common agent messages",
			description:  "Should show macro help for partial matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := New()
			model.Focus()
			model.CompletionProvider = &mockHelpCompletionProvider{}

			// Set the input value
			model.SetValue(tt.input)
			model.SetCursor(len(tt.input))

			// Simulate a key press to trigger help update
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("")})

			// Check if help box shows the expected content
			helpBox := model.HelpBoxView()
			assert.Contains(t, helpBox, tt.expectedHelp, tt.description)
			assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible for "+tt.input)
		})
	}
}

func TestHelpBoxUpdatesOnCompletionNavigation(t *testing.T) {
	// Set up test environment with macros
	t.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro"}`)

	model := New()
	model.Focus()
	model.CompletionProvider = &mockHelpCompletionProvider{}

	// Start with @! to trigger agent control completions
	model.SetValue("@!")
	model.SetCursor(2)

	// Simulate TAB to start completion (ambiguous, so no selection yet)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Simulate another TAB to navigate to the first completion (@!new)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Check that help shows specific help for @!new
	helpBox := model.HelpBoxView()
	assert.Contains(t, helpBox, "**@!new**", "Should show specific help for @!new after selecting first completion")
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible")

	// Simulate another TAB to navigate to next completion (@!tokens)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Check that help updates to show specific help for @!tokens
	helpBox = model.HelpBoxView()
	assert.Contains(t, helpBox, "**@!tokens**", "Should show specific help for @!tokens after second selection")
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should still be visible")
}

func TestHelpBoxFollowsMatchedSuggestions(t *testing.T) {
	model := New()
	model.Focus()
	model.ShowSuggestions = true
	model.CompletionProvider = &mockSuggestionHelpProvider{}

	model.SetValue("ls")
	model.SetCursor(len("ls"))

	model.SetSuggestions([]string{"ls -la"})
	model.UpdateHelpInfo()

	helpBox := model.HelpBoxView()
	assert.Equal(t, "**ls -la** - Lists all files in long format including hidden files", helpBox)
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible when suggestions provide help info")

	model.SetSuggestions([]string{})
	model.UpdateHelpInfo()

	helpBox = model.HelpBoxView()
	assert.Equal(t, "**ls** - List directory contents", helpBox)
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should remain visible when falling back to buffer help")
}

func TestHelpBoxPrefersBufferHelpWhileSuggestionsSuppressed(t *testing.T) {
	model := New()
	model.Focus()
	model.ShowSuggestions = true
	model.CompletionProvider = &mockSuggestionHelpProvider{}

	model.SetValue("ls -la")
	model.SetCursor(len("ls"))

	model.SetSuggestions([]string{"ls -la"})
	model.UpdateHelpInfo()

	helpBox := model.HelpBoxView()
	assert.Equal(t, "**ls -la** - Lists all files in long format including hidden files", helpBox)

	model.deleteAfterCursor()

	// Simulate predictions arriving while suggestions are suppressed.
	model.SetSuggestions([]string{"ls -la"})
	model.matchedSuggestions = [][]rune{[]rune("ls -la")}
	model.UpdateHelpInfo()

	helpBox = model.HelpBoxView()
	assert.Equal(t, "**ls** - List directory contents", helpBox)
	assert.True(t, model.SuggestionsSuppressedUntilInput())
}
