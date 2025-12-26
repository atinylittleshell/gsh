// Package agent provides the grep tool for searching files with regex patterns.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// GrepBackend represents the grep implementation to use.
type GrepBackend int

const (
	GrepBackendNone GrepBackend = iota
	GrepBackendRipgrep
	GrepBackendGitGrep
	GrepBackendGrep
)

// excludeDirs contains directories to exclude when using standard grep.
// These are commonly ignored directories across various language ecosystems.
var excludeDirs = []string{
	// Version control
	".git",
	".svn",
	".hg",

	// Node.js / JavaScript / TypeScript
	"node_modules",
	".npm",
	".yarn",
	".pnpm-store",
	"bower_components",

	// Python
	".venv",
	"venv",
	"env",
	".env",
	"__pycache__",
	".pytest_cache",
	".mypy_cache",
	".ruff_cache",
	"*.egg-info",
	".tox",
	".nox",

	// Go
	"vendor",

	// Rust
	"target",

	// Java / Kotlin / Scala
	".gradle",
	".mvn",
	"build",

	// .NET / C#
	"bin",
	"obj",

	// Ruby
	".bundle",

	// Elixir
	"_build",
	"deps",

	// Build outputs and caches
	"dist",
	"out",
	".cache",
	".parcel-cache",
	".next",
	".nuxt",
	".output",
	".turbo",

	// Coverage and test artifacts
	"coverage",
	".nyc_output",
	"htmlcov",

	// Misc
	".terraform",
	".serverless",
}

// DetectGrepBackend determines which grep backend to use based on available tools.
// Priority: rg > git grep (if in git repo) > grep > none
func DetectGrepBackend() GrepBackend {
	// Check for ripgrep first (fastest and most feature-rich)
	if _, err := exec.LookPath("rg"); err == nil {
		return GrepBackendRipgrep
	}

	// Check if we're in a git repository and git is available
	if _, err := exec.LookPath("git"); err == nil {
		// Check if current directory is inside a git repository
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		if err := cmd.Run(); err == nil {
			return GrepBackendGitGrep
		}
	}

	// Fall back to standard grep
	if _, err := exec.LookPath("grep"); err == nil {
		return GrepBackendGrep
	}

	return GrepBackendNone
}

// BuildGrepCommand builds the appropriate grep command based on the backend and pattern.
func BuildGrepCommand(backend GrepBackend, pattern string) (string, []string, error) {
	switch backend {
	case GrepBackendRipgrep:
		// rg with useful flags:
		// -n: line numbers
		// --hidden: include hidden files
		// (rg respects .gitignore by default)
		// --color=never: no color codes in output
		// -e: pattern (allows patterns starting with -)
		return "rg", []string{"-n", "--hidden", "--color=never", "-e", pattern}, nil

	case GrepBackendGitGrep:
		// git grep with useful flags:
		// -n: line numbers
		// --color=never: no color codes in output
		// --untracked: include untracked files (still respects .gitignore)
		// -E: extended regex (ERE)
		// -e: pattern (allows patterns starting with -)
		return "git", []string{"grep", "-n", "--color=never", "--untracked", "-E", "-e", pattern}, nil

	case GrepBackendGrep:
		// grep with recursive search:
		// -r: recursive
		// -n: line numbers
		// --color=never: no color codes in output
		// -E: extended regex (ERE)
		// -e: pattern (allows patterns starting with -)
		// Note: standard grep doesn't have native gitignore support, so we exclude
		// common non-source directories via excludeDirs list
		args := []string{"-rn", "--color=never", "-E"}
		for _, dir := range excludeDirs {
			args = append(args, "--exclude-dir="+dir)
		}
		args = append(args, "-e", pattern, ".")
		return "grep", args, nil

	case GrepBackendNone:
		return "", nil, fmt.Errorf("no grep tool available: install rg, git, or grep")

	default:
		return "", nil, fmt.Errorf("unknown grep backend: %d", backend)
	}
}

// GrepResult contains the result of a grep search.
type GrepResult struct {
	Output   string // Search results
	ExitCode int    // 0 = matches found, 1 = no matches, 2+ = error
	Backend  string // Which backend was used
}

// ExecuteGrep runs a grep search with the detected backend.
func ExecuteGrep(ctx context.Context, pattern string) (*GrepResult, error) {
	backend := DetectGrepBackend()
	return ExecuteGrepWithBackend(ctx, pattern, backend)
}

// ExecuteGrepWithBackend runs a grep search with a specific backend.
func ExecuteGrepWithBackend(ctx context.Context, pattern string, backend GrepBackend) (*GrepResult, error) {
	cmdName, args, err := BuildGrepCommand(backend, pattern)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Env = os.Environ()

	// Capture stdout and stderr
	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			// Exit code 1 means no matches found (not an error)
			// Exit code 2+ means actual error
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		} else {
			return nil, fmt.Errorf("grep execution failed: %w", err)
		}
	}

	return &GrepResult{
		Output:   outputBuf.String(),
		ExitCode: exitCode,
		Backend:  backendName(backend),
	}, nil
}

// backendName returns a human-readable name for the backend.
func backendName(backend GrepBackend) string {
	switch backend {
	case GrepBackendRipgrep:
		return "rg"
	case GrepBackendGitGrep:
		return "git-grep"
	case GrepBackendGrep:
		return "grep"
	default:
		return "none"
	}
}

// GrepToolDefinition returns the tool definition for the grep tool.
func GrepToolDefinition() interpreter.ChatTool {
	return interpreter.ChatTool{
		Name:        "grep",
		Description: "Search for a regex pattern in files. Automatically uses the best available tool (ripgrep > git grep > grep). Returns matching lines with file names and line numbers.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "The regex pattern to search for",
				},
			},
			"required": []string{"pattern"},
		},
	}
}

// ExecuteGrepTool handles execution of the grep tool.
func ExecuteGrepTool(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("grep tool requires 'pattern' argument as string")
	}

	// Validate that pattern is not empty
	if pattern == "" {
		return "", fmt.Errorf("grep tool requires non-empty 'pattern' argument")
	}

	result, err := ExecuteGrep(ctx, pattern)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	// Truncate very long outputs to avoid overwhelming the model
	output := result.Output
	const maxOutputLen = 50000 // ~50KB limit
	truncated := false
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen]
		truncated = true
	}

	// Determine match status
	matchStatus := "matches_found"
	if result.ExitCode == 1 {
		matchStatus = "no_matches"
	} else if result.ExitCode > 1 {
		matchStatus = "error"
	}

	// Return result as JSON for the agent
	if truncated {
		return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q, "truncated": true}`,
			output, result.ExitCode, result.Backend, matchStatus), nil
	}
	return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q}`,
		output, result.ExitCode, result.Backend, matchStatus), nil
}

// IsGrepAvailable returns true if any grep backend is available.
func IsGrepAvailable() bool {
	return DetectGrepBackend() != GrepBackendNone
}

// GetGrepBackendInfo returns information about the current grep backend.
func GetGrepBackendInfo() (backend string, available bool) {
	b := DetectGrepBackend()
	return backendName(b), b != GrepBackendNone
}

// isInsideGitRepo checks if the current or given directory is inside a git repository.
func isInsideGitRepo(dir string) bool {
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	cmd := exec.Command("git", "-C", absDir, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}
