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
