package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestIdentifierExpression(t *testing.T) {
	input := "foobar;"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	ident, ok := stmt.Expression.(*Identifier)
	if !ok {
		t.Fatalf("exp not *Identifier. got=%T", stmt.Expression)
	}
	if ident.Value != "foobar" {
		t.Errorf("ident.Value not %s. got=%s", "foobar", ident.Value)
	}
}

func TestNumberLiteralExpression(t *testing.T) {
	input := "5;"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	literal, ok := stmt.Expression.(*NumberLiteral)
	if !ok {
		t.Fatalf("exp not *NumberLiteral. got=%T", stmt.Expression)
	}
	if literal.Value != "5" {
		t.Errorf("literal.Value not %s. got=%s", "5", literal.Value)
	}
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world";`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	literal, ok := stmt.Expression.(*StringLiteral)
	if !ok {
		t.Fatalf("exp not *StringLiteral. got=%T", stmt.Expression)
	}
	if literal.Value != "hello world" {
		t.Errorf("literal.Value not %s. got=%s", "hello world", literal.Value)
	}
}

func TestBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true;", true},
		{"false;", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
		}

		boolean, ok := stmt.Expression.(*BooleanLiteral)
		if !ok {
			t.Fatalf("exp not *BooleanLiteral. got=%T", stmt.Expression)
		}
		if boolean.Value != tt.expected {
			t.Errorf("boolean.Value not %t. got=%t", tt.expected, boolean.Value)
		}
	}
}

func TestParsingUnaryExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"!5;", "!"},
		{"-15;", "-"},
		{"!true;", "!"},
		{"!false;", "!"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain %d statements. got=%d\n", 1, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
		}

		exp, ok := stmt.Expression.(*UnaryExpression)
		if !ok {
			t.Fatalf("stmt is not UnaryExpression. got=%T", stmt.Expression)
		}
		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s", tt.operator, exp.Operator)
		}
	}
}

func TestParsingBinaryExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"5 + 5;", "+"},
		{"5 - 5;", "-"},
		{"5 * 5;", "*"},
		{"5 / 5;", "/"},
		{"5 > 5;", ">"},
		{"5 < 5;", "<"},
		{"5 == 5;", "=="},
		{"5 != 5;", "!="},
		{"true && false;", "&&"},
		{"true || false;", "||"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain %d statements. got=%d\n", 1, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
		}

		exp, ok := stmt.Expression.(*BinaryExpression)
		if !ok {
			t.Fatalf("exp is not BinaryExpression. got=%T", stmt.Expression)
		}

		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s", tt.operator, exp.Operator)
		}
	}
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-a * b", "((-a) * b)"},
		{"!-a", "(!(-a))"},
		{"a + b + c", "((a + b) + c)"},
		{"a + b - c", "((a + b) - c)"},
		{"a * b * c", "((a * b) * c)"},
		{"a * b / c", "((a * b) / c)"},
		{"a + b / c", "(a + (b / c))"},
		{"a + b * c + d / e - f", "(((a + (b * c)) + (d / e)) - f)"},
		{"3 + 4; -5 * 5", "(3 + 4)((-5) * 5)"},
		{"5 > 4 == 3 < 4", "((5 > 4) == (3 < 4))"},
		{"5 < 4 != 3 > 4", "((5 < 4) != (3 > 4))"},
		{"3 + 4 * 5 == 3 * 1 + 4 * 5", "((3 + (4 * 5)) == ((3 * 1) + (4 * 5)))"},
		{"true", "true"},
		{"false", "false"},
		{"3 > 5 == false", "((3 > 5) == false)"},
		{"3 < 5 == true", "((3 < 5) == true)"},
		{"1 + (2 + 3) + 4", "((1 + (2 + 3)) + 4)"},
		{"(5 + 5) * 2", "((5 + 5) * 2)"},
		{"2 / (5 + 5)", "(2 / (5 + 5))"},
		{"-(5 + 5)", "(-(5 + 5))"},
		{"!(true == true)", "(!(true == true))"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func TestCallExpressionParsing(t *testing.T) {
	input := "add(1, 2 * 3, 4 + 5);"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain %d statements. got=%d\n", 1, len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not ExpressionStatement. got=%T", program.Statements[0])
	}

	exp, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not CallExpression. got=%T", stmt.Expression)
	}

	if len(exp.Arguments) != 3 {
		t.Fatalf("wrong length of arguments. got=%d", len(exp.Arguments))
	}
}

func TestMemberExpressionParsing(t *testing.T) {
	input := "env.HOME"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	memberExpr, ok := stmt.Expression.(*MemberExpression)
	if !ok {
		t.Fatalf("exp is not MemberExpression. got=%T", stmt.Expression)
	}

	obj, ok := memberExpr.Object.(*Identifier)
	if !ok {
		t.Fatalf("memberExpr.Object is not Identifier. got=%T", memberExpr.Object)
	}
	if obj.Value != "env" {
		t.Errorf("obj.Value not 'env'. got=%s", obj.Value)
	}

	if memberExpr.Property.Value != "HOME" {
		t.Errorf("memberExpr.Property.Value not 'HOME'. got=%s", memberExpr.Property.Value)
	}
}

func TestMemberCallExpression(t *testing.T) {
	input := "filesystem.read_file(path)"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	callExpr, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("exp is not CallExpression. got=%T", stmt.Expression)
	}

	memberExpr, ok := callExpr.Function.(*MemberExpression)
	if !ok {
		t.Fatalf("callExpr.Function is not MemberExpression. got=%T", callExpr.Function)
	}

	obj, ok := memberExpr.Object.(*Identifier)
	if !ok {
		t.Fatalf("memberExpr.Object is not Identifier. got=%T", memberExpr.Object)
	}
	if obj.Value != "filesystem" {
		t.Errorf("obj.Value not 'filesystem'. got=%s", obj.Value)
	}

	if memberExpr.Property.Value != "read_file" {
		t.Errorf("memberExpr.Property.Value not 'read_file'. got=%s", memberExpr.Property.Value)
	}

	if len(callExpr.Arguments) != 1 {
		t.Fatalf("wrong number of arguments. got=%d", len(callExpr.Arguments))
	}
}

func TestArrayLiteralParsing(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3]"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("exp not ExpressionStatement. got=%T", program.Statements[0])
	}

	array, ok := stmt.Expression.(*ArrayLiteral)
	if !ok {
		t.Fatalf("exp not ArrayLiteral. got=%T", stmt.Expression)
	}

	if len(array.Elements) != 3 {
		t.Fatalf("len(array.Elements) not 3. got=%d", len(array.Elements))
	}
}

func TestEmptyArrayLiteral(t *testing.T) {
	input := "[]"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("exp not ExpressionStatement. got=%T", program.Statements[0])
	}

	array, ok := stmt.Expression.(*ArrayLiteral)
	if !ok {
		t.Fatalf("exp not ArrayLiteral. got=%T", stmt.Expression)
	}

	if len(array.Elements) != 0 {
		t.Fatalf("len(array.Elements) not 0. got=%d", len(array.Elements))
	}
}

func TestObjectLiteralParsing(t *testing.T) {
	input := `{name: "Alice", age: 30}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("exp not ExpressionStatement. got=%T", program.Statements[0])
	}

	obj, ok := stmt.Expression.(*ObjectLiteral)
	if !ok {
		t.Fatalf("exp not ObjectLiteral. got=%T", stmt.Expression)
	}

	if len(obj.Pairs) != 2 {
		t.Fatalf("len(obj.Pairs) not 2. got=%d", len(obj.Pairs))
	}

	if _, ok := obj.Pairs["name"]; !ok {
		t.Errorf("obj.Pairs missing 'name' key")
	}

	if _, ok := obj.Pairs["age"]; !ok {
		t.Errorf("obj.Pairs missing 'age' key")
	}
}

func TestEmptyObjectLiteral(t *testing.T) {
	input := "{}"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("exp not ExpressionStatement. got=%T", program.Statements[0])
	}

	obj, ok := stmt.Expression.(*ObjectLiteral)
	if !ok {
		t.Fatalf("exp not ObjectLiteral. got=%T", stmt.Expression)
	}

	if len(obj.Pairs) != 0 {
		t.Fatalf("len(obj.Pairs) not 0. got=%d", len(obj.Pairs))
	}
}

func TestPipeOperator(t *testing.T) {
	input := `"hello" | Agent`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T", program.Statements[0])
	}

	pipeExpr, ok := stmt.Expression.(*PipeExpression)
	if !ok {
		t.Fatalf("exp is not PipeExpression. got=%T", stmt.Expression)
	}

	// Check left side is string literal
	if _, ok := pipeExpr.Left.(*StringLiteral); !ok {
		t.Errorf("pipeExpr.Left is not StringLiteral. got=%T", pipeExpr.Left)
	}

	// Check right side is identifier
	if _, ok := pipeExpr.Right.(*Identifier); !ok {
		t.Errorf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true && false", "(true && false)"},
		{"true || false", "(true || false)"},
		{"x ?? y", "(x ?? y)"},
		{"a && b || c", "((a && b) || c)"},
		{"a || b && c", "(a || (b && c))"},
		{"x ?? y ?? z", "((x ?? y) ?? z)"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}
