package completion

import (
	"os"
	"path/filepath"
	"strings"
)

// fileCompleter is the function type for file completion
type fileCompleter func(prefix string, currentDirectory string) []string

// commandCompleter is the function type for command completion

// getFileCompletions is the default implementation of file completion
var getFileCompletions fileCompleter = func(prefix string, currentDirectory string) []string {
	if prefix == "" {
		// If prefix is empty, use current directory
		entries, err := os.ReadDir(currentDirectory)
		if err != nil {
			return []string{}
		}

		matches := make([]string, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				name += string(os.PathSeparator)
			}
			matches = append(matches, name)
		}
		return matches
	}

	// Determine path type and prepare directory and prefix
	var dir string        // directory to search in
	var filePrefix string // prefix to match file names against
	var pathType string   // "home", "abs", or "rel"
	var prefixDir string  // directory part of the prefix
	var homeDir string    // user's home directory if needed

	// Check if path starts with "~"
	if strings.HasPrefix(prefix, "~") {
		pathType = "home"
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return []string{}
		}
		// Replace "~" with actual home directory for searching
		searchPath := filepath.Join(homeDir, prefix[1:])
		dir = filepath.Dir(searchPath)
		filePrefix = filepath.Base(prefix)
		prefixDir = filepath.Dir(prefix)

		// If prefix ends with "/", adjust accordingly
		if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(os.PathSeparator)) {
			dir = searchPath
			filePrefix = ""
			prefixDir = prefix
		}
	} else if filepath.IsAbs(prefix) {
		// Absolute path
		pathType = "abs"
		dir = filepath.Dir(prefix)
		filePrefix = filepath.Base(prefix)
		prefixDir = filepath.Dir(prefix)

		// If prefix ends with "/", adjust accordingly
		if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(os.PathSeparator)) {
			dir = prefix
			filePrefix = ""
			prefixDir = prefix
		}
	} else {
		// Relative path
		pathType = "rel"
		fullPath := filepath.Join(currentDirectory, prefix)
		filePrefix = filepath.Base(prefix)
		prefixDir = filepath.Dir(prefix)

		// If prefix ends with "/", adjust accordingly
		if strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, string(os.PathSeparator)) {
			dir = fullPath
			filePrefix = ""
			prefixDir = prefix
		} else if prefix == "." || prefix == ".." {
			// Special case: "." means current directory, ".." means parent directory
			dir = fullPath
		} else {
			// For other relative paths, get the directory part
			dir = filepath.Dir(fullPath)
		}
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	// Filter and format matches
	matches := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, filePrefix) {
			continue
		}

		// Build path based on type
		var completionPath string
		if pathType == "home" {
			// For home directory paths, keep the "~" prefix
			if prefixDir == "~" || prefixDir == "." {
				completionPath = "~" + string(os.PathSeparator) + name
			} else {
				completionPath = filepath.Join(prefixDir, name)
			}
		} else if pathType == "abs" {
			// For absolute paths, keep the full path
			completionPath = filepath.Join(prefixDir, name)
		} else {
			// For relative paths, keep them relative
			if prefixDir == "." {
				// Check if the original prefix started with "./"
				if strings.HasPrefix(prefix, "./") {
					completionPath = "./" + name
				} else {
					completionPath = name
				}
			} else {
				completionPath = filepath.Join(prefixDir, name)
			}
		}

		// Add trailing slash for directories
		if entry.IsDir() {
			completionPath += string(os.PathSeparator)
		}

		matches = append(matches, completionPath)
	}

	return matches
}
