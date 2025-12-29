package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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
var excludeDirs = []string{
	".git", ".svn", ".hg",
	"node_modules", ".npm", ".yarn", ".pnpm-store", "bower_components",
	".venv", "venv", "env", ".env", "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", "*.egg-info", ".tox", ".nox",
	"vendor", "target",
	".gradle", ".mvn", "build", "bin", "obj",
	".bundle", "_build", "deps",
	"dist", "out", ".cache", ".parcel-cache", ".next", ".nuxt", ".output", ".turbo",
	"coverage", ".nyc_output", "htmlcov",
	".terraform", ".serverless",
}

// ExecuteNativeGrepTool searches for a regex pattern in files.
func ExecuteNativeGrepTool(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("grep tool requires 'pattern' argument as string")
	}

	if pattern == "" {
		return "", fmt.Errorf("grep tool requires non-empty 'pattern' argument")
	}

	result, err := ExecuteGrep(ctx, pattern)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}

	// Truncate very long outputs
	output := result.Output
	truncated := false
	if len(output) > maxGrepOutputLen {
		output = output[:maxGrepOutputLen]
		truncated = true
	}

	// Determine match status
	matchStatus := "matches_found"
	if result.ExitCode == 1 {
		matchStatus = "no_matches"
	} else if result.ExitCode > 1 {
		matchStatus = "error"
	}

	if truncated {
		return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q, "truncated": true}`,
			output, result.ExitCode, result.Backend, matchStatus), nil
	}
	return fmt.Sprintf(`{"output": %q, "exitCode": %d, "backend": %q, "status": %q}`,
		output, result.ExitCode, result.Backend, matchStatus), nil
}

// GrepResult contains the result of a grep search.
type GrepResult struct {
	Output   string
	ExitCode int
	Backend  string
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

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		} else {
			return nil, fmt.Errorf("grep execution failed: %w", err)
		}
	}

	return &GrepResult{
		Output:   outputBuf.String(),
		ExitCode: exitCode,
		Backend:  GrepBackendName(backend),
	}, nil
}

// DetectGrepBackend determines which grep backend to use based on available tools.
// Priority: rg > git grep (if in git repo) > grep > none
func DetectGrepBackend() GrepBackend {
	if _, err := exec.LookPath("rg"); err == nil {
		return GrepBackendRipgrep
	}
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		if err := cmd.Run(); err == nil {
			return GrepBackendGitGrep
		}
	}
	if _, err := exec.LookPath("grep"); err == nil {
		return GrepBackendGrep
	}
	return GrepBackendNone
}

// BuildGrepCommand builds the appropriate grep command based on the backend and pattern.
func BuildGrepCommand(backend GrepBackend, pattern string) (string, []string, error) {
	switch backend {
	case GrepBackendRipgrep:
		return "rg", []string{"-n", "--hidden", "--color=never", "-e", pattern}, nil
	case GrepBackendGitGrep:
		return "git", []string{"grep", "-n", "--color=never", "--untracked", "-E", "-e", pattern}, nil
	case GrepBackendGrep:
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

// GrepBackendName returns a human-readable name for the backend.
func GrepBackendName(backend GrepBackend) string {
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

const grepToolName = "grep"
const grepToolDescription = "Search for a regex pattern in files. Automatically uses the best available tool (ripgrep > git grep > grep). Returns matching lines with file names and line numbers."

func grepToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
		},
		"required": []string{"pattern"},
	}
}

// GrepToolDefinition returns the ChatTool definition for the grep tool.
func GrepToolDefinition() ChatTool {
	return ChatTool{
		Name:        grepToolName,
		Description: grepToolDescription,
		Parameters:  grepToolParameters(),
	}
}

// CreateGrepNativeTool creates the grep native tool for use in gsh.tools.
func CreateGrepNativeTool() *NativeToolValue {
	return &NativeToolValue{
		Name:        grepToolName,
		Description: grepToolDescription,
		Parameters:  grepToolParameters(),
		Invoke: func(args map[string]interface{}) (interface{}, error) {
			return ExecuteNativeGrepTool(context.Background(), args)
		},
	}
}
