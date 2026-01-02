package telemetry

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleTelemetryCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectHandled  bool
		expectError    bool
		expectedOutput string
	}{
		{
			name:          "not telemetry command",
			args:          []string{"gsh", "script.gsh"},
			expectHandled: false,
		},
		{
			name:          "wrong first arg",
			args:          []string{"bash", "telemetry"},
			expectHandled: false,
		},
		{
			name:          "telemetry status implicit",
			args:          []string{"gsh", "telemetry"},
			expectHandled: true,
		},
		{
			name:          "telemetry status explicit",
			args:          []string{"gsh", "telemetry", "status"},
			expectHandled: true,
		},
		{
			name:           "telemetry -h",
			args:           []string{"gsh", "telemetry", "-h"},
			expectHandled:  true,
			expectedOutput: "Usage: gsh telemetry",
		},
		{
			name:          "unknown subcommand",
			args:          []string{"gsh", "telemetry", "unknown"},
			expectHandled: true,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			handled, err := HandleTelemetryCommand(tt.args)

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stdout = oldStdout

			assert.Equal(t, tt.expectHandled, handled)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expectedOutput != "" {
				assert.Contains(t, buf.String(), tt.expectedOutput)
			}
		})
	}
}

func TestHandleTelemetryOnOff(t *testing.T) {
	// These tests modify the consent file, so we need to be careful
	// The actual file operations are tested in telemetry_test.go

	// Test that on/off commands are handled
	handled, _ := HandleTelemetryCommand([]string{"gsh", "telemetry", "on"})
	assert.True(t, handled)

	handled, _ = HandleTelemetryCommand([]string{"gsh", "telemetry", "off"})
	assert.True(t, handled)
}
