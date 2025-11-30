package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionPreselectionFix(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gsh_permission_preselection_test")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	})

	// Set up the authorized commands file path
	authorizedFile := filepath.Join(tempDir, "authorized_commands")
	t.Setenv("GSH_CONFIG_DIR", tempDir)

	// Use the testing helper functions to set the correct paths
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(authorizedFile)

	// Ensure we start with a clean file - remove any existing file first
	if err := os.Remove(authorizedFile); err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}

	// Write the runtime pattern to the file: "^awk.*" (what GenerateCommandRegex("awk") would generate)
	// This is the actual pattern that gets saved when a user selects "awk" in the permissions menu
	// We want to verify that only the "awk" prefix (not "awk -F'|'" etc.) gets pre-selected
	err = os.WriteFile(authorizedFile, []byte("^awk.*\n"), 0600)
	require.NoError(t, err)

	t.Log("Authorized commands file contains: ^awk.*")

	// Debug: check what's actually in the file
	fileContent, err := os.ReadFile(authorizedFile)
	require.NoError(t, err)
	t.Logf("File content (raw): %q", string(fileContent))

	// Debug: check what patterns are loaded
	patterns, err := environment.LoadAuthorizedCommandsFromFile()
	require.NoError(t, err)
	t.Logf("Loaded patterns: %v", patterns)

	// Test the compound command that includes awk
	command := `stat -f '%N|%p|%z|%Sm|%Su|%Sg' -t '%Y-%m-%d %H:%M:%S' * | awk -F'|' '{size=$3; if(size>1048576) size=sprintf("%.1fM", size/1048576); else if(size>1024) size=sprintf("%.1fK", size/1024); printf "%-30s %s %10s %s %s %s\n", $1, $2, size, $4, $5, $6}' | head -15`

	// Extract individual commands
	individualCommands, err := ExtractCommands(command)
	require.NoError(t, err)

	t.Logf("Individual commands extracted: %v", individualCommands)

	// Track which prefixes should be pre-selected
	expectedPreselected := make(map[string]bool)
	actualPreselected := make(map[string]bool)

	// Test each command's prefixes and check pre-selection
	for _, cmd := range individualCommands {
		t.Logf("=== Testing command: %s ===", cmd)
		prefixes := GenerateCommandPrefixes(cmd)

		for _, prefix := range prefixes {
			// Generate the preselection pattern for this prefix (what would be saved when selected)
			regexPattern := GeneratePreselectionPattern(prefix)

			// Check if this exact pattern exists in the file (literal string matching)
			isPatternAuthorized, err := environment.IsCommandPatternAuthorized(regexPattern)
			require.NoError(t, err)

			// Also check the old way (runtime matching) for comparison
			isRuntimeAuthorized, err := environment.IsCommandAuthorized(prefix)
			require.NoError(t, err)

			t.Logf("  Prefix: %-30s | Pattern: %-15s | PreSelect: %v | Runtime: %v",
				prefix, regexPattern, isPatternAuthorized, isRuntimeAuthorized)

			// Record the results
			actualPreselected[prefix] = isPatternAuthorized

			// Only "awk" should generate the pattern "^awk.*" that matches what's in the file
			// Other prefixes like "awk -F'|'" generate "^awk -F'|'.*" which won't match
			if prefix == "awk" {
				expectedPreselected[prefix] = true
			} else {
				expectedPreselected[prefix] = false
			}
		}
	}

	// Verify the fix: only "awk" should be pre-selected
	t.Log("=== Verification ===")
	for prefix, shouldBePreselected := range expectedPreselected {
		actuallyPreselected := actualPreselected[prefix]
		t.Logf("Prefix '%s': Expected preselected=%v, Actual preselected=%v",
			prefix, shouldBePreselected, actuallyPreselected)

		assert.Equal(t, shouldBePreselected, actuallyPreselected,
			"Prefix '%s' preselection should be %v but was %v",
			prefix, shouldBePreselected, actuallyPreselected)
	}

	// Specifically verify the problematic case from the user's feedback
	t.Log("=== Specific Test Cases ===")

	// These should NOT be pre-selected (they would match ^awk.* at runtime but shouldn't be pre-selected)
	notPreselectedPrefixes := []string{"awk -F'|'", "awk -F'|' '{size=$3; if(size>1048576) size=sprintf(\"%.1fM..."}
	for _, prefix := range notPreselectedPrefixes {
		if actualPreselected[prefix] {
			t.Errorf("FAIL: Prefix '%s' should NOT be pre-selected but was", prefix)
		} else {
			t.Logf("PASS: Prefix '%s' correctly NOT pre-selected", prefix)
		}
	}

	// This should be pre-selected (exact match)
	if actualPreselected["awk"] {
		t.Log("PASS: Prefix 'awk' correctly pre-selected")
	} else {
		t.Error("FAIL: Prefix 'awk' should be pre-selected but wasn't")
	}
}
