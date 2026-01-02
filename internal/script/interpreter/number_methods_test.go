package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestNumberToFixed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "toFixed with 0 decimal places",
			input:    "x = 3.14159\nresult = x.toFixed(0)",
			expected: "3",
		},
		{
			name:     "toFixed with 1 decimal place",
			input:    "x = 3.14159\nresult = x.toFixed(1)",
			expected: "3.1",
		},
		{
			name:     "toFixed with 2 decimal places",
			input:    "x = 3.14159\nresult = x.toFixed(2)",
			expected: "3.14",
		},
		{
			name:     "toFixed with 3 decimal places",
			input:    "x = 3.14159\nresult = x.toFixed(3)",
			expected: "3.142",
		},
		{
			name:     "toFixed on integer",
			input:    "x = 42\nresult = x.toFixed(2)",
			expected: "42.00",
		},
		{
			name:     "toFixed default (no argument)",
			input:    "x = 3.7\nresult = x.toFixed()",
			expected: "4",
		},
		{
			name:     "toFixed on negative number",
			input:    "x = -2.567\nresult = x.toFixed(1)",
			expected: "-2.6",
		},
		{
			name:     "toFixed on expression result",
			input:    "result = (10 / 3).toFixed(2)",
			expected: "3.33",
		},
		{
			name:     "toFixed used in string concatenation",
			input:    "x = 1.5\nresult = \"value: \" + x.toFixed(1) + \"s\"",
			expected: "value: 1.5s",
		},
		{
			name:     "toFixed with percentage calculation",
			input:    "ratio = 80 / 1000 * 100\nresult = ratio.toFixed(0) + \"%\"",
			expected: "8%",
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
				t.Fatalf("unexpected error: %v", err)
			}

			result, ok := interp.env.Get("result")
			if !ok {
				t.Fatal("result variable not found")
			}

			if result.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.String())
			}
		})
	}
}

func TestNumberToFixedErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "toFixed with negative decimals",
			input:       "x = 3.14\nresult = x.toFixed(-1)",
			expectedErr: "toFixed() argument must be non-negative",
		},
		{
			name:        "toFixed with non-number argument",
			input:       "x = 3.14\nresult = x.toFixed(\"2\")",
			expectedErr: "toFixed() argument must be a number",
		},
		{
			name:        "toFixed with too large decimals",
			input:       "x = 3.14\nresult = x.toFixed(101)",
			expectedErr: "toFixed() argument must be at most 100",
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
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if err.Error() != tt.expectedErr {
				t.Errorf("expected error %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}
