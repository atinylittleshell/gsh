package completion

import (
	"testing"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCompleter_GetCompletions(t *testing.T) {
	completer := &DefaultCompleter{}

	tests := []struct {
		name      string
		command   string
		args      []string
		wantFound bool
		wantValue string // Check if at least one completion matches this
	}{
		{
			name:      "cd completion",
			command:   "cd",
			args:      []string{},
			wantFound: true,
			// We can't easily test result values as it depends on filesystem, but we expect found=true
		},
		{
			name:      "export completion",
			command:   "export",
			args:      []string{},
			wantFound: true,
			// Assumes PATH is in environment
			wantValue: "PATH",
		},
		{
			name:      "kill completion",
			command:   "kill",
			args:      []string{"-"},
			wantFound: true,
			wantValue: "-KILL",
		},
		{
			name:      "unknown command",
			command:   "unknown",
			args:      []string{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := completer.GetCompletions(tt.command, tt.args, "", 0)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantValue != "" {
				match := false
				for _, c := range got {
					if c.Value == tt.wantValue || (tt.command == "export" && c.Value == tt.wantValue) {
						match = true
						break
					}
				}
				assert.True(t, match, "Expected to find value %q in completions", tt.wantValue)
			}
		})
	}
}

func TestStaticCompleter(t *testing.T) {
	completer := NewStaticCompleter()

	tests := []struct {
		name      string
		command   string
		args      []string
		wantValue string
	}{
		{
			name:      "docker completion",
			command:   "docker",
			args:      []string{},
			wantValue: "run",
		},
		{
			name:      "docker completion filter",
			command:   "docker",
			args:      []string{"p"},
			wantValue: "ps",
		},
		{
			name:      "npm completion",
			command:   "npm",
			args:      []string{},
			wantValue: "install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := completer.GetCompletions(tt.command, tt.args)

			match := false
			for _, c := range got {
				if c.Value == tt.wantValue {
					match = true
					break
				}
			}
			assert.True(t, match, "Expected to find value %q in completions", tt.wantValue)
		})
	}
}

func TestGitCompleter_Subcommands(t *testing.T) {
	completer := &GitCompleter{}

	// Test subcommands (empty args, line doesn't matter for subcommand completion)
	got := completer.GetCompletions([]string{}, "git ")

	expected := []string{"checkout", "commit", "add", "push", "pull", "status"}
	for _, exp := range expected {
		match := false
		for _, c := range got {
			if c.Value == exp {
				match = true
				break
			}
		}
		assert.True(t, match, "Expected to find git subcommand %q", exp)
	}
}

// Helper to avoid unused import error if we don't use the package explicitly
var _ = shellinput.CompletionCandidate{}
