package appupdate

import (
	"os"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/core"
)

// GetLastUsedVersion reads the last used version from the version marker file.
// Returns empty string if no version marker exists (fresh install or v0.x user).
func GetLastUsedVersion() string {
	data, err := os.ReadFile(core.VersionMarkerFile())
	if err != nil {
		return ""
	}
	return string(data)
}

// UpdateVersionMarker writes the current version to the version marker file.
func UpdateVersionMarker(version string) error {
	return os.WriteFile(core.VersionMarkerFile(), []byte(version), 0644)
}

// IsUpgradeFromV0 checks if the user is upgrading from v0.x to v1.0+.
// This is detected when:
// 1. No version marker file exists (never tracked version before)
// 2. History file exists in the old v0.x location (~/.local/share/gsh/history.db)
func IsUpgradeFromV0() bool {
	// If version marker already exists, this is not an upgrade scenario
	lastVersion := GetLastUsedVersion()
	if lastVersion != "" {
		return false
	}

	// Check if this looks like an existing v0.x user
	// v0.x users had history in ~/.local/share/gsh/history.db
	homeDir := core.HomeDir()
	oldHistoryPath := filepath.Join(homeDir, ".local", "share", "gsh", "history.db")
	if _, err := os.Stat(oldHistoryPath); err == nil {
		return true
	}
	return false
}

// GetMigrationMessage returns the migration message shown to users upgrading from v0.x
func GetMigrationMessage() string {
	return `
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Welcome to gsh v1.0!                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  What's new:                                                                │
│  • Revamped REPL with syntax highlighting and history based completion      │
│  • .gsh agentic scripting language for advanced automation                  │
│  • Deep customization via ~/.gsh/repl.gsh (bash config in .gshrc still works)    │
│                                                                             │
│  Your existing .gshrc and shell history are preserved.                      │
│                                                                             │
│  Learn more: https://github.com/atinylittleshell/gsh                        │
└─────────────────────────────────────────────────────────────────────────────┘
`
}
