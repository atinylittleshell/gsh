package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/environment"
	"go.uber.org/zap"
)

// TestCreateFileAlwaysWorkflow tests the complete 'always' workflow for file creation
func TestCreateFileAlwaysWorkflow(t *testing.T) {
	// This test is no longer relevant since we removed the "always" feature
	t.Skip("Test skipped: 'always' feature has been removed")
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
		_ = os.RemoveAll(tempConfigDir)
		environment.ResetCacheForTesting()
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer func() {
		_ = logger.Sync()
	}()

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
				// Pattern generation function removed - this test is no longer relevant
				t.Skip("Pattern generation function removed")
			})
		}
	})

	t.Run("Very long file paths", func(t *testing.T) {
		// Pattern generation function removed - this test is no longer relevant
		t.Skip("Pattern generation function removed")
	})

	t.Run("Empty extension handling", func(t *testing.T) {
		// Pattern generation function removed - this test is no longer relevant
		t.Skip("Pattern generation function removed")
	})

	t.Run("Root directory files", func(t *testing.T) {
		// Pattern generation function removed - this test is no longer relevant
		t.Skip("Pattern generation function removed")
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
			// Pattern generation function removed - this test is no longer relevant
			t.Skip("Pattern generation function removed")
		})
	}
}
