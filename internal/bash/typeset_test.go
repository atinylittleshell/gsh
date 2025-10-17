package bash

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
)

func TestTypesetFunctionListing(t *testing.T) {
	// Create a test runner
	runner, err := interp.New(interp.StdIO(nil, nil, nil))
	assert.NoError(t, err)

	// Set the global runner for our handler
	SetTypesetRunner(runner)

	// Define some test functions
	testScript := `
testfunc1() { echo "hello world"; }
testfunc2() { ls -la; pwd; }
`

	// Run the test script to define functions
	err = RunBashScriptFromReader(context.Background(), runner, strings.NewReader(testScript), "test")
	assert.NoError(t, err)

	// Test function listing with -f option
	handler := NewTypesetCommandHandler()

	// Create a mock exec handler that captures output
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with -f option
	err = wrappedHandler(context.Background(), []string{"gsh_typeset", "-f"})
	assert.NoError(t, err)

	// The output should contain our function definitions
	// Note: We can't easily capture the output in tests, but we can verify no errors occurred
}

func TestTypesetFunctionNames(t *testing.T) {
	// Create a test runner
	runner, err := interp.New(interp.StdIO(nil, nil, nil))
	assert.NoError(t, err)

	// Set the global runner for our handler
	SetTypesetRunner(runner)

	// Define some test functions
	testScript := `
testfunc1() { echo "hello"; }
testfunc2() { ls; }
`

	// Run the test script to define functions
	err = RunBashScriptFromReader(context.Background(), runner, strings.NewReader(testScript), "test")
	assert.NoError(t, err)

	// Create handler
	handler := NewTypesetCommandHandler()
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with -F option
	err = wrappedHandler(context.Background(), []string{"gsh_typeset", "-F"})
	assert.NoError(t, err)
}

func TestTypesetVariableListing(t *testing.T) {
	// Create a test runner
	runner, err := interp.New(interp.StdIO(nil, nil, nil))
	assert.NoError(t, err)

	// Set the global runner for our handler
	SetTypesetRunner(runner)

	// Set some test variables
	testScript := `
TEST_VAR1="value1"
TEST_VAR2="value2"
export EXPORTED_VAR="exported_value"
`

	// Run the test script to set variables
	err = RunBashScriptFromReader(context.Background(), runner, strings.NewReader(testScript), "test")
	assert.NoError(t, err)

	// Create handler
	handler := NewTypesetCommandHandler()
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with -p option
	err = wrappedHandler(context.Background(), []string{"gsh_typeset", "-p"})
	assert.NoError(t, err)
}

func TestTypesetInvalidOption(t *testing.T) {
	// Create a test runner
	runner, err := interp.New(interp.StdIO(nil, nil, nil))
	assert.NoError(t, err)

	// Set the global runner for our handler
	SetTypesetRunner(runner)

	// Create handler
	handler := NewTypesetCommandHandler()
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with invalid option
	err = wrappedHandler(context.Background(), []string{"gsh_typeset", "-z"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid option")
}

func TestTypesetNoOptions(t *testing.T) {
	// Create a test runner
	runner, err := interp.New(interp.StdIO(nil, nil, nil))
	assert.NoError(t, err)

	// Set the global runner for our handler
	SetTypesetRunner(runner)

	// Create handler
	handler := NewTypesetCommandHandler()
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with no options (should default to variable listing)
	err = wrappedHandler(context.Background(), []string{"gsh_typeset"})
	assert.NoError(t, err)
}

func TestTypesetRunnerNotInitialized(t *testing.T) {
	// Reset global runner to nil for testing uninitialized state
	// This direct access to globalRunner is intentionally used here for testing
	// the error condition when the runner is not initialized. This is the
	// only place where direct global variable manipulation is necessary.
	oldRunner := globalRunner
	globalRunner = nil
	defer func() {
		globalRunner = oldRunner
	}()

	// Create handler
	handler := NewTypesetCommandHandler()
	mockNext := func(ctx context.Context, args []string) error {
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test should return error when runner is not initialized
	err := wrappedHandler(context.Background(), []string{"gsh_typeset", "-f"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "runner not initialized")
}

func TestTypesetNonTypesetCommand(t *testing.T) {
	// Create handler
	handler := NewTypesetCommandHandler()

	called := false
	mockNext := func(ctx context.Context, args []string) error {
		called = true
		return nil
	}

	wrappedHandler := handler(mockNext)

	// Test with non-typeset command (should call next)
	err := wrappedHandler(context.Background(), []string{"echo", "hello"})
	assert.NoError(t, err)
	assert.True(t, called)
}
