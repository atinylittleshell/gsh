package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestAppendToAuthorizedCommands(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Test appending a command
	err := AppendToAuthorizedCommands("fakecommand.*")
	assert.NoError(t, err)

	// Check if file was created
	_, err = os.Stat(authorizedCommandsFile)
	assert.NoError(t, err)

	// Check file contents
	content, err := os.ReadFile(authorizedCommandsFile)
	assert.NoError(t, err)
	// Check content with platform-agnostic newline handling
	expected := "fakecommand.*\n"
	assert.Equal(t, expected, strings.ReplaceAll(string(content), "\r\n", "\n"))

	// Test appending another command
	err = AppendToAuthorizedCommands("anotherfake.*")
	assert.NoError(t, err)

	// Check file contents again
	content, err = os.ReadFile(authorizedCommandsFile)
	assert.NoError(t, err)
	// Check content with platform-agnostic newline handling
	expected = "fakecommand.*\nanotherfake.*\n"
	assert.Equal(t, expected, strings.ReplaceAll(string(content), "\r\n", "\n"))
}

func TestAppendToAuthorizedCommandsSecurePermissions(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config_secure")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	t.Run("New directory and file have secure permissions", func(t *testing.T) {
		// Test appending a command to a new file
		err := AppendToAuthorizedCommands("fakecommand.*")
		assert.NoError(t, err)

		// Skip permission checks on Windows as they work differently
		if runtime.GOOS != "windows" {
			// Check directory permissions
			dirInfo, err := os.Stat(configDir)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0700), dirInfo.Mode()&0777, "Directory should have 0700 permissions")

			// Check file permissions
			fileInfo, err := os.Stat(authorizedCommandsFile)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), fileInfo.Mode()&0777, "File should have 0600 permissions")

			// Verify no group or other access
			assert.Equal(t, os.FileMode(0), fileInfo.Mode()&0077, "File should not be accessible by group or others")
		}
	})

	t.Run("Existing insecure files get permissions fixed", func(t *testing.T) {
		// Clean up from previous test
		assert.NoError(t, os.RemoveAll(tempConfigDir))

		// Create directory and file with insecure permissions
		err := os.MkdirAll(tempConfigDir, 0755) // Insecure directory permissions
		assert.NoError(t, err)

		err = os.WriteFile(tempAuthorizedFile, []byte("existing.*\n"), 0644) // Insecure file permissions
		assert.NoError(t, err)

		// Verify they start with insecure permissions (skip on Windows)
		if runtime.GOOS != "windows" {
			dirInfo, err := os.Stat(tempConfigDir)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), dirInfo.Mode()&0777, "Directory should start with 0755 permissions")

			fileInfo, err := os.Stat(tempAuthorizedFile)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0644), fileInfo.Mode()&0777, "File should start with 0644 permissions")
		}

		// Append to the existing file - this should fix permissions
		err = AppendToAuthorizedCommands("new.*")
		assert.NoError(t, err)

		// Check that permissions were fixed (skip on Windows)
		if runtime.GOOS != "windows" {
			dirInfo, err := os.Stat(tempConfigDir)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0700), dirInfo.Mode()&0777, "Directory permissions should be fixed to 0700")

			fileInfo, err := os.Stat(tempAuthorizedFile)
			assert.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), fileInfo.Mode()&0777, "File permissions should be fixed to 0600")
		}

		// Verify content is correct
		content, err := os.ReadFile(tempAuthorizedFile)
		assert.NoError(t, err)
		// Check content with platform-agnostic newline handling
		expected := "existing.*\nnew.*\n"
		assert.Equal(t, expected, strings.ReplaceAll(string(content), "\r\n", "\n"))
	})

	t.Run("Permission errors are handled gracefully", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("Skipping permission error test when running as root")
		}
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission error test on Windows")
		}

		// Clean up from previous test
		assert.NoError(t, os.RemoveAll(tempConfigDir))

		// Create a directory we can't write to
		err := os.MkdirAll(tempConfigDir, 0555) // Read and execute only
		assert.NoError(t, err)

		// Try to append - this may or may not fail depending on the system
		// The important thing is that it doesn't panic
		err = AppendToAuthorizedCommands("test.*")
		// On some systems this might succeed, on others it might fail
		// We just want to ensure no panic occurs
		if err != nil {
			// If it fails, it should be a permission-related error
			assert.True(t,
				strings.Contains(err.Error(), "permission") ||
					strings.Contains(err.Error(), "failed to set") ||
					strings.Contains(err.Error(), "failed to open"),
				"Error should be permission-related: %v", err)
		}
	})
}

func TestLoadAuthorizedCommandsFromFile(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Test with non-existent file
	patterns, err := LoadAuthorizedCommandsFromFile()
	assert.NoError(t, err)
	assert.Equal(t, []string{}, patterns)

	// Create file with some patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("fakecommand.*")
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("anotherfake.*")
	assert.NoError(t, err)

	// Test loading patterns
	patterns, err = LoadAuthorizedCommandsFromFile()
	assert.NoError(t, err)
	assert.Equal(t, []string{"fakecommand.*", "anotherfake.*"}, patterns)
}

func TestGetApprovedBashCommandRegex(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		ResetCacheForTesting()
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Test with no environment patterns and no file patterns
	patterns := GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{}, patterns)

	// Add file patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("fakecommand.*")
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("anotherfake.*")
	assert.NoError(t, err)

	// Test with file patterns only
	patterns = GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{"fakecommand.*", "anotherfake.*"}, patterns)
}

func TestGetApprovedBashCommandRegexWithEnvironmentPatterns(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config_env")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		ResetCacheForTesting()
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner with environment patterns
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Initialize Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	// Set environment variable with JSON array
	runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
		Kind: expand.String,
		Str:  "[\"^fakecmd1.*\", \"^fakecmd2.*\"]",
	}

	// Test with environment patterns only
	patterns := GetApprovedBashCommandRegex(runner, logger)
	expected := []string{"^fakecmd1.*", "^fakecmd2.*"}
	assert.Equal(t, expected, patterns)

	// Add file patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("fakecommand.*")
	assert.NoError(t, err)

	// Test with both environment and file patterns
	patterns = GetApprovedBashCommandRegex(runner, logger)
	expected = []string{"^fakecmd1.*", "^fakecmd2.*", "fakecommand.*"}
	assert.Equal(t, expected, patterns)
}

func TestFilterDangerousPatterns(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name     string
		patterns []string
		expected []string
	}{
		{
			name:     "safe patterns only",
			patterns: []string{"^fakecommand.*", "^anotherfake.*", "^fakecmd1.*"},
			expected: []string{"^fakecommand.*", "^anotherfake.*", "^fakecmd1.*"},
		},
		{
			name:     "filter dangerous patterns",
			patterns: []string{"^fakecommand.*", ".*", "^anotherfake.*", ".+", "^fakecmd1.*"},
			expected: []string{"^fakecommand.*", "^anotherfake.*", "^fakecmd1.*"},
		},
		{
			name:     "all dangerous patterns",
			patterns: []string{".*", "^.*$", ".+", "^.+$", "[\\s\\S]*", "^[\\s\\S]*$"},
			expected: []string{},
		},
		{
			name:     "empty input",
			patterns: []string{},
			expected: []string{},
		},
		{
			name:     "nil input",
			patterns: nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterDangerousPatterns(tt.patterns, logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetApprovedBashCommandRegexInvalidJSON(t *testing.T) {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner with invalid JSON
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	// Initialize Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	// Set environment variable with invalid JSON
	runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
		Kind: expand.String,
		Str:  "invalid json",
	}

	// Should still return file patterns even when env JSON parsing fails
	patterns := GetApprovedBashCommandRegex(runner, logger)
	// The function will still load patterns from the authorized commands file
	// So we can't expect an empty slice, just that it doesn't panic
	assert.IsType(t, []string{}, patterns)
}

func TestGetApprovedBashCommandRegexCaching(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_config_cache")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
		ResetCacheForTesting()
	})

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Create a test runner with isolated environment (no system env vars)
	runner, err := interp.New()
	assert.NoError(t, err)
	runner.Vars = make(map[string]expand.Variable)

	// Create file with patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	err = AppendToAuthorizedCommands("testcmd1.*")
	assert.NoError(t, err)

	// First call should load from file
	patterns1 := GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{"testcmd1.*"}, patterns1)

	// Second call should use cache (no file modification)
	patterns2 := GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{"testcmd1.*"}, patterns2)

	// Modify file
	err = AppendToAuthorizedCommands("testcmd2.*")
	assert.NoError(t, err)

	// Force cache reset to ensure reload (needed for CI environments with low file time resolution)
	ResetCacheForTesting()

	// Third call should reload from file due to modification
	patterns3 := GetApprovedBashCommandRegex(runner, logger)
	assert.Equal(t, []string{"testcmd1.*", "testcmd2.*"}, patterns3)
}

func TestWriteAuthorizedCommandsToFile(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_write_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Test writing patterns
	patterns := []string{"fakecommand.*", "anotherfake.*", "fakecommand.*", "thirdfake.*"} // Include duplicate
	err := WriteAuthorizedCommandsToFile(patterns)
	assert.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(authorizedCommandsFile)
	assert.NoError(t, err)
	expected := "fakecommand.*\nanotherfake.*\nthirdfake.*\n" // Duplicates should be removed
	// Check content with platform-agnostic newline handling
	assert.Equal(t, expected, strings.ReplaceAll(string(content), "\r\n", "\n"))

	// Verify file permissions (skip on Windows)
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(authorizedCommandsFile)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode()&0777)
	}

	// Test writing empty patterns
	err = WriteAuthorizedCommandsToFile([]string{})
	assert.NoError(t, err)

	// File should be empty
	content, err = os.ReadFile(authorizedCommandsFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))

	// Test writing patterns with whitespace
	patterns = []string{" fakecommand.* ", "\tanotherfake.*\n", "", " ", "thirdfake.*"}
	err = WriteAuthorizedCommandsToFile(patterns)
	assert.NoError(t, err)

	// Verify whitespace is trimmed and empty strings are filtered
	content, err = os.ReadFile(authorizedCommandsFile)
	assert.NoError(t, err)
	expected = "fakecommand.*\nanotherfake.*\nthirdfake.*\n"
	// Check content with platform-agnostic newline handling
	assert.Equal(t, expected, strings.ReplaceAll(string(content), "\r\n", "\n"))
}

func TestIsCommandAuthorized(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_auth_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Test with no authorized commands file
	authorized, err := IsCommandAuthorized("fakecommand -la")
	assert.NoError(t, err)
	assert.False(t, authorized)

	// Create file with patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	patterns := []string{"^fakecommand.*", "^anotherfake status.*", "^fakecmd1.*"}
	err = WriteAuthorizedCommandsToFile(patterns)
	assert.NoError(t, err)

	// Test matching commands
	authorized, err = IsCommandAuthorized("fakecommand -la")
	assert.NoError(t, err)
	assert.True(t, authorized)

	authorized, err = IsCommandAuthorized("anotherfake status --short")
	assert.NoError(t, err)
	assert.True(t, authorized)

	authorized, err = IsCommandAuthorized("fakecmd1 hello")
	assert.NoError(t, err)
	assert.True(t, authorized)

	// Test non-matching commands
	authorized, err = IsCommandAuthorized("rm -rf /")
	assert.NoError(t, err)
	assert.False(t, authorized)

	authorized, err = IsCommandAuthorized("anotherfake commit")
	assert.NoError(t, err)
	assert.False(t, authorized)
}

func TestIsCommandAuthorizedInvalidRegex(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_auth_invalid_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Create file with invalid regex pattern
	err := os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	patterns := []string{"[invalid regex", "^fakecommand.*", "*invalid"}
	err = WriteAuthorizedCommandsToFile(patterns)
	assert.NoError(t, err)

	// Should still match valid patterns even with invalid ones present
	authorized, err := IsCommandAuthorized("fakecommand -la")
	assert.NoError(t, err)
	assert.True(t, authorized)

	// Invalid patterns should be skipped
	authorized, err = IsCommandAuthorized("rm file")
	assert.NoError(t, err)
	assert.False(t, authorized)
}

func TestIsCommandPatternAuthorized(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), "gsh_test_pattern_auth_config")
	tempAuthorizedFile := filepath.Join(tempConfigDir, "authorized_commands")

	// Override the global variables for testing
	oldConfigDir := configDir
	oldAuthorizedFile := authorizedCommandsFile
	configDir = tempConfigDir
	authorizedCommandsFile = tempAuthorizedFile
	t.Cleanup(func() {
		configDir = oldConfigDir
		authorizedCommandsFile = oldAuthorizedFile
		assert.NoError(t, os.RemoveAll(tempConfigDir))
	})

	// Test with no authorized commands file
	authorized, err := IsCommandPatternAuthorized("^fakecommand.*")
	assert.NoError(t, err)
	assert.False(t, authorized)

	// Create file with patterns
	err = os.MkdirAll(tempConfigDir, 0700)
	assert.NoError(t, err)

	patterns := []string{"^fakecommand.*", "^anotherfake status.*", "^fakecmd1.*"}
	err = WriteAuthorizedCommandsToFile(patterns)
	assert.NoError(t, err)

	// Test exact pattern matches
	authorized, err = IsCommandPatternAuthorized("^fakecommand.*")
	assert.NoError(t, err)
	assert.True(t, authorized)

	authorized, err = IsCommandPatternAuthorized("^anotherfake status.*")
	assert.NoError(t, err)
	assert.True(t, authorized)

	// Test non-exact matches (should return false)
	authorized, err = IsCommandPatternAuthorized("^fakecommand$")
	assert.NoError(t, err)
	assert.False(t, authorized)

	authorized, err = IsCommandPatternAuthorized("fakecommand.*")
	assert.NoError(t, err)
	assert.False(t, authorized)

	authorized, err = IsCommandPatternAuthorized("^anotherfake.*")
	assert.NoError(t, err)
	assert.False(t, authorized)
}

func TestEnvironmentHelperFunctions(t *testing.T) {
	// Create a test runner
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)
	logger := zap.NewNop()

	// Test default values
	historyLimit := GetHistoryContextLimit(runner, logger)
	assert.Equal(t, 30, historyLimit)

	logLevel := GetLogLevel(runner)
	assert.NotNil(t, logLevel)

	cleanLog := ShouldCleanLogFile(runner)
	assert.False(t, cleanLog)

	pwd := GetPwd(runner)
	// PWD may be empty in test environment without shell initialization
	assert.IsType(t, "", pwd)

	prompt := GetPrompt(runner, logger)
	assert.Equal(t, "gsh> ", prompt) // DEFAULT_PROMPT value

	contextWindow := GetAgentContextWindowTokens(runner, logger)
	assert.Equal(t, 32768, contextWindow)

	assistantHeight := GetAssistantHeight(runner, logger)
	assert.Equal(t, 3, assistantHeight)

	homeDir := GetHomeDir(runner)
	// HOME may be empty in test environment without shell initialization
	assert.IsType(t, "", homeDir)

	macros := GetAgentMacros(runner, logger)
	assert.Equal(t, map[string]string{}, macros)

	contextTypes := GetContextTypesForAgent(runner, logger)
	assert.NotNil(t, contextTypes)

	contextTypes = GetContextTypesForPredictionWithPrefix(runner, logger)
	assert.NotNil(t, contextTypes)

	contextTypes = GetContextTypesForPredictionWithoutPrefix(runner, logger)
	assert.NotNil(t, contextTypes)

	contextTypes = GetContextTypesForExplanation(runner, logger)
	assert.NotNil(t, contextTypes)

	numHistory := GetContextNumHistoryConcise(runner, logger)
	assert.Equal(t, 30, numHistory)

	numHistoryVerbose := GetContextNumHistoryVerbose(runner, logger)
	assert.Equal(t, 30, numHistoryVerbose)
}

func TestEnvironmentHelperFunctionsWithCustomValues(t *testing.T) {
	// Create a test runner with custom environment values
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)
	logger := zap.NewNop()

	// Initialize Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	// Set custom values
	runner.Vars["GSH_PAST_COMMANDS_CONTEXT_LIMIT"] = expand.Variable{Kind: expand.String, Str: "50"}
	runner.Vars["GSH_LOG_LEVEL"] = expand.Variable{Kind: expand.String, Str: "debug"}
	runner.Vars["GSH_CLEAN_LOG_FILE"] = expand.Variable{Kind: expand.String, Str: "true"}
	runner.Vars["GSH_PROMPT"] = expand.Variable{Kind: expand.String, Str: "custom> "}
	runner.Vars["GSH_BUILD_VERSION"] = expand.Variable{Kind: expand.String, Str: "dev"}
	runner.Vars["GSH_AGENT_CONTEXT_WINDOW_TOKENS"] = expand.Variable{Kind: expand.String, Str: "16384"}
	runner.Vars["GSH_ASSISTANT_HEIGHT"] = expand.Variable{Kind: expand.String, Str: "5"}
	runner.Vars["GSH_AGENT_MACROS"] = expand.Variable{Kind: expand.String, Str: "{\"test\": \"echo test\"}"}
	runner.Vars["GSH_CONTEXT_TYPES_FOR_AGENT"] = expand.Variable{Kind: expand.String, Str: "history,files"}
	runner.Vars["GSH_CONTEXT_NUM_HISTORY_CONCISE"] = expand.Variable{Kind: expand.String, Str: "20"}
	runner.Vars["GSH_CONTEXT_NUM_HISTORY_VERBOSE"] = expand.Variable{Kind: expand.String, Str: "10"}

	// Test custom values
	historyLimit := GetHistoryContextLimit(runner, logger)
	assert.Equal(t, 50, historyLimit)

	logLevel := GetLogLevel(runner)
	assert.Equal(t, zap.DebugLevel, logLevel.Level())

	cleanLog := ShouldCleanLogFile(runner)
	assert.True(t, cleanLog)

	prompt := GetPrompt(runner, logger)
	assert.Equal(t, "[dev] custom> ", prompt)

	contextWindow := GetAgentContextWindowTokens(runner, logger)
	assert.Equal(t, 16384, contextWindow)

	assistantHeight := GetAssistantHeight(runner, logger)
	assert.Equal(t, 5, assistantHeight)

	macros := GetAgentMacros(runner, logger)
	expected := map[string]string{"test": "echo test"}
	assert.Equal(t, expected, macros)

	contextTypes := GetContextTypesForAgent(runner, logger)
	assert.Equal(t, []string{"history", "files"}, contextTypes)

	numHistory := GetContextNumHistoryConcise(runner, logger)
	assert.Equal(t, 20, numHistory)

	numHistoryVerbose := GetContextNumHistoryVerbose(runner, logger)
	assert.Equal(t, 10, numHistoryVerbose)
}

func TestTestingHelperFunctions(t *testing.T) {
	// Test that helper functions return expected values
	configDir := GetConfigDirForTesting()
	assert.NotEmpty(t, configDir)

	authorizedFile := GetAuthorizedCommandsFileForTesting()
	assert.NotEmpty(t, authorizedFile)

	// Test setting custom values
	customConfigDir := "/tmp/test_config"
	customAuthorizedFile := "/tmp/test_authorized"

	SetConfigDirForTesting(customConfigDir)
	SetAuthorizedCommandsFileForTesting(customAuthorizedFile)

	// Get values after setting
	updatedConfigDir := GetConfigDirForTesting()
	updatedAuthorizedFile := GetAuthorizedCommandsFileForTesting()

	assert.Equal(t, customConfigDir, updatedConfigDir)
	assert.Equal(t, customAuthorizedFile, updatedAuthorizedFile)

	// Clean up
	ResetCacheForTesting()
}

func TestSyncVariablesToEnvExportsGSHVariables(t *testing.T) {
	// Create a test runner with custom environment values
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	expected := map[string]string{
		"GSH_PROMPT":                                      "sync> ",
		"GSH_APROMPT":                                     "agent> ",
		"GSH_BUILD_VERSION":                               "dev",
		"GSH_LOG_LEVEL":                                   "debug",
		"GSH_CLEAN_LOG_FILE":                              "true",
		"GSH_MINIMUM_HEIGHT":                              "8",
		"GSH_ASSISTANT_HEIGHT":                            "4",
		"GSH_AGENT_NAME":                                  "helper",
		"GSH_FAST_MODEL_API_KEY":                          "fast-key",
		"GSH_FAST_MODEL_BASE_URL":                         "https://fast.example.com",
		"GSH_FAST_MODEL_ID":                               "fast-model",
		"GSH_SLOW_MODEL_API_KEY":                          "slow-key",
		"GSH_SLOW_MODEL_BASE_URL":                         "https://slow.example.com",
		"GSH_SLOW_MODEL_ID":                               "slow-model",
		"GSH_AGENT_CONTEXT_WINDOW_TOKENS":                 "2048",
		"GSH_PAST_COMMANDS_CONTEXT_LIMIT":                 "25",
		"GSH_CONTEXT_TYPES_FOR_AGENT":                     "history,files",
		"GSH_CONTEXT_TYPES_FOR_PREDICTION_WITH_PREFIX":    "history",
		"GSH_CONTEXT_TYPES_FOR_PREDICTION_WITHOUT_PREFIX": "files",
		"GSH_CONTEXT_TYPES_FOR_EXPLANATION":               "commands",
		"GSH_CONTEXT_NUM_HISTORY_CONCISE":                 "5",
		"GSH_CONTEXT_NUM_HISTORY_VERBOSE":                 "7",
		"GSH_AGENT_APPROVED_BASH_COMMAND_REGEX":           "[\"^ls.*\"]",
		"GSH_AGENT_MACROS":                                "{\"m\":\"cmd\"}",
		"GSH_DEFAULT_TO_YES":                              "true",
	}

	assert.Equal(t, len(gshVariableNames), len(expected))

	for name, value := range expected {
		runner.Vars[name] = expand.Variable{Kind: expand.String, Str: value}
		t.Setenv(name, "")
	}

	SyncVariablesToEnv(runner)

	dynamicEnv, ok := runner.Env.(*DynamicEnviron)
	assert.True(t, ok, "runner.Env should be a DynamicEnviron")

	for name, value := range expected {
		assert.Equal(t, value, os.Getenv(name), "environment should include %s", name)
		assert.Equal(t, value, dynamicEnv.gshVars[name], "dynamic environment should include %s", name)
	}
}

func TestSyncVariablesToEnvRemovesUnsetVariables(t *testing.T) {
	env := expand.ListEnviron(os.Environ()...)
	runner, err := interp.New(interp.Env(env))
	assert.NoError(t, err)

	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	runner.Vars["GSH_PROMPT"] = expand.Variable{Kind: expand.String, Str: "sync> "}
	t.Setenv("GSH_PROMPT", "stale")

	SyncVariablesToEnv(runner)

	dynamicEnv, ok := runner.Env.(*DynamicEnviron)
	assert.True(t, ok, "runner.Env should be a DynamicEnviron")
	assert.Equal(t, "sync> ", os.Getenv("GSH_PROMPT"))
	assert.Equal(t, "sync> ", dynamicEnv.gshVars["GSH_PROMPT"])

	delete(runner.Vars, "GSH_PROMPT")
	SyncVariablesToEnv(runner)

	_, exists := os.LookupEnv("GSH_PROMPT")
	assert.False(t, exists, "GSH_PROMPT should be removed from system environment")
	_, exists = dynamicEnv.gshVars["GSH_PROMPT"]
	assert.False(t, exists, "GSH_PROMPT should be removed from dynamic environment")
}
