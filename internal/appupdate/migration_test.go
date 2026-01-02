package appupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMigrationMessage(t *testing.T) {
	message := GetMigrationMessage()
	assert.Contains(t, message, "Welcome to gsh v1.0")
	assert.Contains(t, message, "repl.gsh")
	assert.Contains(t, message, "github.com/atinylittleshell/gsh")
}

func TestVersionMarker(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override HOME for core.DataDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	core.ResetPaths() // Reset cached paths so new HOME is picked up
	defer func() {
		os.Setenv("HOME", originalHome)
		core.ResetPaths()
	}()

	// Test GetLastUsedVersion when no marker exists
	version := GetLastUsedVersion()
	assert.Equal(t, "", version)

	// Test UpdateVersionMarker
	err := UpdateVersionMarker("1.0.0")
	require.NoError(t, err)

	// Test GetLastUsedVersion after update
	version = GetLastUsedVersion()
	assert.Equal(t, "1.0.0", version)

	// Test updating to a new version
	err = UpdateVersionMarker("1.1.0")
	require.NoError(t, err)

	version = GetLastUsedVersion()
	assert.Equal(t, "1.1.0", version)
}

func TestIsUpgradeFromV0_FreshInstall(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override HOME for core.DataDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	core.ResetPaths() // Reset cached paths so new HOME is picked up
	defer func() {
		os.Setenv("HOME", originalHome)
		core.ResetPaths()
	}()

	// Fresh install: no version marker, no history, no analytics
	// Should NOT be detected as upgrade from v0
	assert.False(t, IsUpgradeFromV0())
}

func TestIsUpgradeFromV0_ExistingUser(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override HOME for core.DataDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	core.ResetPaths() // Reset cached paths so new HOME is picked up
	defer func() {
		os.Setenv("HOME", originalHome)
		core.ResetPaths()
	}()

	// Create the old v0.x data directory (simulating existing user)
	dataDir := filepath.Join(tempDir, ".local", "share", "gsh")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	// Simulate existing v0.x user: create history file in old location
	historyFile := filepath.Join(dataDir, "history.db")
	err = os.WriteFile(historyFile, []byte("test"), 0644)
	require.NoError(t, err)

	// No version marker, but has history in old location = v0.x user upgrading
	assert.True(t, IsUpgradeFromV0())
}

func TestIsUpgradeFromV0_AlreadyOnV1(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override HOME for core.DataDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	core.ResetPaths() // Reset cached paths so new HOME is picked up
	defer func() {
		os.Setenv("HOME", originalHome)
		core.ResetPaths()
	}()

	// Create the old v0.x data directory (simulating existing user)
	dataDir := filepath.Join(tempDir, ".local", "share", "gsh")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	// Simulate existing user with history in old location
	historyFile := filepath.Join(dataDir, "history.db")
	err = os.WriteFile(historyFile, []byte("test"), 0644)
	require.NoError(t, err)

	// User already has version marker (already on v1.x)
	err = UpdateVersionMarker("1.0.0")
	require.NoError(t, err)

	// Should NOT be detected as upgrade since version marker exists
	assert.False(t, IsUpgradeFromV0())
}
