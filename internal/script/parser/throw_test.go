package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestThrowStringLiteral(t *testing.T) {
	input := `throw "error"`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ThrowStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ThrowStatement. got=%T", program.Statements[0])
	}

	if stmt.Token.Type != lexer.KW_THROW {
		t.Errorf("stmt.Token.Type not KW_THROW. got=%v", stmt.Token.Type)
	}

	strLit, ok := stmt.Expression.(*StringLiteral)
	if !ok {
		t.Fatalf("stmt.Expression is not *StringLiteral. got=%T", stmt.Expression)
	}

	if strLit.Value != "error" {
		t.Errorf("strLit.Value not 'error'. got=%s", strLit.Value)
	}
}

func TestThrowObjectLiteral(t *testing.T) {
	input := `throw {message: "err", code: 404}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ThrowStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ThrowStatement. got=%T", program.Statements[0])
	}

	objLit, ok := stmt.Expression.(*ObjectLiteral)
	if !ok {
		t.Fatalf("stmt.Expression is not *ObjectLiteral. got=%T", stmt.Expression)
	}

	if len(objLit.Pairs) != 2 {
		t.Fatalf("objLit.Pairs does not contain 2 pairs. got=%d", len(objLit.Pairs))
	}
}

func TestThrowIdentifier(t *testing.T) {
	input := `throw someVar`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ThrowStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ThrowStatement. got=%T", program.Statements[0])
	}

	ident, ok := stmt.Expression.(*Identifier)
	if !ok {
		t.Fatalf("stmt.Expression is not *Identifier. got=%T", stmt.Expression)
	}

	if ident.Value != "someVar" {
		t.Errorf("ident.Value not 'someVar'. got=%s", ident.Value)
	}
}

func TestThrowBareIsError(t *testing.T) {
	input := `throw
`

	l := lexer.New(input)
	p := New(l)
	p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatalf("expected parser errors for bare throw")
	}

	found := false
	for _, err := range p.Errors() {
		if containsSubstring(err, "throw statement requires an expression") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected error containing 'throw statement requires an expression', got: %v", p.Errors())
	}
}

func TestThrowString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `throw "error"`,
			expected: `throw "error"`,
		},
		{
			input:    `throw someVar`,
			expected: `throw someVar`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ThrowStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not *ThrowStatement. got=%T", program.Statements[0])
		}

		if stmt.String() != tt.expected {
			t.Errorf("stmt.String() wrong.\nexpected: %s\ngot: %s", tt.expected, stmt.String())
		}
	}
}
