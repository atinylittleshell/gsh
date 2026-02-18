package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestStringIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "indexOf - found at beginning",
			input:    "str = \"hello world\"\nresult = str.indexOf(\"hello\")",
			expected: "0",
		},
		{
			name:     "indexOf - found in middle",
			input:    "str = \"hello world\"\nresult = str.indexOf(\"world\")",
			expected: "6",
		},
		{
			name:     "indexOf - not found",
			input:    "str = \"hello world\"\nresult = str.indexOf(\"foo\")",
			expected: "-1",
		},
		{
			name:     "indexOf - with start index",
			input:    "str = \"hello hello\"\nresult = str.indexOf(\"hello\", 1)",
			expected: "6",
		},
		{
			name:     "indexOf - single character",
			input:    "str = \"abcdef\"\nresult = str.indexOf(\"c\")",
			expected: "2",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringLastIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lastIndexOf - last occurrence",
			input:    "str = \"hello hello\"\nresult = str.lastIndexOf(\"hello\")",
			expected: "6",
		},
		{
			name:     "lastIndexOf - single occurrence",
			input:    "str = \"hello world\"\nresult = str.lastIndexOf(\"world\")",
			expected: "6",
		},
		{
			name:     "lastIndexOf - not found",
			input:    "str = \"hello world\"\nresult = str.lastIndexOf(\"foo\")",
			expected: "-1",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringSubstring(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "substring - with start and end",
			input:    "str = \"hello world\"\nresult = str.substring(0, 5)",
			expected: "hello",
		},
		{
			name:     "substring - with only start",
			input:    "str = \"hello world\"\nresult = str.substring(6)",
			expected: "world",
		},
		{
			name:     "substring - middle portion",
			input:    "str = \"hello world\"\nresult = str.substring(3, 8)",
			expected: "lo wo",
		},
		{
			name:     "substring - swapped indices (should swap them)",
			input:    "str = \"hello\"\nresult = str.substring(3, 1)",
			expected: "el",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "slice - with start and end",
			input:    "str = \"hello world\"\nresult = str.slice(0, 5)",
			expected: "hello",
		},
		{
			name:     "slice - with only start",
			input:    "str = \"hello world\"\nresult = str.slice(6)",
			expected: "world",
		},
		{
			name:     "slice - negative start index",
			input:    "str = \"hello world\"\nresult = str.slice(-5)",
			expected: "world",
		},
		{
			name:     "slice - negative end index",
			input:    "str = \"hello world\"\nresult = str.slice(0, -6)",
			expected: "hello",
		},
		{
			name:     "slice - both negative indices",
			input:    "str = \"hello world\"\nresult = str.slice(-5, -1)",
			expected: "worl",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringStartsWith(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "startsWith - true",
			input:    "str = \"hello world\"\nresult = str.startsWith(\"hello\")",
			expected: "true",
		},
		{
			name:     "startsWith - false",
			input:    "str = \"hello world\"\nresult = str.startsWith(\"world\")",
			expected: "false",
		},
		{
			name:     "startsWith - empty string",
			input:    "str = \"hello\"\nresult = str.startsWith(\"\")",
			expected: "true",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringEndsWith(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "endsWith - true",
			input:    "str = \"hello world\"\nresult = str.endsWith(\"world\")",
			expected: "true",
		},
		{
			name:     "endsWith - false",
			input:    "str = \"hello world\"\nresult = str.endsWith(\"hello\")",
			expected: "false",
		},
		{
			name:     "endsWith - empty string",
			input:    "str = \"hello\"\nresult = str.endsWith(\"\")",
			expected: "true",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringIncludes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "includes - true",
			input:    "str = \"hello world\"\nresult = str.includes(\"lo wo\")",
			expected: "true",
		},
		{
			name:     "includes - false",
			input:    "str = \"hello world\"\nresult = str.includes(\"foo\")",
			expected: "false",
		},
		{
			name:     "includes - at beginning",
			input:    "str = \"hello world\"\nresult = str.includes(\"hello\")",
			expected: "true",
		},
		{
			name:     "includes - at end",
			input:    "str = \"hello world\"\nresult = str.includes(\"world\")",
			expected: "true",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringReplace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replace - first occurrence only",
			input:    "str = \"hello hello\"\nresult = str.replace(\"hello\", \"hi\")",
			expected: "hi hello",
		},
		{
			name:     "replace - single occurrence",
			input:    "str = \"hello world\"\nresult = str.replace(\"world\", \"universe\")",
			expected: "hello universe",
		},
		{
			name:     "replace - not found",
			input:    "str = \"hello world\"\nresult = str.replace(\"foo\", \"bar\")",
			expected: "hello world",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringReplaceAll(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaceAll - multiple occurrences",
			input:    "str = \"hello hello\"\nresult = str.replaceAll(\"hello\", \"hi\")",
			expected: "hi hi",
		},
		{
			name:     "replaceAll - single occurrence",
			input:    "str = \"hello world\"\nresult = str.replaceAll(\"world\", \"universe\")",
			expected: "hello universe",
		},
		{
			name:     "replaceAll - not found",
			input:    "str = \"hello world\"\nresult = str.replaceAll(\"foo\", \"bar\")",
			expected: "hello world",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringRepeat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "repeat - multiple times",
			input:    "str = \"ha\"\nresult = str.repeat(3)",
			expected: "hahaha",
		},
		{
			name:     "repeat - once",
			input:    "str = \"hello\"\nresult = str.repeat(1)",
			expected: "hello",
		},
		{
			name:     "repeat - zero times",
			input:    "str = \"hello\"\nresult = str.repeat(0)",
			expected: "",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringPadStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "padStart - with spaces",
			input:    "str = \"hello\"\nresult = str.padStart(10)",
			expected: "     hello",
		},
		{
			name:     "padStart - with custom string",
			input:    "str = \"5\"\nresult = str.padStart(3, \"0\")",
			expected: "005",
		},
		{
			name:     "padStart - already long enough",
			input:    "str = \"hello\"\nresult = str.padStart(3)",
			expected: "hello",
		},
		{
			name:     "padStart - with repeating pattern",
			input:    "str = \"x\"\nresult = str.padStart(5, \"ab\")",
			expected: "ababx",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringPadEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "padEnd - with spaces",
			input:    "str = \"hello\"\nresult = str.padEnd(10)",
			expected: "hello     ",
		},
		{
			name:     "padEnd - with custom string",
			input:    "str = \"5\"\nresult = str.padEnd(3, \"0\")",
			expected: "500",
		},
		{
			name:     "padEnd - already long enough",
			input:    "str = \"hello\"\nresult = str.padEnd(3)",
			expected: "hello",
		},
		{
			name:     "padEnd - with repeating pattern",
			input:    "str = \"x\"\nresult = str.padEnd(5, \"ab\")",
			expected: "xabab",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringCharAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "charAt - first character",
			input:    "str = \"hello\"\nresult = str.charAt(0)",
			expected: "h",
		},
		{
			name:     "charAt - middle character",
			input:    "str = \"hello\"\nresult = str.charAt(2)",
			expected: "l",
		},
		{
			name:     "charAt - last character",
			input:    "str = \"hello\"\nresult = str.charAt(4)",
			expected: "o",
		},
		{
			name:     "charAt - out of bounds",
			input:    "str = \"hello\"\nresult = str.charAt(10)",
			expected: "",
		},
		{
			name:     "charAt - negative index",
			input:    "str = \"hello\"\nresult = str.charAt(-1)",
			expected: "",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "split - by space",
			input:    "str = \"hello world\"\nparts = str.split(\" \")\nresult = parts.length",
			expected: "2",
		},
		{
			name:     "split - by comma",
			input:    "str = \"a,b,c\"\nparts = str.split(\",\")\nresult = parts[1]",
			expected: "b",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringUnicodeSupport(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unicode - length with emoji",
			input:    "str = \"hello 👋\"\nresult = str.length",
			expected: "7",
		},
		{
			name:     "unicode - charAt with emoji",
			input:    "str = \"👋 hello\"\nresult = str.charAt(0)",
			expected: "👋",
		},
		{
			name:     "unicode - substring with emoji",
			input:    "str = \"👋 hello\"\nresult = str.substring(2, 7)",
			expected: "hello",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestStringComparison(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Less than
		{
			name:     "less than - a < b",
			input:    "result = \"a\" < \"b\"",
			expected: "true",
		},
		{
			name:     "less than - b < a",
			input:    "result = \"b\" < \"a\"",
			expected: "false",
		},
		{
			name:     "less than - equal strings",
			input:    "result = \"a\" < \"a\"",
			expected: "false",
		},
		// Less than or equal
		{
			name:     "less than or equal - a <= b",
			input:    "result = \"a\" <= \"b\"",
			expected: "true",
		},
		{
			name:     "less than or equal - b <= a",
			input:    "result = \"b\" <= \"a\"",
			expected: "false",
		},
		{
			name:     "less than or equal - equal strings",
			input:    "result = \"a\" <= \"a\"",
			expected: "true",
		},
		// Greater than
		{
			name:     "greater than - b > a",
			input:    "result = \"b\" > \"a\"",
			expected: "true",
		},
		{
			name:     "greater than - a > b",
			input:    "result = \"a\" > \"b\"",
			expected: "false",
		},
		{
			name:     "greater than - equal strings",
			input:    "result = \"a\" > \"a\"",
			expected: "false",
		},
		// Greater than or equal
		{
			name:     "greater than or equal - b >= a",
			input:    "result = \"b\" >= \"a\"",
			expected: "true",
		},
		{
			name:     "greater than or equal - a >= b",
			input:    "result = \"a\" >= \"b\"",
			expected: "false",
		},
		{
			name:     "greater than or equal - equal strings",
			input:    "result = \"a\" >= \"a\"",
			expected: "true",
		},
		// Digit range check (common use case)
		{
			name:     "digit check - 5 is digit",
			input:    "c = \"5\"\nresult = c >= \"0\" && c <= \"9\"",
			expected: "true",
		},
		{
			name:     "digit check - a is not digit",
			input:    "c = \"a\"\nresult = c >= \"0\" && c <= \"9\"",
			expected: "false",
		},
		// Lexicographic ordering for multi-character strings
		{
			name:     "lexicographic - apple < banana",
			input:    "result = \"apple\" < \"banana\"",
			expected: "true",
		},
		{
			name:     "lexicographic - abc < abd",
			input:    "result = \"abc\" < \"abd\"",
			expected: "true",
		},
		{
			name:     "lexicographic - prefix - ab < abc",
			input:    "result = \"ab\" < \"abc\"",
			expected: "true",
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

			result, ok := interp.globalEnv.Get("result")
			if !ok {
				t.Fatalf("failed to get result")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}
