package parser

import (
	"fmt"
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

func TestIdentifierExpression(t *testing.T) {
	input := "foobar"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program has not enough statements. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
			program.Statements[0])
	}

	ident, ok := stmt.Expression.(*Identifier)
	if !ok {
		t.Fatalf("exp not *Identifier. got=%T", stmt.Expression)
	}

	if ident.Value != "foobar" {
		t.Errorf("ident.Value not %s. got=%s", "foobar", ident.Value)
	}

	if ident.TokenLiteral() != "foobar" {
		t.Errorf("ident.TokenLiteral not %s. got=%s", "foobar",
			ident.TokenLiteral())
	}
}

func TestNumberLiteralExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5", "5"},
		{"10", "10"},
		{"3.14", "3.14"},
		{"0.5", "0.5"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program has not enough statements. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
				program.Statements[0])
		}

		literal, ok := stmt.Expression.(*NumberLiteral)
		if !ok {
			t.Fatalf("exp not *NumberLiteral. got=%T", stmt.Expression)
		}

		if literal.Value != tt.expected {
			t.Errorf("literal.Value not %s. got=%s", tt.expected, literal.Value)
		}
	}
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world"`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ExpressionStatement)
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
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program has not enough statements. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
				program.Statements[0])
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
	unaryTests := []struct {
		input    string
		operator string
		value    any
	}{
		{"!5", "!", 5},
		{"-15", "-", 15},
		{"!true", "!", true},
		{"!false", "!", false},
	}

	for _, tt := range unaryTests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain %d statements. got=%d\n",
				1, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
				program.Statements[0])
		}

		exp, ok := stmt.Expression.(*UnaryExpression)
		if !ok {
			t.Fatalf("stmt is not UnaryExpression. got=%T", stmt.Expression)
		}

		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s",
				tt.operator, exp.Operator)
		}

		if !testLiteralExpression(t, exp.Right, tt.value) {
			return
		}
	}
}

func TestParsingBinaryExpressions(t *testing.T) {
	binaryTests := []struct {
		input      string
		leftValue  any
		operator   string
		rightValue any
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 % 5", 5, "%", 5},
		{"5 > 5", 5, ">", 5},
		{"5 < 5", 5, "<", 5},
		{"5 == 5", 5, "==", 5},
		{"5 != 5", 5, "!=", 5},
		{"5 >= 5", 5, ">=", 5},
		{"5 <= 5", 5, "<=", 5},
		{"true == true", true, "==", true},
		{"true != false", true, "!=", false},
		{"false == false", false, "==", false},
	}

	for _, tt := range binaryTests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain %d statements. got=%d\n",
				1, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
				program.Statements[0])
		}

		if !testBinaryExpression(t, stmt.Expression, tt.leftValue,
			tt.operator, tt.rightValue) {
			return
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
		{"a + b * c + d / e - f", "(((a + (b * c)) + (d / e)) - f)"},
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
	input := "add(1, 2 * 3, 4 + 5)"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain %d statements. got=%d\n",
			1, len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not ExpressionStatement. got=%T",
			program.Statements[0])
	}

	exp, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not CallExpression. got=%T",
			stmt.Expression)
	}

	if !testIdentifier(t, exp.Function, "add") {
		return
	}

	if len(exp.Arguments) != 3 {
		t.Fatalf("wrong length of arguments. got=%d", len(exp.Arguments))
	}

	testLiteralExpression(t, exp.Arguments[0], 1)
	testBinaryExpression(t, exp.Arguments[1], 2, "*", 3)
	testBinaryExpression(t, exp.Arguments[2], 4, "+", 5)
}

func TestMemberExpressionParsing(t *testing.T) {
	tests := []struct {
		input    string
		object   string
		property string
	}{
		{"env.HOME", "env", "HOME"},
		{"filesystem.read_file", "filesystem", "read_file"},
		{"github.create_issue", "github", "create_issue"},
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

		stmt, ok := program.Statements[0].(*ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
				program.Statements[0])
		}

		exp, ok := stmt.Expression.(*MemberExpression)
		if !ok {
			t.Fatalf("exp is not MemberExpression. got=%T", stmt.Expression)
		}

		if !testIdentifier(t, exp.Object, tt.object) {
			return
		}

		if exp.Property.Value != tt.property {
			t.Errorf("exp.Property.Value not %s. got=%s", tt.property, exp.Property.Value)
		}
	}
}

func TestMemberCallExpression(t *testing.T) {
	input := "filesystem.read_file(path)"

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ExpressionStatement. got=%T",
			program.Statements[0])
	}

	exp, ok := stmt.Expression.(*CallExpression)
	if !ok {
		t.Fatalf("exp is not CallExpression. got=%T", stmt.Expression)
	}

	member, ok := exp.Function.(*MemberExpression)
	if !ok {
		t.Fatalf("exp.Function is not MemberExpression. got=%T", exp.Function)
	}

	if !testIdentifier(t, member.Object, "filesystem") {
		return
	}

	if member.Property.Value != "read_file" {
		t.Errorf("member.Property.Value not %s. got=%s", "read_file", member.Property.Value)
	}

	if len(exp.Arguments) != 1 {
		t.Fatalf("wrong number of arguments. got=%d", len(exp.Arguments))
	}

	testIdentifier(t, exp.Arguments[0], "path")
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

	testLiteralExpression(t, array.Elements[0], 1)
	testBinaryExpression(t, array.Elements[1], 2, "*", 2)
	testBinaryExpression(t, array.Elements[2], 3, "+", 3)
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
	input := `{name: "Alice", age: 30, active: true}`

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

	if len(obj.Pairs) != 3 {
		t.Fatalf("obj.Pairs has wrong length. got=%d", len(obj.Pairs))
	}

	expected := map[string]any{
		"name":   "Alice",
		"age":    30,
		"active": true,
	}

	for key, value := range expected {
		val, ok := obj.Pairs[key]
		if !ok {
			t.Errorf("no pair for key %q in obj.Pairs", key)
			continue
		}

		testLiteralExpression(t, val, value)
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
		t.Fatalf("obj.Pairs has wrong length. got=%d", len(obj.Pairs))
	}
}

func TestPipeOperator(t *testing.T) {
	input := `result = "prompt" | Agent`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignmentStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not AssignmentStatement. got=%T",
			program.Statements[0])
	}

	exp, ok := stmt.Value.(*BinaryExpression)
	if !ok {
		t.Fatalf("exp is not BinaryExpression. got=%T", stmt.Value)
	}

	if exp.Operator != "|" {
		t.Fatalf("exp.Operator is not '|'. got=%s", exp.Operator)
	}

	testLiteralExpression(t, exp.Left, "prompt")
	testIdentifier(t, exp.Right, "Agent")
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a && b", "(a && b)"},
		{"a || b", "(a || b)"},
		{"a ?? b", "(a ?? b)"},
		{"a && b || c", "((a && b) || c)"},
		{"a || b && c", "(a || (b && c))"},
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

// Helper functions

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

func testLiteralExpression(
	t *testing.T,
	exp Expression,
	expected any,
) bool {
	switch v := expected.(type) {
	case int:
		return testNumberLiteral(t, exp, int64(v))
	case int64:
		return testNumberLiteral(t, exp, v)
	case string:
		return testStringLiteral(t, exp, v)
	case bool:
		return testBooleanLiteral(t, exp, v)
	}
	t.Errorf("type of exp not handled. got=%T", exp)
	return false
}

func testNumberLiteral(t *testing.T, exp Expression, value int64) bool {
	num, ok := exp.(*NumberLiteral)
	if !ok {
		t.Errorf("exp not *NumberLiteral. got=%T", exp)
		return false
	}

	if num.Value != fmt.Sprintf("%d", value) {
		t.Errorf("num.Value not %d. got=%s", value, num.Value)
		return false
	}

	if num.TokenLiteral() != fmt.Sprintf("%d", value) {
		t.Errorf("num.TokenLiteral not %d. got=%s", value,
			num.TokenLiteral())
		return false
	}

	return true
}

func testStringLiteral(t *testing.T, exp Expression, value string) bool {
	// For identifiers
	if ident, ok := exp.(*Identifier); ok {
		if ident.Value != value {
			t.Errorf("ident.Value not %s. got=%s", value, ident.Value)
			return false
		}
		return true
	}

	// For string literals
	str, ok := exp.(*StringLiteral)
	if !ok {
		t.Errorf("exp not *StringLiteral. got=%T", exp)
		return false
	}

	if str.Value != value {
		t.Errorf("str.Value not %s. got=%s", value, str.Value)
		return false
	}

	return true
}

func testBooleanLiteral(t *testing.T, exp Expression, value bool) bool {
	bo, ok := exp.(*BooleanLiteral)
	if !ok {
		t.Errorf("exp not *BooleanLiteral. got=%T", exp)
		return false
	}

	if bo.Value != value {
		t.Errorf("bo.Value not %t. got=%t", value, bo.Value)
		return false
	}

	if value {
		if bo.TokenLiteral() != "true" {
			t.Errorf("bo.TokenLiteral not 'true'. got=%s", bo.TokenLiteral())
			return false
		}
	} else {
		if bo.TokenLiteral() != "false" {
			t.Errorf("bo.TokenLiteral not 'false'. got=%s", bo.TokenLiteral())
			return false
		}
	}

	return true
}

func testIdentifier(t *testing.T, exp Expression, value string) bool {
	ident, ok := exp.(*Identifier)
	if !ok {
		t.Errorf("exp not *Identifier. got=%T", exp)
		return false
	}

	if ident.Value != value {
		t.Errorf("ident.Value not %s. got=%s", value, ident.Value)
		return false
	}

	if ident.TokenLiteral() != value {
		t.Errorf("ident.TokenLiteral not %s. got=%s", value,
			ident.TokenLiteral())
		return false
	}

	return true
}

func testBinaryExpression(t *testing.T, exp Expression, left any,
	operator string, right any) bool {

	opExp, ok := exp.(*BinaryExpression)
	if !ok {
		t.Errorf("exp is not BinaryExpression. got=%T(%s)", exp, exp)
		return false
	}

	if !testLiteralExpression(t, opExp.Left, left) {
		return false
	}

	if opExp.Operator != operator {
		t.Errorf("exp.Operator is not '%s'. got=%q", operator, opExp.Operator)
		return false
	}

	if !testLiteralExpression(t, opExp.Right, right) {
		return false
	}

	return true
}
