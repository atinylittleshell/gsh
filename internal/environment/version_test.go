package environment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionFileFormat(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	versionFile := filepath.Join(repoRoot, "VERSION")

	t.Run("VERSION file exists", func(t *testing.T) {
		_, err := os.Stat(versionFile)
		require.NoError(t, err, "VERSION file should exist at repository root")
	})

	t.Run("VERSION file is readable", func(t *testing.T) {
		content, err := os.ReadFile(versionFile)
		require.NoError(t, err, "VERSION file should be readable")
		assert.NotEmpty(t, content, "VERSION file should not be empty")
	})

	t.Run("VERSION contains valid semantic version", func(t *testing.T) {
		content, err := os.ReadFile(versionFile)
		require.NoError(t, err)

		version := strings.TrimSpace(string(content))
		assert.NotEmpty(t, version, "VERSION should not be empty after trimming")

		// Parse as semantic version
		_, err = semver.NewVersion(version)
		assert.NoError(t, err, "VERSION should be valid semantic version format (e.g., 0.25.10)")
	})

	t.Run("VERSION format is valid", func(t *testing.T) {
		content, err := os.ReadFile(versionFile)
		require.NoError(t, err)

		version := strings.TrimSpace(string(content))

		// Should not have leading 'v'
		assert.False(t, strings.HasPrefix(version, "v"),
			"VERSION file should not have leading 'v' (will be added by build process)")

		// Should have at least major.minor format
		parts := strings.Split(version, ".")
		assert.GreaterOrEqual(t, len(parts), 2, "VERSION should have at least major.minor")

		// Each part should be numeric
		for i, part := range parts {
			assert.Regexp(t, "^[0-9]+$", part, "VERSION part %d should be numeric", i)
		}
	})

	t.Run("VERSION has no trailing newlines or whitespace", func(t *testing.T) {
		content, err := os.ReadFile(versionFile)
		require.NoError(t, err)

		// Check for single newline at end (Unix convention)
		assert.True(t, len(content) > 0, "VERSION file should not be empty")

		// Trim and check that trimmed version matches when one newline is added
		trimmed := strings.TrimSpace(string(content))
		expected := trimmed + "\n"
		expectedCRLF := trimmed + "\r\n"

		// Allow either no newline, single newline (LF), or single newline (CRLF)
		contentStr := string(content)
		assert.True(t, contentStr == trimmed || contentStr == expected || contentStr == expectedCRLF,
			"VERSION file should have either no newline or single trailing newline (LF or CRLF)")
	})

	t.Run("VERSION is reasonable", func(t *testing.T) {
		content, err := os.ReadFile(versionFile)
		require.NoError(t, err)

		version := strings.TrimSpace(string(content))
		semVer, err := semver.NewVersion(version)
		require.NoError(t, err)

		// Sanity checks
		assert.GreaterOrEqual(t, semVer.Major(), uint64(0), "Major version should be >= 0")
		assert.LessOrEqual(t, semVer.Major(), uint64(100), "Major version should be reasonable (< 100)")
		assert.LessOrEqual(t, semVer.Minor(), uint64(1000), "Minor version should be reasonable (< 1000)")
		assert.LessOrEqual(t, semVer.Patch(), uint64(1000), "Patch version should be reasonable (< 1000)")
	})
}

func TestVersionFileIntegrationWithMakefile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("Makefile reads VERSION file", func(t *testing.T) {
		makefilePath := filepath.Join(repoRoot, "Makefile")
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Skip("Makefile not found")
		}

		makefileContent := string(content)

		// Verify Makefile references VERSION file
		assert.Contains(t, makefileContent, "VERSION",
			"Makefile should reference VERSION file")
		assert.Contains(t, makefileContent, "cat VERSION",
			"Makefile should read VERSION file using cat command")
	})

	t.Run("Makefile build command includes version injection", func(t *testing.T) {
		makefilePath := filepath.Join(repoRoot, "Makefile")
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Skip("Makefile not found")
		}

		makefileContent := string(content)

		// Verify ldflags injection
		assert.Contains(t, makefileContent, "-X main.BUILD_VERSION",
			"Makefile should inject BUILD_VERSION via ldflags")

		// Verify version is prefixed with 'v'
		assert.Contains(t, makefileContent, "v$$VERSION",
			"Makefile should prefix VERSION with 'v'")
	})
}

func TestVersionFileIntegrationWithNix(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("flake.nix reads VERSION file", func(t *testing.T) {
		flakePath := filepath.Join(repoRoot, "flake.nix")
		content, err := os.ReadFile(flakePath)
		if err != nil {
			t.Skip("flake.nix not found")
		}

		flakeContent := string(content)

		// Verify flake.nix references VERSION file
		assert.Contains(t, flakeContent, "builtins.readFile ./VERSION",
			"flake.nix should read VERSION file")

		// Verify version is conditionally prefixed with 'v' (avoids double-prefixing)
		assert.Contains(t, flakeContent, `else "v${rawVersion}"`,
			"flake.nix should conditionally prefix VERSION with 'v'")

		// Also verify that with the current VERSION (0.26.0), the conditional logic holds
		// Since we can't easily execute Nix code here without nix-instantiate,
		// we rely on the structural check above and verifying the source data (VERSION file)
		// doesn't have the prefix, which triggers the 'else' branch we just verified.
	})

	t.Run("flake.nix version string is properly formatted", func(t *testing.T) {
		flakePath := filepath.Join(repoRoot, "flake.nix")
		content, err := os.ReadFile(flakePath)
		if err != nil {
			t.Skip("flake.nix not found")
		}

		flakeContent := string(content)

		// Should define a version variable
		assert.Contains(t, flakeContent, "version =",
			"flake.nix should define version variable")
	})
}

func TestVersionConsistencyAcrossBuildSystems(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	// Read VERSION file
	versionContent, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		t.Skip("VERSION file not found")
	}
	expectedVersion := strings.TrimSpace(string(versionContent))

	t.Run("All build systems use VERSION file as source of truth", func(t *testing.T) {
		// Check Makefile
		makefileContent, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
		if err == nil {
			assert.Contains(t, string(makefileContent), "cat VERSION",
				"Makefile should read from VERSION file")
		}

		// Check flake.nix
		flakeContent, err := os.ReadFile(filepath.Join(repoRoot, "flake.nix"))
		if err == nil {
			assert.Contains(t, string(flakeContent), "builtins.readFile ./VERSION",
				"flake.nix should read from VERSION file")

			// Also check for the conditional logic since we're here
			assert.Contains(t, string(flakeContent), `else "v${rawVersion}"`,
				"flake.nix should have logic to ensure 'v' prefix")
		}
	})

	t.Run("VERSION file version is valid for all build systems", func(t *testing.T) {
		// Should be valid semantic version
		_, err := semver.NewVersion(expectedVersion)
		assert.NoError(t, err, "VERSION should be valid for semantic versioning")

		// Should not have 'v' prefix (added by build systems)
		assert.False(t, strings.HasPrefix(expectedVersion, "v"),
			"VERSION file should not have 'v' prefix")
	})
}

func TestVersionFilePermissions(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	versionFile := filepath.Join(repoRoot, "VERSION")

	t.Run("VERSION file has appropriate permissions", func(t *testing.T) {
		info, err := os.Stat(versionFile)
		require.NoError(t, err)

		// Should be a regular file
		assert.False(t, info.IsDir(), "VERSION should be a file, not directory")

		// Should be readable
		mode := info.Mode()
		assert.True(t, mode&0400 != 0, "VERSION file should be readable by owner")
	})
}

// Helper function to find repository root
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up until we find .git directory or go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}
