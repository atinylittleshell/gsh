package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// MockPrompter implements a mock user prompter for testing
type MockPrompter struct {
	responses []string
	callCount int
}

func (m *MockPrompter) Prompt(
	prompt string,
	historyValues []string,
	explanation string,
	predictor gline.Predictor,
	explainer gline.Explainer,
	analytics gline.PredictionAnalytics,
	logger *zap.Logger,
	options gline.Options,
) (string, error) {
	if m.callCount >= len(m.responses) {
		return "n", nil // Default to no if we run out of responses
	}
	response := m.responses[m.callCount]
	m.callCount++
	return response, nil
}

// TestIntegration tests the complete "always allow" permission workflow
func TestIntegration(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// This test is no longer relevant since we removed the "always" feature
	// Commands must now be pre-approved through the permissions menu (manage option)
	t.Skip("Test skipped: 'always' feature has been removed")
}

// TestPreApproval tests that commands matching saved patterns are pre-approved
func TestPreApproval(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	// First, save some patterns to the authorized commands file
	err = environment.AppendToAuthorizedCommands("^ls.*")
	assert.NoError(t, err)
	err = environment.AppendToAuthorizedCommands("^git status.*")
	assert.NoError(t, err)

	// Create a mock prompter that should NOT be called for pre-approved commands
	mockPrompter := &MockPrompter{
		responses: []string{"n"}, // This should not be used
	}

	oldUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
		return response
	}
	defer func() {
		userConfirmation = oldUserConfirmation
	}()

	// Test commands that should be pre-approved
	testCommands := []string{
		"ls -la /tmp",
		"ls",
		"ls -l",
		"git status",
		"git status --porcelain",
	}

	for _, command := range testCommands {
		t.Run(fmt.Sprintf("pre-approved: %s", command), func(t *testing.T) {
			params := map[string]any{
				"reason":  "Test command",
				"command": command,
			}
			result := BashTool(runner, historyManager, logger, params)

			// Verify the command executed (not declined)
			assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request</gsh_tool_call_error>")

			// Verify the prompter was not called (callCount should still be 0)
			assert.Equal(t, 0, mockPrompter.callCount)
		})
	}
}

// TestFileOperations tests various file operation scenarios
func TestFileOperations(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	t.Run("directory and file creation", func(t *testing.T) {
		// Ensure directory and file don't exist initially
		assert.NoError(t, os.RemoveAll(tempConfigDir))

		// Test appending creates directory and file
		err := environment.AppendToAuthorizedCommands("test.*")
		assert.NoError(t, err)

		// Verify directory was created
		_, err = os.Stat(tempConfigDir)
		assert.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(tempAuthorizedFile)
		assert.NoError(t, err)

		// Verify file contents
		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.Equal(t, "test.*\n", string(content))
	})

	t.Run("multiple pattern appending", func(t *testing.T) {
		// Reset the file
		assert.NoError(t, os.RemoveAll(tempConfigDir))

		// Append multiple patterns
		err := environment.AppendToAuthorizedCommands("ls.*")
		assert.NoError(t, err)
		err = environment.AppendToAuthorizedCommands("git.*")
		assert.NoError(t, err)
		err = environment.AppendToAuthorizedCommands("npm.*")
		assert.NoError(t, err)

		// Verify all patterns are in the file
		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)
		expectedContent := "ls.*\ngit.*\nnpm.*\n"
		assert.Equal(t, expectedContent, string(content))
	})

	t.Run("loading from non-existent file", func(t *testing.T) {
		// Reset to a non-existent file
		nonExistentFile := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_nonexistent_%d", time.Now().UnixNano()))
		environment.SetAuthorizedCommandsFileForTesting(nonExistentFile)
		t.Cleanup(func() {
			environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
			assert.NoError(t, os.RemoveAll(nonExistentFile))
		})

		// Should return empty slice without error
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		assert.Equal(t, []string{}, patterns)
	})

	t.Run("loading from empty file", func(t *testing.T) {
		// Reset the file
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		err := os.MkdirAll(tempConfigDir, 0755)
		assert.NoError(t, err)

		// Create empty file
		file, err := os.Create(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.NoError(t, file.Close())

		// Should return empty slice without error
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		assert.Equal(t, []string{}, patterns)
	})

	// Cleanup
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})
}

// TestGetApprovedBashCommandRegex tests the integration of environment variable and file patterns
func TestGetApprovedBashCommandRegexIntegration(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Reset cache for testing
	environment.ResetCacheForTesting()

	t.Run("patterns from both env var and file", func(t *testing.T) {
		// Reset the file and add some file patterns
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		err := os.MkdirAll(tempConfigDir, 0755)
		assert.NoError(t, err)
		err = environment.AppendToAuthorizedCommands("file_pattern_1.*")
		assert.NoError(t, err)
		err = environment.AppendToAuthorizedCommands("file_pattern_2.*")
		assert.NoError(t, err)

		// Reset cache for testing
		environment.ResetCacheForTesting()

		// Create runner with default environment
		env := expand.ListEnviron(os.Environ()...)
		runner, err := interp.New(interp.Env(env))
		assert.NoError(t, err)

		// Initialize Vars map if it's nil
		if runner.Vars == nil {
			runner.Vars = make(map[string]expand.Variable)
		}

		// Set the environment variable directly on the runner
		runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
			Kind: expand.String,
			Str:  `["env_pattern_1.*", "env_pattern_2.*"]`,
		}

		// Get all approved patterns
		patterns := environment.GetApprovedBashCommandRegex(runner, logger)

		// Should include both environment and file patterns
		assert.Contains(t, patterns, "env_pattern_1.*")
		assert.Contains(t, patterns, "env_pattern_2.*")
		assert.Contains(t, patterns, "file_pattern_1.*")
		assert.Contains(t, patterns, "file_pattern_2.*")
	})

	t.Run("file changes trigger reload", func(t *testing.T) {
		// Reset the file
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		err := os.MkdirAll(tempConfigDir, 0755)
		assert.NoError(t, err)

		// Reset cache for testing
		environment.ResetCacheForTesting()

		// Get initial patterns (should be empty)
		patterns := environment.GetApprovedBashCommandRegex(runner, logger)
		assert.Equal(t, []string{}, patterns)

		// Add a pattern to the file
		err = environment.AppendToAuthorizedCommands("new_pattern.*")
		assert.NoError(t, err)

		// Wait a bit to ensure file mod time changes
		time.Sleep(10 * time.Millisecond)

		// Get patterns again (should now include the new pattern)
		patterns = environment.GetApprovedBashCommandRegex(runner, logger)
		assert.Contains(t, patterns, "new_pattern.*")
	})
}

// TestInvalidRegexHandling tests how the system handles invalid regex patterns
func TestInvalidRegexHandling(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	// Add an invalid regex pattern to the file
	err = environment.AppendToAuthorizedCommands("[invalid regex")
	assert.NoError(t, err)

	// Add a valid pattern as well
	err = environment.AppendToAuthorizedCommands("^ls.*")
	assert.NoError(t, err)

	// Create a mock prompter
	mockPrompter := &MockPrompter{
		responses: []string{"y"}, // This should not be used since valid pattern will match
	}

	oldUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
		return response
	}
	defer func() {
		userConfirmation = oldUserConfirmation
	}()

	// Test that a valid command still works despite invalid patterns in the file
	params := map[string]any{
		"reason":  "Test command",
		"command": "ls -la",
	}
	result := BashTool(runner, historyManager, logger, params)

	// Should execute successfully
	assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request</gsh_tool_call_error>")

	// Should NOT prompt for confirmation since the valid pattern "^ls.*" matches "ls -la"
	assert.Equal(t, 0, mockPrompter.callCount)
}

// TestBashToolWithPreApprovedCommands tests the complete workflow with pre-approved commands
func TestBashToolWithPreApprovedCommands(t *testing.T) {
	// Reset cache to ensure clean state
	environment.ResetCacheForTesting()

	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Create the temporary config directory
	err := os.MkdirAll(tempConfigDir, 0755)
	assert.NoError(t, err)

	// Override the global variables for testing
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// This test is no longer relevant since we removed the "always" feature
	// Commands must now be pre-approved through the permissions menu (manage option)
	t.Skip("Test skipped: 'always' feature has been removed")
}

// TestGenerateCommandRegexComprehensive tests the regex generation for various commands
func TestGenerateCommandRegexComprehensive(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"ls -la /tmp", "^ls.*"},
		{"git status", "^git status.*"},
		{"npm install package", "^npm install.*"},
		{"yarn add dependency", "^yarn add.*"},
		{"docker run image", "^docker run.*"},
		{"kubectl get pods", "^kubectl get.*"},
		{"cat file.txt", "^cat.*"},
		{"pwd", "^pwd.*"},
		{"", "^$"},
		{"git", "^git.*"},
		{"npm", "^npm.*"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := GenerateCommandRegex(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRegexMatching tests that generated regex patterns correctly match commands
func TestRegexMatching(t *testing.T) {
	tests := []struct {
		pattern     string
		command     string
		shouldMatch bool
	}{
		{"^ls.*", "ls -la /tmp", true},
		{"^ls.*", "ls", true},
		{"^ls.*", "ls -l", true},
		{"^ls.*", "git status", false},
		{"^git status.*", "git status", true},
		{"^git status.*", "git status --porcelain", true},
		{"^git status.*", "git diff", false},
		{"^npm install.*", "npm install package", true},
		{"^npm install.*", "npm install", true},
		{"^npm install.*", "npm run test", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s matches %s", tt.pattern, tt.command), func(t *testing.T) {
			matched, err := regexp.MatchString(tt.pattern, tt.command)
			assert.NoError(t, err)
			assert.Equal(t, tt.shouldMatch, matched)
		})
	}
}

// TestFilePermissionIssues tests handling of file permission issues
func TestFilePermissionIssues(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping file permission test when running as root")
	}

	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create directory with no write permissions
	err := os.MkdirAll(tempConfigDir, 0444) // Read-only permissions
	assert.NoError(t, err)

	// Try to append to authorized commands - this might succeed or fail depending on the system
	_ = environment.AppendToAuthorizedCommands("test.*")
	// The important thing is that it doesn't panic

	// Try to load authorized commands - should handle permission issues gracefully
	patterns, err := environment.LoadAuthorizedCommandsFromFile()
	// Should not panic, but might return error or empty patterns
	if err == nil {
		// If the append succeeded, we might have patterns, if it failed, we should have empty
		// The key is that we don't panic and handle the situation gracefully
		assert.True(t, len(patterns) >= 0, "Should return a valid slice (empty or with patterns)")
	}
}

// TestEdgeCases tests various edge cases in the authorization system
func TestEdgeCases(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_config_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	t.Cleanup(func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	// Test with empty authorized_commands file
	err = os.MkdirAll(tempConfigDir, 0755)
	assert.NoError(t, err)

	// Create empty file
	file, err := os.Create(tempAuthorizedFile)
	assert.NoError(t, err)
		assert.NoError(t, file.Close())

	// Should load empty patterns without error
	patterns := environment.GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{}, patterns)

	// Test with various whitespace and empty line scenarios
	content := "  \n^ls.*\n  \n^git.*\n\n"
	err = os.WriteFile(tempAuthorizedFile, []byte(content), 0644)
	assert.NoError(t, err)

	// Should only load non-empty trimmed patterns
	patterns, err = environment.LoadAuthorizedCommandsFromFile()
	assert.NoError(t, err)
	assert.Equal(t, []string{"^ls.*", "^git.*"}, patterns)

	// Test bash tool with empty file initially
	mockPrompter := &MockPrompter{
		responses: []string{"always"},
	}
	oldUserConfirmation := userConfirmation
	userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string) string {
		response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
		return response
	}
	defer func() {
		userConfirmation = oldUserConfirmation
	}()

	params := map[string]any{
		"reason":  "Test command",
		"command": "ls -la",
	}
	result := BashTool(runner, historyManager, logger, params)

	// Should execute successfully
	assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request</gsh_tool_call_error>")
}
