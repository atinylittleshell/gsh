package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestForOfStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic for-of with array literal",
			input: `for (item of [1, 2, 3]) {
				print(item)
			}`,
			expected: "for (item of [1, 2, 3]) {\n  print(item)\n}",
		},
		{
			name: "for-of with identifier",
			input: `for (item of items) {
				process(item)
			}`,
			expected: "for (item of items) {\n  process(item)\n}",
		},
		{
			name: "for-of with member expression",
			input: `for (file of filesystem.list_directory("/home")) {
				print(file)
			}`,
			expected: "for (file of filesystem.list_directory(\"/home\")) {\n  print(file)\n}",
		},
		{
			name: "for-of with string array",
			input: `for (name of ["Alice", "Bob", "Charlie"]) {
				print(name)
			}`,
			expected: "for (name of [\"Alice\", \"Bob\", \"Charlie\"]) {\n  print(name)\n}",
		},
		{
			name: "nested for-of loops",
			input: `for (outer of outerList) {
				for (inner of innerList) {
					print(outer)
					print(inner)
				}
			}`,
			expected: "for (outer of outerList) {\n  for (inner of innerList) {\n  print(outer)\n  print(inner)\n}\n}",
		},
		{
			name: "for-of with multiple statements in body",
			input: `for (item of collection) {
				x = item + 1
				y = x * 2
				print(y)
			}`,
			expected: "for (item of collection) {\n  x = (item + 1)\n  y = (x * 2)\n  print(y)\n}",
		},
		{
			name: "for-of with empty body",
			input: `for (item of items) {
			}`,
			expected: "for (item of items) {\n}",
		},
		{
			name: "for-of with binary expression as iterable",
			input: `for (num of numbers + moreNumbers) {
				print(num)
			}`,
			expected: "for (num of (numbers + moreNumbers)) {\n  print(num)\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			checkParserErrors(t, p)

			if len(program.Statements) != 1 {
				t.Fatalf("program.Statements does not contain 1 statement. got=%d",
					len(program.Statements))
			}

			stmt, ok := program.Statements[0].(*ForOfStatement)
			if !ok {
				t.Fatalf("program.Statements[0] is not *ForOfStatement. got=%T",
					program.Statements[0])
			}

			if stmt.String() != tt.expected {
				t.Errorf("stmt.String() wrong.\nexpected=%q\ngot=%q",
					tt.expected, stmt.String())
			}
		})
	}
}

func TestForOfStatementStructure(t *testing.T) {
	input := `for (item of items) {
		print(item)
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ForOfStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ForOfStatement. got=%T",
			program.Statements[0])
	}

	// Check variable
	if stmt.Variable.Value != "item" {
		t.Errorf("stmt.Variable.Value not 'item'. got=%q", stmt.Variable.Value)
	}

	// Check iterable
	ident, ok := stmt.Iterable.(*Identifier)
	if !ok {
		t.Fatalf("stmt.Iterable is not *Identifier. got=%T", stmt.Iterable)
	}
	if ident.Value != "items" {
		t.Errorf("ident.Value not 'items'. got=%q", ident.Value)
	}

	// Check body
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("stmt.Body.Statements does not contain 1 statement. got=%d",
			len(stmt.Body.Statements))
	}
}

func TestForOfStatementWithArrayLiteral(t *testing.T) {
	input := `for (num of [1, 2, 3]) {
		print(num)
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ForOfStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ForOfStatement. got=%T",
			program.Statements[0])
	}

	// Check iterable is an array
	arr, ok := stmt.Iterable.(*ArrayLiteral)
	if !ok {
		t.Fatalf("stmt.Iterable is not *ArrayLiteral. got=%T", stmt.Iterable)
	}

	if len(arr.Elements) != 3 {
		t.Fatalf("array does not have 3 elements. got=%d", len(arr.Elements))
	}
}

func TestForOfStatementErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "missing opening parenthesis",
			input:         `for item of items) {}`,
			expectedError: "expected next token to be LPAREN",
		},
		{
			name:          "missing variable name",
			input:         `for (of items) {}`,
			expectedError: "expected next token to be IDENT",
		},
		{
			name:          "missing 'of' keyword",
			input:         `for (item items) {}`,
			expectedError: "expected next token to be KW_OF",
		},
		{
			name:          "missing closing parenthesis",
			input:         `for (item of items {}`,
			expectedError: "expected next token to be RPAREN",
		},
		{
			name:          "missing opening brace",
			input:         `for (item of items) print(item) }`,
			expectedError: "expected next token to be LBRACE",
		},
		{
			name:          "missing closing brace",
			input:         `for (item of items) { print(item)`,
			expectedError: "expected '}'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			p.ParseProgram()

			if len(p.Errors()) == 0 {
				t.Fatalf("expected parser errors, got none")
			}

			found := false
			for _, err := range p.Errors() {
				if containsSubstring(err, tt.expectedError) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error containing %q, got errors: %v",
					tt.expectedError, p.Errors())
			}
		})
	}
}

func TestForOfWithIfStatement(t *testing.T) {
	input := `for (item of items) {
		if (item > 5) {
			print(item)
		}
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ForOfStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ForOfStatement. got=%T",
			program.Statements[0])
	}

	// Check body contains if statement
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("stmt.Body.Statements does not contain 1 statement. got=%d",
			len(stmt.Body.Statements))
	}

	_, ok = stmt.Body.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("stmt.Body.Statements[0] is not *IfStatement. got=%T",
			stmt.Body.Statements[0])
	}
}

func TestForOfWithComplexIterable(t *testing.T) {
	input := `for (pr of github.list_pull_requests(repo, {state: "open"})) {
		print(pr)
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ForOfStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ForOfStatement. got=%T",
			program.Statements[0])
	}

	// Check iterable is a call expression
	call, ok := stmt.Iterable.(*CallExpression)
	if !ok {
		t.Fatalf("stmt.Iterable is not *CallExpression. got=%T", stmt.Iterable)
	}

	// Check call function is a member expression
	_, ok = call.Function.(*MemberExpression)
	if !ok {
		t.Fatalf("call.Function is not *MemberExpression. got=%T", call.Function)
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
