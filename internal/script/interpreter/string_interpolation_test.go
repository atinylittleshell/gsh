package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestStringInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			name: "basic variable interpolation",
			input: `name = "World"
msg = ` + "`Hello ${name}!`" + `
msg`,
			expected: "Hello World!",
		},
		{
			name: "multiple interpolations in one string",
			input: `first = "John"
last = "Doe"
msg = ` + "`${first} ${last}`" + `
msg`,
			expected: "John Doe",
		},
		{
			name: "interpolation with no spaces",
			input: `x = "test"
msg = ` + "`prefix${x}suffix`" + `
msg`,
			expected: "prefixtestsuffix",
		},
		{
			name: "empty interpolation expression",
			input: `empty = ""
msg = ` + "`Value: ${empty}`" + `
msg`,
			expected: "Value: ",
		},

		// Arithmetic expressions
		{
			name: "simple arithmetic",
			input: `x = 10
y = 5
result = ` + "`${x + y}`" + `
result`,
			expected: "15",
		},
		{
			name: "complex arithmetic",
			input: `a = 3
b = 4
result = ` + "`${a * a + b * b}`" + `
result`,
			expected: "25",
		},
		{
			name: "arithmetic with parentheses",
			input: `x = 2
y = 3
z = 4
result = ` + "`${(x + y) * z}`" + `
result`,
			expected: "20",
		},
		{
			name: "division and modulo",
			input: `a = 17
b = 5
result = ` + "`${a / b} remainder ${a % b}`" + `
result`,
			expected: "3.4 remainder 2",
		},

		// Member access
		{
			name: "object property access",
			input: `obj = {value: 42}
str = ` + "`Answer: ${obj.value}`" + `
str`,
			expected: "Answer: 42",
		},
		{
			name: "nested object access",
			input: `obj = {nested: {value: "deep"}}
str = ` + "`Value: ${obj.nested.value}`" + `
str`,
			expected: "Value: deep",
		},
		{
			name: "array length property",
			input: `arr = [1, 2, 3, 4, 5]
str = ` + "`Length: ${arr.length}`" + `
str`,
			expected: "Length: 5",
		},
		{
			name: "string length property",
			input: `text = "hello"
str = ` + "`Length: ${text.length}`" + `
str`,
			expected: "Length: 5",
		},

		// Array indexing
		{
			name: "array indexing",
			input: `arr = [1, 2, 3]
output = ` + "`First: ${arr[0]}`" + `
output`,
			expected: "First: 1",
		},
		{
			name: "array indexing with variable",
			input: `arr = ["a", "b", "c"]
idx = 1
output = ` + "`Item: ${arr[idx]}`" + `
output`,
			expected: "Item: b",
		},
		{
			name: "array indexing with expression",
			input: `arr = [10, 20, 30, 40]
i = 1
output = ` + "`Value: ${arr[i + 1]}`" + `
output`,
			expected: "Value: 30",
		},

		// Object indexing
		{
			name: "object string index",
			input: `obj = {name: "Alice"}
key = "name"
output = ` + "`Name: ${obj[key]}`" + `
output`,
			expected: "Name: Alice",
		},

		// Logical expressions
		{
			name: "boolean in interpolation",
			input: `x = 5
y = 10
result = ` + "`Is ${x} less than ${y}? ${x < y}`" + `
result`,
			expected: "Is 5 less than 10? true",
		},
		{
			name: "logical operators",
			input: `a = true
b = false
result = ` + "`${a && b} ${a || b}`" + `
result`,
			expected: "false true",
		},
		{
			name: "comparison operators",
			input: `x = 10
result = ` + "`${x > 5} ${x == 10} ${x != 20}`" + `
result`,
			expected: "true true true",
		},

		// Unary expressions
		{
			name: "unary minus",
			input: `x = 5
result = ` + "`${-x}`" + `
result`,
			expected: "-5",
		},
		{
			name: "unary not",
			input: `flag = true
result = ` + "`${!flag}`" + `
result`,
			expected: "false",
		},

		// String concatenation
		{
			name: "string concatenation in interpolation",
			input: `first = "Hello"
second = "World"
result = ` + "`${first + \" \" + second}`" + `
result`,
			expected: "Hello World",
		},

		// Method calls
		{
			name: "string method call",
			input: `text = "hello"
result = ` + "`${text.toUpperCase()}`" + `
result`,
			expected: "HELLO",
		},
		{
			name: "array method call",
			input: `arr = ["a", "b", "c"]
result = ` + "`${arr.join(\"-\")}`" + `
result`,
			expected: "a-b-c",
		},

		// Function/tool calls
		{
			name: "tool call in interpolation",
			input: `tool greet(name: string): string {
	return "Hello, " + name
}
result = ` + "`${greet(\"Alice\")}`" + `
result`,
			expected: "Hello, Alice",
		},
		{
			name: "nested tool calls",
			input: `tool double(x: number): number {
	return x * 2
}
tool add(a: number, b: number): number {
	return a + b
}
result = ` + "`${double(add(3, 4))}`" + `
result`,
			expected: "14",
		},

		// Complex expressions
		{
			name: "complex nested expression",
			input: `x = 10
y = 20
arr = [1, 2, 3]
obj = {value: 5}
result = ` + "`${(x + y) * arr[0] + obj.value}`" + `
result`,
			expected: "35",
		},

		// Special types
		{
			name: "null value",
			input: `x = null
result = ` + "`Value: ${x}`" + `
result`,
			expected: "Value: null",
		},
		{
			name: "missing property returns null",
			input: `obj = {name: "test"}
result = ` + "`Value: ${obj.unknown}`" + `
result`,
			expected: "Value: null",
		},
		{
			name: "number types",
			input: `int = 42
float = 3.14
result = ` + "`${int} and ${float}`" + `
result`,
			expected: "42 and 3.14",
		},

		// Edge cases
		{
			name: "interpolation at start",
			input: `name = "World"
result = ` + "`${name} says hello`" + `
result`,
			expected: "World says hello",
		},
		{
			name: "interpolation at end",
			input: `name = "World"
result = ` + "`Hello ${name}`" + `
result`,
			expected: "Hello World",
		},
		{
			name: "only interpolation",
			input: `value = 42
result = ` + "`${value}`" + `
result`,
			expected: "42",
		},
		{
			name: "no interpolation",
			input: `result = ` + "`just plain text`" + `
result`,
			expected: "just plain text",
		},
		{
			name: "escaped dollar sign",
			input: `result = ` + "`Price: \\$100`" + `
result`,
			expected: "Price: $100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New()
			result, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if result.FinalResult.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.FinalResult.String())
			}
		})
	}
}

func TestStringInterpolationErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "undefined variable in interpolation",
			input: `result = ` + "`${undefinedVar}`" + `
result`,
			expectError: true,
			errorMsg:    "undefined variable",
		},
		{
			name: "unclosed interpolation",
			input: `x = 5
result = ` + "`Value: ${x`" + `
result`,
			expectError: true,
			errorMsg:    "unclosed template literal interpolation",
		},
		{
			name: "invalid expression in interpolation",
			input: `result = ` + "`${5 +}`" + `
result`,
			expectError: true,
		},
		{
			name: "array index out of bounds",
			input: `arr = [1, 2, 3]
result = ` + "`${arr[10]}`" + `
result`,
			expectError: true,
			errorMsg:    "out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			_, err := interp.Eval(program)

			if tt.expectError && err == nil {
				t.Fatalf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectError && tt.errorMsg != "" && err != nil {
				if !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestStringInterpolationWithBuiltins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "interpolation with JSON.stringify",
			input: `obj = {name: "Alice", age: 30}
result = ` + "`JSON: ${JSON.stringify(obj)}`" + `
result`,
			expected: `JSON: {"age":30,"name":"Alice"}`,
		},
		{
			name: "interpolation with array methods",
			input: `arr = [1, 2, 3]
result = ` + "`Reversed: ${arr.reverse().join(\",\")}`" + `
result`,
			expected: "Reversed: 3,2,1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New()
			result, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if result.FinalResult.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.FinalResult.String())
			}
		})
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
