package completion

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func TestCompletionFunction(t *testing.T) {
	t.Run("basic function execution", func(t *testing.T) {
		// Create a test completion function
		script := `
_test_completion() {
    COMPREPLY=(foo bar baz)
}
`
		file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
		assert.NoError(t, err)

		runner, err := interp.New(
			interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		)
		assert.NoError(t, err)

		err = runner.Run(context.Background(), file)
		assert.NoError(t, err)

		// Create completion function
		fn := NewCompletionFunction("_test_completion", runner)

		// Execute completion
		results, err := fn.Execute(context.Background(), []string{"mycmd", "f"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"foo", "bar", "baz"}, results)
	})

	t.Run("context-aware completion", func(t *testing.T) {
		// Create a test completion function that uses COMP variables
		script := `
_test_completion() {
    if [[ ${COMP_WORDS[COMP_CWORD]} == f* ]]; then
        COMPREPLY=(foo)
    elif [[ ${COMP_WORDS[COMP_CWORD]} == b* ]]; then
        COMPREPLY=(bar baz)
    else
        COMPREPLY=(foo bar baz)
    fi
}
`
		file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
		assert.NoError(t, err)

		runner, err := interp.New(
			interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		)
		assert.NoError(t, err)

		err = runner.Run(context.Background(), file)
		assert.NoError(t, err)

		// Create completion function
		fn := NewCompletionFunction("_test_completion", runner)

		// Test with "f" prefix
		results, err := fn.Execute(context.Background(), []string{"mycmd", "f"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"foo"}, results)

		// Test with "b" prefix
		results, err = fn.Execute(context.Background(), []string{"mycmd", "b"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"bar", "baz"}, results)

		// Test with no prefix
		results, err = fn.Execute(context.Background(), []string{"mycmd", ""})
		assert.NoError(t, err)
		assert.Equal(t, []string{"foo", "bar", "baz"}, results)
	})
}
