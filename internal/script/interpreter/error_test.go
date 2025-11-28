package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestRuntimeError_Error(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		stackTrace []StackFrame
		want       string
	}{
		{
			name:       "error without stack trace",
			message:    "division by zero",
			stackTrace: []StackFrame{},
			want:       "division by zero",
		},
		{
			name:    "error with single stack frame",
			message: "undefined variable: x",
			stackTrace: []StackFrame{
				{FunctionName: "main", Location: "line 5"},
			},
			want: "undefined variable: x\n\nStack trace:\n  at main (line 5)",
		},
		{
			name:    "error with multiple stack frames",
			message: "array index out of bounds",
			stackTrace: []StackFrame{
				{FunctionName: "helper", Location: "tool 'helper'"},
				{FunctionName: "process", Location: "tool 'process'"},
				{FunctionName: "main", Location: "line 10"},
			},
			want: "array index out of bounds\n\nStack trace:\n  at main (line 10)\n  at process (tool 'process')\n  at helper (tool 'helper')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RuntimeError{
				Message:    tt.message,
				StackTrace: tt.stackTrace,
			}
			got := err.Error()
			if got != tt.want {
				t.Errorf("RuntimeError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStackTrace_SimpleError(t *testing.T) {
	input := `
x = 10
y = 0
result = x / y
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "division by zero") {
		t.Errorf("expected error message to contain 'division by zero', got %q", rte.Message)
	}
}

func TestStackTrace_NestedToolCalls(t *testing.T) {
	input := `
tool inner() {
    x = nonexistent
    return x
}

tool outer() {
    return inner()
}

result = outer()
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "undefined variable: nonexistent") {
		t.Errorf("expected error message to contain 'undefined variable: nonexistent', got %q", rte.Message)
	}

	// Check that stack trace contains both tool names
	stackStr := rte.Error()
	if !strings.Contains(stackStr, "inner") {
		t.Errorf("expected stack trace to contain 'inner', got %q", stackStr)
	}
	if !strings.Contains(stackStr, "outer") {
		t.Errorf("expected stack trace to contain 'outer', got %q", stackStr)
	}
}

func TestStackTrace_DeeplyNestedCalls(t *testing.T) {
	input := `
tool level3() {
    arr = [1, 2, 3]
    return arr[10]
}

tool level2() {
    return level3()
}

tool level1() {
    return level2()
}

result = level1()
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "array index out of bounds") {
		t.Errorf("expected error message to contain 'array index out of bounds', got %q", rte.Message)
	}

	// Check that stack trace contains all three tool names
	stackStr := rte.Error()
	if !strings.Contains(stackStr, "level1") {
		t.Errorf("expected stack trace to contain 'level1', got %q", stackStr)
	}
	if !strings.Contains(stackStr, "level2") {
		t.Errorf("expected stack trace to contain 'level2', got %q", stackStr)
	}
	if !strings.Contains(stackStr, "level3") {
		t.Errorf("expected stack trace to contain 'level3', got %q", stackStr)
	}

	// Verify stack trace has the correct order (deepest first)
	level3Pos := strings.Index(stackStr, "level3")
	level2Pos := strings.Index(stackStr, "level2")
	level1Pos := strings.Index(stackStr, "level1")

	if level1Pos >= level2Pos || level2Pos >= level3Pos {
		t.Errorf("stack trace order is incorrect, expected level1 < level2 < level3, got:\n%s", stackStr)
	}
}

func TestStackTrace_ErrorInLoop(t *testing.T) {
	input := `
tool process(n: number) {
    if (n == 3) {
        x = undefined_var
    }
    return n * 2
}

for (i of [1, 2, 3, 4]) {
    result = process(i)
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "undefined variable: undefined_var") {
		t.Errorf("expected error message to contain 'undefined variable: undefined_var', got %q", rte.Message)
	}

	// Check that stack trace contains the tool name
	stackStr := rte.Error()
	if !strings.Contains(stackStr, "process") {
		t.Errorf("expected stack trace to contain 'process', got %q", stackStr)
	}
}

func TestStackTrace_TryCatchDoesNotShowInternalStack(t *testing.T) {
	input := `
tool risky() {
    return nonexistent
}

try {
    result = risky()
} catch (e) {
    print("Caught error: " + e.message)
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	// Should not error since it's caught
	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStackTrace_RecursiveFunction(t *testing.T) {
	input := `
tool factorial(n: number): number {
    if (n == 5) {
        return bad_variable
    }
    if (n <= 1) {
        return 1
    }
    return n * factorial(n - 1)
}

result = factorial(10)
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "undefined variable: bad_variable") {
		t.Errorf("expected error message to contain 'undefined variable: bad_variable', got %q", rte.Message)
	}

	// Check that stack trace contains multiple factorial calls
	stackStr := rte.Error()
	factorialCount := strings.Count(stackStr, "factorial")
	if factorialCount < 2 {
		t.Errorf("expected multiple 'factorial' entries in stack trace, got %d:\n%s", factorialCount, stackStr)
	}
}

func TestStackTrace_BinaryOperationError(t *testing.T) {
	input := `
tool calculate(a: number, b: number) {
    return a / b
}

x = 5
y = 0
result = calculate(x, y)
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "division by zero") {
		t.Errorf("expected error message to contain 'division by zero', got %q", rte.Message)
	}

	// Check that stack trace contains the calculate tool
	stackStr := rte.Error()
	if !strings.Contains(stackStr, "calculate") {
		t.Errorf("expected stack trace to contain 'calculate', got %q", stackStr)
	}
}

func TestStackTrace_TypeMismatchError(t *testing.T) {
	input := `
tool processArray(arr: array) {
    return arr[0]
}

result = processArray("not an array")
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	if !strings.Contains(rte.Message, "expects type") {
		t.Errorf("expected error message to contain 'expects type', got %q", rte.Message)
	}
}

func TestStackTrace_ChainedToolCalls(t *testing.T) {
	input := `
tool a() {
    return b()
}

tool b() {
    return c()
}

tool c() {
    return d()
}

tool d() {
    obj = {}
    return obj.nonexistent.property
}

result = a()
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	rte, ok := err.(*RuntimeError)
	if !ok {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}

	// Check that stack trace contains all tool names
	stackStr := rte.Error()
	for _, toolName := range []string{"a", "b", "c", "d"} {
		if !strings.Contains(stackStr, toolName) {
			t.Errorf("expected stack trace to contain '%s', got %q", toolName, stackStr)
		}
	}
}
