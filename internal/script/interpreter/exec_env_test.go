package interpreter

import (
	"os"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// TestExec_WithDynamicEnvVars tests that environment variables set via env.VAR
// in gsh scripts are available when executing commands via exec()
func TestExec_WithDynamicEnvVars(t *testing.T) {
	// Clean up any existing test env var
	os.Unsetenv("GSH_TEST_VAR")
	defer os.Unsetenv("GSH_TEST_VAR")

	script := `
env.GSH_TEST_VAR = "test_value"
result = exec("echo $GSH_TEST_VAR")
`

	interp := New()
	defer interp.Close()

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	evalResult, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	// Get the result variable
	resultVar, ok := evalResult.Variables()["result"]
	if !ok {
		t.Fatal("result variable not found")
	}

	// Should be an object with stdout, stderr, exitCode
	objVal, ok := resultVar.(*ObjectValue)
	if !ok {
		t.Fatalf("result should be an object, got %s", resultVar.Type())
	}

	// Check stdout
	stdoutVal, ok := objVal.Properties["stdout"]
	if !ok {
		t.Fatal("stdout property not found")
	}

	stdoutStr, ok := stdoutVal.(*StringValue)
	if !ok {
		t.Fatalf("stdout should be a string, got %s", stdoutVal.Type())
	}

	// The output should contain our environment variable value
	stdout := strings.TrimSpace(stdoutStr.Value)
	if stdout != "test_value" {
		t.Errorf("expected stdout to be 'test_value', got '%s'", stdout)
	}

	// Verify the env var was set in OS environment
	if os.Getenv("GSH_TEST_VAR") != "test_value" {
		t.Error("environment variable was not set in OS environment")
	}
}

// TestExec_WithMultipleEnvVars tests that multiple environment variables
// set in the same script are all available in exec()
func TestExec_WithMultipleEnvVars(t *testing.T) {
	// Clean up
	os.Unsetenv("GSH_VAR1")
	os.Unsetenv("GSH_VAR2")
	defer func() {
		os.Unsetenv("GSH_VAR1")
		os.Unsetenv("GSH_VAR2")
	}()

	script := `
env.GSH_VAR1 = "value1"
env.GSH_VAR2 = "value2"
result = exec("echo $GSH_VAR1-$GSH_VAR2")
`

	interp := New()
	defer interp.Close()

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	evalResult, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	resultVar := evalResult.Variables()["result"]
	objVal := resultVar.(*ObjectValue)
	stdoutStr := objVal.Properties["stdout"].(*StringValue)
	stdout := strings.TrimSpace(stdoutStr.Value)

	if stdout != "value1-value2" {
		t.Errorf("expected stdout to be 'value1-value2', got '%s'", stdout)
	}
}
