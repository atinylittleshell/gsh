package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestSimplePipeExpression(t *testing.T) {
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

	// Check left side (string literal)
	leftStr, ok := pipeExpr.Left.(*StringLiteral)
	if !ok {
		t.Fatalf("pipeExpr.Left is not StringLiteral. got=%T", pipeExpr.Left)
	}
	if leftStr.Value != "hello" {
		t.Errorf("leftStr.Value not 'hello'. got=%s", leftStr.Value)
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeExpressionWithVariable(t *testing.T) {
	input := `conv | Agent`

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

	// Check left side (identifier)
	leftIdent, ok := pipeExpr.Left.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Left is not Identifier. got=%T", pipeExpr.Left)
	}
	if leftIdent.Value != "conv" {
		t.Errorf("leftIdent.Value not 'conv'. got=%s", leftIdent.Value)
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestChainedPipeExpression(t *testing.T) {
	input := `"hello" | Agent1 | Agent2`

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

	// The outer pipe should be: ("hello" | Agent1) | Agent2
	outerPipe, ok := stmt.Expression.(*PipeExpression)
	if !ok {
		t.Fatalf("exp is not PipeExpression. got=%T", stmt.Expression)
	}

	// Check right side of outer pipe
	rightIdent, ok := outerPipe.Right.(*Identifier)
	if !ok {
		t.Fatalf("outerPipe.Right is not Identifier. got=%T", outerPipe.Right)
	}
	if rightIdent.Value != "Agent2" {
		t.Errorf("rightIdent.Value not 'Agent2'. got=%s", rightIdent.Value)
	}

	// Check left side of outer pipe (should be another pipe)
	innerPipe, ok := outerPipe.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("outerPipe.Left is not PipeExpression. got=%T", outerPipe.Left)
	}

	// Check inner pipe left side (string)
	leftStr, ok := innerPipe.Left.(*StringLiteral)
	if !ok {
		t.Fatalf("innerPipe.Left is not StringLiteral. got=%T", innerPipe.Left)
	}
	if leftStr.Value != "hello" {
		t.Errorf("leftStr.Value not 'hello'. got=%s", leftStr.Value)
	}

	// Check inner pipe right side
	innerRight, ok := innerPipe.Right.(*Identifier)
	if !ok {
		t.Fatalf("innerPipe.Right is not Identifier. got=%T", innerPipe.Right)
	}
	if innerRight.Value != "Agent1" {
		t.Errorf("innerRight.Value not 'Agent1'. got=%s", innerRight.Value)
	}
}

func TestPipeExpressionWithStringInterpolation(t *testing.T) {
	input := `conv | "Tell me more" | Agent`

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

	// Outer pipe: (conv | "Tell me more") | Agent
	outerPipe, ok := stmt.Expression.(*PipeExpression)
	if !ok {
		t.Fatalf("exp is not PipeExpression. got=%T", stmt.Expression)
	}

	// Check inner pipe
	innerPipe, ok := outerPipe.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("outerPipe.Left is not PipeExpression. got=%T", outerPipe.Left)
	}

	// Check conv
	conv, ok := innerPipe.Left.(*Identifier)
	if !ok {
		t.Fatalf("innerPipe.Left is not Identifier. got=%T", innerPipe.Left)
	}
	if conv.Value != "conv" {
		t.Errorf("conv.Value not 'conv'. got=%s", conv.Value)
	}

	// Check string
	str, ok := innerPipe.Right.(*StringLiteral)
	if !ok {
		t.Fatalf("innerPipe.Right is not StringLiteral. got=%T", innerPipe.Right)
	}
	if str.Value != "Tell me more" {
		t.Errorf("str.Value not 'Tell me more'. got=%s", str.Value)
	}

	// Check Agent
	agent, ok := outerPipe.Right.(*Identifier)
	if !ok {
		t.Fatalf("outerPipe.Right is not Identifier. got=%T", outerPipe.Right)
	}
	if agent.Value != "Agent" {
		t.Errorf("agent.Value not 'Agent'. got=%s", agent.Value)
	}
}

func TestPipeExpressionPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello" | Agent`, `("hello" | Agent)`},
		{`x | y | z`, `((x | y) | z)`},
		{`a + b | Agent`, `((a + b) | Agent)`},
		{`"prompt" | Agent1 | "msg" | Agent2`, `((("prompt" | Agent1) | "msg") | Agent2)`},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, actual)
		}
	}
}

func TestPipeExpressionInAssignment(t *testing.T) {
	input := `result = "analyze data" | DataAnalyst`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignmentStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not AssignmentStatement. got=%T", program.Statements[0])
	}

	if stmt.Name.Value != "result" {
		t.Errorf("stmt.Name.Value not 'result'. got=%s", stmt.Name.Value)
	}

	pipeExpr, ok := stmt.Value.(*PipeExpression)
	if !ok {
		t.Fatalf("stmt.Value is not PipeExpression. got=%T", stmt.Value)
	}

	// Check left side
	leftStr, ok := pipeExpr.Left.(*StringLiteral)
	if !ok {
		t.Fatalf("pipeExpr.Left is not StringLiteral. got=%T", pipeExpr.Left)
	}
	if leftStr.Value != "analyze data" {
		t.Errorf("leftStr.Value not 'analyze data'. got=%s", leftStr.Value)
	}

	// Check right side
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "DataAnalyst" {
		t.Errorf("rightIdent.Value not 'DataAnalyst'. got=%s", rightIdent.Value)
	}
}

func TestComplexPipeExpression(t *testing.T) {
	// Test from the spec: multi-turn conversation with agent handoff
	input := `conv = "Analyze this: " | DataAnalyst | "Write a report" | ReportWriter`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignmentStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not AssignmentStatement. got=%T", program.Statements[0])
	}

	if stmt.Name.Value != "conv" {
		t.Errorf("stmt.Name.Value not 'conv'. got=%s", stmt.Name.Value)
	}

	// The structure should be: ((("Analyze this: " | DataAnalyst) | "Write a report") | ReportWriter)
	pipe3, ok := stmt.Value.(*PipeExpression)
	if !ok {
		t.Fatalf("stmt.Value is not PipeExpression. got=%T", stmt.Value)
	}

	// Check rightmost agent
	if agent, ok := pipe3.Right.(*Identifier); !ok || agent.Value != "ReportWriter" {
		t.Errorf("Expected ReportWriter on right, got=%v", pipe3.Right)
	}

	// Check next level
	pipe2, ok := pipe3.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("pipe3.Left is not PipeExpression. got=%T", pipe3.Left)
	}

	// Check second string
	if str, ok := pipe2.Right.(*StringLiteral); !ok || str.Value != "Write a report" {
		t.Errorf("Expected 'Write a report', got=%v", pipe2.Right)
	}

	// Check next level
	pipe1, ok := pipe2.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("pipe2.Left is not PipeExpression. got=%T", pipe2.Left)
	}

	// Check first agent
	if agent, ok := pipe1.Right.(*Identifier); !ok || agent.Value != "DataAnalyst" {
		t.Errorf("Expected DataAnalyst, got=%v", pipe1.Right)
	}

	// Check first string
	if str, ok := pipe1.Left.(*StringLiteral); !ok || str.Value != "Analyze this: " {
		t.Errorf("Expected 'Analyze this: ', got=%v", pipe1.Left)
	}
}

func TestPipeWithMemberExpression(t *testing.T) {
	// Pipe with function call result
	input := `data | processor.analyze`

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

	// Check left side
	leftIdent, ok := pipeExpr.Left.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Left is not Identifier. got=%T", pipeExpr.Left)
	}
	if leftIdent.Value != "data" {
		t.Errorf("leftIdent.Value not 'data'. got=%s", leftIdent.Value)
	}

	// Check right side (member expression)
	memberExpr, ok := pipeExpr.Right.(*MemberExpression)
	if !ok {
		t.Fatalf("pipeExpr.Right is not MemberExpression. got=%T", pipeExpr.Right)
	}

	obj, ok := memberExpr.Object.(*Identifier)
	if !ok {
		t.Fatalf("memberExpr.Object is not Identifier. got=%T", memberExpr.Object)
	}
	if obj.Value != "processor" {
		t.Errorf("obj.Value not 'processor'. got=%s", obj.Value)
	}

	if memberExpr.Property.Value != "analyze" {
		t.Errorf("memberExpr.Property.Value not 'analyze'. got=%s", memberExpr.Property.Value)
	}
}

func TestPipeOperatorLowestPrecedence(t *testing.T) {
	// Pipe should have lowest precedence (except for assignment)
	// so a + b | c should parse as (a + b) | c, not a + (b | c)
	tests := []struct {
		input    string
		expected string
	}{
		{`a + b | Agent`, `((a + b) | Agent)`},
		{`a * b + c | Agent`, `(((a * b) + c) | Agent)`},
		{`!a | Agent`, `((!a) | Agent)`},
		{`a && b | Agent`, `((a && b) | Agent)`},
		{`a ?? b | Agent`, `((a ?? b) | Agent)`},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, actual)
		}
	}
}

func TestPipeWithNumberLiteral(t *testing.T) {
	input := `42 | Agent`

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

	// Check left side (number literal)
	leftNum, ok := pipeExpr.Left.(*NumberLiteral)
	if !ok {
		t.Fatalf("pipeExpr.Left is not NumberLiteral. got=%T", pipeExpr.Left)
	}
	if leftNum.Value != "42" {
		t.Errorf("leftNum.Value not '42'. got=%s", leftNum.Value)
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeWithBooleanLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true literal", `true | Agent`, true},
		{"false literal", `false | Agent`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
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

			// Check left side (boolean literal)
			leftBool, ok := pipeExpr.Left.(*BooleanLiteral)
			if !ok {
				t.Fatalf("pipeExpr.Left is not BooleanLiteral. got=%T", pipeExpr.Left)
			}
			if leftBool.Value != tt.expected {
				t.Errorf("leftBool.Value not %t. got=%t", tt.expected, leftBool.Value)
			}
		})
	}
}

func TestPipeWithArrayLiteral(t *testing.T) {
	input := `[1, 2, 3] | Agent`

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

	// Check left side (array literal)
	leftArray, ok := pipeExpr.Left.(*ArrayLiteral)
	if !ok {
		t.Fatalf("pipeExpr.Left is not ArrayLiteral. got=%T", pipeExpr.Left)
	}
	if len(leftArray.Elements) != 3 {
		t.Errorf("leftArray.Elements length not 3. got=%d", len(leftArray.Elements))
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeWithObjectLiteral(t *testing.T) {
	input := `{name: "Alice", age: 30} | Agent`

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

	// Check left side (object literal)
	leftObj, ok := pipeExpr.Left.(*ObjectLiteral)
	if !ok {
		t.Fatalf("pipeExpr.Left is not ObjectLiteral. got=%T", pipeExpr.Left)
	}
	if len(leftObj.Pairs) != 2 {
		t.Errorf("leftObj.Pairs length not 2. got=%d", len(leftObj.Pairs))
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeWithFunctionCall(t *testing.T) {
	input := `getData() | Agent`

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

	// Check left side (call expression)
	leftCall, ok := pipeExpr.Left.(*CallExpression)
	if !ok {
		t.Fatalf("pipeExpr.Left is not CallExpression. got=%T", pipeExpr.Left)
	}

	funcIdent, ok := leftCall.Function.(*Identifier)
	if !ok {
		t.Fatalf("leftCall.Function is not Identifier. got=%T", leftCall.Function)
	}
	if funcIdent.Value != "getData" {
		t.Errorf("funcIdent.Value not 'getData'. got=%s", funcIdent.Value)
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeWithMemberCallExpression(t *testing.T) {
	input := `filesystem.read_file("data.json") | Agent`

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

	// Check left side (call expression with member expression)
	leftCall, ok := pipeExpr.Left.(*CallExpression)
	if !ok {
		t.Fatalf("pipeExpr.Left is not CallExpression. got=%T", pipeExpr.Left)
	}

	memberExpr, ok := leftCall.Function.(*MemberExpression)
	if !ok {
		t.Fatalf("leftCall.Function is not MemberExpression. got=%T", leftCall.Function)
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

	if len(leftCall.Arguments) != 1 {
		t.Fatalf("leftCall.Arguments length not 1. got=%d", len(leftCall.Arguments))
	}

	// Check right side (identifier)
	rightIdent, ok := pipeExpr.Right.(*Identifier)
	if !ok {
		t.Fatalf("pipeExpr.Right is not Identifier. got=%T", pipeExpr.Right)
	}
	if rightIdent.Value != "Agent" {
		t.Errorf("rightIdent.Value not 'Agent'. got=%s", rightIdent.Value)
	}
}

func TestPipeWithBinaryExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"addition", `a + b | Agent`, `((a + b) | Agent)`},
		{"subtraction", `a - b | Agent`, `((a - b) | Agent)`},
		{"multiplication", `a * b | Agent`, `((a * b) | Agent)`},
		{"division", `a / b | Agent`, `((a / b) | Agent)`},
		{"modulo", `a % b | Agent`, `((a % b) | Agent)`},
		{"comparison", `a > b | Agent`, `((a > b) | Agent)`},
		{"equality", `a == b | Agent`, `((a == b) | Agent)`},
		{"logical and", `a && b | Agent`, `((a && b) | Agent)`},
		{"logical or", `a || b | Agent`, `((a || b) | Agent)`},
		{"null coalescing", `a ?? b | Agent`, `((a ?? b) | Agent)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			actual := program.String()
			if actual != tt.expected {
				t.Errorf("expected=%q, got=%q", tt.expected, actual)
			}
		})
	}
}

func TestPipeWithUnaryExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"negation", `!active | Agent`, `((!active) | Agent)`},
		{"minus", `-count | Agent`, `((-count) | Agent)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			actual := program.String()
			if actual != tt.expected {
				t.Errorf("expected=%q, got=%q", tt.expected, actual)
			}
		})
	}
}

func TestPipeWithGroupedExpression(t *testing.T) {
	input := `(a + b) | Agent`

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

	// Check left side (binary expression - parentheses are removed during parsing)
	leftBin, ok := pipeExpr.Left.(*BinaryExpression)
	if !ok {
		t.Fatalf("pipeExpr.Left is not BinaryExpression. got=%T", pipeExpr.Left)
	}
	if leftBin.Operator != "+" {
		t.Errorf("leftBin.Operator not '+'. got=%s", leftBin.Operator)
	}
}

func TestRightSideOfPipeWithDifferentExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"identifier", `"hello" | Agent`},
		{"member expression", `"hello" | processor.analyze`},
		{"call expression", `"hello" | process()`},
		{"member call", `"hello" | processor.analyze()`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
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

			// Just verify it's a pipe expression with a left and right side
			if pipeExpr.Left == nil {
				t.Error("pipeExpr.Left is nil")
			}
			if pipeExpr.Right == nil {
				t.Error("pipeExpr.Right is nil")
			}
		})
	}
}

func TestComplexChainedPipeWithMixedOperands(t *testing.T) {
	input := `getData() | "Process: ${result}" | Agent1 | formatOutput() | Agent2`

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

	// Should parse as: ((((getData() | "Process: ${result}") | Agent1) | formatOutput()) | Agent2)
	pipe4, ok := stmt.Expression.(*PipeExpression)
	if !ok {
		t.Fatalf("exp is not PipeExpression. got=%T", stmt.Expression)
	}

	// Verify rightmost is Agent2
	if rightIdent, ok := pipe4.Right.(*Identifier); !ok || rightIdent.Value != "Agent2" {
		t.Errorf("Expected Agent2 on right, got=%v", pipe4.Right)
	}

	// Verify it's properly chained
	pipe3, ok := pipe4.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("pipe4.Left is not PipeExpression. got=%T", pipe4.Left)
	}

	// Verify formatOutput() call
	if _, ok := pipe3.Right.(*CallExpression); !ok {
		t.Errorf("Expected CallExpression, got=%T", pipe3.Right)
	}

	// Continue verifying chain
	pipe2, ok := pipe3.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("pipe3.Left is not PipeExpression. got=%T", pipe3.Left)
	}

	// Verify Agent1
	if rightIdent, ok := pipe2.Right.(*Identifier); !ok || rightIdent.Value != "Agent1" {
		t.Errorf("Expected Agent1, got=%v", pipe2.Right)
	}

	// Verify first pipe
	pipe1, ok := pipe2.Left.(*PipeExpression)
	if !ok {
		t.Fatalf("pipe2.Left is not PipeExpression. got=%T", pipe2.Left)
	}

	// Verify string literal
	if _, ok := pipe1.Right.(*StringLiteral); !ok {
		t.Errorf("Expected StringLiteral, got=%T", pipe1.Right)
	}

	// Verify getData() call
	if _, ok := pipe1.Left.(*CallExpression); !ok {
		t.Errorf("Expected CallExpression, got=%T", pipe1.Left)
	}
}
