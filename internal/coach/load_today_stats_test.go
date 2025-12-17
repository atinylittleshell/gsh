package coach

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"mvdan.cc/sh/v3/interp"
)

// TestLoadTodayStatsErrorHandling tests the error handling improvements in loadTodayStats
func TestLoadTodayStatsErrorHandling(t *testing.T) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Ensure database is closed at the end of the test
	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			t.Logf("Warning: Failed to get underlying SQL connection: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				t.Logf("Warning: Failed to close database connection: %v", err)
			}
		}
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create history manager
	historyManager := &history.HistoryManager{}

	// Create runner
	runner := &interp.Runner{}

	// Create coach manager
	manager, err := NewCoachManager(db, historyManager, runner, logger)
	if err != nil {
		t.Fatalf("Failed to create coach manager: %v", err)
	}

	// Test 1: Verify that loadTodayStats creates a new record when none exists
	today := time.Now().Format("2006-01-02")
	stats := manager.GetTodayStats()
	if stats == nil {
		t.Fatal("Expected todayStats to be initialized")
	}

	if stats.Date != today {
		t.Errorf("Expected date %s, got %s", today, stats.Date)
	}

	if stats.ProfileID != manager.profile.ID {
		t.Errorf("Expected ProfileID %d, got %d", manager.profile.ID, stats.ProfileID)
	}

	// Test 2: Verify backwards compatibility logic
	// Create a record with AvgCommandTimeMs > 0 but CommandCount = 0
	backwardsCompatStats := &CoachDailyStats{
		ProfileID:        manager.profile.ID,
		Date:             today,
		CommandsExecuted: 5,
		CommandCount:     0,
		AvgCommandTimeMs: 100,
	}

	// Delete the current record and create our test record
	db.Delete(&CoachDailyStats{}, "profile_id = ? AND date = ?", manager.profile.ID, today)
	if err := db.Create(backwardsCompatStats).Error; err != nil {
		t.Fatalf("Failed to create backwards compatibility test record: %v", err)
	}

	// Create a new manager instance to test loading the backwards compatible record
	manager2, err := NewCoachManager(db, historyManager, runner, logger)
	if err != nil {
		t.Fatalf("Failed to create second coach manager: %v", err)
	}

	loadedStats := manager2.GetTodayStats()
	if loadedStats == nil {
		t.Fatal("Expected todayStats to be loaded")
	}

	// Verify backwards compatibility: CommandCount should be set to CommandsExecuted
	if loadedStats.CommandCount != loadedStats.CommandsExecuted {
		t.Errorf("Expected CommandCount (%d) to equal CommandsExecuted (%d) for backwards compatibility",
			loadedStats.CommandCount, loadedStats.CommandsExecuted)
	}

	t.Logf("✓ Backwards compatibility verified: CommandCount set to %d", loadedStats.CommandCount)
}

// TestLoadTodayStatsConcurrentAccess tests concurrent access to loadTodayStats
func TestLoadTodayStatsConcurrentAccess(t *testing.T) {
	// Create temporary file database for concurrent access
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Close database connection when test completes
	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			t.Logf("Warning: Failed to get underlying SQL connection: %v", err)
		} else {
			if err := sqlDB.Close(); err != nil {
				t.Logf("Warning: Failed to close database connection: %v", err)
			}
		}
	}()

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create history manager
	historyManager := &history.HistoryManager{}

	// Create runner
	runner := &interp.Runner{}

	// Create coach manager
	manager, err := NewCoachManager(db, historyManager, runner, logger)
	if err != nil {
		t.Fatalf("Failed to create coach manager: %v", err)
	}

	// Simulate concurrent calls to loadTodayStats
	done := make(chan bool, 3)
	errors := make(chan error, 3)
	results := make(chan string, 3)

	// Launch 3 concurrent calls
	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			// Force reload by calling the private method
			manager.loadTodayStats()
			results <- fmt.Sprintf("Goroutine %d completed loadTodayStats", id)
		}(i)
	}

	// Wait for all goroutines to complete and collect results
	completed := 0
	for completed < 3 {
		select {
		case result := <-results:
			t.Log(result)
		case err := <-errors:
			t.Errorf("Concurrent access error: %v", err)
		case <-done:
			completed++
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify that only one record was created (no duplicates)
	var count int64
	db.Model(&CoachDailyStats{}).Where("profile_id = ? AND date = ?", manager.profile.ID, time.Now().Format("2006-01-02")).Count(&count)

	if count != 1 {
		t.Errorf("Expected exactly 1 stats record, found %d (possible race condition)", count)
	} else {
		t.Log("✓ Concurrent access handled correctly: no duplicate records created")
	}
}
