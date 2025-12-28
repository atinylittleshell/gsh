package interpreter

import (
	"testing"
)

func TestBuiltinExec_BasicExecution(t *testing.T) {
	interp := New(nil)

	// Test exec("echo 'hello world'")
	execFn := interp.env.store["exec"].(*BuiltinValue)
	result, err := execFn.Fn([]Value{
		&StringValue{Value: "echo 'hello world'"},
	})

	if err != nil {
		t.Fatalf("exec() failed: %v", err)
	}

	// Check result is an object
	obj, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("exec() result should be an object, got %s", result.Type())
	}

	// Check stdout
	stdout, ok := obj.Properties["stdout"].(*StringValue)
	if !ok {
		t.Fatalf("stdout should be a string")
	}
	if stdout.Value != "hello world\n" {
		t.Errorf("stdout = %q, want %q", stdout.Value, "hello world\n")
	}

	// Check stderr
	stderr, ok := obj.Properties["stderr"].(*StringValue)
	if !ok {
		t.Fatalf("stderr should be a string")
	}
	if stderr.Value != "" {
		t.Errorf("stderr = %q, want empty", stderr.Value)
	}

	// Check exitCode
	exitCode, ok := obj.Properties["exitCode"].(*NumberValue)
	if !ok {
		t.Fatalf("exitCode should be a number")
	}
	if exitCode.Value != 0 {
		t.Errorf("exitCode = %v, want 0", exitCode.Value)
	}
}

func TestBuiltinExec_NonZeroExitCode(t *testing.T) {
	interp := New(nil)

	// Test exec with false command (exits with code 1)
	execFn := interp.env.store["exec"].(*BuiltinValue)
	result, err := execFn.Fn([]Value{
		&StringValue{Value: "false"},
	})

	// Non-zero exit code should not be an error
	if err != nil {
		t.Fatalf("exec() should not fail on non-zero exit code: %v", err)
	}

	// Check result
	obj, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("exec() result should be an object, got %s", result.Type())
	}

	// Check exitCode is 1
	exitCode, ok := obj.Properties["exitCode"].(*NumberValue)
	if !ok {
		t.Fatalf("exitCode should be a number")
	}
	if exitCode.Value != 1 {
		t.Errorf("exitCode = %v, want 1", exitCode.Value)
	}
}

func TestBuiltinExec_StderrCapture(t *testing.T) {
	interp := New(nil)

	// Test exec with command that writes to stderr
	execFn := interp.env.store["exec"].(*BuiltinValue)
	result, err := execFn.Fn([]Value{
		&StringValue{Value: "echo 'error message' >&2"},
	})

	if err != nil {
		t.Fatalf("exec() failed: %v", err)
	}

	// Check result
	obj, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("exec() result should be an object, got %s", result.Type())
	}

	// Check stderr contains the error message
	stderr, ok := obj.Properties["stderr"].(*StringValue)
	if !ok {
		t.Fatalf("stderr should be a string")
	}
	if stderr.Value != "error message\n" {
		t.Errorf("stderr = %q, want %q", stderr.Value, "error message\n")
	}
}

// Removed TestBuiltinExec_NoExecutorConfigured since bash runner is always initialized

func TestBuiltinExec_InvalidArguments(t *testing.T) {
	interp := New(nil)
	execFn := interp.env.store["exec"].(*BuiltinValue)

	tests := []struct {
		name string
		args []Value
		want string
	}{
		{
			name: "no arguments",
			args: []Value{},
			want: "exec() takes 1 or 2 arguments (command: string, options?: object), got 0",
		},
		{
			name: "too many arguments",
			args: []Value{
				&StringValue{Value: "echo"},
				&ObjectValue{Properties: map[string]Value{}},
				&StringValue{Value: "extra"},
			},
			want: "exec() takes 1 or 2 arguments (command: string, options?: object), got 3",
		},
		{
			name: "first arg not string",
			args: []Value{
				&NumberValue{Value: 123},
			},
			want: "exec() first argument must be a string, got number",
		},
		{
			name: "second arg not object",
			args: []Value{
				&StringValue{Value: "echo"},
				&StringValue{Value: "not an object"},
			},
			want: "exec() second argument must be an object, got string",
		},
		{
			name: "timeout not a number",
			args: []Value{
				&StringValue{Value: "echo"},
				&ObjectValue{Properties: map[string]Value{
					"timeout": &StringValue{Value: "1000"},
				}},
			},
			want: "exec() options.timeout must be a number (milliseconds), got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execFn.Fn(tt.args)
			if err == nil {
				t.Fatal("exec() should return error for invalid arguments")
			}
			if err.Error() != tt.want {
				t.Errorf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuiltinExec_WithTimeout(t *testing.T) {
	interp := New(nil)

	// Test exec with timeout option
	execFn := interp.env.store["exec"].(*BuiltinValue)
	result, err := execFn.Fn([]Value{
		&StringValue{Value: "echo 'test output'"},
		&ObjectValue{Properties: map[string]Value{
			"timeout": &NumberValue{Value: 5000}, // 5 seconds
		}},
	})

	if err != nil {
		t.Fatalf("exec() with timeout failed: %v", err)
	}

	// Check result is valid
	obj, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("exec() result should be an object")
	}

	stdout, ok := obj.Properties["stdout"].(*StringValue)
	if !ok || stdout.Value != "test output\n" {
		t.Errorf("stdout = %q, want %q", stdout.Value, "test output\n")
	}
}
