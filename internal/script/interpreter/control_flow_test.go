package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// TestIfStatement tests if/else statements
func TestIfStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple if true",
			input: `
				x = 0
				if (true) {
					x = 1
				}
			`,
			expected: "1",
		},
		{
			name: "simple if false",
			input: `
				x = 0
				if (false) {
					x = 1
				}
			`,
			expected: "0",
		},
		{
			name: "if-else true",
			input: `
				x = 0
				if (true) {
					x = 1
				} else {
					x = 2
				}
			`,
			expected: "1",
		},
		{
			name: "if-else false",
			input: `
				x = 0
				if (false) {
					x = 1
				} else {
					x = 2
				}
			`,
			expected: "2",
		},
		{
			name: "if with condition expression",
			input: `
				x = 5
				if (x > 3) {
					x = 10
				}
			`,
			expected: "10",
		},
		{
			name: "if-else-if chain",
			input: `
				x = 2
				result = 0
				if (x == 1) {
					result = 1
				} else if (x == 2) {
					result = 2
				} else if (x == 3) {
					result = 3
				} else {
					result = 4
				}
			`,
			expected: "2",
		},
		{
			name: "nested if statements",
			input: `
				x = 5
				y = 10
				result = 0
				if (x > 0) {
					if (y > 5) {
						result = 1
					}
				}
			`,
			expected: "1",
		},
		{
			name: "if with truthy/falsy values",
			input: `
				result = 0
				if (0) {
					result = 1
				}
				if (1) {
					result = 2
				}
			`,
			expected: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()

			// Get the appropriate variable based on test
			var varValue Value
			if strings.Contains(tt.input, "result =") {
				varValue = vars["result"]
			} else {
				varValue = vars["x"]
			}

			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestWhileLoop tests while loops
func TestWhileLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		expected string
	}{
		{
			name: "simple while loop",
			input: `
				x = 0
				while (x < 5) {
					x = x + 1
				}
			`,
			varName:  "x",
			expected: "5",
		},
		{
			name: "while loop with counter",
			input: `
				i = 0
				sum = 0
				while (i < 10) {
					sum = sum + i
					i = i + 1
				}
			`,
			varName:  "sum",
			expected: "45",
		},
		{
			name: "while loop false condition",
			input: `
				x = 0
				while (false) {
					x = x + 1
				}
			`,
			varName:  "x",
			expected: "0",
		},
		{
			name: "nested while loops",
			input: `
				i = 0
				j = 0
				count = 0
				while (i < 3) {
					j = 0
					while (j < 2) {
						count = count + 1
						j = j + 1
					}
					i = i + 1
				}
			`,
			varName:  "count",
			expected: "6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.varName]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestForOfLoop tests for-of loops
func TestForOfLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		expected string
	}{
		{
			name: "for-of with array",
			input: `
				sum = 0
				for (x of [1, 2, 3, 4, 5]) {
					sum = sum + x
				}
			`,
			varName:  "sum",
			expected: "15",
		},
		{
			name: "for-of with empty array",
			input: `
				count = 0
				for (x of []) {
					count = count + 1
				}
			`,
			varName:  "count",
			expected: "0",
		},
		{
			name: "for-of with string",
			input: `
				count = 0
				for (ch of "hello") {
					count = count + 1
				}
			`,
			varName:  "count",
			expected: "5",
		},
		{
			name: "for-of building string",
			input: `
				result = ""
				for (ch of "abc") {
					result = result + ch
				}
			`,
			varName:  "result",
			expected: "abc",
		},
		{
			name: "nested for-of loops",
			input: `
				count = 0
				for (x of [1, 2]) {
					for (y of [1, 2, 3]) {
						count = count + 1
					}
				}
			`,
			varName:  "count",
			expected: "6",
		},
		{
			name: "for-of with array expressions",
			input: `
				sum = 0
				arr = [10, 20, 30]
				for (val of arr) {
					sum = sum + val
				}
			`,
			varName:  "sum",
			expected: "60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.varName]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestBreakStatement tests break statements in loops
func TestBreakStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		expected string
	}{
		{
			name: "break in while loop",
			input: `
				x = 0
				while (x < 10) {
					if (x == 5) {
						break
					}
					x = x + 1
				}
			`,
			varName:  "x",
			expected: "5",
		},
		{
			name: "break in for-of loop",
			input: `
				sum = 0
				for (x of [1, 2, 3, 4, 5]) {
					if (x == 3) {
						break
					}
					sum = sum + x
				}
			`,
			varName:  "sum",
			expected: "3",
		},
		{
			name: "break in nested loop - inner",
			input: `
				count = 0
				for (x of [1, 2, 3]) {
					for (y of [1, 2, 3]) {
						if (y == 2) {
							break
						}
						count = count + 1
					}
				}
			`,
			varName:  "count",
			expected: "3",
		},
		{
			name: "break in nested loop - outer",
			input: `
				count = 0
				for (x of [1, 2, 3]) {
					if (x == 2) {
						break
					}
					for (y of [1, 2, 3]) {
						count = count + 1
					}
				}
			`,
			varName:  "count",
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.varName]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestContinueStatement tests continue statements in loops
func TestContinueStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		expected string
	}{
		{
			name: "continue in while loop",
			input: `
				x = 0
				sum = 0
				while (x < 5) {
					x = x + 1
					if (x == 3) {
						continue
					}
					sum = sum + x
				}
			`,
			varName:  "sum",
			expected: "12",
		},
		{
			name: "continue in for-of loop",
			input: `
				sum = 0
				for (x of [1, 2, 3, 4, 5]) {
					if (x == 3) {
						continue
					}
					sum = sum + x
				}
			`,
			varName:  "sum",
			expected: "12",
		},
		{
			name: "continue skipping even numbers",
			input: `
				sum = 0
				for (x of [1, 2, 3, 4, 5, 6]) {
					if (x % 2 == 0) {
						continue
					}
					sum = sum + x
				}
			`,
			varName:  "sum",
			expected: "9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.varName]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestBreakContinueOutsideLoop tests that break/continue outside loops produce errors
func TestBreakContinueOutsideLoop(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "break outside loop",
			input: `break`,
		},
		{
			name:  "continue outside loop",
			input: `continue`,
		},
		{
			name: "break outside loop after if",
			input: `
				if (true) {
					break
				}
			`,
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
			_, err := interp.Eval(program)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

// TestBlockScoping tests that blocks create new scopes
func TestBlockScoping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkVar string
		expected string
	}{
		{
			name: "variable shadowing in if block",
			input: `
				x = 1
				if (true) {
					x = 2
				}
			`,
			checkVar: "x",
			expected: "2",
		},
		{
			name: "variable shadowing in while block",
			input: `
				x = 1
				i = 0
				while (i < 1) {
					x = 10
					i = i + 1
				}
			`,
			checkVar: "x",
			expected: "10",
		},
		{
			name: "variable shadowing in for-of block",
			input: `
				x = 1
				for (item of [5]) {
					x = item
				}
			`,
			checkVar: "x",
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.checkVar]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestComplexControlFlow tests complex combinations of control flow
func TestComplexControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		expected string
	}{
		{
			name: "nested loops with break and continue",
			input: `
				sum = 0
				for (i of [1, 2, 3, 4, 5]) {
					if (i == 3) {
						continue
					}
					for (j of [1, 2, 3]) {
						if (j == 2) {
							break
						}
						sum = sum + 1
					}
				}
			`,
			varName:  "sum",
			expected: "4",
		},
		{
			name: "while with if-else and break",
			input: `
				x = 0
				result = 0
				while (x < 100) {
					x = x + 1
					if (x % 2 == 0) {
						if (x > 10) {
							break
						}
						result = result + x
					}
				}
			`,
			varName:  "result",
			expected: "30",
		},
		{
			name: "fibonacci with while loop",
			input: `
				a = 0
				b = 1
				count = 0
				while (count < 10) {
					temp = a
					a = b
					b = temp + b
					count = count + 1
				}
				result = a
			`,
			varName:  "result",
			expected: "55",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalFull(t, tt.input)
			vars := result.Variables()
			varValue := vars[tt.varName]
			if varValue.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, varValue.String())
			}
		})
	}
}

// TestForOfWithNonIterable tests error handling for non-iterable values
func TestForOfWithNonIterable(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "for-of with number",
			input: `for (x of 123) { x = x + 1 }`,
		},
		{
			name:  "for-of with boolean",
			input: `for (x of true) { x = x + 1 }`,
		},
		{
			name:  "for-of with object",
			input: `for (x of {a: 1}) { x = x + 1 }`,
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
			_, err := interp.Eval(program)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
			if !strings.Contains(err.Error(), "iterable") {
				t.Errorf("expected error to mention 'iterable', got: %v", err)
			}
		})
	}
}
