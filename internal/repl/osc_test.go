package repl

import (
	"bytes"
	"testing"
)

func TestEmitOSC7(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		dir      string
		expected string
	}{
		{
			name:     "normal path",
			hostname: "myhost",
			dir:      "/Users/test/projects",
			expected: "\033]7;file://myhost/Users/test/projects\033\\",
		},
		{
			name:     "path with spaces",
			hostname: "myhost",
			dir:      "/Users/test/my projects",
			expected: "\033]7;file://myhost/Users/test/my%20projects\033\\",
		},
		{
			name:     "path with hash",
			hostname: "myhost",
			dir:      "/Users/test/project#1",
			expected: "\033]7;file://myhost/Users/test/project%231\033\\",
		},
		{
			name:     "empty hostname",
			hostname: "",
			dir:      "/Users/test",
			expected: "\033]7;file:///Users/test\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			emitOSC7(&buf, tt.hostname, tt.dir)
			got := buf.String()
			if got != tt.expected {
				t.Errorf("emitOSC7(%q, %q) = %q, want %q", tt.hostname, tt.dir, got, tt.expected)
			}
		})
	}
}
