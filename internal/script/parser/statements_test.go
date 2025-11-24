package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestIfStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, Statement)
	}{
		{
			name:    "simple if statement",
			input:   `if (x > 5) { y = 10 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				// Check condition
				if ifStmt.Condition == nil {
					t.Fatal("Condition is nil")
				}
				binExpr, ok := ifStmt.Condition.(*BinaryExpression)
				if !ok {
					t.Fatalf("Condition is not *BinaryExpression. got=%T", ifStmt.Condition)
				}
				if binExpr.Operator != ">" {
					t.Errorf("Operator is not '>'. got=%s", binExpr.Operator)
				}

				// Check consequence
				if ifStmt.Consequence == nil {
					t.Fatal("Consequence is nil")
				}
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("Consequence should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				// Check that alternative is nil
				if ifStmt.Alternative != nil {
					t.Errorf("Alternative should be nil. got=%T", ifStmt.Alternative)
				}
			},
		},
		{
			name:    "if-else statement",
			input:   `if (x > 5) { y = 10 } else { y = 20 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				// Check condition
				if ifStmt.Condition == nil {
					t.Fatal("Condition is nil")
				}

				// Check consequence
				if ifStmt.Consequence == nil {
					t.Fatal("Consequence is nil")
				}
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("Consequence should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				// Check alternative
				if ifStmt.Alternative == nil {
					t.Fatal("Alternative is nil")
				}
				altBlock, ok := ifStmt.Alternative.(*BlockStatement)
				if !ok {
					t.Fatalf("Alternative is not *BlockStatement. got=%T", ifStmt.Alternative)
				}
				if len(altBlock.Statements) != 1 {
					t.Fatalf("Alternative should have 1 statement. got=%d", len(altBlock.Statements))
				}
			},
		},
		{
			name:    "if-else if-else statement",
			input:   `if (x > 10) { y = 1 } else if (x > 5) { y = 2 } else { y = 3 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				// Check consequence
				if ifStmt.Consequence == nil {
					t.Fatal("Consequence is nil")
				}

				// Check alternative (should be another IfStatement)
				if ifStmt.Alternative == nil {
					t.Fatal("Alternative is nil")
				}
				elseIfStmt, ok := ifStmt.Alternative.(*IfStatement)
				if !ok {
					t.Fatalf("Alternative is not *IfStatement. got=%T", ifStmt.Alternative)
				}

				// Check else-if consequence
				if elseIfStmt.Consequence == nil {
					t.Fatal("ElseIf Consequence is nil")
				}

				// Check else-if alternative (should be a BlockStatement)
				if elseIfStmt.Alternative == nil {
					t.Fatal("ElseIf Alternative is nil")
				}
				elseBlock, ok := elseIfStmt.Alternative.(*BlockStatement)
				if !ok {
					t.Fatalf("ElseIf Alternative is not *BlockStatement. got=%T", elseIfStmt.Alternative)
				}
				if len(elseBlock.Statements) != 1 {
					t.Fatalf("Else block should have 1 statement. got=%d", len(elseBlock.Statements))
				}
			},
		},
		{
			name:    "nested if statements",
			input:   `if (x > 5) { if (y > 3) { z = 1 } }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				// Check consequence
				if ifStmt.Consequence == nil {
					t.Fatal("Consequence is nil")
				}
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("Consequence should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				// Check that the nested statement is also an IfStatement
				nestedIf, ok := ifStmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("Nested statement is not *IfStatement. got=%T", ifStmt.Consequence.Statements[0])
				}
				if nestedIf.Consequence == nil {
					t.Fatal("Nested Consequence is nil")
				}
			},
		},
		{
			name:    "if with complex condition",
			input:   `if (x > 5 && y < 10) { z = x + y }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				// Check condition is binary expression with &&
				binExpr, ok := ifStmt.Condition.(*BinaryExpression)
				if !ok {
					t.Fatalf("Condition is not *BinaryExpression. got=%T", ifStmt.Condition)
				}
				if binExpr.Operator != "&&" {
					t.Errorf("Operator is not '&&'. got=%s", binExpr.Operator)
				}
			},
		},
		{
			name:    "if with multiple statements in block",
			input:   `if (x > 5) { a = 1 b = 2 c = 3 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				if len(ifStmt.Consequence.Statements) != 3 {
					t.Fatalf("Consequence should have 3 statements. got=%d", len(ifStmt.Consequence.Statements))
				}
			},
		},
		{
			name:    "if with boolean literal",
			input:   `if (true) { x = 1 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				boolLit, ok := ifStmt.Condition.(*BooleanLiteral)
				if !ok {
					t.Fatalf("Condition is not *BooleanLiteral. got=%T", ifStmt.Condition)
				}
				if !boolLit.Value {
					t.Errorf("Boolean value should be true. got=%v", boolLit.Value)
				}
			},
		},
		{
			name:    "if with negation",
			input:   `if (!enabled) { x = 0 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				unaryExpr, ok := ifStmt.Condition.(*UnaryExpression)
				if !ok {
					t.Fatalf("Condition is not *UnaryExpression. got=%T", ifStmt.Condition)
				}
				if unaryExpr.Operator != "!" {
					t.Errorf("Operator is not '!'. got=%s", unaryExpr.Operator)
				}
			},
		},
		{
			name:    "empty if block",
			input:   `if (x > 5) { }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				ifStmt, ok := stmt.(*IfStatement)
				if !ok {
					t.Fatalf("stmt is not *IfStatement. got=%T", stmt)
				}

				if len(ifStmt.Consequence.Statements) != 0 {
					t.Fatalf("Consequence should be empty. got=%d statements", len(ifStmt.Consequence.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				if !tt.wantErr {
					t.Fatalf("parser had errors: %v", p.Errors())
				}
				return
			}

			if tt.wantErr {
				t.Fatal("expected parser error, but got none")
			}

			if len(program.Statements) != 1 {
				t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
			}

			if tt.validate != nil {
				tt.validate(t, program.Statements[0])
			}
		})
	}
}

func TestIfStatementErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing opening parenthesis",
			input: `if x > 5) { y = 10 }`,
		},
		{
			name:  "missing closing parenthesis",
			input: `if (x > 5 { y = 10 }`,
		},
		{
			name:  "missing opening brace",
			input: `if (x > 5) y = 10 }`,
		},
		{
			name:  "missing closing brace",
			input: `if (x > 5) { y = 10`,
		},
		{
			name:  "missing condition",
			input: `if () { y = 10 }`,
		},
		{
			name:  "else without if",
			input: `else { y = 10 }`,
		},
		{
			name:  "else without opening brace",
			input: `if (x > 5) { y = 10 } else y = 20 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			p.ParseProgram()

			if len(p.Errors()) == 0 {
				t.Fatal("expected parser error, but got none")
			}
		})
	}
}

func TestIfStatementString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "simple if",
			input: `if (x > 5) { y = 10 }`,
			expected: `if ((x > 5)) {
  y = 10
}`,
		},
		{
			name:  "if-else",
			input: `if (x > 5) { y = 10 } else { y = 20 }`,
			expected: `if ((x > 5)) {
  y = 10
} else {
  y = 20
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
			}

			result := program.Statements[0].String()
			if result != tt.expected {
				t.Errorf("String() wrong.\nexpected=%q\ngot=%q", tt.expected, result)
			}
		})
	}
}

func TestWhileStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, Statement)
	}{
		{
			name:    "simple while loop",
			input:   `while (x > 0) { x = x - 1 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				// Check condition
				if whileStmt.Condition == nil {
					t.Fatal("Condition is nil")
				}
				binExpr, ok := whileStmt.Condition.(*BinaryExpression)
				if !ok {
					t.Fatalf("Condition is not *BinaryExpression. got=%T", whileStmt.Condition)
				}
				if binExpr.Operator != ">" {
					t.Errorf("Operator is not '>'. got=%s", binExpr.Operator)
				}

				// Check body
				if whileStmt.Body == nil {
					t.Fatal("Body is nil")
				}
				if len(whileStmt.Body.Statements) != 1 {
					t.Fatalf("Body should have 1 statement. got=%d", len(whileStmt.Body.Statements))
				}
			},
		},
		{
			name:    "while with complex condition",
			input:   `while (x > 0 && y < 10) { x = x - 1 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				// Check condition is a binary expression with AND
				binExpr, ok := whileStmt.Condition.(*BinaryExpression)
				if !ok {
					t.Fatalf("Condition is not *BinaryExpression. got=%T", whileStmt.Condition)
				}
				if binExpr.Operator != "&&" {
					t.Errorf("Operator is not '&&'. got=%s", binExpr.Operator)
				}
			},
		},
		{
			name:    "while with multiple statements in body",
			input:   `while (count < 5) { count = count + 1; print(count) }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				if len(whileStmt.Body.Statements) != 2 {
					t.Fatalf("Body should have 2 statements. got=%d", len(whileStmt.Body.Statements))
				}
			},
		},
		{
			name:    "nested while loops",
			input:   `while (i < 3) { while (j < 3) { sum = sum + 1 } }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				// Check that body contains another while statement
				if len(whileStmt.Body.Statements) != 1 {
					t.Fatalf("Body should have 1 statement. got=%d", len(whileStmt.Body.Statements))
				}

				innerWhile, ok := whileStmt.Body.Statements[0].(*WhileStatement)
				if !ok {
					t.Fatalf("Inner statement is not *WhileStatement. got=%T", whileStmt.Body.Statements[0])
				}

				if innerWhile.Body == nil {
					t.Fatal("Inner while body is nil")
				}
			},
		},
		{
			name:    "while with identifier condition",
			input:   `while (running) { count = count + 1 }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				// Check condition is identifier
				if whileStmt.Condition == nil {
					t.Fatal("Condition is nil")
				}
				ident, ok := whileStmt.Condition.(*Identifier)
				if !ok {
					t.Fatalf("Condition is not *Identifier. got=%T", whileStmt.Condition)
				}
				if ident.Value != "running" {
					t.Errorf("Identifier value is not 'running'. got=%s", ident.Value)
				}
			},
		},
		{
			name:    "while missing opening parenthesis",
			input:   `while x > 0) { x = x - 1 }`,
			wantErr: true,
			validate: func(t *testing.T, stmt Statement) {
				// Parser should return nil when it fails to parse
			},
		},
		{
			name:    "while missing closing parenthesis",
			input:   `while (x > 0 { x = x - 1 }`,
			wantErr: true,
			validate: func(t *testing.T, stmt Statement) {
				// Parser should return nil when it fails to parse
			},
		},
		{
			name:    "while missing opening brace",
			input:   `while (x > 0) x = x - 1 }`,
			wantErr: true,
			validate: func(t *testing.T, stmt Statement) {
				// Parser should return nil when it fails to parse
			},
		},
		{
			name:    "while with empty body",
			input:   `while (x > 0) { }`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				whileStmt, ok := stmt.(*WhileStatement)
				if !ok {
					t.Fatalf("stmt is not *WhileStatement. got=%T", stmt)
				}

				if whileStmt.Body == nil {
					t.Fatal("Body is nil")
				}
				if len(whileStmt.Body.Statements) != 0 {
					t.Errorf("Body should be empty. got=%d statements", len(whileStmt.Body.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if tt.wantErr {
				if len(p.Errors()) == 0 {
					t.Fatal("expected parse errors but got none")
				}
			} else {
				checkParserErrors(t, p)

				if len(program.Statements) != 1 {
					t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
				}
			}

			if tt.validate != nil {
				if len(program.Statements) > 0 {
					tt.validate(t, program.Statements[0])
				} else {
					tt.validate(t, nil)
				}
			}
		})
	}
}

func TestWhileStatementString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "simple while",
			input: `while (x > 0) { x = x - 1 }`,
			expected: `while ((x > 0)) {
  x = (x - 1)
}`,
		},
		{
			name:  "while with multiple statements",
			input: `while (count < 5) { count = count + 1; total = total + count }`,
			expected: `while ((count < 5)) {
  count = (count + 1)
  total = (total + count)
}`,
		},
		{
			name:  "while with complex condition",
			input: `while (x > 0 && y < 10) { process() }`,
			expected: `while (((x > 0) && (y < 10))) {
  process()
}`,
		},
		{
			name:  "nested while loops",
			input: `while (i < 3) { while (j < 3) { sum = sum + 1 } }`,
			expected: `while ((i < 3)) {
  while ((j < 3)) {
  sum = (sum + 1)
}
}`,
		},
		{
			name:  "while with empty body",
			input: `while (running) { }`,
			expected: `while (running) {
}`,
		},
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

			got := program.Statements[0].String()
			if got != tt.expected {
				t.Errorf("String() mismatch:\nexpected:\n%s\n\ngot:\n%s", tt.expected, got)
			}
		})
	}
}
