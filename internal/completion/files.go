package completion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/charmbracelet/lipgloss"
)

// fileCompleter is the function type for file completion
type fileCompleter func(prefix string, currentDirectory string) []shellinput.CompletionCandidate

// commandCompleter is the function type for command completion

// formatFileDisplay formats a file name with colors and indicators similar to ls --color -F
func formatFileDisplay(name string, entry os.DirEntry) string {
	var style lipgloss.Style
	var indicator string

	if entry.IsDir() {
		// Directories: blue/bold with trailing /
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true) // Blue
		indicator = "/"
	} else {
		// Check if executable
		info, err := entry.Info()
		if err == nil && info.Mode()&0111 != 0 {
			// Executable files: green/bold with trailing *
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true) // Green
			indicator = "*"
		} else {
			// Check file extension for special types
			ext := filepath.Ext(name)
			switch ext {
			case ".tar", ".gz", ".zip", ".7z", ".bz2", ".xz", ".rar":
				// Archives: red/bold
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("31")).Bold(true) // Red
			case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".ico":
				// Images: magenta/bold
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("35")).Bold(true) // Magenta
			case ".mp3", ".wav", ".flac", ".ogg", ".m4a":
				// Audio: cyan/bold
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true) // Cyan
			case ".mp4", ".avi", ".mkv", ".mov", ".wmv":
				// Video: magenta/bold
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("35")).Bold(true) // Magenta
			default:
				// Regular files: default color
				style = lipgloss.NewStyle()
			}
		}
	}

	return style.Render(name) + indicator
}

// getFileCompletions is the default implementation of file completion
var getFileCompletions fileCompleter = func(prefix string, currentDirectory string) []shellinput.CompletionCandidate {
	if prefix == "" {
		// If prefix is empty, use current directory
		entries, err := os.ReadDir(currentDirectory)
		if err != nil {
			return []shellinput.CompletionCandidate{}
		}

		matches := make([]shellinput.CompletionCandidate, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			candidate := shellinput.CompletionCandidate{
				Value:   name,
				Display: formatFileDisplay(name, entry),
			}
			// Add trailing slash as suffix for directories
			if entry.IsDir() {
				candidate.Suffix = string(os.PathSeparator)
			}
			matches = append(matches, candidate)
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
			return []shellinput.CompletionCandidate{}
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
		return []shellinput.CompletionCandidate{}
	}

	// Filter and format matches
	matches := make([]shellinput.CompletionCandidate, 0, len(entries))
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
				completionPath = "~" + string(os.PathSeparator) + name
			} else {
				completionPath = filepath.Join(prefixDir, name)
			}
		case "abs":
			// For absolute paths, keep the full path
			completionPath = filepath.Join(prefixDir, name)
		default:
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

		// Create completion candidate
		candidate := shellinput.CompletionCandidate{
			Value:   completionPath,
			Display: formatFileDisplay(name, entry),
		}

		// Add trailing slash as suffix for directories (not in Value)
		if entry.IsDir() {
			candidate.Suffix = string(os.PathSeparator)
		}

		matches = append(matches, candidate)
	}

	return matches
}
