package coach

import (
	"sync"
	"testing"
	"time"
)

func TestGetHighPriorityConcurrency(t *testing.T) {
	cache := NewTipCache(10, time.Hour)

	// Add some test tips
	tip1 := &GeneratedTip{
		ID:          "tip1",
		Title:       "Tip 1",
		Priority:    1,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	tip2 := &GeneratedTip{
		ID:          "tip2",
		Title:       "Tip 2",
		Priority:    3,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	tip3 := &GeneratedTip{
		ID:          "tip3",
		Title:       "Tip 3",
		Priority:    2,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	cache.Add(tip1)
	cache.Add(tip2)
	cache.Add(tip3)

	// Test concurrent access to GetHighPriority
	const numGoroutines = 10
	const numIterations = 100

	var wg sync.WaitGroup
	results := make(chan *GeneratedTip, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				tip := cache.GetHighPriority()
				results <- tip
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify all results are valid
	count := 0
	for tip := range results {
		count++
		// All returned tips should be one of our test tips
		if tip != nil && tip.ID != "tip1" && tip.ID != "tip2" && tip.ID != "tip3" {
			t.Errorf("Unexpected tip returned: %s", tip.ID)
		}
	}

	expectedCount := numGoroutines * numIterations
	if count != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, count)
	}
}

func TestGetHighPriorityReturnsHighestPriority(t *testing.T) {
	cache := NewTipCache(10, time.Hour)

	// Add tips with different priorities
	tip1 := &GeneratedTip{
		ID:          "tip1",
		Title:       "Low Priority",
		Priority:    1,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	tip2 := &GeneratedTip{
		ID:          "tip2",
		Title:       "High Priority",
		Priority:    5,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	tip3 := &GeneratedTip{
		ID:          "tip3",
		Title:       "Medium Priority",
		Priority:    3,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	cache.Add(tip1)
	cache.Add(tip2)
	cache.Add(tip3)

	// Should return the highest priority tip
	tip := cache.GetHighPriority()
	if tip == nil || tip.ID != "tip2" {
		t.Errorf("Expected tip2 (priority 5), got %s (priority %d)",
			func() string {
				if tip == nil {
					return "nil"
				}
				return tip.ID
			}(),
			func() int {
				if tip == nil {
					return 0
				}
				return tip.Priority
			}())
	}
}

func TestGetHighPriorityRespectsShownToday(t *testing.T) {
	cache := NewTipCache(10, time.Hour)

	// Add tips
	tip1 := &GeneratedTip{
		ID:          "tip1",
		Title:       "High Priority",
		Priority:    5,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	tip2 := &GeneratedTip{
		ID:          "tip2",
		Title:       "Lower Priority",
		Priority:    3,
		Type:        TipTypeProductivity,
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	cache.Add(tip1)
	cache.Add(tip2)

	// Mark high priority tip as shown
	cache.MarkShown("tip1")

	// Should return the lower priority tip since high priority is already shown
	tip := cache.GetHighPriority()
	if tip == nil || tip.ID != "tip2" {
		t.Errorf("Expected tip2, got %s", func() string {
			if tip == nil {
				return "nil"
			}
			return tip.ID
		}())
	}
}

func TestGetHighPriorityWithNilCache(t *testing.T) {
	cache := NewTipCache(10, time.Hour)

	// Empty cache should return nil
	tip := cache.GetHighPriority()
	if tip != nil {
		t.Errorf("Expected nil for empty cache, got %s", tip.ID)
	}
}
