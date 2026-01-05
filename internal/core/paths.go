package core

import (
	"os"
	"path/filepath"
)

type Paths struct {
	HomeDir           string
	DataDir           string
	LogFile           string
	HistoryFile       string
	LatestVersionFile string
	VersionMarkerFile string
}

var defaultPaths *Paths

func ensureDefaultPaths() {
	if defaultPaths == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		defaultPaths = &Paths{
			HomeDir:           homeDir,
			DataDir:           filepath.Join(homeDir, ".gsh"),
			LogFile:           filepath.Join(homeDir, ".gsh", "gsh.log"),
			HistoryFile:       filepath.Join(homeDir, ".gsh", "history.db"),
			LatestVersionFile: filepath.Join(homeDir, ".gsh", "latest_version.txt"),
			VersionMarkerFile: filepath.Join(homeDir, ".gsh", "version_marker"),
		}

		err = os.MkdirAll(defaultPaths.DataDir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func HomeDir() string {
	ensureDefaultPaths()
	return defaultPaths.HomeDir
}

func DataDir() string {
	ensureDefaultPaths()
	return defaultPaths.DataDir
}

func LogFile() string {
	ensureDefaultPaths()
	return defaultPaths.LogFile
}

func HistoryFile() string {
	ensureDefaultPaths()
	return defaultPaths.HistoryFile
}

func LatestVersionFile() string {
	ensureDefaultPaths()
	return defaultPaths.LatestVersionFile
}

func VersionMarkerFile() string {
	ensureDefaultPaths()
	return defaultPaths.VersionMarkerFile
}

// ResetPaths clears the cached paths, forcing them to be reinitialized.
// This is primarily used for testing purposes.
func ResetPaths() {
	defaultPaths = nil
}
