package completion

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func TestCompletionManager(t *testing.T) {
	t.Run("execute completion", func(t *testing.T) {
		manager := NewCompletionManager()

		// Create a shell runner for function-based completion
		parser := syntax.NewParser()
		runner, err := interp.New()
		assert.NoError(t, err)

		// Set up completion functions
		setupScript := `
			my_completion() {
				COMPREPLY=(foo bar baz)
			}

			prefix_completion() {
				local cur="${COMP_WORDS[COMP_CWORD]}"
				local words="foo bar baz"
				COMPREPLY=()
				for w in $words; do
					if [[ "$w" == "$cur"* ]]; then
						COMPREPLY+=("$w")
					fi
				done
			}
		`
		file, err := parser.Parse(strings.NewReader(setupScript), "")
		assert.NoError(t, err)
		err = runner.Run(context.Background(), file)
		assert.NoError(t, err)

		tests := []struct {
			name    string
			spec    CompletionSpec
			args    []string
			line    string
			pos     int
			want    []shellinput.CompletionCandidate
			wantErr bool
		}{
			{
				name: "word list completion - no filter",
				spec: CompletionSpec{
					Type:  WordListCompletion,
					Value: "apple banana cherry",
				},
				args: []string{},
				line: "",
				pos:  0,
				want: []shellinput.CompletionCandidate{
					{Value: "apple"},
					{Value: "banana"},
					{Value: "cherry"},
				},
			},
			{
				name: "word list completion - with filter",
				spec: CompletionSpec{
					Type:  WordListCompletion,
					Value: "apple banana cherry",
				},
				args: []string{"command", "b"},
				line: "command b",
				pos:  9,
				want: []shellinput.CompletionCandidate{
					{Value: "banana"},
				},
			},
			{
				name: "word list completion - no matches",
				spec: CompletionSpec{
					Type:  WordListCompletion,
					Value: "apple banana cherry",
				},
				args: []string{"command", "x"},
				line: "command x",
				pos:  9,
				want: []shellinput.CompletionCandidate{},
			},
			{
				name: "function completion - basic",
				spec: CompletionSpec{
					Type:  FunctionCompletion,
					Value: "my_completion",
				},
				args: []string{"command", "arg"},
				line: "command arg",
				pos:  11,
				want: []shellinput.CompletionCandidate{
					{Value: "foo"},
					{Value: "bar"},
					{Value: "baz"},
				},
			},
			{
				name: "function completion - with prefix handling",
				spec: CompletionSpec{
					Type:  FunctionCompletion,
					Value: "prefix_completion",
				},
				args: []string{"command", "b"},
				line: "command b",
				pos:  9,
				want: []shellinput.CompletionCandidate{
					{Value: "bar"},
					{Value: "baz"},
				},
			},
			{
				name: "function completion - empty args",
				spec: CompletionSpec{
					Type:  FunctionCompletion,
					Value: "my_completion",
				},
				args: []string{},
				line: "",
				pos:  0,
				want: []shellinput.CompletionCandidate{
					{Value: "foo"},
					{Value: "bar"},
					{Value: "baz"},
				},
			},
			{
				name: "invalid completion type",
				spec: CompletionSpec{
					Type:  CompletionType("invalid"),
					Value: "something",
				},
				args:    []string{"command", "arg"},
				line:    "command arg",
				pos:     11,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := manager.ExecuteCompletion(context.Background(), runner, tt.spec, tt.args, tt.line, tt.pos)

				if tt.wantErr {
					assert.Error(t, err)
					return
				}

				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("basic operations", func(t *testing.T) {
		manager := NewCompletionManager()

		// Test adding a spec
		spec := CompletionSpec{
			Command: "test-cmd",
			Type:    "W",
			Value:   "foo bar baz",
		}
		manager.AddSpec(spec)

		// Test getting the spec
		retrieved, exists := manager.GetSpec("test-cmd")
		assert.True(t, exists)
		assert.Equal(t, spec, retrieved)

		// Test getting non-existent spec
		_, exists = manager.GetSpec("non-existent")
		assert.False(t, exists)

		// Test listing specs
		specs := manager.ListSpecs()
		assert.Len(t, specs, 1)
		assert.Equal(t, spec, specs[0])

		// Test removing spec
		manager.RemoveSpec("test-cmd")
		_, exists = manager.GetSpec("test-cmd")
		assert.False(t, exists)
		specs = manager.ListSpecs()
		assert.Empty(t, specs)
	})
}

func TestCompleteCommandHandler(t *testing.T) {
	t.Run("completion specifications", func(t *testing.T) {
		manager := NewCompletionManager()
		handler := NewCompleteCommandHandler(manager)

		// Create a mock next handler that just returns nil
		nextHandler := func(ctx context.Context, args []string) error {
			return nil
		}

		wrappedHandler := handler(nextHandler)

		// Test adding word list completion
		var captured []string
		oldPrintf := printf
		printf = func(format string, a ...any) (int, error) {
			captured = append(captured, fmt.Sprintf(format, a...))
			return len(format), nil
		}
		defer func() { printf = oldPrintf }()

		// Test word list completion
		err := wrappedHandler(context.Background(), []string{"complete", "-W", "foo bar", "mycmd"})
		assert.NoError(t, err)

		// Verify the word list spec was added correctly
		spec, exists := manager.GetSpec("mycmd")
		assert.True(t, exists)
		assert.Equal(t, WordListCompletion, spec.Type)
		assert.Equal(t, "foo bar", spec.Value)

		// Test function completion
		err = wrappedHandler(context.Background(), []string{"complete", "-F", "_mycmd_completion", "mycmd2"})
		assert.NoError(t, err)

		// Verify the function spec was added correctly
		spec, exists = manager.GetSpec("mycmd2")
		assert.True(t, exists)
		assert.Equal(t, FunctionCompletion, spec.Type)
		assert.Equal(t, "_mycmd_completion", spec.Value)

		// Test command completion
		err = wrappedHandler(context.Background(), []string{"complete", "-C", "mock_completer", "mycmd3"})
		assert.NoError(t, err)

		// Verify the command spec was added correctly
		spec, exists = manager.GetSpec("mycmd3")
		assert.True(t, exists)
		assert.Equal(t, CommandCompletion, spec.Type)
		assert.Equal(t, "mock_completer", spec.Value)

		// Test complete -p
		captured = []string{}
		err = wrappedHandler(context.Background(), []string{"complete", "-p"})
		assert.NoError(t, err)
		assert.Contains(t, captured, "complete -W \"foo bar\" mycmd\n")
		assert.Contains(t, captured, "complete -F _mycmd_completion mycmd2\n")
		assert.Contains(t, captured, "complete -C \"mock_completer\" mycmd3\n")

		// Test complete -p mycmd
		captured = []string{}
		err = wrappedHandler(context.Background(), []string{"complete", "-p", "mycmd"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"complete -W \"foo bar\" mycmd\n"}, captured)

		// Test complete -r mycmd
		err = wrappedHandler(context.Background(), []string{"complete", "-r", "mycmd"})
		assert.NoError(t, err)
		_, exists = manager.GetSpec("mycmd")
		assert.False(t, exists)
	})

	t.Run("error cases", func(t *testing.T) {
		manager := NewCompletionManager()
		handler := NewCompleteCommandHandler(manager)
		nextHandler := func(ctx context.Context, args []string) error {
			return nil
		}
		wrappedHandler := handler(nextHandler)

		testCases := []struct {
			name    string
			args    []string
			wantErr string
		}{
			{
				name:    "missing word list",
				args:    []string{"complete", "-W"},
				wantErr: "option -W requires a word list",
			},
			{
				name:    "unknown option",
				args:    []string{"complete", "-x", "mycmd"},
				wantErr: "unknown option: -x",
			},
			{
				name:    "no command specified",
				args:    []string{"complete", "-W", "foo bar"},
				wantErr: "no command specified",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := wrappedHandler(context.Background(), tc.args)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		}
	})

	t.Run("pass through non-complete commands", func(t *testing.T) {
		manager := NewCompletionManager()
		handler := NewCompleteCommandHandler(manager)

		nextCalled := false
		nextHandler := func(ctx context.Context, args []string) error {
			nextCalled = true
			return nil
		}
		wrappedHandler := handler(nextHandler)

		err := wrappedHandler(context.Background(), []string{"echo", "hello"})
		assert.NoError(t, err)
		assert.True(t, nextCalled)
	})
}
