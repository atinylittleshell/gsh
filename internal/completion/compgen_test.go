package completion

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func TestCompgenCommand(t *testing.T) {
	// Save and restore printf
	oldPrintf := printf
	defer func() { printf = oldPrintf }()

	tests := []struct {
		name          string
		args          []string
		setupScript   string
		want          []string
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name: "word list completion with no filter",
			args: []string{"compgen", "-W", "foo bar baz"},
			want: []string{"foo", "bar", "baz"},
		},
		{
			name: "word list completion with filter",
			args: []string{"compgen", "-W", "foo bar baz", "b"},
			want: []string{"bar", "baz"},
		},
		{
			name: "word list completion with no matches",
			args: []string{"compgen", "-W", "foo bar baz", "x"},
			want: []string{},
		},
		{
			name: "function completion",
			args: []string{"compgen", "-F", "my_completion", "b"},
			setupScript: `
				my_completion() {
					COMPREPLY=(bar baz)
				}
			`,
			want: []string{"bar", "baz"},
		},
		{
			name: "function completion with filter",
			args: []string{"compgen", "-F", "my_completion", "ba"},
			setupScript: `
				my_completion() {
					COMPREPLY=(bar baz foo)
				}
			`,
			want: []string{"bar", "baz"},
		},
		{
			name:          "missing -W argument",
			args:          []string{"compgen", "-W"},
			wantErr:       true,
			wantErrPrefix: "option -W requires a word list",
		},
		{
			name:          "missing -F argument",
			args:          []string{"compgen", "-F"},
			wantErr:       true,
			wantErrPrefix: "option -F requires a function name",
		},
		{
			name:          "unknown option",
			args:          []string{"compgen", "-x"},
			wantErr:       true,
			wantErrPrefix: "unknown option: -x",
		},
		{
			name:          "no options",
			args:          []string{"compgen"},
			wantErr:       true,
			wantErrPrefix: "compgen: no options specified",
		},
		{
			name:          "no completion type",
			args:          []string{"compgen", "word"},
			wantErr:       true,
			wantErrPrefix: "compgen: no completion type specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var output []string
			printf = func(format string, a ...interface{}) (n int, err error) {
				output = append(output, strings.TrimSpace(fmt.Sprintf(format, a...)))
				return 0, nil
			}

			// Create a new runner
			parser := syntax.NewParser()
			runner, err := interp.New()
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			// Set up the completion function if needed
			if tt.setupScript != "" {
				file, err := parser.Parse(strings.NewReader(tt.setupScript), "")
				if err != nil {
					t.Fatalf("failed to parse setup script: %v", err)
				}
				if err := runner.Run(context.Background(), file); err != nil {
					t.Fatalf("failed to run setup script: %v", err)
				}
			}

			// Create the handler
			handler := NewCompgenCommandHandler(runner)
			execHandler := handler(nil)

			// Run the command
			err = execHandler(context.Background(), tt.args)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.HasPrefix(err.Error(), tt.wantErrPrefix) {
					t.Errorf("error = %v, wantPrefix %v", err, tt.wantErrPrefix)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check output
			if len(output) != len(tt.want) {
				t.Errorf("got %d completions, want %d", len(output), len(tt.want))
				return
			}
			for i, w := range tt.want {
				if output[i] != w {
					t.Errorf("completion[%d] = %q, want %q", i, output[i], w)
				}
			}
		})
	}
}
