package shellinput

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockHelpCompletionProvider implements CompletionProvider for testing help box functionality
type mockHelpCompletionProvider struct{}

func (m *mockHelpCompletionProvider) GetCompletions(line string, pos int) []string {
	switch line {
	case "@!":
		return []string{"@!new", "@!tokens"}
	case "@/":
		return []string{"@/test"}
	default:
		return []string{}
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
	os.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro", "help": "Show help information"}`)
	defer os.Unsetenv("GSH_AGENT_MACROS")

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
	os.Setenv("GSH_AGENT_MACROS", `{"test": "This is a test macro"}`)
	defer os.Unsetenv("GSH_AGENT_MACROS")

	model := New()
	model.Focus()
	model.CompletionProvider = &mockHelpCompletionProvider{}

	// Start with @! to trigger agent control completions
	model.SetValue("@!")
	model.SetCursor(2)

	// Simulate TAB to start completion (should select first completion: @!new)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Check that help shows specific help for @!new
	helpBox := model.HelpBoxView()
	assert.Contains(t, helpBox, "**@!new**", "Should show specific help for @!new after first TAB")
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should be visible")

	// Simulate another TAB to navigate to next completion (@!tokens)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Check that help updates to show specific help for @!tokens
	helpBox = model.HelpBoxView()
	assert.Contains(t, helpBox, "**@!tokens**", "Should show specific help for @!tokens after second TAB")
	assert.True(t, model.completion.shouldShowHelpBox(), "Help box should still be visible")
}
