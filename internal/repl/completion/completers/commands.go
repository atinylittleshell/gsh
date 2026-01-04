package completers

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// FileCompleterFunc is the signature for a file completion function.
type FileCompleterFunc func(prefix string, currentDirectory string) []string

// CommandCompleter provides completions for system commands, aliases, and executables.
type CommandCompleter struct {
	runner        *interp.Runner
	pwdGetter     func() string
	fileCompleter FileCompleterFunc
}

// NewCommandCompleter creates a new CommandCompleter.
func NewCommandCompleter(runner *interp.Runner, pwdGetter func() string, fileCompleter FileCompleterFunc) *CommandCompleter {
	return &CommandCompleter{
		runner:        runner,
		pwdGetter:     pwdGetter,
		fileCompleter: fileCompleter,
	}
}

// IsPathBasedCommand determines if a command looks like a path rather than a simple command name.
func (c *CommandCompleter) IsPathBasedCommand(command string) bool {
	// Check for common path patterns
	return strings.HasPrefix(command, "/") || // Absolute path: /bin/ls
		strings.HasPrefix(command, "./") || // Relative path: ./script
		strings.HasPrefix(command, "../") || // Parent directory: ../script
		strings.HasPrefix(command, "~/") || // Home directory: ~/bin/script
		strings.Contains(command, "/") // Any path with directory separator
}

// GetExecutableCompletions returns executable files and directories that match the given path prefix.
// Directories are included so users can navigate into them.
func (c *CommandCompleter) GetExecutableCompletions(pathPrefix string) []string {
	// Handle special case where pathPrefix doesn't contain a directory separator
	if !strings.Contains(pathPrefix, "/") {
		return []string{} // This shouldn't be a path-based command
	}

	// Use the injected file completer for consistent path handling (including "./" preservation)
	allCompletions := c.fileCompleter(pathPrefix, c.pwdGetter())

	// Filter to only executables and directories
	var executableCompletions []string
	for _, comp := range allCompletions {
		// Directories are always included (they end with "/")
		if strings.HasSuffix(comp, "/") {
			executableCompletions = append(executableCompletions, comp)
			continue
		}

		// For files, check if executable
		fullPath := c.resolveCompletionPath(comp)
		if info, err := os.Stat(fullPath); err == nil {
			// On Unix-like systems, check if any execute bit is set
			if info.Mode()&0111 != 0 {
				executableCompletions = append(executableCompletions, comp)
			}
		}
	}

	return executableCompletions
}

// resolveCompletionPath resolves a completion path to an absolute path for checking file info.
func (c *CommandCompleter) resolveCompletionPath(comp string) string {
	// Handle home directory
	if strings.HasPrefix(comp, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, comp[2:])
		}
	}

	// Handle absolute paths
	if filepath.IsAbs(comp) {
		return comp
	}

	// Relative path - resolve against current directory
	return filepath.Join(c.pwdGetter(), comp)
}

// GetAvailableCommands returns available system commands that match the given prefix.
func (c *CommandCompleter) GetAvailableCommands(prefix string) []string {
	// Use a map to avoid duplicates
	commands := make(map[string]bool)

	// First, add shell aliases
	aliasCompletions := c.GetAliasCompletions(prefix)
	for _, alias := range aliasCompletions {
		commands[alias] = true
	}

	// Then, get PATH from environment for system commands
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		// Split PATH into directories
		pathDirs := strings.Split(pathEnv, string(os.PathListSeparator))

		// Search each directory in PATH
		for _, dir := range pathDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue // Skip directories we can't read
			}

			for _, entry := range entries {
				// Only consider regular files that are executable
				if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
					// Check if file is executable (this is a simplified check)
					// In a real implementation, you'd want to check file permissions
					commands[entry.Name()] = true
				}
			}
		}
	}

	// Convert map to sorted slice
	var completions []string
	for cmd := range commands {
		completions = append(completions, cmd)
	}

	// Sort alphabetically for consistent ordering
	sort.Strings(completions)
	return completions
}

// GetAliasCompletions returns shell aliases that match the given prefix.
func (c *CommandCompleter) GetAliasCompletions(prefix string) []string {
	if c.runner == nil {
		return []string{}
	}

	// Use reflection to access the unexported alias field
	runnerValue := reflect.ValueOf(c.runner).Elem()
	aliasField := runnerValue.FieldByName("alias")

	if !aliasField.IsValid() || aliasField.IsNil() {
		return []string{}
	}

	// The alias field is a map[string]interp.alias
	// We need to iterate over the keys (alias names)
	var completions []string

	// Get the map keys using reflection
	for _, key := range aliasField.MapKeys() {
		aliasName := key.String()
		if strings.HasPrefix(aliasName, prefix) {
			completions = append(completions, aliasName)
		}
	}

	// Sort alphabetically for consistent ordering
	sort.Strings(completions)
	return completions
}
