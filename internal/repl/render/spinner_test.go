package render

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
