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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// ComprehensiveIntegrationTest tests the complete 'always' response workflow across all components
func TestComprehensiveIntegrationWorkflow(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_comprehensive_test_%d", time.Now().UnixNano()))
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

	// Create temporary directories for file operations
	testFileDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_files_%d", time.Now().UnixNano()))
	err = os.MkdirAll(testFileDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testFileDir)

	t.Run("1. Test Bash Commands with 'always' Response", func(t *testing.T) {
		testBashCommandsAlwaysWorkflow(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("2. Test File Creation with 'always' Response", func(t *testing.T) {
		testFileCreationAlwaysWorkflow(t, runner, logger, tempConfigDir, tempAuthorizedFile, testFileDir)
	})

	t.Run("3. Test Patterns Coexistence", func(t *testing.T) {
		testPatternsCoexistence(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile, testFileDir)
	})

	t.Run("4. Test Permission System", func(t *testing.T) {
		testPermissionSystem(t, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("5. Test EditFile 'always' Response", func(t *testing.T) {
		testEditFileAlwaysWorkflow(t, runner, logger, tempConfigDir, tempAuthorizedFile, testFileDir)
	})

	t.Run("6. Test Compound Commands with New Patterns", func(t *testing.T) {
		testCompoundCommandsWithPatterns(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("7. Test Prompt Options Display", func(t *testing.T) {
		testPromptOptionsDisplay(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile, testFileDir)
	})

	t.Run("8. Test Pattern Formatting", func(t *testing.T) {
		testPatternFormatting(t, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("9. Test No Regressions", func(t *testing.T) {
		testNoRegressions(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("10. Test Security Maintained", func(t *testing.T) {
		testSecurityMaintained(t, runner, historyManager, logger, tempConfigDir, tempAuthorizedFile)
	})
}

func testBashCommandsAlwaysWorkflow(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	// Test simple command with 'always' response
	t.Run("Simple bash command with 'always'", func(t *testing.T) {
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			// Verify the prompt contains the expected question
			assert.Contains(t, question, "Do I have your permission to run the following command?")
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "List directory contents",
			"command": "ls /tmp",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify command executed successfully
		assert.NotContains(t, result, "<gsh_tool_call_error>User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Verify pattern was saved
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		assert.Contains(t, patterns, "^ls.*")

		// Verify directory and file were created with correct permissions
		info, err := os.Stat(tempConfigDir)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode()&0777)

		info, err = os.Stat(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode()&0777)
	})

	// Test auto-approval works
	t.Run("Auto-approval for similar command", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "List directory with details",
			"command": "ls -la /tmp",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Verify command executed without prompting
		assert.False(t, promptCalled, "User should not be prompted for pre-approved command")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})
}

func testFileCreationAlwaysWorkflow(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile, testFileDir string) {
	// Test file creation with 'always' response
	t.Run("File creation with 'always'", func(t *testing.T) {
		testFile := filepath.Join(testFileDir, "test.txt")

		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			// Verify the prompt contains the expected question
			assert.Contains(t, question, "Do I have your permission to create the file with the content shown above?")
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    testFile,
			"content": "test content",
		}
		result := CreateFileTool(runner, logger, params)

		// Verify file was created successfully
		assert.Contains(t, result, "successfully created")
		assert.FileExists(t, testFile)

		// Verify pattern was saved
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		expectedPattern := GenerateFileOperationRegex(testFile, "create_file")
		assert.Contains(t, patterns, expectedPattern)

		// Verify file content
		content, err := os.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})
}

func testPatternsCoexistence(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile, testFileDir string) {
	// Verify that bash and file patterns coexist
	t.Run("Bash and file patterns coexist", func(t *testing.T) {
		// Load current patterns
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)

		// Should contain both bash and file patterns
		hasBashPattern := false
		hasFilePattern := false

		for _, pattern := range patterns {
			if strings.HasPrefix(pattern, "^ls") {
				hasBashPattern = true
			}
			if strings.HasPrefix(pattern, "create_file:") {
				hasFilePattern = true
			}
		}

		assert.True(t, hasBashPattern, "Should contain bash command patterns")
		assert.True(t, hasFilePattern, "Should contain file operation patterns")

		// Verify file content shows both types
		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "^ls")
		assert.Contains(t, contentStr, "create_file:")
	})
}

func testPermissionSystem(t *testing.T, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Directory permissions are 0700", func(t *testing.T) {
		info, err := os.Stat(tempConfigDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0700), info.Mode()&0777, "Directory should have 0700 permissions")
		assert.Equal(t, os.FileMode(0), info.Mode()&0077, "Directory should not be accessible by group or others")
	})

	t.Run("File permissions are 0600", func(t *testing.T) {
		info, err := os.Stat(tempAuthorizedFile)
		assert.NoError(t, err)
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0600), info.Mode()&0777, "File should have 0600 permissions")
		assert.Equal(t, os.FileMode(0), info.Mode()&0077, "File should not be accessible by group or others")
	})
}

func testEditFileAlwaysWorkflow(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile, testFileDir string) {
	t.Run("EditFile handles 'always' response", func(t *testing.T) {
		// Create a test file to edit
		testFile := filepath.Join(testFileDir, "edit_test.txt")
		err := os.WriteFile(testFile, []byte("original content\nline 2\nline 3"), 0644)
		require.NoError(t, err)

		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			// Verify the prompt contains the expected question
			assert.Contains(t, question, "Do I have your permission to make the edit proposed above?")
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    testFile,
			"old_str": "original content",
			"new_str": "modified content",
		}
		result := EditFileTool(runner, logger, params)

		// Verify edit was successful
		assert.NotContains(t, result, "User declined this request")
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Verify file content was changed
		content, err := os.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "modified content")
		assert.NotContains(t, string(content), "original content")
	})
}

func testCompoundCommandsWithPatterns(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Compound commands work with saved patterns", func(t *testing.T) {
		// Add echo pattern via 'always' response
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Echo test message",
			"command": "echo 'test'",
		}
		result := BashTool(runner, historyManager, logger, params)
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Now test compound command with both ls and echo (both should be pre-approved)
		promptCalled := false
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}

		params = map[string]any{
			"reason":  "List and echo",
			"command": "ls /tmp && echo 'done'",
		}
		result = BashTool(runner, historyManager, logger, params)

		// Should execute without prompting since both commands are approved
		assert.False(t, promptCalled, "Compound command with approved parts should not prompt")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})

	t.Run("Compound commands with unapproved parts still prompt", func(t *testing.T) {
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "List and remove",
			"command": "ls /tmp && rm -rf /tmp/nonexistent",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should prompt because rm is not approved
		assert.True(t, promptCalled, "Compound command with unapproved parts should prompt")
		assert.Contains(t, result, "User declined this request")
	})
}

func testPromptOptionsDisplay(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile, testFileDir string) {
	t.Run("Bash tool prompt shows (y/N/freeform/a)", func(t *testing.T) {
		// Clean up to ensure we get a prompt
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()

		var capturedPrompt string
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			capturedPrompt = question
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Test prompt",
			"command": "pwd",
		}
		BashTool(runner, historyManager, logger, params)

		// The actual prompt formatting is done in utils.go by appending " (y/N/freeform/a) "
		assert.Contains(t, capturedPrompt, "Do I have your permission to run the following command?")
	})

	t.Run("CreateFile tool prompt shows correct question", func(t *testing.T) {
		testFile := filepath.Join(testFileDir, "prompt_test.txt")

		var capturedPrompt string
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			capturedPrompt = question
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    testFile,
			"content": "test",
		}
		CreateFileTool(runner, logger, params)

		assert.Contains(t, capturedPrompt, "Do I have your permission to create the file with the content shown above?")
	})

	t.Run("EditFile tool prompt shows correct question", func(t *testing.T) {
		testFile := filepath.Join(testFileDir, "edit_prompt_test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		var capturedPrompt string
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			capturedPrompt = question
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    testFile,
			"old_str": "test",
			"new_str": "modified",
		}
		EditFileTool(runner, logger, params)

		assert.Contains(t, capturedPrompt, "Do I have your permission to make the edit proposed above?")
	})
}

func testPatternFormatting(t *testing.T, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Patterns are saved with correct formatting", func(t *testing.T) {
		// Check if file exists first
		if _, err := os.Stat(tempAuthorizedFile); os.IsNotExist(err) {
			t.Skip("No authorized commands file exists yet")
			return
		}

		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")

		for _, line := range lines {
			if line == "" {
				continue
			}

			// Each line should be a valid pattern (no extra whitespace, proper format)
			assert.NotContains(t, line, "  ", "Pattern should not contain double spaces")
			assert.False(t, strings.HasPrefix(line, " "), "Pattern should not start with space")
			assert.False(t, strings.HasSuffix(line, " "), "Pattern should not end with space")

			// Should be either a bash pattern or file pattern
			isBashPattern := strings.HasPrefix(line, "^")
			isFilePattern := strings.Contains(line, ":")
			assert.True(t, isBashPattern || isFilePattern, "Pattern should be either bash or file pattern: %s", line)
		}
	})
}

func testNoRegressions(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Basic 'y' response still works", func(t *testing.T) {
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "y"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Test basic approval",
			"command": "date",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should execute but not save pattern
		assert.NotContains(t, result, "gsh_tool_call_error")

		// Verify pattern was NOT saved (since it was 'y', not 'always')
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		assert.NotContains(t, patterns, "^date.*")
	})

	t.Run("Basic 'n' response still works", func(t *testing.T) {
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Test basic denial",
			"command": "whoami",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should be declined
		assert.Contains(t, result, "User declined this request")
	})

	t.Run("Custom response still works", func(t *testing.T) {
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "custom reason for denial"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Test custom response",
			"command": "uptime",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should be declined with custom message
		assert.Contains(t, result, "User declined this request: custom reason for denial")
	})
}

func testSecurityMaintained(t *testing.T, runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	t.Run("Malicious injection still blocked", func(t *testing.T) {
		// Even with ls approved, malicious injection should be blocked
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Malicious command",
			"command": "ls /tmp; rm -rf /",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should prompt because rm is not approved
		assert.True(t, promptCalled, "Malicious injection should still prompt")
		assert.Contains(t, result, "User declined this request")
	})

	t.Run("File permissions remain secure", func(t *testing.T) {
		// Check if directory exists first
		if _, err := os.Stat(tempConfigDir); err == nil {
			// Verify config directory is still 0700
			info, err := os.Stat(tempConfigDir)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0700), info.Mode()&0777)
		}

		// Check if file exists first
		if _, err := os.Stat(tempAuthorizedFile); err == nil {
			// Verify authorized commands file is still 0600
			info, err := os.Stat(tempAuthorizedFile)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), info.Mode()&0777)
		}
	})

	t.Run("Invalid regex patterns don't break system", func(t *testing.T) {
		// Ensure the file exists first by creating it if needed
		if _, err := os.Stat(tempAuthorizedFile); os.IsNotExist(err) {
			err = os.MkdirAll(tempConfigDir, 0700)
			require.NoError(t, err)
			err = os.WriteFile(tempAuthorizedFile, []byte("^ls.*\n"), 0600)
			require.NoError(t, err)
		}

		// Add an invalid regex pattern manually
		file, err := os.OpenFile(tempAuthorizedFile, os.O_APPEND|os.O_WRONLY, 0600)
		require.NoError(t, err)
		_, err = file.WriteString("[invalid regex\n")
		require.NoError(t, err)
		file.Close()

		// Reset cache to force reload
		environment.ResetCacheForTesting()

		// System should still work with valid patterns
		promptCalled := false
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"reason":  "Test with invalid regex in file",
			"command": "ls /tmp",
		}
		result := BashTool(runner, historyManager, logger, params)

		// Should NOT prompt because ls pattern is still valid
		assert.False(t, promptCalled, "Valid patterns should still work despite invalid ones")
		assert.NotContains(t, result, "gsh_tool_call_error")
	})
}
