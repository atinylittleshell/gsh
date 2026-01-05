package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/analytics/telemetry"
	"github.com/atinylittleshell/gsh/internal/core"
)

// BenchmarkStartupInitialization measures the cost of preparing core gsh
// components for an interactive session without running the Bubble Tea loop.
func BenchmarkStartupInitialization(b *testing.B) {
	b.ReportAllocs()

	originalArgs := os.Args
	b.Cleanup(func() {
		os.Args = originalArgs
	})

	baseTempDir := b.TempDir()
	homeDir := filepath.Join(baseTempDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		b.Fatalf("failed to create temp home: %v", err)
	}

	b.Setenv("GSH_NO_TELEMETRY", "1")
	b.Setenv("HOME", homeDir)

	for i := 0; i < b.N; i++ {
		core.ResetPaths()
		os.Args = []string{"gsh"}

		telemetryClient, _ := telemetry.NewClient(telemetry.Config{
			Version: BUILD_VERSION,
		})
		if telemetryClient != nil {
			_ = telemetryClient.Close()
		}

		historyManager, err := initializeHistoryManager()
		if err != nil {
			b.Fatalf("failed to initialize history manager: %v", err)
		}

		completionManager := initializeCompletionManager()

		runner, err := initializeRunner(historyManager, completionManager, false)
		if err != nil {
			b.Fatalf("failed to initialize runner: %v", err)
		}

		logger, _, err := initializeLogger(runner)
		if err != nil {
			b.Fatalf("failed to initialize logger: %v", err)
		}
		_ = logger.Sync()
	}
}
