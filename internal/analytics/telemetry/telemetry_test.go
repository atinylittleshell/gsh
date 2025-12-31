package telemetry

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAnonymousID(t *testing.T) {
	id1 := generateAnonymousID()
	id2 := generateAnonymousID()

	// Should be 32 hex characters (16 bytes)
	assert.Len(t, id1, 32)
	assert.Len(t, id2, 32)

	// Should be different (random component)
	assert.NotEqual(t, id1, id2)
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()

	// Should be 16 hex characters (8 bytes)
	assert.Len(t, id1, 16)

	// Verify it's valid hex
	_, err := hex.DecodeString(id1)
	assert.NoError(t, err)
}

func TestSetTelemetryEnabled(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	consentPath := filepath.Join(tempDir, "analytics_consent")

	// Override the consent file path for testing
	originalDataDir := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalDataDir)

	// Test disabling
	err := os.WriteFile(consentPath, []byte("0"), 0644)
	require.NoError(t, err)

	data, err := os.ReadFile(consentPath)
	require.NoError(t, err)
	assert.Equal(t, "0", string(data))

	// Test enabling
	err = os.WriteFile(consentPath, []byte("1"), 0644)
	require.NoError(t, err)

	data, err = os.ReadFile(consentPath)
	require.NoError(t, err)
	assert.Equal(t, "1", string(data))
}

func TestGetTelemetryStatus(t *testing.T) {
	// Test with environment variable
	os.Setenv(envNoTelemetry, "1")
	status := GetTelemetryStatus()
	assert.Contains(t, status, "disabled")
	assert.Contains(t, status, "GSH_NO_TELEMETRY")
	os.Unsetenv(envNoTelemetry)
}

func TestGetFirstRunNotification(t *testing.T) {
	notification := GetFirstRunNotification()
	assert.Contains(t, notification, "anonymous usage statistics")
	assert.Contains(t, notification, "gsh telemetry off")
	assert.Contains(t, notification, "github.com/atinylittleshell/gsh")
}

func TestClientDebugMode(t *testing.T) {
	os.Setenv(envTelemetryDebug, "1")
	defer os.Unsetenv(envTelemetryDebug)

	client, err := NewClient(Config{Version: "test"})
	require.NoError(t, err)
	defer client.Close()

	assert.True(t, client.debugMode)
	// In debug mode, client should be nil (no actual PostHog connection)
	assert.Nil(t, client.client)
}

func TestClientDisabledByEnv(t *testing.T) {
	os.Setenv(envNoTelemetry, "1")
	defer os.Unsetenv(envNoTelemetry)

	client, err := NewClient(Config{Version: "test"})
	require.NoError(t, err)
	defer client.Close()

	assert.False(t, client.IsEnabled())
}

func TestClientTracking(t *testing.T) {
	// Disable actual sending
	os.Setenv(envNoTelemetry, "1")
	defer os.Unsetenv(envNoTelemetry)

	client, err := NewClient(Config{Version: "test"})
	require.NoError(t, err)
	defer client.Close()

	// These should not panic even when disabled
	client.TrackSessionStart("repl")
	client.TrackScriptExecution()
	client.TrackError(ErrorCategoryParse)
	client.TrackStartupTime(100)
	client.TrackSessionEnd()
}

func TestCounters(t *testing.T) {
	os.Setenv(envNoTelemetry, "1")
	defer os.Unsetenv(envNoTelemetry)

	client, err := NewClient(Config{Version: "test"})
	require.NoError(t, err)
	defer client.Close()

	// Track multiple script executions
	client.TrackScriptExecution()
	client.TrackScriptExecution()
	client.TrackScriptExecution()

	client.mu.Lock()
	count := client.counters[EventScriptExecution]
	client.mu.Unlock()

	assert.Equal(t, int64(3), count)
}
