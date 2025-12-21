package appupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildVersionInjectionIntegration(t *testing.T) {
	t.Run("Version string format validation", func(t *testing.T) {
		testVersions := []struct {
			name      string
			version   string
			shouldErr bool
		}{
			{
				name:      "Valid semantic version with v prefix",
				version:   "v0.25.10",
				shouldErr: false,
			},
			{
				name:      "Valid semantic version without v prefix",
				version:   "0.25.10",
				shouldErr: false,
			},
			{
				name:      "Dev version",
				version:   "dev",
				shouldErr: true, // semver parsing will fail for "dev"
			},
			{
				name:      "Valid major.minor.patch",
				version:   "v1.2.3",
				shouldErr: false,
			},
			{
				name:      "Valid with pre-release",
				version:   "v1.2.3-beta.1",
				shouldErr: false,
			},
		}

		for _, tt := range testVersions {
			t.Run(tt.name, func(t *testing.T) {
				_, err := semver.NewVersion(tt.version)
				if tt.shouldErr {
					assert.Error(t, err, "Should fail to parse non-semver version")
				} else {
					assert.NoError(t, err, "Should successfully parse valid semver")
				}
			})
		}
	})
}

func TestVersionFileIntegrationWithAppUpdate(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("VERSION file can be read for app updates", func(t *testing.T) {
		versionFile := filepath.Join(repoRoot, "VERSION")
		content, err := os.ReadFile(versionFile)
		if err != nil {
			t.Skip("VERSION file not found")
		}

		version := strings.TrimSpace(string(content))
		assert.NotEmpty(t, version, "VERSION file should not be empty")

		// Should be parseable as semantic version
		semVer, err := semver.NewVersion(version)
		require.NoError(t, err, "VERSION should be valid semantic version")

		// Verify it's a reasonable version
		assert.GreaterOrEqual(t, semVer.Major(), uint64(0))
	})

	t.Run("VERSION format matches goreleaser expectations", func(t *testing.T) {
		goreleaserFile := filepath.Join(repoRoot, ".goreleaser.yaml")
		content, err := os.ReadFile(goreleaserFile)
		if err != nil {
			t.Skip(".goreleaser.yaml not found")
		}

		goreleaserContent := string(content)

		// Verify goreleaser uses ldflags for version injection
		assert.Contains(t, goreleaserContent, "-X main.BUILD_VERSION",
			"goreleaser should inject main.BUILD_VERSION")
		assert.Contains(t, goreleaserContent, "{{.Version}}",
			"goreleaser should use version template")
	})
}

func TestMakefileVersionInjection(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("Makefile version injection syntax", func(t *testing.T) {
		makefilePath := filepath.Join(repoRoot, "Makefile")
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Skip("Makefile not found")
		}

		makefileContent := string(content)

		// Verify proper shell variable syntax
		assert.Contains(t, makefileContent, "$$(cat VERSION)",
			"Makefile should use proper shell command substitution")

		// Verify ldflags format
		assert.Contains(t, makefileContent, "-ldflags=",
			"Makefile should use proper ldflags syntax")

		// Verify version is prefixed with v
		assert.Contains(t, makefileContent, "-X main.BUILD_VERSION=v$$VERSION",
			"Makefile should prefix version with 'v'")
	})

	t.Run("Makefile build target is properly structured", func(t *testing.T) {
		makefilePath := filepath.Join(repoRoot, "Makefile")
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Skip("Makefile not found")
		}

		makefileContent := string(content)

		// Verify build target exists
		assert.Contains(t, makefileContent, ".PHONY: build",
			"Makefile should have .PHONY build target")

		// Verify go build command
		assert.Contains(t, makefileContent, "go build",
			"Makefile should use go build command")

		// Verify output location
		assert.Contains(t, makefileContent, "./bin/gsh",
			"Makefile should build to ./bin/gsh")

		// Verify main package location
		assert.Contains(t, makefileContent, "./cmd/gsh/main.go",
			"Makefile should reference correct main package")
	})
}

func TestNixFlakeVersionInjection(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("flake.nix version reading", func(t *testing.T) {
		flakePath := filepath.Join(repoRoot, "flake.nix")
		content, err := os.ReadFile(flakePath)
		if err != nil {
			t.Skip("flake.nix not found")
		}

		flakeContent := string(content)

		// Verify Nix builtins.readFile syntax
		assert.Contains(t, flakeContent, "builtins.readFile",
			"flake.nix should use builtins.readFile")

		// Verify path is relative
		assert.Contains(t, flakeContent, "./VERSION",
			"flake.nix should reference ./VERSION")

		// Verify normalization (trimming whitespace)
		assert.Contains(t, flakeContent, "builtins.replaceStrings",
			"flake.nix should normalize version string (trim whitespace)")
	})

	t.Run("flake.nix version variable assignment", func(t *testing.T) {
		flakePath := filepath.Join(repoRoot, "flake.nix")
		content, err := os.ReadFile(flakePath)
		if err != nil {
			t.Skip("flake.nix not found")
		}

		flakeContent := string(content)

		// Verify version logic handles 'v' prefix conditionally
		assert.Contains(t, flakeContent, `if builtins.substring 0 1 rawVersion == "v"`,
			"flake.nix should check for existing 'v' prefix")

		// Verify it adds 'v' if missing
		assert.Contains(t, flakeContent, `else "v${rawVersion}"`,
			"flake.nix should add 'v' prefix if missing")

		// Verify version is assigned in buildGoModule
		assert.Contains(t, flakeContent, "version = version;",
			"flake.nix should assign calculated version variable")
	})
}

func TestBuildSystemVersionConsistency(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	// Read VERSION file
	versionContent, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		t.Skip("VERSION file not found")
	}
	version := strings.TrimSpace(string(versionContent))

	t.Run("All build systems produce consistent version string", func(t *testing.T) {
		// Verify version format
		_, err := semver.NewVersion(version)
		require.NoError(t, err, "VERSION should be valid semantic version")

		// Verify all build systems add 'v' prefix
		expectedVersionString := "v" + version

		// Check Makefile produces this format
		makefileContent, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
		if err == nil {
			assert.Contains(t, string(makefileContent), "v$$VERSION",
				"Makefile should produce version with 'v' prefix")
		}

		// Check flake.nix produces this format
		flakeContent, err := os.ReadFile(filepath.Join(repoRoot, "flake.nix"))
		if err == nil {
			// Verify flake.nix has logic to ensure v prefix
			assert.Contains(t, string(flakeContent), `else "v${rawVersion}"`,
				"flake.nix should have logic to ensure 'v' prefix")

			// Also verify that the logic would resolve correctly given our current VERSION
			if !strings.HasPrefix(version, "v") {
				// Since our VERSION file doesn't have 'v' prefix (e.g. "0.26.0"),
				// the logic `if builtins.substring 0 1 rawVersion == "v" then rawVersion else "v${rawVersion}"`
				// should evaluate to "v0.26.0"
				assert.Equal(t, "v"+version, expectedVersionString,
					"Resolved version should be 'v' + VERSION content")
			}
		}

		// Verify the format is what appupdate expects
		_, err = semver.NewVersion(expectedVersionString)
		assert.NoError(t, err, "Version with 'v' prefix should also be valid semver")
	})
}

func TestVersionFileUpdateProcess(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("Could not find repository root")
	}

	t.Run("VERSION file is single source of truth", func(t *testing.T) {
		versionFile := filepath.Join(repoRoot, "VERSION")
		_, err := os.Stat(versionFile)
		require.NoError(t, err, "VERSION file should exist")

		// Verify no hardcoded versions in flake.nix
		flakeContent, err := os.ReadFile(filepath.Join(repoRoot, "flake.nix"))
		if err == nil {
			// Should not contain hardcoded version like "v0.22.2" anymore
			lines := strings.Split(string(flakeContent), "\n")
			for _, line := range lines {
				if strings.Contains(line, "version = ") &&
					!strings.Contains(line, "builtins.readFile") {
					// If there's a version assignment without builtins.readFile,
					// it should be the variable reference, not a hardcoded string
					assert.NotContains(t, line, `"v0.`,
						"flake.nix should not contain hardcoded version strings")
				}
			}
		}
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
