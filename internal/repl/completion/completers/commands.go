package completers

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// osReadDir is a variable that can be overridden for testing.
var osReadDir = os.ReadDir

// CommandCompleter provides completions for system commands, aliases, and executables.
type CommandCompleter struct {
	runner    *interp.Runner
	pwdGetter func() string
}

// NewCommandCompleter creates a new CommandCompleter.
func NewCommandCompleter(runner *interp.Runner, pwdGetter func() string) *CommandCompleter {
	return &CommandCompleter{
		runner:    runner,
		pwdGetter: pwdGetter,
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

// GetExecutableCompletions returns executable files that match the given path prefix.
func (c *CommandCompleter) GetExecutableCompletions(pathPrefix string) []string {
	// Determine the directory to search and the filename prefix
	var searchDir, filePrefix string

	if strings.HasSuffix(pathPrefix, "/") {
		// Path ends with /, so we want all executables in that directory
		searchDir = pathPrefix
		filePrefix = ""
	} else {
		// Extract directory and filename parts
		searchDir = filepath.Dir(pathPrefix)
		filePrefix = filepath.Base(pathPrefix)

		// Handle special case where pathPrefix doesn't contain a directory separator
		if searchDir == "." && !strings.Contains(pathPrefix, "/") {
			return []string{} // This shouldn't be a path-based command
		}
	}

	// Resolve the search directory
	var resolvedDir string
	if strings.HasPrefix(searchDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return []string{}
		}
		resolvedDir = filepath.Join(homeDir, searchDir[2:])
	} else if filepath.IsAbs(searchDir) {
		resolvedDir = searchDir
	} else {
		// Relative path
		currentDir := c.pwdGetter()
		resolvedDir = filepath.Join(currentDir, searchDir)
	}

	// Read directory contents
	entries, err := osReadDir(resolvedDir)
	if err != nil {
		return []string{}
	}

	var completions []string
	for _, entry := range entries {
		// Skip directories and non-matching files
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), filePrefix) {
			continue
		}

		// Check if file is executable (simplified check)
		// In a more complete implementation, we'd check file permissions
		if info, err := entry.Info(); err == nil {
			// On Unix-like systems, check if any execute bit is set
			if info.Mode()&0111 != 0 {
				// Build the completion preserving the original path structure
				if strings.HasSuffix(pathPrefix, "/") {
					completions = append(completions, pathPrefix+entry.Name())
				} else {
					// Replace the filename part with the matched file
					completions = append(completions, filepath.Join(searchDir, entry.Name()))
				}
			}
		}
	}

	// Sort alphabetically for consistent ordering
	sort.Strings(completions)
	return completions
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
			entries, err := osReadDir(dir)
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
