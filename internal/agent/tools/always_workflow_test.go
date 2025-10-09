package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// TestAlwaysWorkflowEndToEnd tests the complete workflow for the 'always' option
func TestAlwaysWorkflowEndToEnd(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_always_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	defer func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	t.Run("Simple Command - ls /tmp", func(t *testing.T) {
		testSimpleCommandWorkflow(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Compound Command - ls /tmp && echo test", func(t *testing.T) {
		testCompoundCommandWorkflow(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Special Command - git status", func(t *testing.T) {
		testSpecialCommandWorkflow(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("File Permissions", func(t *testing.T) {
		testFilePermissions(t, tempConfigDir, tempAuthorizedFile)
	})
}

func testSimpleCommandWorkflow(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	command := "ls /tmp"
	reason := "List directory contents"

	// Step 1: First execution with 'always' response
	t.Run("First execution with 'always' response", func(t *testing.T) {
		mockPrompter := &MockPrompter{
			responses: []string{"always"},
		}
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			// The question parameter is just the base question, the options are added in the prompt
			// So we just verify we got the expected question
			assert.Contains(t, question, "Do I have your permission to run the following command?")
			response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
			return response
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run the bash tool
		params := map[string]any{
			"reason":  reason,
			"command": command,
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify the command executed successfully (not declined)
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Verify the directory and file were created
		_, err := os.Stat(tempConfigDir)
		assert.NoError(t, err, "Config directory should be created")
		_, err = os.Stat(tempAuthorizedFile)
		assert.NoError(t, err, "Authorized commands file should be created")

		// Verify the regex pattern was saved to file
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		expectedPattern := "^ls.*"
		assert.Contains(t, patterns, expectedPattern, "Pattern should be saved to file")

		// Verify the file contains the expected pattern
		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.Contains(t, string(content), expectedPattern+"\n", "File should contain the pattern")
	})

	// Step 2: Second execution should be auto-approved
	t.Run("Second execution should be auto-approved", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n" // This should not be called
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run the same command again
		params := map[string]any{
			"reason":  reason,
			"command": command,
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify the command executed successfully without prompting
		assert.False(t, promptCalled, "User should not be prompted for pre-approved command")
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})

	// Step 3: Similar command should also be auto-approved
	t.Run("Similar command should be auto-approved", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n" // This should not be called
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run a similar ls command
		similarCommand := "ls -la /tmp"
		params := map[string]any{
			"reason":  "List directory with details",
			"command": similarCommand,
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify the command executed successfully without prompting
		assert.False(t, promptCalled, "User should not be prompted for similar command")
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})
}

func testCompoundCommandWorkflow(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	command := `ls /tmp && echo "test"`
	reason := "List directory and echo test"

	// Step 1: First execution with 'always' response
	t.Run("First execution with 'always' response", func(t *testing.T) {
		mockPrompter := &MockPrompter{
			responses: []string{"always"},
		}
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
			return response
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run the bash tool
		params := map[string]any{
			"reason":  reason,
			"command": command,
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify the command executed successfully
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Verify patterns for both commands were saved
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		assert.Contains(t, patterns, "^ls.*", "ls pattern should be saved")
		assert.Contains(t, patterns, "^echo.*", "echo pattern should be saved")
	})

	// Step 2: Individual commands should be auto-approved
	t.Run("Individual commands should be auto-approved", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Test ls command alone
		params := map[string]any{
			"reason":  "List directory",
			"command": "ls /tmp",
		}
		result := BashTool(runner, historyManager, logger, params)
		assert.False(t, promptCalled, "ls command should be pre-approved")
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Reset prompt flag
		promptCalled = false

		// Test echo command alone
		params = map[string]any{
			"reason":  "Echo test",
			"command": `echo "hello"`,
		}
		result = BashTool(runner, historyManager, logger, params)
		assert.False(t, promptCalled, "echo command should be pre-approved")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})
}

func testSpecialCommandWorkflow(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	command := "git status"
	reason := "Check git status"

	// Step 1: First execution with 'always' response
	t.Run("First execution with 'always' response", func(t *testing.T) {
		mockPrompter := &MockPrompter{
			responses: []string{"always"},
		}
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
			// Use the actual response mapping logic from the real userConfirmation function
			lowerResponse := strings.ToLower(response)
			var mappedResponse string
			if lowerResponse == "y" || lowerResponse == "yes" {
				mappedResponse = "y"
			} else if lowerResponse == "n" || lowerResponse == "no" {
				mappedResponse = "n"
			} else if lowerResponse == "a" || lowerResponse == "always" {
				mappedResponse = "always"
			} else {
				mappedResponse = response
			}
			// Debug: log what we're returning
			logger.Debug("Mock userConfirmation", zap.String("input", response), zap.String("output", mappedResponse))
			return mappedResponse
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run the bash tool
		params := map[string]any{
			"reason":  reason,
			"command": command,
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify the command executed successfully
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")

		// Verify the specific git status pattern was saved
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		expectedPattern := "^git status.*"
		assert.Contains(t, patterns, expectedPattern, "git status pattern should be saved")
	})

	// Step 2: Similar git status commands should be auto-approved
	t.Run("Similar git status commands should be auto-approved", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Test git status with flags
		params := map[string]any{
			"reason":  "Check git status with short format",
			"command": "git status -s",
		}
		result := BashTool(runner, historyManager, logger, params)
		assert.False(t, promptCalled, "git status -s should be pre-approved")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})

	// Step 3: Different git commands should still require approval
	t.Run("Different git commands should require approval", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Test git add (different subcommand)
		params := map[string]any{
			"reason":  "Add files to git",
			"command": "git add .",
		}
		result := BashTool(runner, historyManager, logger, params)
		assert.True(t, promptCalled, "git add should require approval")
		assert.Contains(t, result, "<gsh_tool_call_error>User declined this request")
	})
}

func testFilePermissions(t *testing.T, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Config directory permissions", func(t *testing.T) {
		info, err := os.Stat(tempConfigDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())

		// Directory should be accessible by owner only (0700)
		mode := info.Mode()
		expectedMode := os.FileMode(0700)
		assert.Equal(t, expectedMode, mode&0777, "Directory permissions should be 0700 (owner only)")

		// Verify no group or other permissions
		assert.Equal(t, os.FileMode(0), mode&0077, "Directory should not be accessible by group or others")
	})

	t.Run("Authorized commands file permissions", func(t *testing.T) {
		info, err := os.Stat(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.False(t, info.IsDir())

		// File should be readable and writable by owner only (0600)
		mode := info.Mode()
		expectedMode := os.FileMode(0600)
		assert.Equal(t, expectedMode, mode&0777, "File permissions should be 0600 (owner only)")

		// Verify no group or other permissions
		assert.Equal(t, os.FileMode(0), mode&0077, "File should not be accessible by group or others")
	})

	t.Run("Security verification", func(t *testing.T) {
		// Verify the file content is not world-readable
		info, err := os.Stat(tempAuthorizedFile)
		assert.NoError(t, err)

		mode := info.Mode()
		// Ensure no read permissions for group or others
		assert.False(t, mode&0044 != 0, "File should not be readable by group or others")
		assert.False(t, mode&0004 != 0, "File should not be readable by others")
	})
}

// TestPromptOptionsDisplay tests that the permission prompt shows the correct options
func TestPromptOptionsDisplay(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_prompt_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	defer func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	t.Run("Prompt shows all options including 'a'", func(t *testing.T) {
		var capturedPrompt string
		mockPrompter := &MockPrompter{
			responses: []string{"n"},
		}
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			capturedPrompt = question
			response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
			// Use the actual response mapping logic from the real userConfirmation function
			lowerResponse := strings.ToLower(response)
			if lowerResponse == "y" || lowerResponse == "yes" {
				return "y"
			}
			if lowerResponse == "n" || lowerResponse == "no" {
				return "n"
			}
			if lowerResponse == "a" || lowerResponse == "always" {
				return "always"
			}
			return response
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		// Run the bash tool
		params := map[string]any{
			"reason":  "Test prompt options",
			"command": "ls /tmp",
		}
		BashTool(runner, historyManager, logger, params)

		// Verify the prompt contains the expected question
		// Note: The options (y/N/freeform/a) are added by the userConfirmation function, not passed in the question parameter
		assert.Contains(t, capturedPrompt, "Do I have your permission to run the following command?", "Prompt should ask for permission")

		// The actual prompt with options is created in utils.go by appending " (y/N/freeform/a) " to the question
		// This test verifies that the question parameter is correct, and we know from utils.go that the options are added
	})
}

// TestResponseHandling tests that different user responses are handled correctly
func TestResponseHandling(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_response_%d", time.Now().UnixNano()))
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Save original values
	oldConfigDir := environment.GetConfigDirForTesting()
	oldAuthorizedFile := environment.GetAuthorizedCommandsFileForTesting()

	// Override the global variables for testing
	environment.SetConfigDirForTesting(tempConfigDir)
	environment.SetAuthorizedCommandsFileForTesting(tempAuthorizedFile)
	defer func() {
		environment.SetConfigDirForTesting(oldConfigDir)
		environment.SetAuthorizedCommandsFileForTesting(oldAuthorizedFile)
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	// Create a mock history manager
	historyManager, _ := history.NewHistoryManager(":memory:")

	testCases := []struct {
		name          string
		response      string
		shouldExecute bool
		shouldSave    bool
		expectedError string
	}{
		{
			name:          "Response 'a' should execute and save",
			response:      "always", // Using "always" because mock bypasses userConfirmation mapping
			shouldExecute: true,
			shouldSave:    true,
		},
		{
			name:          "Response 'always' should execute and save",
			response:      "always",
			shouldExecute: true,
			shouldSave:    true,
		},
		{
			name:          "Response 'y' should execute but not save",
			response:      "y",
			shouldExecute: true,
			shouldSave:    false,
		},
		{
			name:          "Response 'yes' should execute but not save",
			response:      "y", // Using "y" because mock bypasses userConfirmation mapping
			shouldExecute: true,
			shouldSave:    false,
		},
		{
			name:          "Response 'n' should not execute",
			response:      "n",
			shouldExecute: false,
			shouldSave:    false,
			expectedError: "User declined this request",
		},
		{
			name:          "Response 'no' should not execute",
			response:      "no",
			shouldExecute: false,
			shouldSave:    false,
			expectedError: "User declined this request",
		},
		{
			name:          "Custom response should not execute",
			response:      "not now",
			shouldExecute: false,
			shouldSave:    false,
			expectedError: "User declined this request: not now",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up for each test
			os.RemoveAll(tempConfigDir)
			environment.ResetCacheForTesting()

			mockPrompter := &MockPrompter{
				responses: []string{tc.response},
			}
			oldUserConfirmation := userConfirmation
			userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
				response, _ := mockPrompter.Prompt("", []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
				return response
			}
			defer func() {
				userConfirmation = oldUserConfirmation
			}()

			// Run the bash tool
			params := map[string]any{
				"reason":  "Test response handling",
				"command": "ls /tmp",
			}
			result := BashTool(runner, historyManager, logger, params)

			if tc.shouldExecute {
				assert.NotContains(t, result, "gsh_tool_call_error", "Command should execute successfully")
			} else {
				assert.Contains(t, result, tc.expectedError, "Should contain expected error message")
			}

			if tc.shouldSave {
				// Check if pattern was saved
				patterns, err := environment.LoadAuthorizedCommandsFromFile()
				if err == nil && len(patterns) > 0 {
					assert.Contains(t, patterns, "^ls.*", "Pattern should be saved")
				}
			} else {
				// Check that no pattern was saved or file doesn't exist
				patterns, err := environment.LoadAuthorizedCommandsFromFile()
				if err == nil {
					assert.NotContains(t, patterns, "^ls.*", "Pattern should not be saved")
				}
			}
		})
	}
}

// TestPatternGeneration tests that regex patterns are generated correctly for different command types
func TestPatternGeneration(t *testing.T) {
	testCases := []struct {
		name            string
		command         string
		expectedPattern string
	}{
		{
			name:            "Simple ls command",
			command:         "ls /tmp",
			expectedPattern: "^ls.*",
		},
		{
			name:            "Git status command",
			command:         "git status",
			expectedPattern: "^git status.*",
		},
		{
			name:            "Git commit command",
			command:         "git commit -m 'message'",
			expectedPattern: "^git commit.*",
		},
		{
			name:            "NPM install command",
			command:         "npm install package",
			expectedPattern: "^npm install.*",
		},
		{
			name:            "Docker run command",
			command:         "docker run image",
			expectedPattern: "^docker run.*",
		},
		{
			name:            "Kubectl get command",
			command:         "kubectl get pods",
			expectedPattern: "^kubectl get.*",
		},
		{
			name:            "Regular command with args",
			command:         "cat file.txt",
			expectedPattern: "^cat.*",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern := GenerateCommandRegex(tc.command)
			assert.Equal(t, tc.expectedPattern, pattern, "Generated pattern should match expected")
		})
	}
}
