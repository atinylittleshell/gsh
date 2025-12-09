package completion

import (
	"os"
	"path/filepath"
	"strings"
)

// osReadDir is a variable that can be overridden for testing.
var osReadDir = os.ReadDir

// getFileCompletions returns file completions for the given prefix in the current directory.
func getFileCompletions(prefix string, currentDirectory string) []string {
	if prefix == "" {
		// If prefix is empty, use current directory
		entries, err := osReadDir(currentDirectory)
		if err != nil {
			return []string{}
		}

		matches := make([]string, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				name += "/"
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
		if strings.HasSuffix(prefix, "/") {
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
		if strings.HasSuffix(prefix, "/") {
			dir = prefix
			filePrefix = ""
			prefixDir = prefix
		}
	} else {
		// Relative path
		pathType = "rel"
		fullPath := filepath.Join(currentDirectory, prefix)
		dir = filepath.Dir(fullPath)
		filePrefix = filepath.Base(prefix)
		prefixDir = filepath.Dir(prefix)

		// If prefix ends with "/", adjust accordingly
		if strings.HasSuffix(prefix, "/") {
			dir = fullPath
			filePrefix = ""
			prefixDir = prefix
		}
	}

	// Read directory contents
	entries, err := osReadDir(dir)
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
		switch pathType {
		case "home":
			// For home directory paths, keep the "~" prefix
			if prefixDir == "~" || prefixDir == "." {
				completionPath = "~/" + name
			} else {
				completionPath = filepath.Join(prefixDir, name)
			}
		case "abs":
			// For absolute paths, keep the full path
			completionPath = filepath.Join(prefixDir, name)
		default:
			// For relative paths, keep them relative
			if prefixDir == "." {
				completionPath = name
			} else {
				completionPath = filepath.Join(prefixDir, name)
			}
		}

		// Add trailing slash for directories
		if entry.IsDir() {
			completionPath += "/"
		}

		matches = append(matches, completionPath)
	}

	return matches
}
