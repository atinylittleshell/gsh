// Package telemetry provides privacy-respecting analytics for gsh.
// It follows the Homebrew analytics model - opt-out with notification,
// strictly anonymous, and silent failures.
package telemetry

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/posthog/posthog-go"
)

const (
	// PostHog EU endpoint for GDPR compliance
	posthogEndpoint = "https://eu.i.posthog.com"

	// PostHog project API key (public, safe to embed)
	// This key can only be used to send events, not read data
	posthogAPIKey = "phc_WSvCBQuNjNtw2s2CHQZhLLRZhYJB0EsW7fMxmxWKHno"

	// File names for consent and ID storage
	consentFileName    = "analytics_consent"
	anonymousIDFile    = "anonymous_id"
	firstRunMarkerFile = "first_run_complete"

	// Environment variables
	envNoTelemetry    = "GSH_NO_TELEMETRY"
	envTelemetryDebug = "GSH_TELEMETRY_DEBUG"
)

// Event names - what we track
const (
	EventSessionStart = "session_start"
	EventSessionEnd   = "session_end"

	// Feature adoption events (counts only)
	EventScriptExecution = "script_execution"

	// Error categories (no content)
	EventError = "error"
)

// Error categories for tracking
const (
	ErrorCategoryParse         = "parse_error"
	ErrorCategoryRuntime       = "runtime_error"
	ErrorCategoryMCPConnection = "mcp_connection_failed"
	ErrorCategoryAgent         = "agent_error"
	ErrorCategoryScript        = "script_error"
)

// Client manages telemetry for gsh
type Client struct {
	mu          sync.Mutex
	client      posthog.Client
	anonymousID string
	enabled     bool
	debugMode   bool
	version     string
	sessionID   string
	startTime   time.Time

	// Counters for batch reporting
	counters map[string]int64
}

// Config holds configuration for the telemetry client
type Config struct {
	Version string
	Enabled bool // Override enabled state (for testing)
}

// NewClient creates a new telemetry client
func NewClient(cfg Config) (*Client, error) {
	c := &Client{
		version:   cfg.Version,
		startTime: time.Now(),
		sessionID: generateSessionID(),
		counters:  make(map[string]int64),
		debugMode: os.Getenv(envTelemetryDebug) == "1",
	}

	// Check if telemetry is enabled
	c.enabled = c.isTelemetryEnabled()
	if cfg.Enabled {
		c.enabled = true // Allow override for testing
	}

	// Generate or load anonymous ID
	c.anonymousID = c.getOrCreateAnonymousID()

	// Only create PostHog client if enabled and not in debug mode
	if c.enabled && !c.debugMode {
		client, err := posthog.NewWithConfig(
			posthogAPIKey,
			posthog.Config{
				Endpoint: posthogEndpoint,
				// Send events immediately instead of batching to avoid
				// delays during client.Close() which flushes the queue
				BatchSize: 10,
				Interval:  100 * time.Millisecond,
			},
		)
		if err != nil {
			// Silent failure - analytics should never break the app
			c.enabled = false
			return c, nil
		}
		c.client = client
	}

	return c, nil
}

// Close flushes pending events and closes the client
func (c *Client) Close() error {
	if c.client != nil {
		// Send session end event with duration
		c.TrackSessionEnd()
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns whether analytics is currently enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// IsFirstRun checks if this is the first time gsh is being run
func IsFirstRun() bool {
	markerPath := filepath.Join(core.DataDir(), firstRunMarkerFile)
	_, err := os.Stat(markerPath)
	return os.IsNotExist(err)
}

// MarkFirstRunComplete marks that the first run notification has been shown
func MarkFirstRunComplete() error {
	markerPath := filepath.Join(core.DataDir(), firstRunMarkerFile)
	return os.WriteFile(markerPath, []byte("1"), 0644)
}

// GetFirstRunNotification returns the notification message to show on first run
func GetFirstRunNotification() string {
	return `gsh collects anonymous usage statistics to help improve the product.
No commands, prompts, or personal data are ever collected.

To opt out: gsh telemetry off
Learn more: https://github.com/atinylittleshell/gsh#telemetry
`
}

// isTelemetryEnabled checks all conditions for telemetry being enabled
func (c *Client) isTelemetryEnabled() bool {
	// Check environment variable opt-out
	if os.Getenv(envNoTelemetry) == "1" {
		return false
	}

	// Check consent file
	consentPath := filepath.Join(core.DataDir(), consentFileName)
	data, err := os.ReadFile(consentPath)
	if err != nil {
		// No consent file means default to enabled (opt-out model)
		return true
	}

	// If file contains "0" or "off", analytics is disabled
	consent := string(data)
	return consent != "0" && consent != "off"
}

// SetTelemetryEnabled enables or disables telemetry
func SetTelemetryEnabled(enabled bool) error {
	consentPath := filepath.Join(core.DataDir(), consentFileName)
	value := "1"
	if !enabled {
		value = "0"
	}
	return os.WriteFile(consentPath, []byte(value), 0644)
}

// GetTelemetryStatus returns the current telemetry status as a string
func GetTelemetryStatus() string {
	// Check environment variable
	if os.Getenv(envNoTelemetry) == "1" {
		return "disabled (via GSH_NO_TELEMETRY environment variable)"
	}

	// Check consent file
	consentPath := filepath.Join(core.DataDir(), consentFileName)
	data, err := os.ReadFile(consentPath)
	if err != nil {
		return "enabled (opt out with 'gsh telemetry off')"
	}

	consent := string(data)
	if consent == "0" || consent == "off" {
		return "disabled"
	}
	return "enabled (opt out with 'gsh telemetry off')"
}

// getOrCreateAnonymousID generates or loads a persistent anonymous ID
func (c *Client) getOrCreateAnonymousID() string {
	idPath := filepath.Join(core.DataDir(), anonymousIDFile)

	// Try to read existing ID
	data, err := os.ReadFile(idPath)
	if err == nil && len(data) > 0 {
		return string(data)
	}

	// Generate new anonymous ID (hash of machine identifier + random component)
	id := generateAnonymousID()

	// Save for future sessions
	_ = os.WriteFile(idPath, []byte(id), 0600)

	return id
}

// generateAnonymousID creates a hashed anonymous identifier
func generateAnonymousID() string {
	// Use hostname + user home dir as machine identifier
	// This is hashed so no PII is stored
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	// Add cryptographically secure random component for additional privacy
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to less secure but still functional random if crypto/rand fails
		randomBytes = fmt.Appendf(nil, "%d-%d", time.Now().UnixNano(), os.Getpid())
	}

	data := hostname + homeDir + string(randomBytes)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	// Use cryptographically secure random bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback if crypto/rand fails
		data := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
		hash := sha256.Sum256([]byte(data))
		return hex.EncodeToString(hash[:8])
	}
	return hex.EncodeToString(randomBytes)
}

// track sends an event to PostHog (internal helper)
func (c *Client) track(event string, properties map[string]any) {
	if !c.enabled {
		return
	}

	// Add common properties
	if properties == nil {
		properties = make(map[string]any)
	}
	properties["gsh_version"] = c.version
	properties["os"] = runtime.GOOS
	properties["arch"] = runtime.GOARCH
	properties["session_id"] = c.sessionID

	if c.debugMode {
		fmt.Fprintf(os.Stderr, "[telemetry debug] Would send: {event: '%s', properties: %v}\n", event, properties)
		return
	}

	if c.client == nil {
		return
	}

	// Enqueue event (non-blocking)
	_ = c.client.Enqueue(posthog.Capture{
		DistinctId: c.anonymousID,
		Event:      event,
		Properties: properties,
	})
}

// TrackSessionStart records the start of a gsh session
func (c *Client) TrackSessionStart(mode string) {
	c.track(EventSessionStart, map[string]any{
		"mode": mode, // "repl" or "script"
	})
}

// TrackSessionEnd records the end of a gsh session with duration
func (c *Client) TrackSessionEnd() {
	duration := time.Since(c.startTime).Seconds()
	c.track(EventSessionEnd, map[string]any{
		"duration_seconds": duration,
	})
}

// TrackScriptExecution records a .gsh script execution (count only)
func (c *Client) TrackScriptExecution() {
	c.mu.Lock()
	c.counters[EventScriptExecution]++
	c.mu.Unlock()
	c.track(EventScriptExecution, nil)
}

// TrackError records an error by category (no error content)
func (c *Client) TrackError(category string) {
	c.track(EventError, map[string]any{
		"category": category,
	})
}

// TrackStartupTime records the startup time in milliseconds
func (c *Client) TrackStartupTime(durationMs int64) {
	c.track("startup_time", map[string]any{
		"duration_ms": durationMs,
	})
}
