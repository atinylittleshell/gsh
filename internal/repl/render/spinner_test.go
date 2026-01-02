package render

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerFrames(t *testing.T) {
	// Verify spinner frames are defined
	assert.Len(t, SpinnerFrames, 10)
	assert.Equal(t, "⠋", SpinnerFrames[0])
}

func TestGetCurrentFrame(t *testing.T) {
	// Test frame retrieval wraps around
	assert.Equal(t, SpinnerFrames[0], GetCurrentFrame(0))
	assert.Equal(t, SpinnerFrames[5], GetCurrentFrame(5))
	assert.Equal(t, SpinnerFrames[0], GetCurrentFrame(10)) // Wraps around
	assert.Equal(t, SpinnerFrames[3], GetCurrentFrame(13)) // Wraps around
}

func TestNewSpinner(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf)

	assert.NotNil(t, spinner)
	assert.Equal(t, &buf, spinner.writer)
	assert.Equal(t, SpinnerFrames, spinner.frames)
	assert.Equal(t, 80*time.Millisecond, spinner.interval)
}

func TestSpinnerSetMessage(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf)

	spinner.SetMessage("Loading...")
	assert.Equal(t, "Loading...", spinner.message)

	spinner.SetMessage("Processing...")
	assert.Equal(t, "Processing...", spinner.message)
}

func TestSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf)
	spinner.SetMessage("Test")

	ctx := context.Background()
	cancel := spinner.Start(ctx)

	// Give the spinner time to render at least one frame
	time.Sleep(100 * time.Millisecond)

	// Cancel the spinner
	cancel()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify something was written (spinner frames + message)
	output := buf.String()
	assert.NotEmpty(t, output)
}

func TestSpinnerDoubleStart(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewSpinner(&buf)

	ctx := context.Background()
	cancel1 := spinner.Start(ctx)

	// Starting again should return a cancel function without starting a new goroutine
	cancel2 := spinner.Start(ctx)

	cancel1()
	cancel2()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)
}

// =============================================================================
// SpinnerManager Tests
// =============================================================================

func TestNewSpinnerManager(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	assert.NotNil(t, manager)
	assert.Equal(t, &buf, manager.writer)
	assert.Equal(t, SpinnerFrames, manager.frames)
	assert.Equal(t, 80*time.Millisecond, manager.interval)
	assert.Empty(t, manager.liveSpinners)
}

func TestSpinnerManager_SingleSpinner(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner := manager.NewSpinner()
	spinner.SetMessage("Loading...")

	ctx := context.Background()
	stop := spinner.Start(ctx)

	// Verify spinner is live and active
	assert.Equal(t, 1, manager.LiveCount())
	assert.Equal(t, spinner.id, manager.ActiveID())

	// Give time to render
	time.Sleep(100 * time.Millisecond)

	stop()

	// Verify spinner is removed
	assert.Equal(t, 0, manager.LiveCount())
	assert.Equal(t, uint64(0), manager.ActiveID())

	// Verify output contains the message
	output := buf.String()
	assert.Contains(t, output, "Loading...")
}

func TestSpinnerManager_MultipleSpinners_OnlyMostRecentRenders(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Create and start first spinner
	spinner1 := manager.NewSpinner()
	spinner1.SetMessage("First")
	stop1 := spinner1.Start(context.Background())

	// Create and start second spinner (should become active)
	spinner2 := manager.NewSpinner()
	spinner2.SetMessage("Second")
	stop2 := spinner2.Start(context.Background())

	// Create and start third spinner (should become active)
	spinner3 := manager.NewSpinner()
	spinner3.SetMessage("Third")
	stop3 := spinner3.Start(context.Background())

	// Verify all three are live but only the third is active
	assert.Equal(t, 3, manager.LiveCount())
	assert.Equal(t, spinner3.id, manager.ActiveID())

	// Give time to render
	time.Sleep(100 * time.Millisecond)

	// Stop all spinners before checking output (to avoid race with buffer)
	stop3()
	stop2()
	stop1()

	// Check output - should contain "Third" (the active spinner's message)
	output := buf.String()
	assert.Contains(t, output, "Third")

	assert.Equal(t, 0, manager.LiveCount())
}

func TestSpinnerManager_StoppingActiveSpinner_ActivatesNextMostRecent(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Create and start spinners in order
	spinner1 := manager.NewSpinner()
	spinner1.SetMessage("First")
	stop1 := spinner1.Start(context.Background())

	spinner2 := manager.NewSpinner()
	spinner2.SetMessage("Second")
	stop2 := spinner2.Start(context.Background())

	spinner3 := manager.NewSpinner()
	spinner3.SetMessage("Third")
	stop3 := spinner3.Start(context.Background())

	// spinner3 should be active
	assert.Equal(t, spinner3.id, manager.ActiveID())

	// Stop spinner3 - spinner2 should become active
	stop3()
	assert.Equal(t, 2, manager.LiveCount())
	assert.Equal(t, spinner2.id, manager.ActiveID())

	// Give time to render the newly active spinner
	time.Sleep(100 * time.Millisecond)

	// Stop spinner2 - spinner1 should become active
	stop2()
	assert.Equal(t, 1, manager.LiveCount())
	assert.Equal(t, spinner1.id, manager.ActiveID())

	// Stop spinner1 - no spinners left
	stop1()
	assert.Equal(t, 0, manager.LiveCount())
	assert.Equal(t, uint64(0), manager.ActiveID())
}

func TestSpinnerManager_StoppingNonActiveSpinner_DoesNotChangeActive(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner1 := manager.NewSpinner()
	spinner1.SetMessage("First")
	stop1 := spinner1.Start(context.Background())

	spinner2 := manager.NewSpinner()
	spinner2.SetMessage("Second")
	stop2 := spinner2.Start(context.Background())

	spinner3 := manager.NewSpinner()
	spinner3.SetMessage("Third")
	stop3 := spinner3.Start(context.Background())

	// spinner3 is active
	assert.Equal(t, spinner3.id, manager.ActiveID())

	// Stop spinner1 (not active) - spinner3 should still be active
	stop1()
	assert.Equal(t, 2, manager.LiveCount())
	assert.Equal(t, spinner3.id, manager.ActiveID())

	// Stop spinner2 (not active) - spinner3 should still be active
	stop2()
	assert.Equal(t, 1, manager.LiveCount())
	assert.Equal(t, spinner3.id, manager.ActiveID())

	// Stop spinner3
	stop3()
	assert.Equal(t, 0, manager.LiveCount())
}

func TestSpinnerManager_StopInReverseOrder(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Start spinners 1, 2, 3
	spinner1 := manager.NewSpinner()
	stop1 := spinner1.Start(context.Background())

	spinner2 := manager.NewSpinner()
	stop2 := spinner2.Start(context.Background())

	spinner3 := manager.NewSpinner()
	stop3 := spinner3.Start(context.Background())

	// Stop in order: 3, 2, 1 (reverse of creation)
	// Each time the active spinner should change to the next most recent
	stop3()
	assert.Equal(t, spinner2.id, manager.ActiveID())

	stop2()
	assert.Equal(t, spinner1.id, manager.ActiveID())

	stop1()
	assert.Equal(t, uint64(0), manager.ActiveID())
}

func TestSpinnerManager_StopInCreationOrder(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Start spinners 1, 2, 3
	spinner1 := manager.NewSpinner()
	stop1 := spinner1.Start(context.Background())

	spinner2 := manager.NewSpinner()
	stop2 := spinner2.Start(context.Background())

	spinner3 := manager.NewSpinner()
	stop3 := spinner3.Start(context.Background())

	// Stop in order: 1, 2, 3 (same as creation)
	// Active should stay as spinner3 until it's stopped
	stop1()
	assert.Equal(t, spinner3.id, manager.ActiveID())

	stop2()
	assert.Equal(t, spinner3.id, manager.ActiveID())

	stop3()
	assert.Equal(t, uint64(0), manager.ActiveID())
}

func TestSpinnerManager_DoubleStop(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner := manager.NewSpinner()
	stop := spinner.Start(context.Background())

	// Stop twice should be safe
	stop()
	stop() // Should not panic or cause issues

	assert.Equal(t, 0, manager.LiveCount())
}

func TestSpinnerManager_StartAlreadyStoppedSpinner(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner := manager.NewSpinner()
	stop := spinner.Start(context.Background())
	stop()

	// Starting an already stopped spinner should be a no-op
	stop2 := spinner.Start(context.Background())
	stop2()

	assert.Equal(t, 0, manager.LiveCount())
}

func TestSpinnerManager_MessageChange(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner := manager.NewSpinner()
	spinner.SetMessage("Initial")
	stop := spinner.Start(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Change message while running
	spinner.SetMessage("Changed")

	time.Sleep(100 * time.Millisecond)

	stop()

	output := buf.String()
	assert.Contains(t, output, "Initial")
	assert.Contains(t, output, "Changed")
}

func TestSpinnerManager_ConcurrentStops(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	const numSpinners = 10
	spinners := make([]*ManagedSpinner, numSpinners)
	stopFuncs := make([]func(), numSpinners)

	// Create and start all spinners
	for i := 0; i < numSpinners; i++ {
		spinners[i] = manager.NewSpinner()
		spinners[i].SetMessage("Spinner " + string(rune('0'+i)))
		stopFuncs[i] = spinners[i].Start(context.Background())
	}

	assert.Equal(t, numSpinners, manager.LiveCount())

	// Stop all concurrently
	var wg sync.WaitGroup
	for i := 0; i < numSpinners; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			stopFuncs[idx]()
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 0, manager.LiveCount())
	assert.Equal(t, uint64(0), manager.ActiveID())
}

func TestSpinnerManager_RapidStartStop(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Rapidly create, start, and stop spinners
	for i := 0; i < 20; i++ {
		spinner := manager.NewSpinner()
		stop := spinner.Start(context.Background())
		stop()
	}

	assert.Equal(t, 0, manager.LiveCount())
}

func TestSpinnerManager_AnimationContinuesAcrossActiveChange(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner1 := manager.NewSpinner()
	spinner1.SetMessage("First")
	stop1 := spinner1.Start(context.Background())

	spinner2 := manager.NewSpinner()
	spinner2.SetMessage("Second")
	stop2 := spinner2.Start(context.Background())

	// Let animation run
	time.Sleep(100 * time.Millisecond)

	// Stop active spinner
	stop2()

	// Animation should continue with spinner1
	time.Sleep(100 * time.Millisecond)

	stop1()

	output := buf.String()
	// Should have rendered both messages at different times
	assert.True(t, strings.Contains(output, "First") || strings.Contains(output, "Second"),
		"Output should contain spinner messages")
}

func TestSpinnerManager_NoRenderWhenNoSpinners(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Don't start any spinners, just verify initial state
	assert.Equal(t, 0, manager.LiveCount())
	assert.Equal(t, uint64(0), manager.ActiveID())
	assert.Empty(t, buf.String())
}

func TestSpinnerManager_SpinnerIDsAreUnique(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	spinner1 := manager.NewSpinner()
	spinner2 := manager.NewSpinner()
	spinner3 := manager.NewSpinner()

	// IDs should be unique and increasing
	require.NotEqual(t, spinner1.id, spinner2.id)
	require.NotEqual(t, spinner2.id, spinner3.id)
	require.NotEqual(t, spinner1.id, spinner3.id)

	// IDs should be in increasing order
	assert.True(t, spinner1.id < spinner2.id)
	assert.True(t, spinner2.id < spinner3.id)
}

func TestSpinnerManager_StopAll(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)
	ctx := context.Background()

	// Create and start multiple spinners
	spinner1 := manager.NewSpinnerWithID("s1")
	spinner1.SetMessage("First")
	spinner1.Start(ctx)

	spinner2 := manager.NewSpinnerWithID("s2")
	spinner2.SetMessage("Second")
	spinner2.Start(ctx)

	spinner3 := manager.NewSpinnerWithID("s3")
	spinner3.SetMessage("Third")
	spinner3.Start(ctx)

	// Verify all are running
	assert.Equal(t, 3, manager.LiveCount())
	assert.True(t, manager.HasActiveSpinners())

	// Give time to render
	time.Sleep(100 * time.Millisecond)

	// Stop all
	manager.StopAll()

	// Verify all stopped
	assert.Equal(t, 0, manager.LiveCount())
	assert.False(t, manager.HasActiveSpinners())

	// Verify spinners are no longer running
	assert.False(t, spinner1.IsRunning())
	assert.False(t, spinner2.IsRunning())
	assert.False(t, spinner3.IsRunning())

	// Verify they're removed from ID map
	_, exists := manager.GetSpinnerByID("s1")
	assert.False(t, exists)
	_, exists = manager.GetSpinnerByID("s2")
	assert.False(t, exists)
	_, exists = manager.GetSpinnerByID("s3")
	assert.False(t, exists)
}

func TestSpinnerManager_StopAll_Empty(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// StopAll on empty manager should be safe
	manager.StopAll()

	assert.Equal(t, 0, manager.LiveCount())
	assert.False(t, manager.HasActiveSpinners())
}

func TestSpinnerManager_StopAll_ThenStartNew(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)
	ctx := context.Background()

	// Start some spinners
	spinner1 := manager.NewSpinnerWithID("old1")
	spinner1.SetMessage("Old 1")
	spinner1.Start(ctx)

	spinner2 := manager.NewSpinnerWithID("old2")
	spinner2.SetMessage("Old 2")
	spinner2.Start(ctx)

	// Stop all
	manager.StopAll()
	assert.Equal(t, 0, manager.LiveCount())

	// Start a new spinner - should work fine
	spinner3 := manager.NewSpinnerWithID("new1")
	spinner3.SetMessage("New 1")
	spinner3.Start(ctx)

	assert.Equal(t, 1, manager.LiveCount())
	assert.True(t, spinner3.IsRunning())

	// Cleanup
	manager.StopAll()
}

func TestSpinnerManager_NewSpinnerBecomesActiveEvenIfOthersExist(t *testing.T) {
	var buf bytes.Buffer
	manager := NewSpinnerManager(&buf)

	// Start first spinner
	spinner1 := manager.NewSpinner()
	stop1 := spinner1.Start(context.Background())
	assert.Equal(t, spinner1.id, manager.ActiveID())

	// Start second spinner - should become active
	spinner2 := manager.NewSpinner()
	stop2 := spinner2.Start(context.Background())
	assert.Equal(t, spinner2.id, manager.ActiveID())

	// Start third spinner - should become active
	spinner3 := manager.NewSpinner()
	stop3 := spinner3.Start(context.Background())
	assert.Equal(t, spinner3.id, manager.ActiveID())

	stop3()
	stop2()
	stop1()
}

func TestNewInlineSpinner(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewInlineSpinner(&buf)

	assert.NotNil(t, spinner)
	assert.Equal(t, &buf, spinner.writer)
	assert.Equal(t, SpinnerFrames, spinner.frames)
}

func TestInlineSpinnerSetPrefixSuffix(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewInlineSpinner(&buf)

	spinner.SetPrefix("○ tool")
	assert.Equal(t, "○ tool", spinner.prefix)

	spinner.SetSuffix(" done")
	assert.Equal(t, " done", spinner.suffix)
}

func TestInlineSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewInlineSpinner(&buf)
	spinner.SetPrefix("○ test")

	ctx := context.Background()
	cancel := spinner.Start(ctx)

	// Give the spinner time to render
	time.Sleep(100 * time.Millisecond)

	cancel()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "○ test")
}

func TestInlineSpinnerClearAndFinish(t *testing.T) {
	var buf bytes.Buffer
	spinner := NewInlineSpinner(&buf)

	spinner.ClearAndFinish()

	// Should contain ANSI clear line sequence
	output := buf.String()
	assert.Contains(t, output, "\r\033[K")
}
