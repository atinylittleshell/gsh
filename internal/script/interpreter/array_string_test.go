package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestArrayIndexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic array indexing",
			input:    "arr = [1, 2, 3]\nresult = arr[1]",
			expected: "2",
		},
		{
			name:     "array indexing with string elements",
			input:    "arr = [\"a\", \"b\", \"c\"]\nresult = arr[0]",
			expected: "a",
		},
		{
			name:     "array indexing at end",
			input:    "arr = [10, 20, 30]\nresult = arr[2]",
			expected: "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("interpreter error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestArrayIndexAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "assign to array index",
			input:    "arr = [1, 2, 3]\narr[1] = 99\nresult = arr[1]",
			expected: "99",
		},
		{
			name:     "assign string to array index",
			input:    "arr = [\"a\", \"b\", \"c\"]\narr[0] = \"z\"\nresult = arr[0]",
			expected: "z",
		},
		{
			name:     "assign to last index",
			input:    "arr = [10, 20, 30]\narr[2] = 100\nresult = arr[2]",
			expected: "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("interpreter error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestArrayLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty array length",
			input:    "arr = []\nresult = arr.length",
			expected: "0",
		},
		{
			name:     "array with 3 elements",
			input:    "arr = [1, 2, 3]\nresult = arr.length",
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("interpreter error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestArrayPushPop(t *testing.T) {
	input := "arr = [1, 2, 3]\narr.push(4)\nlast = arr.pop()\nresult = arr.length"

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("interpreter error: %v", err)
	}

	result, ok := interp.env.Get("result")
	if !ok {
		t.Fatalf("failed to get result")
	}

	if result.String() != "3" {
		t.Errorf("expected 3, got %s", result.String())
	}

	last, ok := interp.env.Get("last")
	if !ok {
		t.Fatalf("failed to get last")
	}

	if last.String() != "4" {
		t.Errorf("expected last to be 4, got %s", last.String())
	}
}

func TestArrayJoin(t *testing.T) {
	input := "arr = [1, 2, 3]\nresult = arr.join(\"-\")"

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("interpreter error: %v", err)
	}

	result, ok := interp.env.Get("result")
	if !ok {
		t.Fatalf("failed to get result")
	}

	if result.String() != "1-2-3" {
		t.Errorf("expected '1-2-3', got %s", result.String())
	}
}

func TestArraySlice(t *testing.T) {
	input := "arr = [1, 2, 3, 4, 5]\nsliced = arr.slice(1, 3)\nresult = sliced[0]"

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("interpreter error: %v", err)
	}

	result, ok := interp.env.Get("result")
	if !ok {
		t.Fatalf("failed to get result")
	}

	if result.String() != "2" {
		t.Errorf("expected 2, got %s", result.String())
	}
}

func TestStringLength(t *testing.T) {
	input := "str = \"hello\"\nresult = str.length"

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("interpreter error: %v", err)
	}

	result, ok := interp.env.Get("result")
	if !ok {
		t.Fatalf("failed to get result")
	}

	if result.String() != "5" {
		t.Errorf("expected 5, got %s", result.String())
	}
}

func TestStringMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "toUpperCase",
			input:    "str = \"hello\"\nresult = str.toUpperCase()",
			expected: "HELLO",
		},
		{
			name:     "toLowerCase",
			input:    "str = \"HELLO\"\nresult = str.toLowerCase()",
			expected: "hello",
		},
		{
			name:     "trim",
			input:    "str = \"  hello  \"\nresult = str.trim()",
			expected: "hello",
		},
		{
			name:     "trimStart",
			input:    "str = \"  hello  \"\nresult = str.trimStart()",
			expected: "hello  ",
		},
		{
			name:     "trimEnd",
			input:    "str = \"  hello  \"\nresult = str.trimEnd()",
			expected: "  hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("interpreter error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestObjectIndexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "object string index",
			input:    "obj = {name: \"Alice\", age: 30}\nresult = obj[\"name\"]",
			expected: "Alice",
		},
		{
			name:     "object index assignment",
			input:    "obj = {x: 10}\nobj[\"x\"] = 20\nresult = obj[\"x\"]",
			expected: "20",
		},
		{
			name:     "object add new property via index",
			input:    "obj = {}\nobj[\"key\"] = \"value\"\nresult = obj[\"key\"]",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("interpreter error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}
