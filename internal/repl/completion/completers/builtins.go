package completers

import (
	"sort"
	"strings"
)

// BuiltinCommand represents a built-in command with its help text.
type BuiltinCommand struct {
	Name        string
	Description string
	Help        string
}

// BuiltinCompleter provides completions for built-in commands (prefixed with #!).
type BuiltinCompleter struct {
	commands []BuiltinCommand
}

// NewBuiltinCompleter creates a new BuiltinCompleter with default commands.
func NewBuiltinCompleter() *BuiltinCompleter {
	return &BuiltinCompleter{
		commands: []BuiltinCommand{
			{
				Name:        "new",
				Description: "Start a new chat session",
				Help:        "**#!new** - Start a new chat session with the agent\n\nThis command resets the conversation history and starts fresh.",
			},
			{
				Name:        "tokens",
				Description: "Show token usage statistics",
				Help:        "**#!tokens** - Display token usage statistics\n\nShows information about token consumption for the current chat session.",
			},
		},
	}
}

// GetCompletions returns built-in command completions for the given prefix.
func (c *BuiltinCompleter) GetCompletions(prefix string) []string {
	var completions []string
	prefixAfterBang := strings.TrimPrefix(prefix, "#!")

	for _, cmd := range c.commands {
		if strings.HasPrefix(cmd.Name, prefixAfterBang) {
			completions = append(completions, "#!"+cmd.Name)
		}
	}

	// Sort alphabetically for consistent ordering
	sort.Strings(completions)
	return completions
}

// GetHelp returns help information for a built-in command.
func (c *BuiltinCompleter) GetHelp(command string) string {
	// Check for exact match
	for _, cmd := range c.commands {
		if cmd.Name == command {
			return cmd.Help
		}
	}

	// Empty or partial - show general help
	if command == "" {
		return c.getGeneralHelp()
	}

	// Check for partial matches
	for _, cmd := range c.commands {
		if strings.HasPrefix(cmd.Name, command) {
			return c.getGeneralHelp()
		}
	}

	return ""
}

// getGeneralHelp returns general help for all built-in commands.
func (c *BuiltinCompleter) getGeneralHelp() string {
	var lines []string
	lines = append(lines, "**Agent Controls** - Built-in commands for managing the agent\n\nAvailable commands:")
	for _, cmd := range c.commands {
		lines = append(lines, "• **#!"+cmd.Name+"** - "+cmd.Description)
	}
	return strings.Join(lines, "\n")
}
