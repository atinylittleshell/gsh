package analytics

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsCommand(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err)

	handler := NewAnalyticsCommandHandler(analyticsManager)
	nextHandler := func(ctx context.Context, args []string) error {
		return nil
	}
	wrappedHandler := handler(nextHandler)

	// Test non-analytics command passes through
	err = wrappedHandler(context.Background(), []string{"echo", "hello"})
	assert.NoError(t, err)

	tests := []struct {
		name          string
		args          []string
		expectedError bool
		setupFn       func()
		verify        func(t *testing.T, am *AnalyticsManager)
	}{
		{
			name:          "Show help",
			args:          []string{"gsh_analytics", "--help"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				// Help message is printed to stdout, we can't easily verify it
				// but at least we can verify no state change
				entries, err := am.GetRecentEntries(10)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
		},
		{
			name:          "List with default limit",
			args:          []string{"gsh_analytics"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
				_ = analyticsManager.NewEntry("test3", "test3", "test3")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				entries, err := am.GetRecentEntries(20)
				assert.NoError(t, err)
				assert.Len(t, entries, 3)
			},
		},
		{
			name:          "List with custom limit",
			args:          []string{"gsh_analytics", "2"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
				_ = analyticsManager.NewEntry("test3", "test3", "test3")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				entries, err := am.GetRecentEntries(2)
				assert.NoError(t, err)
				assert.Len(t, entries, 2)
			},
		},
		{
			name:          "Clear analytics",
			args:          []string{"gsh_analytics", "-c"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				entries, err := am.GetRecentEntries(10)
				assert.NoError(t, err)
				assert.Len(t, entries, 0)
			},
		},
		{
			name:          "Show count with short flag",
			args:          []string{"gsh_analytics", "-n"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				count, err := am.GetTotalCount()
				assert.NoError(t, err)
				assert.Equal(t, int64(2), count)
			},
		},
		{
			name:          "Show count with long flag",
			args:          []string{"gsh_analytics", "--count"},
			expectedError: false,
			setupFn: func() {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
				_ = analyticsManager.NewEntry("test3", "test3", "test3")
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				count, err := am.GetTotalCount()
				assert.NoError(t, err)
				assert.Equal(t, int64(3), count)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFn != nil {
				tc.setupFn()
			}
			err := wrappedHandler(context.Background(), tc.args)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tc.verify != nil {
				tc.verify(t, analyticsManager)
			}
		})
	}
}

func TestAnalyticsCommandDelete(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err)

	handler := NewAnalyticsCommandHandler(analyticsManager)
	nextHandler := func(ctx context.Context, args []string) error {
		return nil
	}
	wrappedHandler := handler(nextHandler)

	tests := []struct {
		name          string
		args          []string
		expectedError bool
		setupFn       func() uint
		verify        func(t *testing.T, am *AnalyticsManager)
	}{
		{
			name:          "Delete without entry number",
			args:          []string{"gsh_analytics", "--delete"},
			expectedError: true,
			setupFn: func() uint {
				_ = analyticsManager.ResetAnalytics()
				return 0
			},
		},
		{
			name:          "Delete with invalid entry number",
			args:          []string{"gsh_analytics", "--delete", "invalid"},
			expectedError: true,
			setupFn: func() uint {
				_ = analyticsManager.ResetAnalytics()
				return 0
			},
		},
		{
			name:          "Delete non-existent entry",
			args:          []string{"gsh_analytics", "--delete", "999"},
			expectedError: true,
			setupFn: func() uint {
				_ = analyticsManager.ResetAnalytics()
				return 0
			},
		},
		{
			name:          "Delete existing entry",
			args:          []string{"gsh_analytics", "--delete", "%d"},
			expectedError: false,
			setupFn: func() uint {
				_ = analyticsManager.ResetAnalytics()
				_ = analyticsManager.NewEntry("test1", "test1", "test1")
				_ = analyticsManager.NewEntry("test2", "test2", "test2")
				entries, _ := analyticsManager.GetRecentEntries(10)
				return entries[0].ID
			},
			verify: func(t *testing.T, am *AnalyticsManager) {
				entries, err := am.GetRecentEntries(10)
				assert.NoError(t, err)
				assert.Len(t, entries, 1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.setupFn()
			args := tc.args
			if id > 0 {
				args = []string{args[0], args[1], fmt.Sprintf(args[2], id)}
			}
			err := wrappedHandler(context.Background(), args)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tc.verify != nil {
				tc.verify(t, analyticsManager)
			}
		})
	}
}

func TestAnalyticsCommandEdgeCases(t *testing.T) {
	analyticsManager, err := NewAnalyticsManager(":memory:")
	assert.NoError(t, err)

	handler := NewAnalyticsCommandHandler(analyticsManager)
	nextHandler := func(ctx context.Context, args []string) error {
		return nil
	}
	wrappedHandler := handler(nextHandler)

	// Test empty args
	err = wrappedHandler(context.Background(), []string{})
	assert.NoError(t, err)

	// Test invalid limit number
	err = wrappedHandler(context.Background(), []string{"gsh_analytics", "invalid"})
	assert.NoError(t, err) // Should default to 20

	// Test negative limit number
	err = wrappedHandler(context.Background(), []string{"gsh_analytics", "-5"})
	assert.NoError(t, err) // Should default to 20

	// Test with zero limit
	err = wrappedHandler(context.Background(), []string{"gsh_analytics", "0"})
	assert.NoError(t, err) // Should default to 20

	// Test count after clearing analytics
	_ = analyticsManager.ResetAnalytics()
	_ = analyticsManager.NewEntry("test1", "test1", "test1")
	_ = analyticsManager.NewEntry("test2", "test2", "test2")
	err = wrappedHandler(context.Background(), []string{"gsh_analytics", "-c"})
	assert.NoError(t, err)
	err = wrappedHandler(context.Background(), []string{"gsh_analytics", "--count"})
	assert.NoError(t, err) // Should show count as 0
}

