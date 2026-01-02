package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestBreakStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, Statement)
	}{
		{
			name:    "simple break statement",
			input:   `break`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				breakStmt, ok := stmt.(*BreakStatement)
				if !ok {
					t.Fatalf("stmt is not *BreakStatement. got=%T", stmt)
				}
				if breakStmt.TokenLiteral() != "break" {
					t.Errorf("TokenLiteral is not 'break'. got=%s", breakStmt.TokenLiteral())
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
					t.Fatal("expected parser errors but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
			}

			tt.validate(t, program.Statements[0])
		})
	}
}

func TestContinueStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, Statement)
	}{
		{
			name:    "simple continue statement",
			input:   `continue`,
			wantErr: false,
			validate: func(t *testing.T, stmt Statement) {
				continueStmt, ok := stmt.(*ContinueStatement)
				if !ok {
					t.Fatalf("stmt is not *ContinueStatement. got=%T", stmt)
				}
				if continueStmt.TokenLiteral() != "continue" {
					t.Errorf("TokenLiteral is not 'continue'. got=%s", continueStmt.TokenLiteral())
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
					t.Fatal("expected parser errors but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
			}

			tt.validate(t, program.Statements[0])
		})
	}
}

func TestBreakInLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *Program)
	}{
		{
			name: "break in while loop",
			input: `while (x > 0) {
				if (x == 5) {
					break
				}
				x = x - 1
			}`,
			wantErr: false,
			validate: func(t *testing.T, program *Program) {
				if len(program.Statements) != 1 {
					t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
				}

				whileStmt, ok := program.Statements[0].(*WhileStatement)
				if !ok {
					t.Fatalf("statement is not *WhileStatement. got=%T", program.Statements[0])
				}

				if len(whileStmt.Body.Statements) != 2 {
					t.Fatalf("while body should have 2 statements. got=%d", len(whileStmt.Body.Statements))
				}

				// First statement should be an if statement
				ifStmt, ok := whileStmt.Body.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("first statement is not *IfStatement. got=%T", whileStmt.Body.Statements[0])
				}

				// If body should contain break
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("if body should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				breakStmt, ok := ifStmt.Consequence.Statements[0].(*BreakStatement)
				if !ok {
					t.Fatalf("statement in if body is not *BreakStatement. got=%T", ifStmt.Consequence.Statements[0])
				}

				if breakStmt.TokenLiteral() != "break" {
					t.Errorf("break token literal wrong. got=%s", breakStmt.TokenLiteral())
				}
			},
		},
		{
			name: "break in for-of loop",
			input: `for (item of items) {
				if (item == "stop") {
					break
				}
				print(item)
			}`,
			wantErr: false,
			validate: func(t *testing.T, program *Program) {
				if len(program.Statements) != 1 {
					t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
				}

				forStmt, ok := program.Statements[0].(*ForOfStatement)
				if !ok {
					t.Fatalf("statement is not *ForOfStatement. got=%T", program.Statements[0])
				}

				if len(forStmt.Body.Statements) != 2 {
					t.Fatalf("for body should have 2 statements. got=%d", len(forStmt.Body.Statements))
				}

				// First statement should be an if statement
				ifStmt, ok := forStmt.Body.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("first statement is not *IfStatement. got=%T", forStmt.Body.Statements[0])
				}

				// If body should contain break
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("if body should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				breakStmt, ok := ifStmt.Consequence.Statements[0].(*BreakStatement)
				if !ok {
					t.Fatalf("statement in if body is not *BreakStatement. got=%T", ifStmt.Consequence.Statements[0])
				}

				if breakStmt.TokenLiteral() != "break" {
					t.Errorf("break token literal wrong. got=%s", breakStmt.TokenLiteral())
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
					t.Fatal("expected parser errors but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			tt.validate(t, program)
		})
	}
}

func TestContinueInLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *Program)
	}{
		{
			name: "continue in while loop",
			input: `while (x > 0) {
				if (x == 5) {
					continue
				}
				x = x - 1
			}`,
			wantErr: false,
			validate: func(t *testing.T, program *Program) {
				if len(program.Statements) != 1 {
					t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
				}

				whileStmt, ok := program.Statements[0].(*WhileStatement)
				if !ok {
					t.Fatalf("statement is not *WhileStatement. got=%T", program.Statements[0])
				}

				if len(whileStmt.Body.Statements) != 2 {
					t.Fatalf("while body should have 2 statements. got=%d", len(whileStmt.Body.Statements))
				}

				// First statement should be an if statement
				ifStmt, ok := whileStmt.Body.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("first statement is not *IfStatement. got=%T", whileStmt.Body.Statements[0])
				}

				// If body should contain continue
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("if body should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				continueStmt, ok := ifStmt.Consequence.Statements[0].(*ContinueStatement)
				if !ok {
					t.Fatalf("statement in if body is not *ContinueStatement. got=%T", ifStmt.Consequence.Statements[0])
				}

				if continueStmt.TokenLiteral() != "continue" {
					t.Errorf("continue token literal wrong. got=%s", continueStmt.TokenLiteral())
				}
			},
		},
		{
			name: "continue in for-of loop",
			input: `for (item of items) {
				if (item == "skip") {
					continue
				}
				print(item)
			}`,
			wantErr: false,
			validate: func(t *testing.T, program *Program) {
				if len(program.Statements) != 1 {
					t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
				}

				forStmt, ok := program.Statements[0].(*ForOfStatement)
				if !ok {
					t.Fatalf("statement is not *ForOfStatement. got=%T", program.Statements[0])
				}

				if len(forStmt.Body.Statements) != 2 {
					t.Fatalf("for body should have 2 statements. got=%d", len(forStmt.Body.Statements))
				}

				// First statement should be an if statement
				ifStmt, ok := forStmt.Body.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("first statement is not *IfStatement. got=%T", forStmt.Body.Statements[0])
				}

				// If body should contain continue
				if len(ifStmt.Consequence.Statements) != 1 {
					t.Fatalf("if body should have 1 statement. got=%d", len(ifStmt.Consequence.Statements))
				}

				continueStmt, ok := ifStmt.Consequence.Statements[0].(*ContinueStatement)
				if !ok {
					t.Fatalf("statement in if body is not *ContinueStatement. got=%T", ifStmt.Consequence.Statements[0])
				}

				if continueStmt.TokenLiteral() != "continue" {
					t.Errorf("continue token literal wrong. got=%s", continueStmt.TokenLiteral())
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
					t.Fatal("expected parser errors but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			tt.validate(t, program)
		})
	}
}

func TestBreakContinueString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "break statement",
			input:    `break`,
			expected: `break`,
		},
		{
			name:     "continue statement",
			input:    `continue`,
			expected: `continue`,
		},
		{
			name: "break in loop",
			input: `for (item of items) {
				break
			}`,
			expected: `for (item of items) {
  break
}`,
		},
		{
			name: "continue in loop",
			input: `while (x > 0) {
				continue
			}`,
			expected: `while ((x > 0)) {
  continue
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

func TestNestedLoopsWithBreakContinue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *Program)
	}{
		{
			name: "nested loops with break and continue",
			input: `for (i of outer) {
				for (j of inner) {
					if (j == "skip") {
						continue
					}
					if (j == "stop") {
						break
					}
					print(j)
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, program *Program) {
				if len(program.Statements) != 1 {
					t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
				}

				outerFor, ok := program.Statements[0].(*ForOfStatement)
				if !ok {
					t.Fatalf("statement is not *ForOfStatement. got=%T", program.Statements[0])
				}

				if len(outerFor.Body.Statements) != 1 {
					t.Fatalf("outer for body should have 1 statement. got=%d", len(outerFor.Body.Statements))
				}

				innerFor, ok := outerFor.Body.Statements[0].(*ForOfStatement)
				if !ok {
					t.Fatalf("inner statement is not *ForOfStatement. got=%T", outerFor.Body.Statements[0])
				}

				if len(innerFor.Body.Statements) != 3 {
					t.Fatalf("inner for body should have 3 statements. got=%d", len(innerFor.Body.Statements))
				}

				// First if should have continue
				ifStmt1, ok := innerFor.Body.Statements[0].(*IfStatement)
				if !ok {
					t.Fatalf("first statement is not *IfStatement. got=%T", innerFor.Body.Statements[0])
				}

				_, ok = ifStmt1.Consequence.Statements[0].(*ContinueStatement)
				if !ok {
					t.Fatalf("first if should contain continue. got=%T", ifStmt1.Consequence.Statements[0])
				}

				// Second if should have break
				ifStmt2, ok := innerFor.Body.Statements[1].(*IfStatement)
				if !ok {
					t.Fatalf("second statement is not *IfStatement. got=%T", innerFor.Body.Statements[1])
				}

				_, ok = ifStmt2.Consequence.Statements[0].(*BreakStatement)
				if !ok {
					t.Fatalf("second if should contain break. got=%T", ifStmt2.Consequence.Statements[0])
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
					t.Fatal("expected parser errors but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser had errors: %v", p.Errors())
			}

			tt.validate(t, program)
		})
	}
}

func TestMultipleBreakContinueInSameLoop(t *testing.T) {
	input := `for (item of items) {
		if (item == "skip") {
			continue
		}
		if (item == "stop") {
			break
		}
		print(item)
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser had errors: %v", p.Errors())
	}

	if len(program.Statements) != 1 {
		t.Fatalf("program should have 1 statement. got=%d", len(program.Statements))
	}

	forStmt, ok := program.Statements[0].(*ForOfStatement)
	if !ok {
		t.Fatalf("statement is not *ForOfStatement. got=%T", program.Statements[0])
	}

	if len(forStmt.Body.Statements) != 3 {
		t.Fatalf("for body should have 3 statements. got=%d", len(forStmt.Body.Statements))
	}

	// First if should have continue
	ifStmt1, ok := forStmt.Body.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("first statement is not *IfStatement. got=%T", forStmt.Body.Statements[0])
	}

	continueStmt, ok := ifStmt1.Consequence.Statements[0].(*ContinueStatement)
	if !ok {
		t.Fatalf("first if should contain continue. got=%T", ifStmt1.Consequence.Statements[0])
	}

	if continueStmt.String() != "continue" {
		t.Errorf("continue string representation wrong. got=%s", continueStmt.String())
	}

	// Second if should have break
	ifStmt2, ok := forStmt.Body.Statements[1].(*IfStatement)
	if !ok {
		t.Fatalf("second statement is not *IfStatement. got=%T", forStmt.Body.Statements[1])
	}

	breakStmt, ok := ifStmt2.Consequence.Statements[0].(*BreakStatement)
	if !ok {
		t.Fatalf("second if should contain break. got=%T", ifStmt2.Consequence.Statements[0])
	}

	if breakStmt.String() != "break" {
		t.Errorf("break string representation wrong. got=%s", breakStmt.String())
	}
}
