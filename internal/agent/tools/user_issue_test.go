package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestUserReportedIssue(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gsh_user_issue_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Set up the authorized commands file path
	authorizedFile := filepath.Join(tempDir, "authorized_commands")

	// Use the testing helper functions to set the correct paths
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(authorizedFile)

	// Write the user's authorized_commands file content
	fileContent := "^ls.*\n^awk.*\n^sort.*\n"
	err = os.WriteFile(authorizedFile, []byte(fileContent), 0600)
	require.NoError(t, err)

	t.Logf("Created authorized_commands file with content:\n%s", fileContent)

	// Test the user's command
	userCommand := `find . -maxdepth 1 -type f -exec stat -f '%N|%p|%z|%Sm' -t '%Y-%m-%d %H:%M:%S' {} \; | awk -F'|' '{size=$3; if(size>1024*1024) size=sprintf("%.1fM", size/(1024*1024)); else if(size>1024) size=sprintf("%.1fK", size/1024); else size=sprintf("%dB", size); printf "%-40s %-10s %8s %s\n", $1, $2, size, $4}' | head -20`

	t.Logf("Testing command: %s", userCommand)

	// Extract individual commands
	individualCommands, err := ExtractCommands(userCommand)
	require.NoError(t, err)

	t.Logf("Individual commands extracted:")
	for i, cmd := range individualCommands {
		t.Logf("  %d: %s", i+1, cmd)
	}

	// Create a mock runner and logger
	runner := &interp.Runner{}
	logger := zap.NewNop()

	// Get approved patterns (should include file patterns)
	approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)
	t.Logf("All approved patterns:")
	for i, pattern := range approvedPatterns {
		t.Logf("  %d: %s", i+1, pattern)
	}

	// Test validation
	isPreApproved, err := ValidateCompoundCommand(userCommand, approvedPatterns)
	require.NoError(t, err)

	t.Logf("Command validation result: %v", isPreApproved)

	// Test each individual command
	t.Logf("Testing each individual command:")
	for _, cmd := range individualCommands {
		approved := false
		matchedPattern := ""
		for _, pattern := range approvedPatterns {
			if matched, _ := regexp.MatchString(pattern, cmd); matched {
				approved = true
				matchedPattern = pattern
				break
			}
		}

		status := "❌ BLOCKED"
		if approved {
			status = "✅ APPROVED"
		}

		t.Logf("  %s: %s", status, cmd)
		if approved {
			t.Logf("    (matched: %s)", matchedPattern)
		}
	}

	// The command should be blocked because it contains 'find' and 'stat' which are not authorized
	assert.False(t, isPreApproved, "Command should be blocked because it contains unauthorized commands (find, stat)")
}

func TestUserReportedIssueWithEnvironmentVariable(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gsh_user_issue_env_test")
	require.NoError(t, err)

	// Set up the authorized commands file path
	authorizedFile := filepath.Join(tempDir, "authorized_commands")

	// Use the testing helper functions to set the correct paths
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(authorizedFile)
	defer environment.ResetCacheForTesting()

	// Write the user's authorized_commands file content
	fileContent := "^ls.*\n^awk.*\n^sort.*\n"
	err = os.WriteFile(authorizedFile, []byte(fileContent), 0600)
	require.NoError(t, err)

	t.Logf("Created authorized_commands file with content:\n%s", fileContent)

	// Test the user's command
	userCommand := `find . -maxdepth 1 -type f -exec stat -f '%N|%p|%z|%Sm' -t '%Y-%m-%d %H:%M:%S' {} \; | awk -F'|' '{size=$3; if(size>1024*1024) size=sprintf("%.1fM", size/(1024*1024)); else if(size>1024) size=sprintf("%.1fK", size/1024); else size=sprintf("%dB", size); printf "%-40s %-10s %8s %s\n", $1, $2, size, $4}' | head -20`

	t.Logf("Testing command: %s", userCommand)

	// Create a runner with environment variable that might allow broader patterns
	runner := &interp.Runner{
		Vars: make(map[string]expand.Variable),
	}

	// Simulate the user having a broad environment variable pattern
	runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
		Kind: expand.String,
		Str:  `[".*"]`, // This would allow everything - simulating a potential security issue
	}

	logger := zap.NewNop()

	// Get approved patterns (now includes environment variable)
	approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)

	t.Logf("All approved patterns (including environment variable):")
	for i, pattern := range approvedPatterns {
		t.Logf("  %d: %s", i+1, pattern)
	}

	// Test compound command validation
	isValid, err := ValidateCompoundCommand(userCommand, approvedPatterns)
	require.NoError(t, err)

	t.Logf("Command validation result with environment variable: %t", isValid)

	// This test demonstrates that dangerous environment variable patterns are now filtered out
	if isValid {
		t.Logf("⚠️  WARNING: Command was allowed despite security filtering!")
	} else {
		t.Logf("✅ SUCCESS: Dangerous environment variable pattern was filtered out, command properly blocked")
	}

	// The command should now be blocked because dangerous patterns are filtered out
	assert.False(t, isValid, "Command should be blocked because dangerous environment variable patterns are filtered out")
}

func TestEnvironmentVariablePatternFiltering(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gsh_env_filter_test")
	require.NoError(t, err)

	// Set up the authorized commands file path
	authorizedFile := filepath.Join(tempDir, "authorized_commands")

	// Use the testing helper functions to set the correct paths
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(authorizedFile)
	defer environment.ResetCacheForTesting()

	// Create authorized_commands file with basic patterns
	fileContent := "^ls.*\n^echo.*\n"
	err = os.WriteFile(authorizedFile, []byte(fileContent), 0600)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		envPatterns    []string
		expectFiltered []string
		expectKept     []string
	}{
		{
			name:           "legitimate patterns should be kept",
			envPatterns:    []string{"^git.*", "^npm.*", "^docker.*"},
			expectFiltered: []string{},
			expectKept:     []string{"^git.*", "^npm.*", "^docker.*", "^ls.*", "^echo.*"},
		},
		{
			name:           "dangerous patterns should be filtered",
			envPatterns:    []string{".*", "^.*$", ".+", "^.+$"},
			expectFiltered: []string{".*", "^.*$", ".+", "^.+$"},
			expectKept:     []string{"^ls.*", "^echo.*"},
		},
		{
			name:           "mixed patterns - keep legitimate, filter dangerous",
			envPatterns:    []string{"^git.*", ".*", "^npm.*", ".+", "^docker.*"},
			expectFiltered: []string{".*", ".+"},
			expectKept:     []string{"^git.*", "^npm.*", "^docker.*", "^ls.*", "^echo.*"},
		},
		{
			name:           "complex dangerous patterns should be filtered",
			envPatterns:    []string{"[\\\\s\\\\S]*", "^[\\\\s\\\\S]*$", "^valid.*"},
			expectFiltered: []string{"[\\\\s\\\\S]*", "^[\\\\s\\\\S]*$"},
			expectKept:     []string{"^valid.*", "^ls.*", "^echo.*"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner with the test environment patterns
			runner := &interp.Runner{
				Vars: make(map[string]expand.Variable),
			}

			if len(tc.envPatterns) > 0 {
				patternsJSON := `["` + tc.envPatterns[0]
				for i := 1; i < len(tc.envPatterns); i++ {
					patternsJSON += `", "` + tc.envPatterns[i]
				}
				patternsJSON += `"]`

				runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
					Kind: expand.String,
					Str:  patternsJSON,
				}
			}

			logger := zap.NewNop()

			// Get approved patterns (should filter dangerous ones)
			approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)

			t.Logf("Environment patterns: %v", tc.envPatterns)
			t.Logf("Final approved patterns: %v", approvedPatterns)

			// Verify that all expected patterns are kept
			for _, expectedPattern := range tc.expectKept {
				found := false
				for _, actualPattern := range approvedPatterns {
					if actualPattern == expectedPattern {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected pattern '%s' should be kept", expectedPattern)
			}

			// Verify that dangerous patterns are filtered out
			for _, filteredPattern := range tc.expectFiltered {
				found := false
				for _, actualPattern := range approvedPatterns {
					if actualPattern == filteredPattern {
						found = true
						break
					}
				}
				assert.False(t, found, "Dangerous pattern '%s' should be filtered out", filteredPattern)
			}
		})
	}
}
