package interpreter

import (
	"context"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"mvdan.cc/sh/v3/syntax"
)

// TestExec_WithDynamicEnvVars tests that environment variables set via env.VAR
// in gsh scripts are available when executing commands via exec()
func TestExec_WithDynamicEnvVars(t *testing.T) {
	script := `
env.GSH_TEST_VAR = "test_value"
result = exec("echo $GSH_TEST_VAR")
`

	interp := New(nil)
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
}

// TestExec_WithMultipleEnvVars tests that multiple environment variables
// set in the same script are all available in exec()
func TestExec_WithMultipleEnvVars(t *testing.T) {
	script := `
env.GSH_VAR1 = "value1"
env.GSH_VAR2 = "value2"
result = exec("echo $GSH_VAR1-$GSH_VAR2")
`

	interp := New(nil)
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

// TestExec_UnsetEnvVar tests that unsetting an environment variable via env.VAR = null
// properly removes it from exec() subshells
func TestExec_UnsetEnvVar(t *testing.T) {
	script := `
env.GSH_UNSET_TEST = "initial_value"
result1 = exec("echo $GSH_UNSET_TEST")
env.GSH_UNSET_TEST = null
result2 = exec("echo $GSH_UNSET_TEST")
`

	interp := New(nil)
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

	// Check result1 - should have the value
	result1Var := evalResult.Variables()["result1"]
	obj1 := result1Var.(*ObjectValue)
	stdout1 := strings.TrimSpace(obj1.Properties["stdout"].(*StringValue).Value)
	if stdout1 != "initial_value" {
		t.Errorf("result1: expected stdout to be 'initial_value', got '%s'", stdout1)
	}

	// Check result2 - should be empty after unset
	result2Var := evalResult.Variables()["result2"]
	obj2 := result2Var.(*ObjectValue)
	stdout2 := strings.TrimSpace(obj2.Properties["stdout"].(*StringValue).Value)
	if stdout2 != "" {
		t.Errorf("result2: expected stdout to be empty after unset, got '%s'", stdout2)
	}
}

// TestExec_WorkingDirectory tests that exec() uses the interpreter's working directory
// and that changes made via bash commands are reflected
func TestExec_WorkingDirectory(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	// Get initial working directory
	result1, err := interp.EvalString(`exec("pwd")`)
	if err != nil {
		t.Fatalf("initial pwd failed: %v", err)
	}
	obj1 := result1.FinalResult.(*ObjectValue)
	initialDir := strings.TrimSpace(obj1.Properties["stdout"].(*StringValue).Value)

	// Change to /tmp using a bash command through the runner
	runner := interp.Runner()
	mu := interp.RunnerMutex()
	mu.Lock()
	prog, _ := syntax.NewParser().Parse(strings.NewReader("cd /tmp"), "")
	runner.Run(context.Background(), prog)
	mu.Unlock()

	// Now exec("pwd") should return /tmp
	result2, err := interp.EvalString(`exec("pwd")`)
	if err != nil {
		t.Fatalf("pwd after cd failed: %v", err)
	}
	obj2 := result2.FinalResult.(*ObjectValue)
	newDir := strings.TrimSpace(obj2.Properties["stdout"].(*StringValue).Value)

	if newDir != "/tmp" {
		t.Errorf("expected working directory to be '/tmp', got '%s'", newDir)
	}

	// Verify it changed from the initial directory
	if newDir == initialDir {
		t.Errorf("working directory didn't change: still '%s'", initialDir)
	}
}

// TestExec_EnvVarInheritedBySubshell tests that env vars set via env.VAR
// are properly inherited by exec() subshells (the core fix for starship integration)
func TestExec_EnvVarInheritedBySubshell(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	// Set env var using the interpreter's SetEnv method (simulating env.VAR = "value")
	interp.SetEnv("GSH_SUBSHELL_TEST", "subshell_value")

	// The env var should be visible in exec()
	result, err := interp.EvalString(`exec("echo $GSH_SUBSHELL_TEST")`)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	obj := result.FinalResult.(*ObjectValue)
	stdout := strings.TrimSpace(obj.Properties["stdout"].(*StringValue).Value)

	if stdout != "subshell_value" {
		t.Errorf("expected subshell to see env var 'subshell_value', got '%s'", stdout)
	}
}

// TestExec_PWDEnvVarMatchesWorkingDir tests that the PWD environment variable
// matches the actual working directory (important for tools like starship)
func TestExec_PWDEnvVarMatchesWorkingDir(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	// Change to /tmp
	runner := interp.Runner()
	mu := interp.RunnerMutex()
	mu.Lock()
	prog, _ := syntax.NewParser().Parse(strings.NewReader("cd /tmp"), "")
	runner.Run(context.Background(), prog)
	mu.Unlock()

	// Check both pwd command output and $PWD env var
	result, err := interp.EvalString(`exec("echo pwd=$(pwd) PWD=$PWD")`)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	obj := result.FinalResult.(*ObjectValue)
	stdout := strings.TrimSpace(obj.Properties["stdout"].(*StringValue).Value)

	// Both should be /tmp
	expected := "pwd=/tmp PWD=/tmp"
	if stdout != expected {
		t.Errorf("expected '%s', got '%s'", expected, stdout)
	}
}
