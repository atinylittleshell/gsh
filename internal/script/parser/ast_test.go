package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestString(t *testing.T) {
	program := &Program{
		Statements: []Statement{
			&AssignmentStatement{
				Token: lexer.Token{Type: lexer.IDENT, Literal: "myVar"},
				Name:  &Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "myVar"}, Value: "myVar"},
				Value: &Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "anotherVar"}, Value: "anotherVar"},
			},
		},
	}

	if program.String() != "myVar = anotherVar" {
		t.Errorf("program.String() wrong. got=%q", program.String())
	}
}

func TestIdentifierNode(t *testing.T) {
	ident := &Identifier{
		Token: lexer.Token{Type: lexer.IDENT, Literal: "foobar"},
		Value: "foobar",
	}

	if ident.TokenLiteral() != "foobar" {
		t.Errorf("ident.TokenLiteral() wrong. got=%q", ident.TokenLiteral())
	}

	if ident.String() != "foobar" {
		t.Errorf("ident.String() wrong. got=%q", ident.String())
	}
}

func TestNumberLiteral(t *testing.T) {
	num := &NumberLiteral{
		Token: lexer.Token{Type: lexer.NUMBER, Literal: "42"},
		Value: "42",
	}

	if num.TokenLiteral() != "42" {
		t.Errorf("num.TokenLiteral() wrong. got=%q", num.TokenLiteral())
	}

	if num.String() != "42" {
		t.Errorf("num.String() wrong. got=%q", num.String())
	}
}

func TestStringLiteral(t *testing.T) {
	str := &StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "hello"},
		Value: "hello",
	}

	if str.TokenLiteral() != "hello" {
		t.Errorf("str.TokenLiteral() wrong. got=%q", str.TokenLiteral())
	}

	if str.String() != `"hello"` {
		t.Errorf("str.String() wrong. got=%q", str.String())
	}
}

func TestBooleanLiteral(t *testing.T) {
	tests := []struct {
		value    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		b := &BooleanLiteral{
			Token: lexer.Token{Type: lexer.IDENT, Literal: tt.expected},
			Value: tt.value,
		}

		if b.String() != tt.expected {
			t.Errorf("b.String() wrong. expected=%q, got=%q", tt.expected, b.String())
		}
	}
}

func TestBinaryExpression(t *testing.T) {
	expr := &BinaryExpression{
		Token:    lexer.Token{Type: lexer.OP_PLUS, Literal: "+"},
		Left:     &NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "5"}, Value: "5"},
		Operator: "+",
		Right:    &NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "10"}, Value: "10"},
	}

	if expr.String() != "(5 + 10)" {
		t.Errorf("expr.String() wrong. got=%q", expr.String())
	}
}

func TestUnaryExpression(t *testing.T) {
	expr := &UnaryExpression{
		Token:    lexer.Token{Type: lexer.OP_MINUS, Literal: "-"},
		Operator: "-",
		Right:    &NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "5"}, Value: "5"},
	}

	if expr.String() != "(-5)" {
		t.Errorf("expr.String() wrong. got=%q", expr.String())
	}
}

func TestCallExpression(t *testing.T) {
	expr := &CallExpression{
		Token:    lexer.Token{Type: lexer.LPAREN, Literal: "("},
		Function: &Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "add"}, Value: "add"},
		Arguments: []Expression{
			&NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "1"}, Value: "1"},
			&NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "2"}, Value: "2"},
		},
	}

	if expr.String() != "add(1, 2)" {
		t.Errorf("expr.String() wrong. got=%q", expr.String())
	}
}

func TestMemberExpression(t *testing.T) {
	expr := &MemberExpression{
		Token:    lexer.Token{Type: lexer.DOT, Literal: "."},
		Object:   &Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "env"}, Value: "env"},
		Property: &Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "HOME"}, Value: "HOME"},
	}

	if expr.String() != "env.HOME" {
		t.Errorf("expr.String() wrong. got=%q", expr.String())
	}
}

func TestArrayLiteral(t *testing.T) {
	expr := &ArrayLiteral{
		Token: lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: []Expression{
			&NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "1"}, Value: "1"},
			&NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "2"}, Value: "2"},
			&NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "3"}, Value: "3"},
		},
	}

	if expr.String() != "[1, 2, 3]" {
		t.Errorf("expr.String() wrong. got=%q", expr.String())
	}
}

func TestObjectLiteral(t *testing.T) {
	expr := &ObjectLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: map[string]Expression{
			"name": &StringLiteral{Token: lexer.Token{Type: lexer.STRING, Literal: "Alice"}, Value: "Alice"},
			"age":  &NumberLiteral{Token: lexer.Token{Type: lexer.NUMBER, Literal: "30"}, Value: "30"},
		},
		Order: []string{"name", "age"},
	}

	expected := `{name: "Alice", age: 30}`
	if expr.String() != expected {
		t.Errorf("expr.String() wrong. expected=%q, got=%q", expected, expr.String())
	}
}

func TestBlockStatement(t *testing.T) {
	block := &BlockStatement{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Statements: []Statement{
			&ExpressionStatement{
				Token: lexer.Token{Type: lexer.IDENT, Literal: "x"},
				Expression: &Identifier{
					Token: lexer.Token{Type: lexer.IDENT, Literal: "x"},
					Value: "x",
				},
			},
		},
	}

	if block.String() != "{\n  x\n}" {
		t.Errorf("block.String() wrong. got=%q", block.String())
	}
}
