package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// TestCreateFileAlwaysWorkflow tests the complete 'always' workflow for file creation
func TestCreateFileAlwaysWorkflow(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_createfile_always_%d", time.Now().UnixNano()))
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

	t.Run("Files with Extensions", func(t *testing.T) {
		testFilesWithExtensions(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Files without Extensions", func(t *testing.T) {
		testFilesWithoutExtensions(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Files in Different Directories", func(t *testing.T) {
		testFilesInDifferentDirectories(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Files with Multiple Dots", func(t *testing.T) {
		testFilesWithMultipleDots(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Pattern Matching Logic", func(t *testing.T) {
		testPatternMatchingLogic(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})

	t.Run("Integration with Existing Patterns", func(t *testing.T) {
		testIntegrationWithExistingPatterns(t, runner, logger, tempConfigDir, tempAuthorizedFile)
	})
}

func testFilesWithExtensions(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	testCases := []struct {
		name            string
		filePath        string
		expectedPattern string
		similarFiles    []string
		differentFiles  []string
	}{
		{
			name:            "Go file in src directory",
			filePath:        "/tmp/test/src/main.go",
			expectedPattern: "create_file:/tmp/test/src/.*\\\\.go$",
			similarFiles:    []string{"/tmp/test/src/utils.go", "/tmp/test/src/handler.go"},
			differentFiles:  []string{"/tmp/test/src/main.js", "/tmp/other/main.go"},
		},
		{
			name:            "Text file in root",
			filePath:        "/tmp/test.txt",
			expectedPattern: "create_file:/tmp/.*\\\\.txt$",
			similarFiles:    []string{"/tmp/another.txt", "/tmp/data.txt"},
			differentFiles:  []string{"/tmp/test.log", "/other/test.txt"},
		},
		{
			name:            "JSON config file",
			filePath:        "/home/user/config.json",
			expectedPattern: "create_file:/home/user/.*\\\\.json$",
			similarFiles:    []string{"/home/user/package.json", "/home/user/settings.json"},
			differentFiles:  []string{"/home/user/config.yaml", "/home/other/config.json"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up for each test case
			os.RemoveAll(tempConfigDir)
			environment.ResetCacheForTesting()

			// Create temporary file for testing
			tempFile, err := os.CreateTemp("", "gsh_createfile_test")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			// Step 1: First execution with 'always' response
			t.Run("First execution with 'always' response", func(t *testing.T) {
				oldUserConfirmation := userConfirmation
				userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
					return "always"
				}
				defer func() {
					userConfirmation = oldUserConfirmation
				}()

				params := map[string]any{
					"path":    tempFile.Name(),
					"content": "test content",
				}
				result := CreateFileTool(runner, logger, params)

				// Verify the file was created successfully
				assert.Contains(t, result, "successfully created", "File should be created")

				// Verify the pattern was saved
				patterns, err := environment.LoadAuthorizedCommandsFromFile()
				assert.NoError(t, err)

				// Generate expected pattern for the temp file
				expectedPattern := GenerateFileOperationRegex(tempFile.Name(), "create_file")
				assert.Contains(t, patterns, expectedPattern, "Pattern should be saved to file")

				// Verify file content
				content, err := os.ReadFile(tempFile.Name())
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(content))
			})

			// Step 2: Test pattern generation matches expected
			t.Run("Pattern generation", func(t *testing.T) {
				actualPattern := GenerateFileOperationRegex(tc.filePath, "create_file")
				assert.Equal(t, tc.expectedPattern, actualPattern)
			})
		})
	}
}

func testFilesWithoutExtensions(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	testCases := []struct {
		name            string
		filePath        string
		expectedPattern string
	}{
		{
			name:            "README file",
			filePath:        "/home/user/README",
			expectedPattern: "create_file:/home/user/README$",
		},
		{
			name:            "Makefile",
			filePath:        "/project/Makefile",
			expectedPattern: "create_file:/project/Makefile$",
		},
		{
			name:            "LICENSE file",
			filePath:        "/repo/LICENSE",
			expectedPattern: "create_file:/repo/LICENSE$",
		},
		{
			name:            "Dockerfile",
			filePath:        "/app/Dockerfile",
			expectedPattern: "create_file:/app/Dockerfile$",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test pattern generation
			actualPattern := GenerateFileOperationRegex(tc.filePath, "create_file")
			assert.Equal(t, tc.expectedPattern, actualPattern)

			// Test with actual file creation
			tempFile, err := os.CreateTemp("", "gsh_createfile_noext_test")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			oldUserConfirmation := userConfirmation
			userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
				return "always"
			}
			defer func() {
				userConfirmation = oldUserConfirmation
			}()

			params := map[string]any{
				"path":    tempFile.Name(),
				"content": "test content for file without extension",
			}
			result := CreateFileTool(runner, logger, params)

			// Verify the file was created successfully
			assert.Contains(t, result, "successfully created", "File should be created")

			// Verify the pattern was saved
			patterns, err := environment.LoadAuthorizedCommandsFromFile()
			assert.NoError(t, err)

			expectedPattern := GenerateFileOperationRegex(tempFile.Name(), "create_file")
			assert.Contains(t, patterns, expectedPattern, "Pattern should be saved to file")
		})
	}
}

func testFilesInDifferentDirectories(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	testCases := []struct {
		name            string
		filePath        string
		expectedPattern string
	}{
		{
			name:            "File in /tmp",
			filePath:        "/tmp/test.txt",
			expectedPattern: "create_file:/tmp/.*\\\\.txt$",
		},
		{
			name:            "File in nested directory",
			filePath:        "/home/user/projects/myapp/src/main.go",
			expectedPattern: "create_file:/home/user/projects/myapp/src/.*\\\\.go$",
		},
		{
			name:            "File in relative path",
			filePath:        "./local/file.go",
			expectedPattern: "create_file:local/.*\\\\.go$",
		},
		{
			name:            "File in current directory",
			filePath:        "./config.yaml",
			expectedPattern: "create_file:\\./.*\\\\.yaml$",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualPattern := GenerateFileOperationRegex(tc.filePath, "create_file")
			assert.Equal(t, tc.expectedPattern, actualPattern)
		})
	}
}

func testFilesWithMultipleDots(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	testCases := []struct {
		name            string
		filePath        string
		expectedPattern string
	}{
		{
			name:            "Archive with multiple extensions",
			filePath:        "/backup/archive.tar.gz",
			expectedPattern: "create_file:/backup/.*\\\\.gz$",
		},
		{
			name:            "Backup file with date",
			filePath:        "/logs/backup.2023.txt",
			expectedPattern: "create_file:/logs/.*\\\\.txt$",
		},
		{
			name:            "Config file with environment",
			filePath:        "/config/app.prod.json",
			expectedPattern: "create_file:/config/.*\\\\.json$",
		},
		{
			name:            "Test file with multiple dots",
			filePath:        "/tests/unit.test.js",
			expectedPattern: "create_file:/tests/.*\\\\.js$",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualPattern := GenerateFileOperationRegex(tc.filePath, "create_file")
			assert.Equal(t, tc.expectedPattern, actualPattern)
		})
	}
}

func testPatternMatchingLogic(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	// Create a temporary directory for test files
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_pattern_test_%d", time.Now().UnixNano()))
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	t.Run("After approving txt files, pattern is saved correctly", func(t *testing.T) {
		// NOTE: The current createfile implementation does NOT check for existing patterns
		// before prompting. This test verifies that patterns are saved correctly, but
		// auto-approval functionality would need to be implemented in the createfile tool.

		// Clean up
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()

		// Step 1: Create first file with 'always' response
		firstFile := filepath.Join(testDir, "first.txt")
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    firstFile,
			"content": "first file content",
		}
		result := CreateFileTool(runner, logger, params)
		assert.Contains(t, result, "successfully created")

		// Verify the pattern was saved
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)
		expectedPattern := GenerateFileOperationRegex(firstFile, "create_file")
		assert.Contains(t, patterns, expectedPattern, "Pattern should be saved to file")

		// Verify file exists
		assert.FileExists(t, firstFile)
	})

	t.Run("Different extensions should still prompt", func(t *testing.T) {
		// Clean up
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()

		// Step 1: Create txt file with 'always' response
		txtFile := filepath.Join(testDir, "test.txt")
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    txtFile,
			"content": "txt file content",
		}
		result := CreateFileTool(runner, logger, params)
		assert.Contains(t, result, "successfully created")

		// Step 2: Try to create js file - should prompt
		jsFile := filepath.Join(testDir, "test.js")
		promptCalled := false
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}

		params = map[string]any{
			"path":    jsFile,
			"content": "js file content",
		}
		result = CreateFileTool(runner, logger, params)

		// Verify user was prompted for different extension
		assert.True(t, promptCalled, "User should be prompted for different extension")
		assert.Contains(t, result, "User declined this request")
	})

	t.Run("Different directories should still prompt", func(t *testing.T) {
		// Clean up
		os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()

		// Create another test directory
		otherDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_other_test_%d", time.Now().UnixNano()))
		err := os.MkdirAll(otherDir, 0755)
		require.NoError(t, err)
		defer os.RemoveAll(otherDir)

		// Step 1: Create file in first directory with 'always' response
		firstFile := filepath.Join(testDir, "test.txt")
		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    firstFile,
			"content": "first directory file",
		}
		result := CreateFileTool(runner, logger, params)
		assert.Contains(t, result, "successfully created")

		// Step 2: Try to create file in different directory - should prompt
		otherFile := filepath.Join(otherDir, "test.txt")
		promptCalled := false
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			promptCalled = true
			return "n"
		}

		params = map[string]any{
			"path":    otherFile,
			"content": "other directory file",
		}
		result = CreateFileTool(runner, logger, params)

		// Verify user was prompted for different directory
		assert.True(t, promptCalled, "User should be prompted for different directory")
		assert.Contains(t, result, "User declined this request")
	})
}

func testIntegrationWithExistingPatterns(t *testing.T, runner *interp.Runner, logger *zap.Logger, tempConfigDir, tempAuthorizedFile string) {
	// Clean up any existing patterns
	os.RemoveAll(tempConfigDir)
	environment.ResetCacheForTesting()

	t.Run("File patterns should coexist with bash command patterns", func(t *testing.T) {
		// Pre-populate with some bash command patterns
		err := os.MkdirAll(tempConfigDir, 0755)
		require.NoError(t, err)

		initialPatterns := []string{
			"^ls.*",
			"^git status.*",
			"^echo.*",
		}

		// Write initial patterns to file
		file, err := os.Create(tempAuthorizedFile)
		require.NoError(t, err)
		for _, pattern := range initialPatterns {
			_, err = file.WriteString(pattern + "\n")
			require.NoError(t, err)
		}
		file.Close()

		// Create a file with 'always' response
		tempFile, err := os.CreateTemp("", "gsh_integration_test")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		oldUserConfirmation := userConfirmation
		userConfirmation = func(logger *zap.Logger, question string, explanation string) string {
			return "always"
		}
		defer func() {
			userConfirmation = oldUserConfirmation
		}()

		params := map[string]any{
			"path":    tempFile.Name(),
			"content": "integration test content",
		}
		result := CreateFileTool(runner, logger, params)
		assert.Contains(t, result, "successfully created")

		// Verify all patterns are present
		patterns, err := environment.LoadAuthorizedCommandsFromFile()
		assert.NoError(t, err)

		// Check original bash patterns are still there
		for _, pattern := range initialPatterns {
			assert.Contains(t, patterns, pattern, "Original bash pattern should be preserved")
		}

		// Check new file pattern was added
		expectedFilePattern := GenerateFileOperationRegex(tempFile.Name(), "create_file")
		assert.Contains(t, patterns, expectedFilePattern, "New file pattern should be added")

		// Verify total count
		assert.GreaterOrEqual(t, len(patterns), len(initialPatterns)+1, "Should have at least original patterns plus new file pattern")
	})

	t.Run("File patterns should not interfere with bash command matching", func(t *testing.T) {
		// This test would require integration with bash tool testing
		// For now, we verify that file patterns have the correct prefix
		testCases := []string{
			"/tmp/test.txt",
			"/home/user/config.json",
			"/project/README",
		}

		for _, filePath := range testCases {
			pattern := GenerateFileOperationRegex(filePath, "create_file")
			assert.True(t, strings.HasPrefix(pattern, "create_file:"), "File patterns should have create_file: prefix")
			assert.False(t, strings.HasPrefix(pattern, "^"), "File patterns should not start with ^ like bash patterns")
		}
	})
}

// TestCreateFileAlwaysEdgeCases tests edge cases and error handling
func TestCreateFileAlwaysEdgeCases(t *testing.T) {
	// Create a temporary config directory for testing
	tempConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("gsh_test_createfile_edge_%d", time.Now().UnixNano()))
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

	// Note: runner not needed for these edge case tests

	t.Run("Special characters in file paths", func(t *testing.T) {
		testCases := []struct {
			name     string
			filePath string
		}{
			{
				name:     "File with spaces",
				filePath: "/tmp/file with spaces.txt",
			},
			{
				name:     "File with special characters",
				filePath: "/tmp/file-with_special.chars.txt",
			},
			{
				name:     "File with parentheses",
				filePath: "/tmp/file(1).txt",
			},
			{
				name:     "File with brackets",
				filePath: "/tmp/file[backup].txt",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test that pattern generation doesn't crash
				pattern := GenerateFileOperationRegex(tc.filePath, "create_file")
				assert.NotEmpty(t, pattern, "Pattern should be generated for special characters")
				assert.Contains(t, pattern, "create_file:", "Pattern should have correct prefix")
			})
		}
	})

	t.Run("Very long file paths", func(t *testing.T) {
		longPath := "/very/long/path/that/goes/on/and/on/and/on/and/on/and/on/and/on/file.txt"
		pattern := GenerateFileOperationRegex(longPath, "create_file")
		assert.NotEmpty(t, pattern, "Pattern should be generated for long paths")
		assert.Contains(t, pattern, "create_file:", "Pattern should have correct prefix")
	})

	t.Run("Empty extension handling", func(t *testing.T) {
		filePath := "/tmp/file."
		pattern := GenerateFileOperationRegex(filePath, "create_file")
		// File with just a dot should be treated as having an empty extension
		assert.Contains(t, pattern, "\\.$", "Should match files ending with just a dot")
	})

	t.Run("Root directory files", func(t *testing.T) {
		filePath := "/file.txt"
		pattern := GenerateFileOperationRegex(filePath, "create_file")
		expectedPattern := "create_file://.*\\\\.txt$"
		assert.Equal(t, expectedPattern, pattern, "Root directory files should be handled correctly")
	})
}

// TestCreateFilePatternFormat tests that the pattern format is correct
func TestCreateFilePatternFormat(t *testing.T) {
	testCases := []struct {
		name            string
		filePath        string
		operation       string
		expectedPattern string
	}{
		{
			name:            "Standard create_file pattern",
			filePath:        "/home/user/test.go",
			operation:       "create_file",
			expectedPattern: "create_file:/home/user/.*\\\\.go$",
		},
		{
			name:            "Edit_file pattern",
			filePath:        "/home/user/test.go",
			operation:       "edit_file",
			expectedPattern: "edit_file:/home/user/.*\\\\.go$",
		},
		{
			name:            "Pattern format consistency",
			filePath:        "/tmp/config.json",
			operation:       "create_file",
			expectedPattern: "create_file:/tmp/.*\\\\.json$",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualPattern := GenerateFileOperationRegex(tc.filePath, tc.operation)
			assert.Equal(t, tc.expectedPattern, actualPattern)

			// Verify pattern format
			parts := strings.Split(actualPattern, ":")
			assert.Len(t, parts, 2, "Pattern should have exactly one colon separator")
			assert.Equal(t, tc.operation, parts[0], "First part should be the operation")
			assert.True(t, strings.HasSuffix(parts[1], "$"), "Pattern should end with $")
		})
	}
}
