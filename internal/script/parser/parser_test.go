package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestAssignmentStatements(t *testing.T) {
	tests := []struct {
		input              string
		expectedIdentifier string
		expectedValue      string
	}{
		{"x = 5", "x", "5"},
		{"y = 10", "y", "10"},
		{"foobar = 838383", "foobar", "838383"},
		{"name = \"Alice\"", "name", "\"Alice\""},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt := program.Statements[0]
		if !testAssignmentStatement(t, stmt, tt.expectedIdentifier) {
			return
		}

		val := stmt.(*AssignmentStatement).Value
		if val.String() != tt.expectedValue {
			t.Errorf("val.String() not %s. got=%s", tt.expectedValue, val.String())
		}
	}
}

func TestAssignmentWithTypeAnnotation(t *testing.T) {
	tests := []struct {
		input              string
		expectedIdentifier string
		expectedType       string
		expectedValue      string
	}{
		{"x: number = 5", "x", "number", "5"},
		{"name: string = \"Alice\"", "name", "string", "\"Alice\""},
		{"isActive: boolean = true", "isActive", "boolean", "true"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt := program.Statements[0]
		assignStmt, ok := stmt.(*AssignmentStatement)
		if !ok {
			t.Fatalf("stmt not *AssignmentStatement. got=%T", stmt)
		}

		if assignStmt.Name.Value != tt.expectedIdentifier {
			t.Errorf("assignStmt.Name.Value not %s. got=%s",
				tt.expectedIdentifier, assignStmt.Name.Value)
		}

		if assignStmt.TypeAnnotation == nil {
			t.Fatalf("assignStmt.TypeAnnotation is nil")
		}

		if assignStmt.TypeAnnotation.Value != tt.expectedType {
			t.Errorf("assignStmt.TypeAnnotation.Value not %s. got=%s",
				tt.expectedType, assignStmt.TypeAnnotation.Value)
		}

		if assignStmt.Value.String() != tt.expectedValue {
			t.Errorf("assignStmt.Value.String() not %s. got=%s",
				tt.expectedValue, assignStmt.Value.String())
		}
	}
}

func testAssignmentStatement(t *testing.T, s Statement, name string) bool {
	if s.TokenLiteral() != name {
		t.Errorf("s.TokenLiteral not '%s'. got=%s", name, s.TokenLiteral())
		return false
	}

	assignStmt, ok := s.(*AssignmentStatement)
	if !ok {
		t.Errorf("s not *AssignmentStatement. got=%T", s)
		return false
	}

	if assignStmt.Name.Value != name {
		t.Errorf("assignStmt.Name.Value not '%s'. got=%s", name, assignStmt.Name.Value)
		return false
	}

	if assignStmt.Name.TokenLiteral() != name {
		t.Errorf("assignStmt.Name.TokenLiteral() not '%s'. got=%s",
			name, assignStmt.Name.TokenLiteral())
		return false
	}

	return true
}
